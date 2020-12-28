package pipelinerun

import (
	"reflect"
	"testing"
	"time"

	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"

	workflowsv1alpha1 "github.com/nubank/workflows/pkg/apis/workflows/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func newBasicWorkflow() *workflowsv1alpha1.Workflow {
	return &workflowsv1alpha1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "test-1",
			Namespace: "dev"},
		Spec: workflowsv1alpha1.WorkflowSpec{Description: "FIXME",
			Tasks: map[string]*workflowsv1alpha1.Task{"build": &workflowsv1alpha1.Task{Use: "golang-builder",
				ServiceAccount: "sa-1",
			},

				"test": &workflowsv1alpha1.Task{Use: "golang-testing",
					ServiceAccount: "sa-2",
					PodTemplate:    &pipelinev1beta1.PodTemplate{NodeSelector: map[string]string{"x": "y"}},
					Retries:        2,
					Timeout:        &metav1.Duration{Duration: 1 * time.Hour},
				},
			},
		},
	}
}

func TestPipelineRunGenerateName(t *testing.T) {
	workflow := newBasicWorkflow()
	pipelineRun := From(workflow).Build()

	want := "test-1-run"
	got := pipelineRun.ObjectMeta.GenerateName
	if want != got {
		t.Errorf("Want %s, got %s", want, got)
	}
}

func TestPipelineRunNamespace(t *testing.T) {
	workflow := newBasicWorkflow()
	pipelineRun := From(workflow).Build()

	want := "dev"
	got := pipelineRun.ObjectMeta.Namespace
	if want != got {
		t.Errorf("Want %s, got %s", want, got)
	}
}

func TestPipelineDescription(t *testing.T) {
	workflow := newBasicWorkflow()
	pipelineRun := From(workflow).Build()

	want := "FIXME"
	got := pipelineRun.Spec.PipelineSpec.Description
	if want != got {
		t.Errorf("Want %s, got %s", want, got)
	}
}

func TestPipelineTasks(t *testing.T) {
	workflow := newBasicWorkflow()
	pipelineRun := From(workflow).Build()

	tests := []struct {
		name    string
		retries int
		timeout *metav1.Duration
	}{{"build", 0, nil},
		{"test", 2, &metav1.Duration{Duration: 1 * time.Hour}},
	}

	for _, test := range tests {
		var task *pipelinev1beta1.PipelineTask
		for _, x := range pipelineRun.Spec.PipelineSpec.Tasks {
			if test.name == x.Name {
				task = &x
				break
			}
		}

		if task == nil {
			t.Errorf("No such PipelineTask %s in the graph", test.name)
			t.FailNow()
		}

		if test.retries != task.Retries {
			t.Errorf("Error at PipelineTask %s: want retries %d, got %d", test.name, test.retries, task.Retries)
		}

		if !reflect.DeepEqual(test.timeout, task.Timeout) {
			t.Errorf("Error at PipelineTask %s: want timeout %+v, got %+v", test.name, test.timeout, task.Timeout)
		}
	}
}

func TestTaskRunSpecs(t *testing.T) {
	workflow := newBasicWorkflow()
	pipelineRun := From(workflow).Build()

	tests := []struct {
		name           string
		serviceAccount string
		podTemplate    *pipelinev1beta1.PodTemplate
	}{{"build", "sa-1", nil},
		{"test", "sa-2", &pipelinev1beta1.PodTemplate{NodeSelector: map[string]string{"x": "y"}}},
	}

	for _, test := range tests {
		var taskRunSpec *pipelinev1beta1.PipelineTaskRunSpec

		for _, x := range pipelineRun.Spec.TaskRunSpecs {
			if test.name == x.PipelineTaskName {
				taskRunSpec = &x
				break
			}
		}

		if taskRunSpec == nil {
			t.Errorf("No such TaskRunSpec %s", test.name)
			t.FailNow()
		}

		if test.serviceAccount != taskRunSpec.TaskServiceAccountName {
			t.Errorf("Error at TaskRunSpec %s: want service account %s, got %s", test.name, test.serviceAccount, taskRunSpec.TaskServiceAccountName)
		}

		if !reflect.DeepEqual(test.podTemplate, taskRunSpec.TaskPodTemplate) {
			t.Errorf("Error at TaskRunSpec %s: want pod template %+v, got %+v", test.name, test.podTemplate, taskRunSpec.TaskPodTemplate)
		}
	}
}
