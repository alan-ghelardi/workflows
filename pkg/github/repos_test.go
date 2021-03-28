package github

import (
	"context"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-github/v33/github"
	workflowsv1alpha1 "github.com/nubank/workflows/pkg/apis/workflows/v1alpha1"
	githubmocks "github.com/nubank/workflows/pkg/github/mocks"
)

func TestReconcileReposSuccessfully(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	repositoryService := githubmocks.NewMockrepositoryService(mockCtrl)
	reconciler := &defaultRepoReconciler{service: repositoryService}

	repo1 := &workflowsv1alpha1.Repository{
		Owner: "john-doe",
		Name:  "my-repo",
	}
	repo2 := workflowsv1alpha1.Repository{
		Owner: "john-doe",
		Name:  "my-other-repo",
	}
	workflow := &workflowsv1alpha1.Workflow{
		Spec: workflowsv1alpha1.WorkflowSpec{
			Repository:             repo1,
			AdditionalRepositories: []workflowsv1alpha1.Repository{repo2},
		},
	}

	ctx := context.Background()

	// Mock setup
	repositoryService.EXPECT().
		Get(ctx, "john-doe", "my-repo").
		Return(&github.Repository{
			DefaultBranch: github.String("main"),
			Private:       github.Bool(true),
		}, nil, nil)

	repositoryService.EXPECT().
		Get(ctx, "john-doe", "my-other-repo").
		Return(&github.Repository{
			DefaultBranch: github.String("default"),
			Private:       github.Bool(false),
		}, nil, nil)

	want := &workflowsv1alpha1.Workflow{
		Spec: workflowsv1alpha1.WorkflowSpec{
			Repository: &workflowsv1alpha1.Repository{
				Owner:         "john-doe",
				Name:          "my-repo",
				DefaultBranch: "main",
				Private:       true,
			},
			AdditionalRepositories: []workflowsv1alpha1.Repository{workflowsv1alpha1.Repository{
				Owner:         "john-doe",
				Name:          "my-other-repo",
				DefaultBranch: "default",
				Private:       false,
			}},
		},
	}

	err := reconciler.ReconcileRepos(ctx, workflow)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if diff := cmp.Diff(want, workflow); diff != "" {
		t.Errorf("Fail to reconcile repos\nMismatch (-want +got): %s", diff)
	}
}

func TestReturnsAnErrorWhenTheRepoReconcileFails(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	repositoryService := githubmocks.NewMockrepositoryService(mockCtrl)
	reconciler := &defaultRepoReconciler{service: repositoryService}

	workflow := &workflowsv1alpha1.Workflow{
		Spec: workflowsv1alpha1.WorkflowSpec{
			Repository: &workflowsv1alpha1.Repository{
				Owner: "john-doe",
				Name:  "my-repo",
			},
		},
	}

	ctx := context.Background()

	// Mock setup
	repositoryService.EXPECT().
		Get(ctx, "john-doe", "my-repo").
		Return(nil, nil, fmt.Errorf("Error"))

	err := reconciler.ReconcileRepos(ctx, workflow)
	if err == nil {
		t.Error("Want an error, but the reconcile succeeded unexpectedly")
	}
}
