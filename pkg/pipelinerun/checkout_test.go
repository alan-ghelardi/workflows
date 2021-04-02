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
			workflow: "checking-out-private-repo.yaml",
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
			workflow: "checking-out-public-repo.yaml",
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
			workflow: "checking-out-public-repo.yaml",
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
