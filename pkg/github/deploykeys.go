package github

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"

	"knative.dev/pkg/logging"

	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"

	"github.com/google/go-github/v33/github"
	"github.com/nubank/workflows/pkg/apis/workflows/v1alpha1"

	"golang.org/x/crypto/ssh"
)

// Size of private keys.
const keySize = 4096

// DeployKeySyncer keeps Github deploy keys in sync with the desired state
// declared in workflows.
type DeployKeySyncer struct {
	githubClient *github.Client
}

// Sync creates or updates all Github deploy keys associated to the supplied workflow.
func (d *DeployKeySyncer) Sync(ctx context.Context, workflow *v1alpha1.Workflow) (*SyncResult, error) {
	result := EmptySyncResult()
	repos := append(workflow.Spec.SecondaryRepositories, *workflow.Spec.Repository)

	for _, repo := range repos {
		if entry, err := d.syncDeployKey(ctx, workflow, &repo); err != nil {
			return nil, err
		} else {
			result.Add(entry)
		}
	}
	return result, nil
}

// SyncDeployKey creates or updates a Github DeployKey.
func (d *DeployKeySyncer) syncDeployKey(ctx context.Context, workflow *v1alpha1.Workflow, repo *v1alpha1.Repository) (SyncResultEntry, error) {
	var (
		id    *int64
		err   error
		key   *github.Key
		entry SyncResultEntry
	)

	logger := logging.FromContext(ctx).With("repository", repo)

	id = workflow.GetDeployKeyID(repo)
	if id == nil {
		logger.Info("There are no recognized deploy keys associated to the workflow. Creating a new one")
		entry, err = d.createDeployKey(ctx, workflow, repo)
		if err == nil {
			workflow.SetDeployKeyID(repo, entry.ID)
		}
		return entry, err
	}

	key, err = d.getDeployKey(ctx, repo, *id)
	if IsNotFound(err) {
		logger.Infow("Unable to find a deploy key for the supplied id. It might have been deleted by mistaken. Creating a new one", "deploy-key-id", *id)
		entry, err = d.createDeployKey(ctx, workflow, repo)
		if err == nil {
			workflow.SetDeployKeyID(repo, entry.ID)
		}
		return entry, err
	}

	// Unexpected error getting the deploy key
	if err != nil {
		return entry, err
	}

	if d.changedSinceLastSync(repo, key) {
		logger.Infow("Deploy key and workflow settings are out of sync. Rotating deploy key", "deploy-key-id", *id)
		entry, err = d.updateDeployKey(ctx, workflow, repo, *id)
		return entry, err
	}

	logger.Infow("Deploy key settings are up to date", "deploy-key-id", *id)

	return entry, nil
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
func (d *DeployKeySyncer) createDeployKey(ctx context.Context, workflow *v1alpha1.Workflow, repo *v1alpha1.Repository) (SyncResultEntry, error) {
	var (
		entry SyncResultEntry
		err   error
	)

	privateKey, err := generateRSAPrivateKey()
	if err != nil {
		return entry, err
	}

	publicKey, err := generateRSAPublicKey(privateKey)
	if err != nil {
		return entry, err
	}

	key, _, err := d.githubClient.Repositories.CreateKey(ctx,
		repo.Owner,
		repo.Name,
		&github.Key{Title: github.String(fmt.Sprintf("%s-public-ssh-key", workflow.GetName())),
			Key:      github.String(string(publicKey)),
			ReadOnly: github.Bool(repo.IsReadOnlyDeployKey()),
		})

	if err != nil {
		return entry, fmt.Errorf("unable to create Github deploy key for repository %s: %w", repo, err)
	}

	logger := logging.FromContext(ctx)
	logger.Infow("DeployKey has been successfully created",
		"repository", repo,
		"deploy-key-id", key.ID)

	return SyncResultEntry{ID: *key.ID,
		Repository: repo,
		Action:     Created,
		Secret:     encodePrivateKeyToPEM(privateKey),
	}, nil
}

// updateDeployKey updates an existing Github DeployKey.
func (d *DeployKeySyncer) updateDeployKey(ctx context.Context, workflow *v1alpha1.Workflow, repo *v1alpha1.Repository, id int64) (SyncResultEntry, error) {
	_, err := d.githubClient.Repositories.DeleteKey(ctx,
		repo.Owner,
		repo.Name,
		id)

	if err != nil {
		return SyncResultEntry{}, fmt.Errorf("unable to delete Github deploy key for repository %s: %w", repo, err)
	}

	return d.createDeployKey(ctx, workflow, repo)
}

// generateRSAPrivateKey returns a new RSA private key.
func generateRSAPrivateKey() (*rsa.PrivateKey, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, keySize)
	if err != nil {
		return nil, fmt.Errorf("Error generating private RSA key: %w", err)
	}
	return privateKey, nil
}

// encodePrivateKeyToPEM encodes the supplied RSA private key to PEM format.
func encodePrivateKeyToPEM(privateKey *rsa.PrivateKey) []byte {
	keyContent := x509.MarshalPKCS1PrivateKey(privateKey)
	block := pem.Block{Type: "RSA PRIVATE KEY",
		Bytes: keyContent,
	}
	return pem.EncodeToMemory(&block)
}

// generatePublicKey returns the public key part of the supplied RSA private key.
func generateRSAPublicKey(privateKey *rsa.PrivateKey) ([]byte, error) {
	publicKey, err := ssh.NewPublicKey(privateKey.Public())
	if err != nil {
		return nil, fmt.Errorf("Error generating public RSA key: %w", err)
	}

	return ssh.MarshalAuthorizedKey(publicKey), nil
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
func GetDeployKeySyncerOrDie(ctx context.Context) Syncer {
	if syncer, ok := ctx.Value(deployKeySyncerKey{}).(Syncer); ok {
		return syncer
	}
	log.Fatal("Unable to get a valid DeployKeySyncer instance from context")
	return nil
}
