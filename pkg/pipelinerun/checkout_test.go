package pipelinerun

import (
	"testing"

	corev1 "k8s.io/api/core/v1"

	"github.com/google/go-cmp/cmp"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"

	"github.com/nubank/workflows/pkg/testutils"
)

// newTestCheckoutStep returns a Checkout instance.
func newTestCheckoutStep(filename string) (*Checkout, error) {
	workflow, err := testutils.ReadWorkflow(filename)
	if err != nil {
		return nil, err
	}

	event, err := testutils.ReadEvent("event.json")
	if err != nil {
		return nil, err
	}

	return &Checkout{
		workflow: workflow,
		event:    event,
	}, nil
}

func TestCheckoutBuildStep(t *testing.T) {
	tests := []struct {
		name     string
		workflow string
		task     string
		want     pipelinev1beta1.Step
	}{
		{
			name:     "checks out a private repository",
			workflow: "checking-out-private-repos.yaml",
			task:     "lint",
			want: pipelinev1beta1.Step{
				Container: corev1.Container{
					Name:  "checkout",
					Image: gitInitImage,
					VolumeMounts: []corev1.VolumeMount{
						{Name: sshPrivateKeysVolumeName,
							ReadOnly:  true,
							MountPath: sshPrivateKeysMountPath,
						},
					},
				},
				Script: `#!/usr/bin/env sh
set -euo pipefail

mkdir -p ~/.ssh
cat > ~/.ssh/config<<EOF
Host github.com
  User git
  Hostname github.com
  IdentityFile /var/run/secrets/workflows/my-repo_id_rsa
EOF

/ko-app/git-init \
    -url="git@github.com:john-doe/my-repo.git" \
    -revision="833568e" \
    -path="$(workspaces.projects.path)/my-repo" \
    -sslVerify="true" \
    -submodules="true" \
    -depth="1"

cd $(workspaces.projects.path)/my-repo
echo -n "$(git rev-parse HEAD)" > /tekton/results/my-repo-commit`,
			},
		},
		{
			name:     "checks out a public repository",
			workflow: "checking-out-public-repos.yaml",
			task:     "lint",
			want: pipelinev1beta1.Step{
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
		},
		{
			name:     "preserve the step name",
			workflow: "checking-out-public-repos.yaml",
			task:     "test",
			want: pipelinev1beta1.Step{
				Container: corev1.Container{
					Name:  "checkout-code",
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
		},
	}

	for _, test := range tests {
		checkout, err := newTestCheckoutStep(test.workflow)
		if err != nil {
			t.Fatalf("Fail in %s: %v", test.name, err)
		}

		task, exists := checkout.workflow.Spec.Tasks[test.task]
		if !exists {
			t.Fatalf("Fail in %s: task %s not found", test.name, test.task)
		}

		step := task.Steps[0]
		got := checkout.BuildStep(step)
		if diff := cmp.Diff(test.want, got); diff != "" {
			t.Errorf("Fail in %s\nMismatch (-want +got):\n%s", test.name, diff)
		}
	}
}

func TestCheckoutPostEmbeddedTaskCreation(t *testing.T) {
	tests := []struct {
		name     string
		workflow string
		task     pipelinev1beta1.EmbeddedTask
		want     pipelinev1beta1.EmbeddedTask
	}{
		{
			name:     "modify a task with two private repositories",
			workflow: "checking-out-private-repos.yaml",
			task: pipelinev1beta1.EmbeddedTask{
				TaskSpec: pipelinev1beta1.TaskSpec{
					Steps: []pipelinev1beta1.Step{{
						Container: corev1.Container{
							Name: "checkout",
						},
					},
						pipelinev1beta1.Step{
							Container: corev1.Container{
								Name: "lint",
							},
						},
					},
				},
			},
			want: pipelinev1beta1.EmbeddedTask{
				TaskSpec: pipelinev1beta1.TaskSpec{
					Workspaces: []pipelinev1beta1.WorkspaceDeclaration{{
						Name: projectsWorkspace,
					},
					},
					Results: []pipelinev1beta1.TaskResult{{
						Name: "my-repo-commit",
					},
						{
							Name: "my-other-repo-commit",
						},
					},
					Volumes: []corev1.Volume{{
						Name: sshPrivateKeysVolumeName,
						VolumeSource: corev1.VolumeSource{
							Secret: &corev1.SecretVolumeSource{
								SecretName:  "lorem-ipsum-ssh-private-keys",
								DefaultMode: &defaultVolumeMode,
							},
						},
					},
					},
					Steps: []pipelinev1beta1.Step{{
						Container: corev1.Container{
							Name: "checkout",
						},
					},
						{
							Container: corev1.Container{
								Name:  "checkout-my-other-repo",
								Image: gitInitImage,
								VolumeMounts: []corev1.VolumeMount{{Name: sshPrivateKeysVolumeName,
									ReadOnly:  true,
									MountPath: sshPrivateKeysMountPath,
								},
								},
							},
							Script: `#!/usr/bin/env sh
set -euo pipefail

mkdir -p ~/.ssh
cat > ~/.ssh/config<<EOF
Host github.com
  User git
  Hostname github.com
  IdentityFile /var/run/secrets/workflows/my-other-repo_id_rsa
EOF

/ko-app/git-init \
    -url="git@github.com:john-doe/my-other-repo.git" \
    -revision="main" \
    -path="$(workspaces.projects.path)/my-other-repo" \
    -sslVerify="true" \
    -submodules="true" \
    -depth="1"

cd $(workspaces.projects.path)/my-other-repo
echo -n "$(git rev-parse HEAD)" > /tekton/results/my-other-repo-commit`,
						},
						{
							Container: corev1.Container{
								Name:       "lint",
								WorkingDir: "$(workspaces.projects.path)/my-repo",
							},
						},
					},
				},
			},
		},

		{
			name:     "modify a task containing two public repositories and existing workspaces and results",
			workflow: "checking-out-public-repos.yaml",
			task: pipelinev1beta1.EmbeddedTask{
				TaskSpec: pipelinev1beta1.TaskSpec{
					Workspaces: []pipelinev1beta1.WorkspaceDeclaration{{
						Name: "workspace1",
					},
					},
					Results: []pipelinev1beta1.TaskResult{{
						Name: "result1",
					},
					},
					Steps: []pipelinev1beta1.Step{{
						Container: corev1.Container{
							Name: "checkout",
						},
					},
						pipelinev1beta1.Step{
							Container: corev1.Container{
								Name: "lint",
							},
						},
					},
				},
			},
			want: pipelinev1beta1.EmbeddedTask{
				TaskSpec: pipelinev1beta1.TaskSpec{
					Workspaces: []pipelinev1beta1.WorkspaceDeclaration{{
						Name: "workspace1",
					},
						{
							Name: projectsWorkspace,
						},
					},
					Results: []pipelinev1beta1.TaskResult{{
						Name: "result1",
					},
						{
							Name: "my-repo-commit",
						},
						{
							Name: "my-other-repo-commit",
						},
					},
					Steps: []pipelinev1beta1.Step{{
						Container: corev1.Container{
							Name: "checkout",
						},
					},
						{
							Container: corev1.Container{
								Name:  "checkout-my-other-repo",
								Image: gitInitImage,
							},
							Script: `#!/usr/bin/env sh
set -euo pipefail

/ko-app/git-init \
    -url="https://github.com/john-doe/my-other-repo.git" \
    -revision="main" \
    -path="$(workspaces.projects.path)/my-other-repo" \
    -sslVerify="true" \
    -submodules="true" \
    -depth="1"

cd $(workspaces.projects.path)/my-other-repo
echo -n "$(git rev-parse HEAD)" > /tekton/results/my-other-repo-commit`,
						},
						{
							Container: corev1.Container{
								Name:       "lint",
								WorkingDir: "$(workspaces.projects.path)/my-repo",
							},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		checkout, err := newTestCheckoutStep(test.workflow)
		if err != nil {
			t.Fatalf("Fail in %s: %v", test.name, err)
		}

		got := test.task
		checkout.PostEmbeddedTaskCreation(&got)
		if diff := cmp.Diff(test.want, got); diff != "" {
			t.Errorf("Fail in %s\nMismatch (-want +got):\n%s", test.name, diff)
		}
	}
}

func TestCheckoutPostPipelineTaskCreation(t *testing.T) {
	tests := []struct {
		name string
		in   pipelinev1beta1.PipelineTask
		want pipelinev1beta1.PipelineTask
	}{{
		name: "add the project workspace",
		in:   pipelinev1beta1.PipelineTask{},
		want: pipelinev1beta1.PipelineTask{
			Workspaces: []pipelinev1beta1.WorkspacePipelineTaskBinding{{
				Name:      projectsWorkspace,
				Workspace: projectsWorkspace,
			},
			},
		},
	},
		{
			name: "append the projects workspace to an existing list of workspaces",
			in: pipelinev1beta1.PipelineTask{
				Workspaces: []pipelinev1beta1.WorkspacePipelineTaskBinding{{
					Name:      "workspace1",
					Workspace: "workspace1",
				},
				},
			},
			want: pipelinev1beta1.PipelineTask{
				Workspaces: []pipelinev1beta1.WorkspacePipelineTaskBinding{{
					Name:      "workspace1",
					Workspace: "workspace1",
				},
					{
						Name:      projectsWorkspace,
						Workspace: projectsWorkspace,
					},
				},
			},
		},
		{
			name: "do not duplicate the project workspace",
			in: pipelinev1beta1.PipelineTask{
				Workspaces: []pipelinev1beta1.WorkspacePipelineTaskBinding{{
					Name:      projectsWorkspace,
					Workspace: projectsWorkspace,
				},
				},
			},
			want: pipelinev1beta1.PipelineTask{
				Workspaces: []pipelinev1beta1.WorkspacePipelineTaskBinding{{
					Name:      projectsWorkspace,
					Workspace: projectsWorkspace,
				},
				},
			},
		},
	}

	checkout, err := newTestCheckoutStep("checking-out-private-repos.yaml")
	if err != nil {
		t.Fatalf("Fail to create Checkout: %v", err)
	}

	for _, test := range tests {
		got := test.in
		checkout.PostPipelineTaskCreation(&got)
		if diff := cmp.Diff(test.want, got); diff != "" {
			t.Errorf("Fail in %s\nMismatch (-want +got):\n%s", test.name, diff)
		}
	}
}

func TestCheckoutPostPipelineSpecCreation(t *testing.T) {
	tests := []struct {
		name string
		in   pipelinev1beta1.PipelineSpec
		want pipelinev1beta1.PipelineSpec
	}{{
		name: "add the project workspace",
		in:   pipelinev1beta1.PipelineSpec{},
		want: pipelinev1beta1.PipelineSpec{
			Workspaces: []pipelinev1beta1.PipelineWorkspaceDeclaration{{
				Name: projectsWorkspace,
			},
			},
		},
	},
		{
			name: "append the projects workspace to an existing list of workspaces",
			in: pipelinev1beta1.PipelineSpec{
				Workspaces: []pipelinev1beta1.PipelineWorkspaceDeclaration{{
					Name: "workspace1",
				},
				},
			},
			want: pipelinev1beta1.PipelineSpec{
				Workspaces: []pipelinev1beta1.PipelineWorkspaceDeclaration{{
					Name: "workspace1",
				},
					{
						Name: projectsWorkspace,
					},
				},
			},
		},
		{
			name: "do not duplicate the project workspace",
			in: pipelinev1beta1.PipelineSpec{
				Workspaces: []pipelinev1beta1.PipelineWorkspaceDeclaration{{
					Name: projectsWorkspace,
				},
				},
			},
			want: pipelinev1beta1.PipelineSpec{
				Workspaces: []pipelinev1beta1.PipelineWorkspaceDeclaration{{
					Name: projectsWorkspace,
				},
				},
			},
		},
	}

	checkout, err := newTestCheckoutStep("checking-out-private-repos.yaml")
	if err != nil {
		t.Fatalf("Fail to create Checkout: %v", err)
	}

	for _, test := range tests {
		got := test.in
		checkout.PostPipelineSpecCreation(&got)
		if diff := cmp.Diff(test.want, got); diff != "" {
			t.Errorf("Fail in %s\nMismatch (-want +got):\n%s", test.name, diff)
		}
	}
}

func TestCheckoutPostPipelineRunCreation(t *testing.T) {
	workspaceBinding := pipelinev1beta1.WorkspaceBinding{
		Name:     projectsWorkspace,
		EmptyDir: &corev1.EmptyDirVolumeSource{},
	}

	tests := []struct {
		name string
		in   pipelinev1beta1.PipelineRun
		want pipelinev1beta1.PipelineRun
	}{{
		name: "add the project workspace",
		in:   pipelinev1beta1.PipelineRun{},
		want: pipelinev1beta1.PipelineRun{
			Spec: pipelinev1beta1.PipelineRunSpec{
				Workspaces: []pipelinev1beta1.WorkspaceBinding{workspaceBinding},
			},
		},
	},
		{
			name: "append the projects workspace to an existing list of workspaces",
			in: pipelinev1beta1.PipelineRun{
				Spec: pipelinev1beta1.PipelineRunSpec{
					Workspaces: []pipelinev1beta1.WorkspaceBinding{{
						Name: "workspace1",
					},
					},
				},
			},
			want: pipelinev1beta1.PipelineRun{
				Spec: pipelinev1beta1.PipelineRunSpec{
					Workspaces: []pipelinev1beta1.WorkspaceBinding{{
						Name: "workspace1",
					},
						workspaceBinding,
					},
				},
			},
		},
		{
			name: "do not duplicate the project workspace",
			in: pipelinev1beta1.PipelineRun{
				Spec: pipelinev1beta1.PipelineRunSpec{
					Workspaces: []pipelinev1beta1.WorkspaceBinding{workspaceBinding},
				},
			},
			want: pipelinev1beta1.PipelineRun{
				Spec: pipelinev1beta1.PipelineRunSpec{
					Workspaces: []pipelinev1beta1.WorkspaceBinding{workspaceBinding},
				},
			},
		},
	}

	checkout, err := newTestCheckoutStep("checking-out-private-repos.yaml")
	if err != nil {
		t.Fatalf("Fail to create Checkout: %v", err)
	}

	for _, test := range tests {
		got := test.in
		checkout.PostPipelineRunCreation(&got)
		if diff := cmp.Diff(test.want, got); diff != "" {
			t.Errorf("Fail in %s\nMismatch (-want +got):\n%s", test.name, diff)
		}
	}
}
