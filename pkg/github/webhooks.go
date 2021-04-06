package github

import (
	"context"
	"fmt"
	"log"
	"reflect"

	"knative.dev/pkg/logging"

	"github.com/google/go-github/v33/github"
	"github.com/nubank/workflows/pkg/apis/workflows/v1alpha1"
	"github.com/nubank/workflows/pkg/secrets"
)

// DefaultWebhookReconciler keeps Github Webhooks in sync to the desired state declared in
// workflows.
type WebhookReconciler interface {
	ReconcileHook(ctx context.Context, workflow *v1alpha1.Workflow) (*Webhook, error)
	Delete(ctx context.Context, workflow *v1alpha1.Workflow) error
}

// defaultWebhookReconciler implements WebhookReconciler.
type defaultWebhookReconciler struct {
	service hooksService
}

// Webhook contains information about a created or updated Github Webhook.
type Webhook struct {
	ID     int64
	Secret []byte
}

// ReconcileHook creates or updates a Github Webhook.
func (w *defaultWebhookReconciler) ReconcileHook(ctx context.Context, workflow *v1alpha1.Workflow) (*Webhook, error) {
	var (
		id         *int64
		err        error
		githubHook *github.Hook
		repo       = workflow.Spec.Repository
		webhook    *Webhook
	)

	logger := logging.FromContext(ctx).With("repository", repo)

	id = workflow.GetWebhookID()
	if id == nil {
		logger.Info("There are no recognized webhooks associated to the workflow. Creating a new one")
		webhook, err = w.createWebhook(ctx, workflow)
		if err == nil {
			workflow.SetWebhookID(webhook.ID)
		}
		return webhook, err
	}

	githubHook, err = w.getWebhook(ctx, workflow, *id)
	if IsNotFound(err) {
		logger.Infow("Unable to find Webhook for supplied id. It might have been deleted by mistaken. Creating a new one", "webhook-id", *id)
		webhook, err = w.createWebhook(ctx, workflow)
		if err == nil {
			workflow.SetWebhookID(webhook.ID)
		}
		return webhook, err
	}

	// Unexpected error getting the Webhook
	if err != nil {
		return webhook, err
	}

	if w.changedSinceLastSync(workflow, githubHook) {
		logger.Infow("Webhook and workflow settings are out of sync. Updating Webhook", "webhook-id", *id)
		webhook, err = w.updateWebhook(ctx, workflow, *id)
		return webhook, err
	}

	logger.Infow("Webhook settings are up to date", "webhook-id", *id)

	return webhook, nil
}

// getWebhook returns the Webhook associated to the supplied workflow.
func (w *defaultWebhookReconciler) getWebhook(ctx context.Context, workflow *v1alpha1.Workflow, id int64) (*github.Hook, error) {
	repo := workflow.Spec.Repository
	hook, response, err := w.service.GetHook(ctx,
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
func (w *defaultWebhookReconciler) changedSinceLastSync(workflow *v1alpha1.Workflow, hook *github.Hook) bool {
	return !hook.GetActive() ||
		!reflect.DeepEqual(workflow.Spec.Events, hook.Events) ||
		workflow.GetHooksURL() != hook.Config["url"] ||
		hook.Config["content_type"] != "json" ||
		hook.Config["insecure_ssl"] != "0"
}

// createWebhook creates a new Github Webhook.
func (w *defaultWebhookReconciler) createWebhook(ctx context.Context, workflow *v1alpha1.Workflow) (*Webhook, error) {
	repo := workflow.Spec.Repository

	secretToken := secrets.GenerateRandomToken()

	hook, _, err := w.service.CreateHook(ctx,
		repo.Owner,
		repo.Name,
		w.newHook(workflow, github.String(secretToken)))

	if err != nil {
		return nil, fmt.Errorf("unable to create Github Webhook for repository %s: %w", repo, err)
	}
	logger := logging.FromContext(ctx)
	logger.Infow("Webhook has been successfully created",
		"repository", repo,
		"webhook-id", hook.ID)

	return &Webhook{
		ID:     *hook.ID,
		Secret: []byte(secretToken),
	}, nil
}

// newHook returns a new Hook object.
func (w *defaultWebhookReconciler) newHook(workflow *v1alpha1.Workflow, secret *string) *github.Hook {
	hook := &github.Hook{Active: github.Bool(true),
		Events: workflow.Spec.Events,
		Config: map[string]interface{}{
			"url":          workflow.GetHooksURL(),
			"content_type": "json",
			"insecure_ssl": "0",
		},
	}

	if secret != nil {
		hook.Config["secret"] = *secret
	}

	return hook
}

// updateWebhook updates an existing Github Webhook.
func (w *defaultWebhookReconciler) updateWebhook(ctx context.Context, workflow *v1alpha1.Workflow, id int64) (*Webhook, error) {
	repo := workflow.Spec.Repository
	hook, _, err := w.service.EditHook(ctx,
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

	return &Webhook{ID: *hook.ID}, nil
}

// Delete deletes the Webhook associated to the workflow in question.
func (w *defaultWebhookReconciler) Delete(ctx context.Context, workflow *v1alpha1.Workflow) error {
	var (
		id   *int64
		err  error
		repo = workflow.Spec.Repository
	)

	logger := logging.FromContext(ctx).With("repository", repo)

	id = workflow.GetWebhookID()
	if id == nil {
		return fmt.Errorf("Unable to delete Webhook because its identifier is unknown")
	}

	_, err = w.service.DeleteHook(ctx,
		repo.Owner,
		repo.Name,
		*id)

	if err != nil {
		return fmt.Errorf("Error deleting Github Webhook: %w", err)
	}

	logger.Infow("Webhook has been successfully deleted", "repository", repo, "webhook-id", id)

	return nil
}

// webhookReconcilerKey is used to store WebhookReconciler objects into context.Context.
type webhookReconcilerKey struct {
}

// WithWebhookReconciler returns a copy of the supplied context with a new WebhookReconciler object added.
func WithWebhookReconciler(ctx context.Context, client *github.Client) context.Context {
	return context.WithValue(ctx, webhookReconcilerKey{}, &defaultWebhookReconciler{service: client.Repositories})
}

// GetWebhookReconcilerOrDie returns a WebhookReconciler instance from the supplied
// context or dies by calling log.fatal if the context doesn't contain a
// WebhookReconciler object.
func GetWebhookReconcilerOrDie(ctx context.Context) WebhookReconciler {
	if webhookReconciler, ok := ctx.Value(webhookReconcilerKey{}).(*defaultWebhookReconciler); ok {
		return webhookReconciler
	}
	log.Fatal("Unable to get a valid WebhookReconciler instance from context")
	return nil
}
