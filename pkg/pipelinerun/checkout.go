package pipelinerun

import (
	"bytes"
	"fmt"
	"text/template"

	workflowsv1alpha1 "github.com/nubank/workflows/pkg/apis/workflows/v1alpha1"
	"github.com/nubank/workflows/pkg/github"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
)

const (

	// Image used for checking out code.
	gitInitImage = "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init:v0.18.1"

	// Name of the workspace used to host Github projects that are checked out.
	projectsWorkspace = "projects"

	// Path to the projects workspace using parameter expansion.
	projectsWorkspaceExpr = "$(workspaces.projects.path)"

	// Path inside the container where SSH private keys will be mounted.
	sshPrivateKeysMountPath = "/var/run/secrets/workflows"

	// Name of the volume used to mount SSH private keys into steps.
	sshPrivateKeysVolumeName = "ssh-private-keys"
)

// Shell script that performs the logic to check out repositories.
// It is parsed as a Go template to facilitate the interpolation of parameters
// that are resolved dynamically.
var checkoutScript = template.Must(template.New("checkout").
	Parse(`#!/usr/bin/env sh
set -euo pipefail
{{if .sshPrivateKey}}
mkdir -p ~/.ssh
cat > ~/.ssh/config<<EOF
Host github.com
  User git
  Hostname github.com
  IdentityFile {{.sshPrivateKey}}
EOF
{{end}}
/ko-app/git-init \
    -url="{{.URL}}" \
    -revision="{{.Revision}}" \
    -path "{{.Destination}}" \
    -sslVerify="true" \
    -submodules="true" \
    -depth "{{.Dept}}"

cd {{.Destination}}
echo -n "$(git rev-parse HEAD)" > /tekton/results/{{.ResultName}}`))

// Checkout is a built-in action for checking out Github repositories into Tekton task.
type Checkout struct {
	workflow *workflowsv1alpha1.Workflow
	event    *github.Event
}

// CheckoutOptions represents a few options to be passed to the checkout script.
type CheckoutOptions struct {
	Dept          int
	Destination   string
	ResultName    string
	Revision      string
	sshPrivateKey string
	URL           string
}

// BuildStep implements BuiltInAction.
func (c *Checkout) BuildStep(embeddedStep workflowsv1alpha1.EmbeddedStep) pipelinev1beta1.Step {
	return buildCheckoutStep(embeddedStep, c.workflow.Spec.Repository, c.event)
}

func buildCheckoutStep(embeddedStep workflowsv1alpha1.EmbeddedStep, repo *workflowsv1alpha1.Repository, event *github.Event) pipelinev1beta1.Step {
	options := BuildCheckoutOptions(repo, event)
	step := pipelinev1beta1.Step{
		Container: corev1.Container{
			Image: gitInitImage,
		},
		Script: renderCheckoutScript(options),
	}

	if embeddedStep.Name != "" {
		step.Name = embeddedStep.Name
	} else {
		step.Name = fmt.Sprintf("checkout-%s", repo.Name)
	}

	if repo.Private {
		step.VolumeMounts = []corev1.VolumeMount{
			{Name: sshPrivateKeysVolumeName,
				ReadOnly:  true,
				MountPath: sshPrivateKeysMountPath,
			},
		}
	}

	return step
}

// BuildCheckoutOptions returns options that control the behavior of the
// checkout process for the supplied repository.
func BuildCheckoutOptions(repo *workflowsv1alpha1.Repository, event *github.Event) CheckoutOptions {
	options := CheckoutOptions{
		Dept:        1,
		Destination: fmt.Sprintf("%s/%s", projectsWorkspaceExpr, repo.Name),
		ResultName:  resultName(repo),
	}

	if repo.Private {
		options.sshPrivateKey = fmt.Sprintf("%s/%s", projectsWorkspaceExpr, repo.GetSSHPrivateKeyName())
		options.URL = fmt.Sprintf("git@github.com:%s/%s.git", repo.Owner, repo.Name)
	} else {
		options.URL = fmt.Sprintf("https://github.com/%s/%s", repo.Owner, repo.Name)
	}

	if event.HeadCommitSHA != "" {
		options.Revision = event.HeadCommitSHA
	} else {
		options.Revision = repo.DefaultBranch
	}

	return options
}

// resultName returns the name of the task result that store the precise commit
// fetched from the supplied repository.
func resultName(repo *workflowsv1alpha1.Repository) string {
	return fmt.Sprintf("%s-commit", repo.Name)
}

