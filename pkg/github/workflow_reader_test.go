package github

import (
	"context"
	"os"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-github/v33/github"
	workflowsv1alpha1 "github.com/nubank/workflows/pkg/apis/workflows/v1alpha1"
	githubmocks "github.com/nubank/workflows/pkg/github/mocks"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestReadsTheWorkflowDeclaredInTheRepository(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	contents := githubmocks.NewMockcontentsService(mockCtrl)
	reader := &DefaultWorkflowReader{service: contents}
	var welcome *github.RepositoryContent
	if content, err := os.ReadFile("testdata/welcome.yaml"); err != nil {
		t.Fatal(err)
	} else {
		welcome = &github.RepositoryContent{
			Content: github.String(string(content)),
		}
	}

	ctx := context.Background()

	originalWorkflow := &workflowsv1alpha1.Workflow{
		Spec: workflowsv1alpha1.WorkflowSpec{
			Repository: &workflowsv1alpha1.Repository{
				Owner: "john-doe",
				Name:  "my-repo",
			},
			AdditionalRepositories: []workflowsv1alpha1.Repository{{
				Owner: "john-doe",
				Name:  "my-other-repo",
			},
			},
			Tasks: map[string]*workflowsv1alpha1.Task{
				"welcome": &workflowsv1alpha1.Task{},
			},
		},
	}

	contents.EXPECT().
		GetContents(gomock.Eq(ctx),
			gomock.Eq("john-doe"),
			gomock.Eq("my-repo"),
			gomock.Eq(".tektoncd/workflows/welcome.yaml"),
			gomock.Eq(&github.RepositoryContentGetOptions{Ref: "ref"})).
		Return(welcome, nil, nil, nil)

	want := &workflowsv1alpha1.Workflow{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "workflows.dev/v1alpha1",
			Kind:       "Workflow",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "welcome",
		},
		Spec: workflowsv1alpha1.WorkflowSpec{
			Repository: &workflowsv1alpha1.Repository{
				Owner: "john-doe",
				Name:  "my-repo",
			},
			AdditionalRepositories: []workflowsv1alpha1.Repository{{
				Owner: "john-doe",
				Name:  "my-other-repo",
			},
			},
			Tasks: map[string]*workflowsv1alpha1.Task{
				"welcome": &workflowsv1alpha1.Task{
					Steps: []workflowsv1alpha1.EmbeddedStep{{
						Run: `echo "Welcome!"`,
					},
					},
				},
			},
		},
	}

	got, err := reader.GetWorkflowContent(ctx, originalWorkflow, ".tektoncd/workflows/welcome.yaml", "ref")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Mismatch (- want + got):\n%s", diff)
	}
}
