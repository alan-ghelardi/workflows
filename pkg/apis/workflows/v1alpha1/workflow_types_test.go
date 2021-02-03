package v1alpha1

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestTriggerURL(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{{
		name: "URL without a trailing slash",
		in:   "https://hooks.example.com",
		want: "https://hooks.example.com/api/v1alpha1/namespaces/dev/workflows/test/trigger",
	},
		{
			name: "URL with a trailing slash",
			in:   "https://hooks.example.com/",
			want: "https://hooks.example.com/api/v1alpha1/namespaces/dev/workflows/test/trigger",
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

		got := workflow.GetTriggerURL()

		if test.want != got {
			t.Errorf("Fail in %s.\nWant URL %s, but got %s", test.name, test.want, got)
		}
	}
}
