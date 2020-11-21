package pipelinerun

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"k8s.io/apimachinery/pkg/api/resource"

	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"

	"github.com/nubank/workflows/pkg/apis/config"
	"github.com/nubank/workflows/pkg/github"
	"github.com/nubank/workflows/pkg/testutils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func findPipelineTaskOrFail(pipelineRun *pipelinev1beta1.PipelineRun, taskName string) (pipelinev1beta1.PipelineTask, error) {
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

	gotBuild, err := findPipelineTaskOrFail(pipelineRun, "build")
	if err != nil {
		t.Fatal(err)
	}

	gotTest, err := findPipelineTaskOrFail(pipelineRun, "test")
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
					ImagePullPolicy: corev1.PullAlways,
				},
					Script: `#!/usr/bin/env sh
set -eu
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
					ImagePullPolicy: corev1.PullAlways,
					Env: []corev1.EnvVar{
						{Name: "ENV_VAR_2", Value: "b"},
					},
				},
					Script: `#!/usr/bin/env sh
set -eu
go test ./...`,
				},
				},
			},
		},
	}

	gotLint, err := findPipelineTaskOrFail(pipelineRun, "lint")
	if err != nil {
		t.Fatal(err)
	}

	gotTest, err := findPipelineTaskOrFail(pipelineRun, "test")
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
	workflow, err := testutils.ReadWorkflow("variable-substitution.yaml")
	if err != nil {
		t.Fatal(err)
	}

	event, err := testutils.ReadEvent("event.json")
	if err != nil {
		t.Fatal(err)
	}

	pipelineRun := NewBuilder(workflow, event).Build()

	wantAnnotations := map[string]string{
		"workflows.dev/author": "john-doe",
	}

	gotAnnotations := pipelineRun.Annotations

	if diff := cmp.Diff(wantAnnotations, gotAnnotations); diff != "" {
		t.Errorf("Mismatch (-want +got):\n%s", diff)
	}

	wantScript := `#!/usr/bin/env sh
set -eu
echo "Hello john-doe!"
echo "Thank you for running the workflow hello"
echo "The PipelineRun $(context.pipelineRun.name) has been created"
`

	gotScript := pipelineRun.Spec.PipelineSpec.Tasks[0].TaskSpec.Steps[0].Script

	if diff := cmp.Diff(wantScript, gotScript); diff != "" {
		t.Errorf("Mismatch (-want +got): %s\n", diff)
	}
}

func TestLabelsAndAnnotations(t *testing.T) {
	workflow, err := testutils.ReadWorkflow("labels-and-annotations.yaml")
	if err != nil {
		t.Fatal(err)
	}

	event, err := testutils.ReadEvent("event.json")
	if err != nil {
		t.Fatal(err)
	}

	defaults := &config.Defaults{
		Labels: map[string]string{
			"workflows.dev/head-commit":   "$(workflow.head-commit)",
			"workflows.dev/example-label": "def",
		},
		Annotations: map[string]string{
			"workflows.dev/example-annotation": "ghi",
			"workflows.dev/other-annotation":   "xyz",
		},
	}

	pipelineRun := NewBuilder(workflow, event).WithDefaults(defaults).Build()

	wantLabels := map[string]string{
		"workflows.dev/workflow":      "hello",
		"workflows.dev/example-label": "abc",
		"workflows.dev/head-commit":   "833568e",
	}

	gotLabels := pipelineRun.Labels

	if diff := cmp.Diff(wantLabels, gotLabels); diff != "" {
		t.Errorf("Mismatch in labels (-want +got):\n%s", diff)
	}

	wantAnnotations := map[string]string{
		"workflows.dev/author":             "john-doe",
		"workflows.dev/example-annotation": "def",
		"workflows.dev/other-annotation":   "xyz",
	}

	gotAnnotations := pipelineRun.Annotations

	if diff := cmp.Diff(wantAnnotations, gotAnnotations); diff != "" {
		t.Errorf("Mismatch (-want +got):\n%s", diff)
	}

}

func TestGraph(t *testing.T) {
	workflow, err := testutils.ReadWorkflow("creating-graphs.yaml")
	if err != nil {
		t.Fatal(err)
	}

	event, err := testutils.ReadEvent("event.json")
	if err != nil {
		t.Fatal(err)
	}

	pipelineRun := NewBuilder(workflow, event).Build()

	task, err := findPipelineTaskOrFail(pipelineRun, "release")
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"build"}
	got := task.RunAfter
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Fail to convert attribut Need to RunAfter\nMismatch (-want +got):\n%s", diff)
	}
}

func TestCheckout(t *testing.T) {
	workflow, err := testutils.ReadWorkflow("checking-out-one-repo.yaml")
	if err != nil {
		t.Fatal(err)
	}

	event, err := testutils.ReadEvent("event.json")
	if err != nil {
		t.Fatal(err)
	}

	pipelineRun := NewBuilder(workflow, event).Build()

	want := pipelinev1beta1.PipelineRunSpec{
		PipelineSpec: &pipelinev1beta1.PipelineSpec{
			Tasks: []pipelinev1beta1.PipelineTask{{
				Name: "lint",
				TaskSpec: &pipelinev1beta1.EmbeddedTask{
					TaskSpec: pipelinev1beta1.TaskSpec{
						Workspaces: []pipelinev1beta1.WorkspaceDeclaration{{
							Name: projectsWorkspace,
						},
						},
						Results: []pipelinev1beta1.TaskResult{{
							Name: "my-repo-commit",
						},
						},
						Steps: []pipelinev1beta1.Step{{
							Container: corev1.Container{
								Name:  "checkout",
								Image: gitInitImage,
							},
							Script: `#!/usr/bin/env sh
set -euo pipefail

/ko-app/git-init \
    -url="https://github.com/john-doe/my-repo.git" \
    -revision="833568e" \
    -path="$(workspaces.projects.path)/my-repo" \
    -sslVerify="true" \
    -submodules="true" \
    -depth="1"

cd $(workspaces.projects.path)/my-repo
echo -n "$(git rev-parse HEAD)" > /tekton/results/my-repo-commit`,
						},
							{
								Container: corev1.Container{
									WorkingDir:      "$(workspaces.projects.path)/my-repo",
									ImagePullPolicy: corev1.PullAlways,
								},
								Script: `#!/usr/bin/env sh
set -eu
ls`,
							},
						},
					},
				},
				Workspaces: []pipelinev1beta1.WorkspacePipelineTaskBinding{{
					Name:      projectsWorkspace,
					Workspace: projectsWorkspace,
				},
				},
			},
			},
			Workspaces: []pipelinev1beta1.PipelineWorkspaceDeclaration{{
				Name: projectsWorkspace,
			},
			},
		},
		Workspaces: []pipelinev1beta1.WorkspaceBinding{{
			Name:     projectsWorkspace,
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
		},
		TaskRunSpecs: []pipelinev1beta1.PipelineTaskRunSpec{},
	}

	got := pipelineRun.Spec

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Mismatch (-want +got):\n%s", diff)
	}
}
