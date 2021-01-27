package pipelinerun

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"k8s.io/apimachinery/pkg/api/resource"

	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"

	"github.com/nubank/workflows/pkg/github"
	"github.com/nubank/workflows/pkg/testutils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func getPipelineTaskOrFail(t *testing.T, pipelineRun *pipelinev1beta1.PipelineRun, taskName string) (pipelinev1beta1.PipelineTask, error) {
	for _, x := range pipelineRun.Spec.PipelineSpec.Tasks {
		if taskName == x.Name {
			return x, nil
		}
	}

	return pipelinev1beta1.PipelineTask{}, fmt.Errorf("No such pipeline task `%s` in the graph", taskName)
}

func TestPipelineRunGenerateName(t *testing.T) {
	workflow, err := testutils.ReadWorkflow("referencing-existing-tasks.yaml")
	if err != nil {
		t.Fatal(err)
	}

	pipelineRun := NewBuilder(workflow, &github.Event{}).Build()

	want := "hello-world-run-"
	got := pipelineRun.ObjectMeta.GenerateName
	if want != got {
		t.Errorf("Want %s, got %s", want, got)
	}
}

func TestPipelineRunNamespace(t *testing.T) {
	workflow, err := testutils.ReadWorkflow("referencing-existing-tasks.yaml")
	if err != nil {
		t.Fatal(err)
	}

	pipelineRun := NewBuilder(workflow, &github.Event{}).Build()
	want := "dev"
	got := pipelineRun.ObjectMeta.Namespace
	if want != got {
		t.Errorf("Want %s, got %s", want, got)
	}
}

func TestPipelineDescription(t *testing.T) {
	workflow, err := testutils.ReadWorkflow("referencing-existing-tasks.yaml")
	if err != nil {
		t.Fatal(err)
	}

	pipelineRun := NewBuilder(workflow, &github.Event{}).Build()

	want := "FIXME"
	got := pipelineRun.Spec.PipelineSpec.Description
	if want != got {
		t.Errorf("Want %s, got %s", want, got)
	}
}

