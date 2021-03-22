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
	"strings"

	corev1 "k8s.io/api/core/v1"

	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"

	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/kmeta"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	hooksEndpointPath = "/api/%s/namespaces/%s/workflows/%s/hooks"

	webhookIDFormat = "workflows.dev/github.%s.%s.webhook-id"

	deployKeyIDFormat = "workflows.dev/github.%s.%s.key-id"
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

// GetRepositories returns the list of Github repositories associated to this workflow.
func (w *Workflow) GetRepositories() []Repository {
	repos := []Repository{*w.Spec.Repository}
	repos = append(repos, w.Spec.AdditionalRepositories...)
	return repos
}

// GetDeployKeyID returns the id of a deploy key associated to the repository in
// question or nil if no deploy key has been created yet.
func (w *Workflow) GetDeployKeyID(repo *Repository) *int64 {
	key := fmt.Sprintf(deployKeyIDFormat, repo.Owner, repo.Name)
	value, exists := w.Status.Annotations[key]

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
	if w.Status.Annotations == nil {
		w.Status.Annotations = make(map[string]string)
	}
	w.Status.Annotations[fmt.Sprintf(deployKeyIDFormat, repo.Owner, repo.Name)] = fmt.Sprint(id)
}

// GetWebhookID returns the id of a Webhook associated to the repository in
// question or nil if no Webhook has been created yet.
func (w *Workflow) GetWebhookID() *int64 {
	repo := w.Spec.Repository
	key := fmt.Sprintf(webhookIDFormat, repo.Owner, repo.Name)
	value, exists := w.Status.Annotations[key]

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
	if w.Status.Annotations == nil {
		w.Status.Annotations = make(map[string]string)
	}
	repo := w.Spec.Repository
	w.Status.Annotations[fmt.Sprintf(webhookIDFormat, repo.Owner, repo.Name)] = fmt.Sprint(id)
}

// GetDeployKeysSecretName returns the name of the private SSH keys associated to
// this workflow.
func (w *Workflow) GetDeployKeysSecretName() string {
	return fmt.Sprintf("%s-ssh-private-keys", w.GetName())
}

// GetWebhookSecretName returns the name of the Webhook secret associated to this workflow.
func (w *Workflow) GetWebhookSecretName() string {
	return fmt.Sprintf("%s-webhook-secret", w.GetName())
}

// GetHooksURL returns the URL that Github Webhooks must use to triger this
// workflow.
func (w *Workflow) GetHooksURL() string {
	baseURL := w.Spec.Webhook.URL
	if strings.HasSuffix(baseURL, "/") {
		// Drop the trailing slash
		baseURL = baseURL[:len(baseURL)-1]
	}
	return baseURL + fmt.Sprintf(hooksEndpointPath, workflowsVersion, w.GetNamespace(), w.GetName())
}

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
	AdditionalRepositories []Repository `json:"additionalRepos,omitempty"`

	// Names of Github events that trigger this workflow.
	// +optional
	Events []string `json:"triggersOn,omitempty"`

	// Configures the workflow to run on events related to those branches.
	// +optional
	Branches []string `json:"branches,omitempty"`

	// Configures the workflow to run on push or pull_request events where
	// those paths have been modified (created, changed or deleted).
	// +optional
	Paths []string `json:"paths,omitempty"`

	// Default settings that will apply to all tasks in the workflow.
	// +optional
	Defaults *Defaults `json:"defaults,omitempty"`

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

	// Default branch for this repository.
	// +optional
	DefaultBranch string `json:"defaultBranch,omitempty"`

	// Authorizes task runs that make up this workflow to access the
	// repository in question.
	// +optional
	DeployKey *DeployKey `json:"deployKey,omitempty"`

	// Whether or not the repository is private.
	// +optional
	Private bool `json:"private,omitempty"`
}

