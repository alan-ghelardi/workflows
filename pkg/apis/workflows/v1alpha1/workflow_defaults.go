/*
Copyright 2020 The Workflows Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"context"

	"github.com/nubank/workflows/pkg/apis/config"
	"knative.dev/pkg/apis"
)

// SetDefaults implements apis.Defaultable
func (w *Workflow) SetDefaults(ctx context.Context) {
	w.Spec.SetDefaults(apis.WithinSpec(ctx))
}

// SetDefaults implements apis.Defautable.
func (ws *WorkflowSpec) SetDefaults(ctx context.Context) {
	defaults := config.Get(ctx).Defaults

	if ws.Events == nil {
		ws.Events = defaults.DefaultEvents
	}

	if ws.Webhook == nil && defaults.Webhook != "" {
		ws.Webhook = &Webhook{URL: defaults.Webhook}
	}

	if ws.Defaults == nil {
		// Initialize workflow defaults.
		ws.Defaults = &Defaults{}
	}

	for _, task := range ws.Tasks {
		if ws.Defaults.PodTemplate != nil && task.PodTemplate == nil {
			task.PodTemplate = ws.Defaults.PodTemplate
		}

		if ws.Defaults.ServiceAccount != "" && task.ServiceAccount == "" {
			task.ServiceAccount = ws.Defaults.ServiceAccount
		}

		for i, step := range task.Steps {
			if step.Image == "" {
				if ws.Defaults.Image != "" {
					task.Steps[i].Image = ws.Defaults.Image
				} else {
					task.Steps[i].Image = defaults.DefaultImage
				}
			}
		}
	}
}
