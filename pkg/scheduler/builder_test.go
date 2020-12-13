package scheduler

import (
	"testing"

	workflowsv1alpha1 "github.com/nubank/workflows/pkg/apis/workflows/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var workflow = &workflowsv1alpha1.Workflow{
	ObjectMeta: metav1.ObjectMeta{Name: "fake-library",
		Namespace: "dev"},
	Spec: workflowsv1alpha1.WorkflowSpec{Description: "FIXME",
		Tasks: []workflowsv1alpha1.Task{workflowsv1alpha1.Task{Name: "build",
			TaskRef:            "golang-builder",
			ServiceAccountName: "sa1",
		},
		},
	},
}

var event = map[string]interface{}{}

var pipelineRun = buildPipelineRun(workflow, event)

func TestPipelineRunGenerateName(t *testing.T) {
	want := "fake-library-run"
	got := pipelineRun.ObjectMeta.GenerateName
	if want != got {
		t.Errorf("Want %s, got %s", want, got)
	}
}

func TestPipelineRunNamespace(t *testing.T) {
	want := "dev"
	got := pipelineRun.ObjectMeta.Namespace
	if want != got {
		t.Errorf("Want %s, got %s", want, got)
	}
}

func TestPipelineDescription(t *testing.T) {
	want := "FIXME"
	got := pipelineRun.Spec.PipelineSpec.Description
	if want != got {
		t.Errorf("Want %s, got %s", want, got)
	}
}

func TestPipelineTasks(t *testing.T) {
	tests := []struct {
		wantedName    string
		wantedRetries int
	}{{wantedName: "build",
		wantedRetries: 0,
	},
	}

	for i, test := range tests {
		task := pipelineRun.Spec.PipelineSpec.Tasks[i]

		if test.wantedName != task.Name {
			t.Errorf("Error at task #%d: want name %s, got %s", i, test.wantedName, task.Name)
		}

		if test.wantedRetries != task.Retries {
			t.Errorf("Error at task #%d: want retries %d, got %d", i, test.wantedRetries, task.Retries)
		}

	}
}