// GetSSHPrivateKeyName returns the name of the SSH private key associated to this repository.
func (r *Repository) GetSSHPrivateKeyName() string {
	return fmt.Sprintf("%s_id_rsa", r.Name)
}

// IsReadOnlyDeployKey returns true if the associated deploy key is read only or false otherwise.
func (r *Repository) IsReadOnlyDeployKey() bool {
	if r.DeployKey == nil {
		return true
	}
	return r.DeployKey.ReadOnly
}

// NeedsSSHPrivateKeys returns true if the repository in question requires SSH
// private keys (i.e. it's a private repository or the configured deploy key has
// write permissions).
func (r *Repository) NeedsSSHPrivateKeys() bool {
	return r.Private || r.IsReadOnlyDeployKey()
}

// String satisfies fmt.Stringer interface.
func (r *Repository) String() string {
	return fmt.Sprintf("%s/%s", r.Owner, r.Name)
}

// Webhook contains information about a Github Webhook that triggers the workflow
// in question.
type Webhook struct {

	// The URL to which the payloads will be delivered
	URL string `json:"url"`
}

// DeployKey contains a few settings for the deploy keys associated to the workflow.
type DeployKey struct {

	// Whether or not the deploy key has read-only permissions on the repository.
	ReadOnly bool `json:"readOnly"`
}

// Defaults defines default settings to all tasks in the workflow.
type Defaults struct {

	// Docker/OCI image to serve as the container for the step in question.
	// +optional
	Image string `json:"image,omitempty"`

	// Specifies the template to create the pod associated to underlying Tekton TaskRun objects.
	// +optional
	PodTemplate *pipelinev1beta1.PodTemplate `json:"podTemplate,omitempty"`

	// Service account to be assigned to the underlying TaskRun object.
	// +optional
	ServiceAccount string `json:"serviceAccount,omitempty"`
}

// Task contains information about the tasks that make up the workflow.
type Task struct {

	// A map of environment variables that are available to all steps in the task.
	// +optional
	Env map[string]string `json:"env,omitempty"`

	// List of upstream tasks this task depends on.
	// +optional
	Need []string `json:"needs,omitempty"`

	// Execution parameters for this task.
	// +optional
	Params map[string]string `json:"params,omitempty"`

	// Specifies the template to create the pod associated to the underlying Tekton TaskRun object.
	// +optional
	PodTemplate *pipelinev1beta1.PodTemplate `json:"podTemplate,omitempty"`

	// How many times the task should be retried in case of failures.
	// +optional
	Retries int `json:"retries,omitempty"`

	// Assigns resources (CPU and memory) to the task.
	// +optional
	Resources corev1.ResourceList `json:"resources,omitempty"`

	// Service account to be assigned to the underlying TaskRun object.
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
	Use string `json:"uses,omitempty"`
}

// EmbeddedStep defines a step to be executed as part of a task.
type EmbeddedStep struct {

	// A map of environment variables that are available to step in question.
	// +optional
	Env map[string]string `json:"env,omitempty"`

	// Docker/OCI image to serve as the container for the step in question.
	// +optional
	Image string `json:"image,omitempty"`

	// The step's name.
	// +optional
	Name string `json:"name,omitempty"`

	// Runs command-line programs using the container's shell.
	// +optional
	Run string `json:"run,omitempty"`

	// Selects a built-in action to run as part of the task in question.
	// +optional
	Use BuiltInAction `json:"uses,omitempty"`

	// Step's working directory.
	// +optional
	WorkingDir string `json:"workingDir,omitempty"`
}

// BuiltInAction represents a set of comon actions in CI pipelines (such as
// checking out code) which are provided out of the box by workflows.
type BuiltInAction string

const (
	CheckoutAction BuiltInAction = "checkout"
)

// WorkflowStatus defines the observed state of Workflow
type WorkflowStatus struct {
	duckv1.Status `json:",inline"`
}

const (
	WorkflowConditionReady = apis.ConditionReady
)

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
