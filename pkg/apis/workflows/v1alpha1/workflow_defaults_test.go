package v1alpha1

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/nubank/workflows/pkg/apis/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestWorkflowDefaulting(t *testing.T) {
	configs := &config.Config{
		Defaults: &config.Defaults{
			DefaultImage: "ubuntu",
			Labels: map[string]string{
				"workflows.dev/sample-label": "abc",
			},
			Annotations: map[string]string{
				"workflows.dev/sample-annotation": "def",
			},
		},
	}

	ctx := config.WithConfig(context.Background(), configs)

	tests := []struct {
		name string
		in   *Workflow
		want *Workflow
	}{{
		name: "empty",
		in:   &Workflow{},
		want: &Workflow{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"workflows.dev/sample-label": "abc",
				},
				Annotations: map[string]string{
					"workflows.dev/sample-annotation": "def",
				},
			},
		},
	},
		{
			name: "do not override existing labels and/or annotations",
			in: &Workflow{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"workflows.dev/sample-label": "ghi",
					},
					Annotations: map[string]string{
						"workflows.dev/sample-annotation": "jkl",
					},
				},
			},
			want: &Workflow{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"workflows.dev/sample-label": "ghi",
					},
					Annotations: map[string]string{
						"workflows.dev/sample-annotation": "jkl",
					},
				},
			},
		},
	}

	for _, test := range tests {
		want := test.want
		got := test.in
		got.SetDefaults(ctx)

		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("Fail in %s.\nMismatch (-want +got):\n%s", test.name, diff)
		}
	}
}

func TestWorkflowSpecDefaulting(t *testing.T) {
	configs := &config.Config{
		Defaults: &config.Defaults{
			DefaultImage: "ubuntu",
			Webhook:      "https://hooks.example.com",
		},
	}

	ctx := config.WithConfig(context.Background(), configs)

	tests := []struct {
		name string
		in   *WorkflowSpec
		want *WorkflowSpec
	}{{
		name: "empty",
		in:   &WorkflowSpec{},
		want: &WorkflowSpec{
			Webhook: &Webhook{
				URL: "https://hooks.example.com",
			},
		},
	},
		{
			name: "add default image",
			in: &WorkflowSpec{
				Webhook: &Webhook{
					URL: "https://hooks.example.dev",
				},
				Tasks: map[string]*Task{
					"build": &Task{
						Steps: []EmbeddedStep{{
							Name: "build",
						}},
					},
				},
			},
			want: &WorkflowSpec{
				Webhook: &Webhook{
					URL: "https://hooks.example.dev",
				},
				Tasks: map[string]*Task{
					"build": &Task{
						Steps: []EmbeddedStep{{
							Name:  "build",
							Image: "ubuntu",
						}},
					},
				},
			},
		},
	}

	for _, test := range tests {
		want := test.want
		got := test.in
		got.SetDefaults(ctx)

		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("Fail in %s.\nMismatch (-want +got):\n%s", test.name, diff)
		}
	}
}
