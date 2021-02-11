package v1alpha1

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/nubank/workflows/pkg/apis/config"
)

func TestWorkflowSpecDefaulting(t *testing.T) {
	configs := &config.Config{
		Defaults: &config.Defaults{
			DefaultEvents: []string{"push"},
			DefaultImage:  "ubuntu",
			Webhook:       "https://hooks.example.com",
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
			Events: []string{"push"},
		},
	},
		{
			name: "add default events",
			in: &WorkflowSpec{
				Webhook: &Webhook{
					URL: "https://hooks.example.dev",
				},
			},
			want: &WorkflowSpec{
				Webhook: &Webhook{
					URL: "https://hooks.example.dev",
				},
				Events: []string{"push"},
			},
		},
		{
			name: "add default image",
			in: &WorkflowSpec{
				Webhook: &Webhook{
					URL: "https://hooks.example.dev",
				},
				Events: []string{"pull_request"},
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
				Events: []string{"pull_request"},
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
