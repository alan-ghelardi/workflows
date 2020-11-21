/*
Copyright 2019 The Tekton Authors

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

package v1beta1

import (
	"context"
	"fmt"
	"strings"

	"github.com/tektoncd/pipeline/pkg/apis/validate"
	"k8s.io/apimachinery/pkg/util/sets"
	"knative.dev/pkg/apis"
)

var _ apis.Validatable = (*TaskRun)(nil)

// Validate taskrun
func (tr *TaskRun) Validate(ctx context.Context) *apis.FieldError {
	errs := validate.ObjectMetadata(tr.GetObjectMeta()).ViaField("metadata")
	return errs.Also(tr.Spec.Validate(apis.WithinSpec(ctx)).ViaField("spec"))
}

// Validate taskrun spec
func (ts *TaskRunSpec) Validate(ctx context.Context) (errs *apis.FieldError) {
	// can't have both taskRef and taskSpec at the same time
	if (ts.TaskRef != nil && ts.TaskRef.Name != "") && ts.TaskSpec != nil {
		errs = errs.Also(apis.ErrDisallowedFields("taskref", "taskspec"))
	}

	// Check that one of TaskRef and TaskSpec is present
	if (ts.TaskRef == nil || (ts.TaskRef != nil && ts.TaskRef.Name == "")) && ts.TaskSpec == nil {
		errs = errs.Also(apis.ErrMissingField("taskref.name", "taskspec"))
	}

	// Validate TaskSpec if it's present
	if ts.TaskSpec != nil {
		errs = errs.Also(ts.TaskSpec.Validate(ctx).ViaField("taskspec"))
	}

	errs = errs.Also(validateParameters(ts.Params).ViaField("params"))
	errs = errs.Also(validateWorkspaceBindings(ctx, ts.Workspaces).ViaField("workspaces"))
	errs = errs.Also(ts.Resources.Validate(ctx).ViaField("resources"))

	if ts.Status != "" {
		if ts.Status != TaskRunSpecStatusCancelled {
			errs = errs.Also(apis.ErrInvalidValue(fmt.Sprintf("%s should be %s", ts.Status, TaskRunSpecStatusCancelled), "status"))
		}
	}
	if ts.Timeout != nil {
		// timeout should be a valid duration of at least 0.
		if ts.Timeout.Duration < 0 {
			errs = errs.Also(apis.ErrInvalidValue(fmt.Sprintf("%s should be >= 0", ts.Timeout.Duration.String()), "timeout"))
		}
	}

	return errs
}

// validateWorkspaceBindings makes sure the volumes provided for the Task's declared workspaces make sense.
func validateWorkspaceBindings(ctx context.Context, wb []WorkspaceBinding) (errs *apis.FieldError) {
	seen := sets.NewString()
	for idx, w := range wb {
		if seen.Has(w.Name) {
			errs = errs.Also(apis.ErrMultipleOneOf("name").ViaIndex(idx))
		}
		seen.Insert(w.Name)

		errs = errs.Also(w.Validate(ctx).ViaIndex(idx))
	}

	return errs
}

func validateParameters(params []Param) (errs *apis.FieldError) {
	// Template must not duplicate parameter names.
	seen := sets.NewString()
	for _, p := range params {
		if seen.Has(strings.ToLower(p.Name)) {
			errs = errs.Also(apis.ErrMultipleOneOf("name").ViaKey(p.Name))
		}
		seen.Insert(p.Name)
	}
	return errs
}
