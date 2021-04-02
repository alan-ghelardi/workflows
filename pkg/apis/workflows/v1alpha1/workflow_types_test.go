package v1alpha1

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestHooksURL(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{{
		name: "URL without a trailing slash",
		in:   "https://hooks.example.com",
		want: "https://hooks.example.com/api/v1alpha1/namespaces/dev/workflows/test/hooks",
	},
		{
			name: "URL with a trailing slash",
			in:   "https://hooks.example.com/",
			want: "https://hooks.example.com/api/v1alpha1/namespaces/dev/workflows/test/hooks",
		},
	}

	for _, test := range tests {
		workflow := &Workflow{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "dev",
			},
			Spec: WorkflowSpec{
				Webhook: &Webhook{
					URL: test.in,
				},
			},
		}

		got := workflow.GetHooksURL()

		if test.want != got {
			t.Errorf("Fail in %s.\nWant URL %s, but got %s", test.name, test.want, got)
		}
	}
}

func TestNeedsSSHPrivateKeys(t *testing.T) {
	tests := []struct {
		name string
		in   *Repository
		want bool
	}{
		{
			name: "private repository",
			in:   &Repository{Private: true},
			want: true,
		},
		{
			name: "public repository",
			in:   &Repository{Private: false},
			want: false,
		},
		{
			name: "private repository with write permissions",
			in: &Repository{
				Private: true,
				DeployKey: &DeployKey{
					ReadOnly: false,
				},
			},
			want: true,
		},
		{
			name: "public repository with write permissions",
			in: &Repository{
				Private: false,
				DeployKey: &DeployKey{
					ReadOnly: false,
				},
			},
			want: true,
		},
	}

	for _, test := range tests {
		got := test.in.NeedsSSHPrivateKeys()
		if test.want != got {
			t.Errorf("Fail in %s: want %t, but got %t", test.name, test.want, got)
		}
	}
}
