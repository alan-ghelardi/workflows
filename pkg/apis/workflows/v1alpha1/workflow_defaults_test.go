package v1alpha1

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/nubank/workflows/pkg/apis/config"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
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
			Events:   []string{"push"},
			Defaults: &Defaults{},
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
				Events:   []string{"push"},
				Defaults: &Defaults{},
			},
		},
		{
			name: "add default image using the config-defaults",
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
				Events:   []string{"pull_request"},
				Defaults: &Defaults{},
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
		{
			name: "add default settings using workflow defaults",
			in: &WorkflowSpec{
				Webhook: &Webhook{
					URL: "https://hooks.example.dev",
				},
				Events: []string{"pull_request"},
				Defaults: &Defaults{
					Image: "golang",
					PodTemplate: &pipelinev1beta1.PodTemplate{
						NodeSelector: map[string]string{
							"a.b/c": "y",
						},
					},
					ServiceAccount: "sa-1",
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
				Events: []string{"pull_request"},
				Defaults: &Defaults{
					Image: "golang",
					PodTemplate: &pipelinev1beta1.PodTemplate{
						NodeSelector: map[string]string{
							"a.b/c": "y",
						},
					},
					ServiceAccount: "sa-1",
				},
				Tasks: map[string]*Task{
					"build": &Task{
						PodTemplate: &pipelinev1beta1.PodTemplate{
							NodeSelector: map[string]string{
								"a.b/c": "y",
							},
						},
						ServiceAccount: "sa-1",
						Steps: []EmbeddedStep{{
							Name:  "build",
							Image: "golang",
						}},
					},
				},
			},
		},
		{
			name: "do not overwrite more specific settings with workflow defaults",
			in: &WorkflowSpec{
				Webhook: &Webhook{
					URL: "https://hooks.example.dev",
				},
				Events: []string{"pull_request"},
				Defaults: &Defaults{
					Image: "golang",
					PodTemplate: &pipelinev1beta1.PodTemplate{
						NodeSelector: map[string]string{
							"a.b/c": "y",
						},
					},
					ServiceAccount: "sa-1",
				},

				Tasks: map[string]*Task{
					"build": &Task{
						PodTemplate: &pipelinev1beta1.PodTemplate{
							NodeSelector: map[string]string{
								"d.e/f": "z",
							},
						},
						ServiceAccount: "sa-2",
						Steps: []EmbeddedStep{{
							Name:  "build",
							Image: "ubuntu",
						}},
					},
				},
			},
			want: &WorkflowSpec{
				Webhook: &Webhook{
					URL: "https://hooks.example.dev",
				},
				Events: []string{"pull_request"},
				Defaults: &Defaults{
					Image: "golang",
					PodTemplate: &pipelinev1beta1.PodTemplate{
						NodeSelector: map[string]string{
							"a.b/c": "y",
						},
					},
					ServiceAccount: "sa-1",
				},
				Tasks: map[string]*Task{
					"build": &Task{
						PodTemplate: &pipelinev1beta1.PodTemplate{
							NodeSelector: map[string]string{
								"d.e/f": "z",
							},
						},
						ServiceAccount: "sa-2",
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
