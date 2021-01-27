package variables

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	workflowsv1alpha1 "github.com/nubank/workflows/pkg/apis/workflows/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestExpand(t *testing.T) {
	workflow := &workflowsv1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name: "hello-world",
		},
	}

	event, err := readEvent()
	if err != nil {
		t.Fatalf("Error reading event from testdata: %v", err)
	}

	replacements := MakeReplacements(workflow, event)

	tests := []struct {
		expr   string
		result string
	}{
		{"$(workflow.name)", "hello-world"},
		{"Hello $(event{.sender.login}), thank you for the commit $(event{.head_commit.id})", "Hello john-doe, thank you for the commit 833568e"},
		{"$(workspaces.project.path)", "$(workspaces.project.path)"},
	}

	for _, test := range tests {
		gotResult := Expand(test.expr, replacements)

		if diff := cmp.Diff(test.result, gotResult); diff != "" {
			t.Errorf("Mismatch (-want +got): %s\n", diff)
		}
	}
}
