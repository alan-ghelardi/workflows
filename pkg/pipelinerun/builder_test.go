package pipelinerun

import (
	"reflect"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"k8s.io/apimachinery/pkg/api/resource"

	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"

	workflowsv1alpha1 "github.com/nubank/workflows/pkg/apis/workflows/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func newWorkflowWithExistingTasks() *workflowsv1alpha1.Workflow {
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

func newWorkflowWithEmbeddedTasks() *workflowsv1alpha1.Workflow {
	return &workflowsv1alpha1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "test-2",
			Namespace: "dev"},
		Spec: workflowsv1alpha1.WorkflowSpec{Tasks: map[string]*workflowsv1alpha1.Task{"lint": &workflowsv1alpha1.Task{Env: map[string]string{"ENV_VAR_1": "x"},
			Steps: []workflowsv1alpha1.EmbeddedStep{{Name: "golangci-lint",
				Image: "golang",
				Run:   "golangci-lint run",
			},
			},
		},

			"test": &workflowsv1alpha1.Task{Resources: basicContainerResources(),
				Steps: []workflowsv1alpha1.EmbeddedStep{{Image: "golang",
					Env: map[string]string{"ENV_VAR_2": "y"},
					Run: "go test ./...",
				},
				},
			},
		},
		},
	}
}

func basicContainerResources() corev1.ResourceList {
	cpu, _ := resource.ParseQuantity("1m")
	memory, _ := resource.ParseQuantity("2Gi")
	return corev1.ResourceList{corev1.ResourceCPU: cpu,
		corev1.ResourceMemory: memory,
	}
}

func getPipelineTaskOrFail(t *testing.T, pipelineRun *pipelinev1beta1.PipelineRun, taskName string) pipelinev1beta1.PipelineTask {
	for _, x := range pipelineRun.Spec.PipelineSpec.Tasks {
		if taskName == x.Name {
			return x
		}
	}

	t.Errorf("No such pipeline task `%s` in the graph", taskName)
	t.FailNow()

	return pipelinev1beta1.PipelineTask{}
}

func comparePipelineTasks(t *testing.T, want, got pipelinev1beta1.PipelineTask) {
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Mismatch (-want +got):\n%s", diff)
	}
}

func TestPipelineRunGenerateName(t *testing.T) {
	workflow := newWorkflowWithExistingTasks()
	pipelineRun := From(workflow).Build()

	want := "test-1-run"
	got := pipelineRun.ObjectMeta.GenerateName
	if want != got {
		t.Errorf("Want %s, got %s", want, got)
	}
}

func TestPipelineRunNamespace(t *testing.T) {
	workflow := newWorkflowWithExistingTasks()
	pipelineRun := From(workflow).Build()

	want := "dev"
	got := pipelineRun.ObjectMeta.Namespace
	if want != got {
		t.Errorf("Want %s, got %s", want, got)
	}
}

func TestPipelineDescription(t *testing.T) {
	workflow := newWorkflowWithExistingTasks()
	pipelineRun := From(workflow).Build()

	want := "FIXME"
	got := pipelineRun.Spec.PipelineSpec.Description
	if want != got {
		t.Errorf("Want %s, got %s", want, got)
	}
}

func TestPipelineTasksWithTaskRefs(t *testing.T) {
	workflow := newWorkflowWithExistingTasks()
	pipelineRun := From(workflow).Build()

	wantBuild := pipelinev1beta1.PipelineTask{Name: "build",
		TaskRef: &pipelinev1beta1.TaskRef{Name: "golang-builder"},
	}

	wantTest := pipelinev1beta1.PipelineTask{Name: "test",
		TaskRef: &pipelinev1beta1.TaskRef{Name: "golang-testing"},
		Retries: 2,
		Timeout: &metav1.Duration{Duration: 1 * time.Hour},
	}

	gotBuild := getPipelineTaskOrFail(t, pipelineRun, "build")

	gotTest := getPipelineTaskOrFail(t, pipelineRun, "test")

	comparePipelineTasks(t, wantBuild, gotBuild)

	comparePipelineTasks(t, wantTest, gotTest)
}

func TestPipelineTasksWithEmbeddedTasks(t *testing.T) {
	workflow := newWorkflowWithEmbeddedTasks()
	pipelineRun := From(workflow).Build()

	wantLint := pipelinev1beta1.PipelineTask{Name: "lint",
		TaskSpec: &pipelinev1beta1.EmbeddedTask{TaskSpec: pipelinev1beta1.TaskSpec{StepTemplate: &corev1.Container{Env: []corev1.EnvVar{{Name: "ENV_VAR_1",
			Value: "x",
		},
		},
		},
			Steps: []pipelinev1beta1.Step{{Container: corev1.Container{Name: "golangci-lint",
				Image:           "golang",
				ImagePullPolicy: "Always",
			},
				Script: `#!/usr/bin/env sh
set -o errexit
set -o nounset
golangci-lint run`,
			},
			},
		},
		},
	}

	wantTest := pipelinev1beta1.PipelineTask{Name: "test",
		TaskSpec: &pipelinev1beta1.EmbeddedTask{TaskSpec: pipelinev1beta1.TaskSpec{StepTemplate: &corev1.Container{Resources: corev1.ResourceRequirements{Limits: basicContainerResources(),
			Requests: basicContainerResources(),
		},
		},
			Steps: []pipelinev1beta1.Step{{Container: corev1.Container{Image: "golang",
				ImagePullPolicy: "Always",
				Env: []corev1.EnvVar{{Name: "ENV_VAR_2",
					Value: "y",
				},
				},
			},
				Script: `#!/usr/bin/env sh
set -o errexit
set -o nounset
go test ./...`,
			},
			},
		},
		},
	}

	gotLint := getPipelineTaskOrFail(t, pipelineRun, "lint")
	gotTest := getPipelineTaskOrFail(t, pipelineRun, "test")

	comparePipelineTasks(t, wantLint, gotLint)
	comparePipelineTasks(t, wantTest, gotTest)
}

func TestTaskRunSpecs(t *testing.T) {
	workflow := newWorkflowWithExistingTasks()
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
