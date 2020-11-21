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
	"k8s.io/apimachinery/pkg/runtime/schema"
	"knative.dev/pkg/apis"
)

var condSet = apis.NewLivingConditionSet()

// GetGroupVersionKind implements kmeta.OwnerRefable
func (*Workflow) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("Workflow")
}

// GetConditionSet retrieves the condition set for this resource. Implements the KRShaped interface.
func (w *Workflow) GetConditionSet() apis.ConditionSet {
	return condSet
}

// InitializeConditions sets the initial values to the conditions.
func (w *WorkflowStatus) InitializeConditions() {
	condSet.Manage(w).InitializeConditions()
}

func (w *WorkflowStatus) MarkServiceUnavailable(name string) {
	condSet.Manage(w).MarkFalse(
		WorkflowConditionReady,
		"ServiceUnavailable",
		"Service %q wasn't found.", name)
}

func (w *WorkflowStatus) MarkServiceAvailable() {
	condSet.Manage(w).MarkTrue(WorkflowConditionReady)
}