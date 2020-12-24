package scheduling

import (
	"reflect"
	"testing"

	"github.com/google/go-github/v33/github"

	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"

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

var event = &github.Event{}

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

func TestTaskRunSpecs(t *testing.T) {
	tests := []struct {
		wantedName           string
		wantedServiceAccount string
		wantedPodTemplate    *pipelinev1beta1.PodTemplate
	}{{wantedName: "build",
		wantedServiceAccount: "sa1"}}

	for i, test := range tests {
		taskRunSpec := pipelineRun.Spec.TaskRunSpecs[i]

		if test.wantedName != taskRunSpec.PipelineTaskName {
			t.Errorf("Error at task run spec #%d: want pipeline task name %s, got %s", i, test.wantedName, taskRunSpec.PipelineTaskName)
		}

		if test.wantedServiceAccount != taskRunSpec.TaskServiceAccountName {
			t.Errorf("Error at task run spec #%d: want service account %s, got %s", i, test.wantedServiceAccount, taskRunSpec.TaskServiceAccountName)
		}

		if !reflect.DeepEqual(test.wantedPodTemplate, taskRunSpec.TaskPodTemplate) {
			t.Errorf("Error at task run spec #%d: want pod template %+v, got %+v", i, test.wantedPodTemplate, taskRunSpec.TaskPodTemplate)
		}
	}
}
