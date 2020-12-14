package github

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"reflect"

	"knative.dev/pkg/logging"

	"github.com/google/go-github/v33/github"
	"github.com/nubank/workflows/pkg/apis/workflows/v1alpha1"
	securerandom "github.com/theckman/go-securerandom"
)

// WebhookSyncer keeps Github Webhooks in sync to the desired state declared in
// workflows.
type WebhookSyncer struct {
	githubClient *github.Client
}

// Sync creates or updates a Github Webhook.
func (w *WebhookSyncer) Sync(ctx context.Context, workflow *v1alpha1.Workflow) (*SyncResult, error) {
	var (
		id     *int64
		err    error
		repo   = workflow.Spec.Repository
		hook   *github.Hook
		result = EmptySyncResult()
	)

	logger := logging.FromContext(ctx).With("repository", repo)

	id = workflow.GetWebhookID()
	if id == nil {
		logger.Info("There are no recognized webhooks associated to the workflow. Creating a new one")
		result, err = w.createWebhook(ctx, workflow)
		if err == nil {
			workflow.SetWebhookID(result.Entries[0].ID)
		}
		return result, err
	}

	hook, err = w.getWebhook(ctx, workflow, *id)
	if IsNotFound(err) {
		logger.Infow("Unable to find Webhook for supplied id. It might have been deleted by mistaken. Creating a new one", "webhook-id", *id)
		result, err = w.createWebhook(ctx, workflow)
		if err == nil {
			workflow.SetWebhookID(result.Entries[0].ID)
		}
		return result, err
	}

	// Unexpected error getting the Webhook
	if err != nil {
		return result, err
	}

	if w.changedSinceLastSync(workflow, hook) {
		logger.Infow("Webhook and workflow settings are out of sync. Updating Webhook", "webhook-id", *id)
		result, err = w.updateWebhook(ctx, workflow, *id)
		return result, err
	}

	logger.Infow("Webhook settings are up to date", "webhook-id", *id)

	return result, nil
}

// getWebhook returns the Webhook associated to the supplied workflow.
func (w *WebhookSyncer) getWebhook(ctx context.Context, workflow *v1alpha1.Workflow, id int64) (*github.Hook, error) {
	repo := workflow.Spec.Repository
	hook, response, err := w.githubClient.Repositories.GetHook(ctx,
		repo.Owner,
		repo.Name,
		id)

	if response != nil && response.StatusCode == 404 {
		return nil, &NotFoundError{msg: fmt.Sprintf("Unable to find Webhook #%d. It might be deleted by mistaken directly on Github", id)}
	}

	if err != nil {
		return nil, fmt.Errorf("Unable to get Webhook #%d: %w", id, err)
	}

	return hook, nil
}

// changedSinceLastSync returns true if the Webhook settings have been changed
// since the last sync or false otherwise.
func (w *WebhookSyncer) changedSinceLastSync(workflow *v1alpha1.Workflow, hook *github.Hook) bool {
	return !hook.GetActive() ||
		!reflect.DeepEqual(workflow.Spec.Webhook.Events, hook.Events) ||
		workflow.Spec.Webhook.URL != hook.Config["url"] ||
		hook.Config["content_type"] != "json" ||
		hook.Config["insecure_ssl"] != "0"
}

// createWebhook creates a new Github Webhook.
func (w *WebhookSyncer) createWebhook(ctx context.Context, workflow *v1alpha1.Workflow) (*SyncResult, error) {
	repo := workflow.Spec.Repository

	secret, err := securerandom.Bytes(94)
	if err != nil {
		return nil, fmt.Errorf("unable to generate Webhook secret token for repository %s: %w", repo, err)
	}

	hook, _, err := w.githubClient.Repositories.CreateHook(ctx,
		repo.Owner,
		repo.Name,
		w.newHook(workflow, secret))

	if err != nil {
		return nil, fmt.Errorf("unable to create Github Webhook for repository %s: %w", repo, err)
	}
	logger := logging.FromContext(ctx)
	logger.Infow("Webhook has been successfully created",
		"repository", repo,
		"webhook-id", hook.ID)

	return newSyncResult(*hook.ID, workflow.Spec.Repository, Created, secret), nil
}

// newSyncResult is a helper function to create a SyncResult with a single entry.
func newSyncResult(id int64, repo *v1alpha1.Repository, action ActionType, secret []byte) *SyncResult {
	result := EmptySyncResult()
	return result.Add(SyncResultEntry{ID: id,
		Repository: repo,
		Action:     action,
		Secret:     secret,
	})
}

// newHook returns a new Hook object.
func (w *WebhookSyncer) newHook(workflow *v1alpha1.Workflow, secret []byte) *github.Hook {
	hook := &github.Hook{Active: github.Bool(true),
		Events: workflow.Spec.Webhook.Events,
		Config: map[string]interface{}{
			"url":          workflow.Spec.Webhook.URL,
			"content_type": "json",
			"insecure_ssl": "0",
		},
	}

	if secret != nil {
		hook.Config["secret"] = base64.StdEncoding.EncodeToString(secret)
	}

	return hook
}

// updateWebhook updates an existing Github Webhook.
func (w *WebhookSyncer) updateWebhook(ctx context.Context, workflow *v1alpha1.Workflow, id int64) (*SyncResult, error) {
	repo := workflow.Spec.Repository
	hook, _, err := w.githubClient.Repositories.EditHook(ctx,
		repo.Owner,
		repo.Name,
		id,
		w.newHook(workflow, nil))

	if err != nil {
		return nil, fmt.Errorf("unable to update Github Webhook for repository %s: %w", repo, err)
	}

	logger := logging.FromContext(ctx)
	logger.Infow("Webhook has been successfully updated",
		"repository", repo,
		"webhook-id", id)

	return newSyncResult(*hook.ID, repo, Updated, nil), nil
}

// webhookSyncerKey is used to store WebhookSyncer objects into context.Context.
type webhookSyncerKey struct {
}

// WithWebhookSyncer returns a copy of the supplied context with a new WebhookSyncer object added.
func WithWebhookSyncer(ctx context.Context, client *github.Client) context.Context {
	return context.WithValue(ctx, webhookSyncerKey{}, &WebhookSyncer{githubClient: client})
}

// GetWebhookSyncerOrDie returns a WebhookSyncer instance from the supplied
// context or dies by calling log.fatal if the context doesn't contain a
// WebhookSyncer object.
func GetWebhookSyncerOrDie(ctx context.Context) Syncer {
	if syncer, ok := ctx.Value(webhookSyncerKey{}).(Syncer); ok {
		return syncer
	}
	log.Fatal("Unable to get a valid WebhookSyncer instance from context")
	return nil
}