func TestPipelineTasksWithTaskRefs(t *testing.T) {
	workflow, err := testutils.ReadWorkflow("referencing-existing-tasks.yaml")
	if err != nil {
		t.Fatal(err)
	}

	pipelineRun := NewBuilder(workflow, &github.Event{}).Build()

	wantBuild := pipelinev1beta1.PipelineTask{Name: "build",
		TaskRef: &pipelinev1beta1.TaskRef{Name: "golang-builder"},
	}

	wantTest := pipelinev1beta1.PipelineTask{Name: "test",
		TaskRef: &pipelinev1beta1.TaskRef{Name: "golang-testing"},
		Retries: 2,
		Timeout: &metav1.Duration{Duration: 1 * time.Hour},
	}

	gotBuild, err := getPipelineTaskOrFail(t, pipelineRun, "build")
	if err != nil {
		t.Fatal(err)
	}

	gotTest, err := getPipelineTaskOrFail(t, pipelineRun, "test")
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(wantBuild, gotBuild); diff != "" {
		t.Errorf("Mismatch (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff(wantTest, gotTest); diff != "" {
		t.Errorf("Mismatch (-want +got):\n%s", diff)
	}
}

func TestPipelineTasksWithEmbeddedTasks(t *testing.T) {
	workflow, err := testutils.ReadWorkflow("embedding-tasks.yaml")
	if err != nil {
		t.Fatal(err)
	}

	pipelineRun := NewBuilder(workflow, &github.Event{}).Build()

	wantLint := pipelinev1beta1.PipelineTask{
		Name: "lint",
		TaskSpec: &pipelinev1beta1.EmbeddedTask{
			TaskSpec: pipelinev1beta1.TaskSpec{
				StepTemplate: &corev1.Container{
					Env: []corev1.EnvVar{
						{Name: "ENV_VAR_1", Value: "a"},
					},
				},
				Steps: []pipelinev1beta1.Step{{Container: corev1.Container{
					Name:            "golangci-lint",
					Image:           "golang",
					ImagePullPolicy: "Always",
				},
					Script: `#!/usr/bin/env sh
set -euo pipefail
golangci-lint run`,
				},
				},
			},
		},
	}

	cpu, _ := resource.ParseQuantity("1m")
	memory, _ := resource.ParseQuantity("2Gi")
	resources := corev1.ResourceList{
		corev1.ResourceCPU:    cpu,
		corev1.ResourceMemory: memory,
	}

	wantTest := pipelinev1beta1.PipelineTask{
		Name: "test",
		TaskSpec: &pipelinev1beta1.EmbeddedTask{
			TaskSpec: pipelinev1beta1.TaskSpec{
				StepTemplate: &corev1.Container{
					Resources: corev1.ResourceRequirements{
						Limits:   resources,
						Requests: resources,
					},
				},
				Steps: []pipelinev1beta1.Step{{Container: corev1.Container{
					Name:            "test",
					Image:           "golang",
					ImagePullPolicy: "Always",
					Env: []corev1.EnvVar{
						{Name: "ENV_VAR_2", Value: "b"},
					},
				},
					Script: `#!/usr/bin/env sh
set -euo pipefail
go test ./...`,
				},
				},
			},
		},
	}

	gotLint, err := getPipelineTaskOrFail(t, pipelineRun, "lint")
	if err != nil {
		t.Fatal(err)
	}

	gotTest, err := getPipelineTaskOrFail(t, pipelineRun, "test")
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(wantLint, gotLint); diff != "" {
		t.Errorf("Mismatch (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff(wantTest, gotTest); diff != "" {
		t.Errorf("Mismatch (-want +got):\n%s", diff)
	}
}

func TestTaskRunSpecs(t *testing.T) {
	workflow, err := testutils.ReadWorkflow("referencing-existing-tasks.yaml")
	if err != nil {
		t.Fatal(err)
	}

	pipelineRun := NewBuilder(workflow, &github.Event{}).Build()

	tests := []struct {
		name           string
		serviceAccount string
		podTemplate    *pipelinev1beta1.PodTemplate
	}{
		{"build", "sa-1", nil},
		{"test", "sa-2", &pipelinev1beta1.PodTemplate{NodeSelector: map[string]string{"label": "value"}}},
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
			t.Fatalf("Error: no such TaskRunSpec %s", test.name)
		}

		if test.serviceAccount != taskRunSpec.TaskServiceAccountName {
			t.Errorf("Fail at task %s.\nMismatch in TaskRunSpec.ServiceAccount: want service account %s, got %s", test.name, test.serviceAccount, taskRunSpec.TaskServiceAccountName)
		}

		if diff := cmp.Diff(test.podTemplate, taskRunSpec.TaskPodTemplate); diff != "" {
			t.Errorf("Fail at task %s.\nMismatches in TaskRunSpec.PodTemplate (-want +got):\n%s", test.name, diff)
		}
	}
}

func TestVariableExpansion(t *testing.T) {
	workflow, err := testutils.ReadWorkflow("variables.yaml")
	if err != nil {
		t.Fatal(err)
	}

	event, err := testutils.ReadEvent("event.json")
	if err != nil {
		t.Fatal(err)
	}

	pipelineRun := NewBuilder(workflow, event).Build()

	wantLabels := map[string]string{
		"workflows.dev/workflow":    "hello",
		"workflows.dev/head-commit": "833568e",
	}

	gotLabels := pipelineRun.Labels

	if diff := cmp.Diff(wantLabels, gotLabels); diff != "" {
		t.Errorf("Mismatch (-want +got):\n%s", diff)
	}

	wantAnnotations := map[string]string{
		"workflows.dev/author": "john-doe",
	}

	gotAnnotations := pipelineRun.Annotations

	if diff := cmp.Diff(wantAnnotations, gotAnnotations); diff != "" {
		t.Errorf("Mismatch (-want +got):\n%s", diff)
	}

	wantScript := `#!/usr/bin/env sh
set -euo pipefail
echo "Hello john-doe!"
echo "Thank you for running the workflow hello"
echo "The PipelineRun $(context.pipelineRun.name) has been created"
`

	gotScript := pipelineRun.Spec.PipelineSpec.Tasks[0].TaskSpec.Steps[0].Script

	if diff := cmp.Diff(wantScript, gotScript); diff != "" {
		t.Errorf("Mismatch (-want +got): %s\n", diff)
	}
}
