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
	"fmt"
	"strconv"

	corev1 "k8s.io/api/core/v1"

	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"

	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/kmeta"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genreconciler
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Workflow is the Schema for the workflows API
type Workflow struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WorkflowSpec   `json:"spec,omitempty"`
	Status WorkflowStatus `json:"status,omitempty"`
}

var (
	// Check that Workflow can be validated and defaulted.
	_ apis.Validatable   = (*Workflow)(nil)
	_ apis.Defaultable   = (*Workflow)(nil)
	_ kmeta.OwnerRefable = (*Workflow)(nil)
	// Check that the type conforms to the duck Knative Resource shape.
	_ duckv1.KRShaped = (*Workflow)(nil)
)

// WorkflowSpec defines the desired state of Workflow objects.
type WorkflowSpec struct {

	// User-facing description of this workflow.
	// +optional
	Description string `json:"description,omitempty"`

	// The repository whose changes trigger the workflow.
	Repository *Repository `json:"repo"`

	// Github Webhook that triggers this workflow.
	Webhook *Webhook `json:"webhook"`

	// Other repositories that must be checked out during the execution of this workflow.
	// +optional
	SecondaryRepositories []Repository `json:"secondaryRepos,omitempty"`

	// The tasks that make up the workflow.
	Tasks map[string]*Task `json:"tasks"`
}

// Repository contains relevant information about a Github repository associated
// to a workflow.
type Repository struct {

	// The repository's name.
	Name string `json:"name"`

	// The repository's owner.
	Owner string `json:"owner"`

	// Authorizes task runs that make up this workflow to access the
	// repository in question.
	// +optional
	DeployKey *DeployKey `json:"deployKey,omitempty"`
}

// Webhook contains information about a Github Webhook that triggers the workflow
// in question.
type Webhook struct {

	// The URL to which the payloads will be delivered
	URL string `json:"url"`

	// Determines what events the Webhook is triggered for.
	Events []string `json:"events"`
}

// DeployKey contains a few settings for the deploy keys associated to the workflow.
type DeployKey struct {

	// Whether or not the deploy key has read-only permissions on the repository.
	ReadOnly bool `json:"readOnly"`
}

// Task contains information about the tasks that make up the workflow.
type Task struct {

	// A map of environment variables that are available to all steps in the task.
	// +optional
	Env map[string]string `json:"env,omitempty"`

	// Execution parameters for this task.
	// +optional
	Params map[string]string `json:"params,omitempty"`

	// Specifies the template to create the pod associated to the underwing Tekton TaskRun object.
	// +optional
	PodTemplate *pipelinev1beta1.PodTemplate `json:"podTemplate,omitempty"`

	// How many times the task should be retried in case of failures.
	// +optional
	Retries int `json:"retries,omitempty"`

	// Assigns resources (CPU and memory) to the task.
	// +optional
	Resources corev1.ResourceList `json:"resources,omitempty"`

	// Service account to be assigned to the underwing TaskRun object.
	// +optional
	ServiceAccount string `json:"serviceAccount,omitempty"`

	// Sequential steps to be executed in this task.
	// +optional
	Steps []EmbeddedStep `json:"steps,omitempty"`

	// Time after which the task times out.
	// +optional
	Timeout *metav1.Duration `json:"timeout,omitempty"`

	// Selects an existing Tekton Task to run in this workflow.
	// +optional
	Use string `json:"use,omitempty"`

	// List of workspaces to be bound to the underwing TaskRun object.
	// +optional
	Workspace []string `json:"workspaces,omitempty"`
}

// EmbeddedStep defines a step to be executed as part of a task.
type EmbeddedStep struct {

	// Docker/OCI image to serve as the container for the step in question.
	// +optional
	Image string `json:"image,omitempty"`

	// The step's name.
	// +optional
	Name string `json:"name,omitempty"`

	// Runs command-line programs using the container's shell.
	// +optional
	Run string `json:"run,omitempty"`

	// Selects a built-in step to run as part of the task in question.
	// +optional
	Use string `json:"use,omitempty"`
}

// WorkflowStatus defines the observed state of Workflow
type WorkflowStatus struct {
	duckv1.Status `json:",inline"`
}

const (
	// WorkflowConditionReady is set when the revision is starting to materialize
	// runtime resources, and becomes true when those resources are ready.
	WorkflowConditionReady = apis.ConditionReady
)

// IsReadOnlyDeployKey returns true if the associated deploy key is read only or false otherwise.
func (r *Repository) IsReadOnlyDeployKey() bool {
	if r.DeployKey == nil {
		return true
	}
	return r.DeployKey.ReadOnly
}

// String satisfies fmt.Stringer interface.
func (r *Repository) String() string {
	return fmt.Sprintf("%s/%s", r.Owner, r.Name)
}

// GetWebhookSecretName returns the name of the Webhook secret associated to this workflow.
func (w *Workflow) GetWebhookSecretName() string {
	return fmt.Sprintf("%s-webhook-secret", w.GetName())
}

// GetDeployKeysSecretName returns the name of the private SSH key associated to
// this workflow.
func (w *Workflow) GetDeployKeysSecretName() string {
	return fmt.Sprintf("%s-private-ssh-key", w.GetName())
}

const (
	webhookIDFormat   = "workflows.dev/github.%s.%s.webhook-id"
	deployKeyIDFormat = "workflows.dev/github.%s.%s.deploy-key-id"
)

// GetWebhookID returns the id of a Webhook associated to the repository in
// question or nil if no Webhook has been created yet.
func (w *Workflow) GetWebhookID() *int64 {
	repo := w.Spec.Repository
	key := fmt.Sprintf(webhookIDFormat, repo.Owner, repo.Name)
	value, exists := w.ObjectMeta.Annotations[key]

	if exists {
		if id, err := strconv.ParseInt(value, 10, 64); err == nil {
			return &id
		}
	}
	return nil
}

// SetWebhookID stores the Webhook id associated to the supplied repository as a
// metadata in the workflow in question.
func (w *Workflow) SetWebhookID(id int64) {
	repo := w.Spec.Repository
	w.ObjectMeta.Annotations[fmt.Sprintf(webhookIDFormat, repo.Owner, repo.Name)] = fmt.Sprint(id)
}

// GetDeployKeyID returns the id of a deploy key associated to the repository in
// question or nil if no deploy key has been created yet.
func (w *Workflow) GetDeployKeyID(repo *Repository) *int64 {
	key := fmt.Sprintf(deployKeyIDFormat, repo.Owner, repo.Name)
	value, exists := w.ObjectMeta.Annotations[key]

	if exists {
		if id, err := strconv.ParseInt(value, 10, 64); err == nil {
			return &id
		}
	}
	return nil
}

// SetDeployKeyID stores the deploy key id associated to the supplied repository as a
// metadata in the workflow in question.
func (w *Workflow) SetDeployKeyID(repo *Repository, id int64) {
	w.ObjectMeta.Annotations[fmt.Sprintf(deployKeyIDFormat, repo.Owner, repo.Name)] = fmt.Sprint(id)
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// WorkflowList contains a list of Workflow objects.
type WorkflowList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Workflow `json:"items"`
}

// GetStatus retrieves the status of the resource. Implements the KRShaped interface.
func (w *Workflow) GetStatus() *duckv1.Status {
	return &w.Status.Status
}
