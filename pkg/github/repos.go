package github

import (
	"context"
	"fmt"
	"log"

	"github.com/google/go-github/v33/github"
	workflowsv1alpha1 "github.com/nubank/workflows/pkg/apis/workflows/v1alpha1"
)

type RepoReconciler interface {
	ReconcileRepos(ctx context.Context, workflow *workflowsv1alpha1.Workflow) error
}

type defaultRepoReconciler struct {
	service repositoryService
}

// ReconcileRepos implements RepoReconciler.ReconcileRepos.
func (r *defaultRepoReconciler) ReconcileRepos(ctx context.Context, workflow *workflowsv1alpha1.Workflow) error {
	if err := r.setRepoInfo(ctx, workflow.Spec.Repository); err != nil {
		return err
	}

	for i := range workflow.Spec.AdditionalRepositories {
		if err := r.setRepoInfo(ctx, &workflow.Spec.AdditionalRepositories[i]); err != nil {
			return err
		}
	}

	return nil
}

// setRepoInfo takes a workflowv1alpha1.Repository object and sets the
// DefaultBranch and Private attributes according to the corresponding Github
// repository.
func (r *defaultRepoReconciler) setRepoInfo(ctx context.Context, repo *workflowsv1alpha1.Repository) error {
	actualRepo, _, err := r.service.Get(ctx, repo.Owner, repo.Name)
	if err != nil {
		return fmt.Errorf("Error fetching Github repository %s: %w", repo, err)
	}

	repo.DefaultBranch = *actualRepo.DefaultBranch
	repo.Private = *actualRepo.Private

	return nil
}

// repoReconcilerKey is used to store repoReconciler objects into context.Context.
type repoReconcilerKey struct {
}

// WithRepoReconciler returns a copy of the supplied context with a new RepoReconciler object added.
func WithRepoReconciler(ctx context.Context, client *github.Client) context.Context {
	return context.WithValue(ctx, repoReconcilerKey{}, &defaultRepoReconciler{service: client.Repositories})
}

// GetRepoReconcilerOrDie returns a RepoReconciler instance from the supplied
// context or dies by calling log.fatal if the context doesn't contain a
// RepoReconciler object.
func GetRepoReconcilerOrDie(ctx context.Context) RepoReconciler {
	if repoReconciler, ok := ctx.Value(repoReconcilerKey{}).(RepoReconciler); ok {
		return repoReconciler
	}
	log.Fatal("Unable to get a valid RepoReconciler instance from context")
	return nil
}
