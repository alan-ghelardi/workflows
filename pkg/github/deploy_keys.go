package github

import (
	"context"
	"fmt"
	"log"

	"knative.dev/pkg/logging"

	"github.com/google/go-github/v33/github"
	"github.com/nubank/workflows/pkg/apis/workflows/v1alpha1"

	"github.com/nubank/workflows/pkg/secret"
)

// DeployKeySyncer keeps Github deploy keys in sync with the desired state
// declared in workflows.
type DeployKeySyncer struct {
	githubClient *github.Client
}

// Sync creates or updates all Github deploy keys associated to the supplied workflow.
func (d *DeployKeySyncer) Sync(ctx context.Context, workflow *v1alpha1.Workflow) ([]secret.KeyPair, error) {
	keyPairs := make([]secret.KeyPair, 0)
	repos := workflow.GetRepositories()

	for _, repo := range repos {
		if keyPair, err := d.syncDeployKey(ctx, workflow, &repo); err != nil {
			return nil, err
		} else if keyPair != nil {
			keyPairs = append(keyPairs, *keyPair)
		}
	}
	return keyPairs, nil
}

// SyncDeployKey creates or updates a Github DeployKey.
func (d *DeployKeySyncer) syncDeployKey(ctx context.Context, workflow *v1alpha1.Workflow, repo *v1alpha1.Repository) (*secret.KeyPair, error) {
	var (
		id      *int64
		err     error
		key     *github.Key
		keyPair *secret.KeyPair
	)

	logger := logging.FromContext(ctx).With("repository", repo)

	id = workflow.GetDeployKeyID(repo)
	if id == nil {
		logger.Info("There are no recognized deploy keys associated to the workflow. Creating a new one")
		key, keyPair, err = d.createDeployKey(ctx, workflow, repo)
		if err == nil {
			workflow.SetDeployKeyID(repo, *key.ID)
		}
		return keyPair, err
	}

	key, err = d.getDeployKey(ctx, repo, *id)
	if IsNotFound(err) {
		logger.Infow("Unable to find a deploy key for the supplied id. It might have been deleted by mistaken. Creating a new one", "deploy-key-id", *id)
		key, keyPair, err = d.createDeployKey(ctx, workflow, repo)
		if err == nil {
			workflow.SetDeployKeyID(repo, *key.ID)
		}
		return keyPair, err
	}

	// Unexpected error getting the deploy key
	if err != nil {
		return keyPair, err
	}

	if d.changedSinceLastSync(repo, key) {
		logger.Infow("Deploy key and workflow settings are out of sync. Rotating deploy key", "deploy-key-id", *id)
		key, keyPair, err = d.updateDeployKey(ctx, workflow, repo, *id)
		if err == nil {
			workflow.SetDeployKeyID(repo, *key.ID)
		}
		return keyPair, err
	}

	logger.Infow("Deploy key settings are up to date", "deploy-key-id", *id)

	return keyPair, nil
}

// getDeployKey returns the deploy key that matches the supplied id.
func (d *DeployKeySyncer) getDeployKey(ctx context.Context, repo *v1alpha1.Repository, id int64) (*github.Key, error) {
	key, response, err := d.githubClient.Repositories.GetKey(ctx,
		repo.Owner,
		repo.Name,
		id)

	if response != nil && response.StatusCode == 404 {
		return nil, &NotFoundError{msg: fmt.Sprintf("Unable to find deploy key #%d. It might be deleted by mistaken directly on Github", id)}
	}

	if err != nil {
		return nil, fmt.Errorf("Unable to get deploy key #%d: %w", id, err)
	}

	return key, nil
}

// changedSinceLastSync returns true if the deploy key settings have been changed
// since the last sync or false otherwise.
func (d *DeployKeySyncer) changedSinceLastSync(repo *v1alpha1.Repository, key *github.Key) bool {
	return repo.IsReadOnlyDeployKey() != key.GetReadOnly()
}

// createDeployKey creates a new Github DeployKey.
func (d *DeployKeySyncer) createDeployKey(ctx context.Context, workflow *v1alpha1.Workflow, repo *v1alpha1.Repository) (*github.Key, *secret.KeyPair, error) {
	var (
		key     *github.Key
		keyPair *secret.KeyPair
		err     error
	)

	keyPair, err = secret.GenerateKeyPair(repo)
	if err != nil {
		return key, keyPair, err
	}

	key, _, err = d.githubClient.Repositories.CreateKey(ctx,
		repo.Owner,
		repo.Name,
		&github.Key{Title: github.String(fmt.Sprintf("%s-ssh-public-key", workflow.GetName())),
			Key:      github.String(string(keyPair.PublicKey)),
			ReadOnly: github.Bool(repo.IsReadOnlyDeployKey()),
		})

	if err != nil {
		return key, keyPair, fmt.Errorf("unable to create Github deploy key for repository %s: %w", repo, err)
	}

	logger := logging.FromContext(ctx)
	logger.Infow("DeployKey has been successfully created",
		"repository", repo,
		"deploy-key-id", key.ID)

	return key, keyPair, err
}

// updateDeployKey updates an existing Github DeployKey.
func (d *DeployKeySyncer) updateDeployKey(ctx context.Context, workflow *v1alpha1.Workflow, repo *v1alpha1.Repository, id int64) (*github.Key, *secret.KeyPair, error) {
	err := d.deleteDeployKey(ctx, repo, id)
	if err != nil {
		return nil, nil, err
	}

	return d.createDeployKey(ctx, workflow, repo)
}

// deleteDeployKey deletes an existing Github DeployKey.
func (d *DeployKeySyncer) deleteDeployKey(ctx context.Context, repo *v1alpha1.Repository, id int64) error {
	logger := logging.FromContext(ctx)

	_, err := d.githubClient.Repositories.DeleteKey(ctx,
		repo.Owner,
		repo.Name,
		id)

	if err != nil {
		return fmt.Errorf("unable to delete Github deploy key for repository %s: %w", repo, err)
	}

	logger.Infow("Github deploy key has been successfully deleted", "repository", repo, "deploy-key-id", id)

	return nil
}

// Delete implements Syncer by deleting all deploy keys associated to the
// workflow in question.
func (d *DeployKeySyncer) Delete(ctx context.Context, workflow *v1alpha1.Workflow) error {
	repos := workflow.GetRepositories()

	for _, repo := range repos {
		id := workflow.GetDeployKeyID(&repo)

		if id == nil {
			return fmt.Errorf("Error deleting deploy key for repository %s: the key's identifier is unknown", repo.String())
		}

		if err := d.deleteDeployKey(ctx, &repo, *id); err != nil {
			return err
		}
	}
	return nil
}

// deployKeySyncerKey is used to store DeployKeySyncer objects into context.Context.
type deployKeySyncerKey struct {
}

// WithDeployKeySyncer returns a copy of the supplied context with a new DeployKeySyncer object added.
func WithDeployKeySyncer(ctx context.Context, client *github.Client) context.Context {
	return context.WithValue(ctx, deployKeySyncerKey{}, &DeployKeySyncer{githubClient: client})
}

// GetDeployKeySyncerOrDie returns a DeployKeySyncer instance from the supplied
// context or dies by calling log.fatal if the context doesn't contain a
// DeployKeySyncer object.
func GetDeployKeySyncerOrDie(ctx context.Context) *DeployKeySyncer {
	if deployKeySyncer, ok := ctx.Value(deployKeySyncerKey{}).(*DeployKeySyncer); ok {
		return deployKeySyncer
	}
	log.Fatal("Unable to get a valid DeployKeySyncer instance from context")
	return nil
}