func renderCheckoutScript(options CheckoutOptions) string {
	var buffer bytes.Buffer

	err := checkoutScript.ExecuteTemplate(&buffer, "checkout", options)
	if err != nil {
		panic(err)
	}

	return buffer.String()
}

// ModifyEmbeddedTask implements BuiltInAction.
func (c *Checkout) ModifyEmbeddedTask(task *pipelinev1beta1.EmbeddedTask) {
	// Create the projects workspace
	if task.Workspaces == nil {
		task.Workspaces = make([]pipelinev1beta1.WorkspaceDeclaration, 1)
	}
	task.Workspaces = append(task.Workspaces, pipelinev1beta1.WorkspaceDeclaration{
		Name: projectsWorkspace,
	})

	// For convenience, if steps don't declare a working directory, set the
	// path where the main project was checked out as the default one.
	for _, step := range task.Steps[1:] {
		if step.WorkingDir == "" {
			step.WorkingDir = fmt.Sprintf("%s/%s", projectsWorkspaceExpr, c.workflow.Spec.Repository.Name)
		}
	}

	// Create results for exposing the precise commit used to fetch repositories
	repositories := c.workflow.GetRepositories()

	if task.Results != nil {
		task.Results = make([]pipelinev1beta1.TaskResult, len(repositories))
	}

	for _, repo := range repositories {
		task.Results = append(task.Results, pipelinev1beta1.TaskResult{
			Name: resultName(&repo),
		})
	}

	// Control whether a volume for projecting SSH private keys should be
	// mounted
	needsSSHPrivateKeys := c.workflow.Spec.Repository.Private

	// Inject steps for checking out secondary repositories
	if len(c.workflow.Spec.AdditionalRepositories) != 0 {
		steps := task.Steps[:1]
		// Create an empty event since secondary repositories aren't bound to Github events.
		event := &github.Event{}

		for _, repo := range c.workflow.Spec.AdditionalRepositories {
			embeddedStep := workflowsv1alpha1.EmbeddedStep{
				Name: fmt.Sprintf("checkout-%s", repo.Name),
				Use:  workflowsv1alpha1.CheckoutAction,
			}

			steps = append(steps, buildCheckoutStep(embeddedStep, &repo, event))

			if repo.Private {
				needsSSHPrivateKeys = true
			}
		}

		steps = append(steps, task.Steps[1:]...)
		task.Steps = steps
	}

	// If SSH private keys are required (i.e. there are private repositories
	// associated to this workflow), create the volume to mount the secret
	// containing the deploy key into the step.
	if needsSSHPrivateKeys {
		task.Volumes = []corev1.Volume{{
			Name: sshPrivateKeysVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: c.workflow.GetDeployKeysSecretName(),
				},
			},
		},
		}
	}
}

// ModifyPipelineTask implements BuitInAction.
func (c *Checkout) ModifyPipelineTask(task *pipelinev1beta1.PipelineTask) {
	if task.Workspaces == nil {
		task.Workspaces = make([]pipelinev1beta1.WorkspacePipelineTaskBinding, 1)
	}

	task.Workspaces = append(task.Workspaces, pipelinev1beta1.WorkspacePipelineTaskBinding{
		Name:      projectsWorkspace,
		Workspace: projectsWorkspace,
	})
}

// ModifyPipelineSpec implements BuiltInAction.
func (c *Checkout) ModifyPipelineSpec(pipeline *pipelinev1beta1.PipelineSpec) {
	if pipeline.Workspaces == nil {
		pipeline.Workspaces = make([]pipelinev1beta1.PipelineWorkspaceDeclaration, 1)
	}
	pipeline.Workspaces = append(pipeline.Workspaces, pipelinev1beta1.PipelineWorkspaceDeclaration{
		Name: projectsWorkspace,
	})
}

// ModifyPipelineRun implements BuiltInAction.
func (c *Checkout) ModifyPipelineRun(pipelineRun *pipelinev1beta1.PipelineRun) {
	if pipelineRun.Spec.Workspaces == nil {
		pipelineRun.Spec.Workspaces = make([]pipelinev1beta1.WorkspaceBinding, 1)
	}
	pipelineRun.Spec.Workspaces = append(pipelineRun.Spec.Workspaces, pipelinev1beta1.WorkspaceBinding{
		Name:     projectsWorkspace,
		EmptyDir: &corev1.EmptyDirVolumeSource{},
	})
}
