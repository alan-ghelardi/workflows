package variables

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	workflowsv1alpha1 "github.com/nubank/workflows/pkg/apis/workflows/v1alpha1"
	"github.com/nubank/workflows/pkg/testutils"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestExpand(t *testing.T) {
	workflow := &workflowsv1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name: "hello-world",
		},
		Spec: workflowsv1alpha1.WorkflowSpec{
			Repository: &workflowsv1alpha1.Repository{
				Owner: "john-doe",
				Name:  "my-repo",
			},
		},
	}

	event, err := testutils.ReadEvent("push-event.json")
	if err != nil {
		t.Fatalf("Error reading event from testdata: %v", err)
	}

	replacements := MakeReplacements(workflow, event)

	tests := []struct {
		expr   string
		result string
	}{
		{"$(workflow.name)", "hello-world"},
		{"--repo $(workflow.repo.owner)/$(workflow.repo.name)", "--repo john-doe/my-repo"},
		{"--revision=$(workflow.head-commit)", "--revision=833568e"},
		{"$(event {.forced})", "false"},
		{"Hello $(event{.sender.login}), thank you for the commit $(event{.head_commit.id})", "Hello john-doe, thank you for the commit 833568e"},
		{"This is an invalid variable $(workflow.invalid-key)", "This is an invalid variable $(workflow.invalid-key)"},
		{"$(workspaces.project.path)", "$(workspaces.project.path)"},
	}

	for _, test := range tests {
		gotResult := Expand(test.expr, replacements)

		if diff := cmp.Diff(test.result, gotResult); diff != "" {
			t.Errorf("Mismatch (-want +got): %s\n", diff)
		}
	}
}
