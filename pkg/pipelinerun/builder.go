package pipelinerun

import (
	"fmt"

	"github.com/nubank/workflows/pkg/github"
	workflowsv1alpha1 "github.com/nubank/workflows/pkg/apis/workflows/v1alpha1"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// defaultImage is the image used in steps when none is set.
const defaultImage = "gcr.io/google-containers/busybox"

// Builder builds Tekton PipelineRun objects.
type Builder struct {
	workflow *workflowsv1alpha1.Workflow
	event    *github.Event
}

// From returns a new Builder object that builds Tekton PipelineRuns from the provided Workflow.
func From(workflow *workflowsv1alpha1.Workflow) *Builder {
	return &Builder{workflow: workflow}
}

// And returns the same Builder object with the supplied Event object set.
func (b *Builder) And(event *github.Event) *Builder {
	b.event = event
	return b
}

// Build returns a new PipelineRun object.
func (b *Builder) Build() *pipelinev1beta1.PipelineRun {
	pipelineRun := &pipelinev1beta1.PipelineRun{ObjectMeta: metav1.ObjectMeta{GenerateName: fmt.Sprintf("%s-run", b.workflow.GetName()),
		Namespace: b.workflow.GetNamespace()},
		Spec: pipelinev1beta1.PipelineRunSpec{PipelineSpec: b.buildPipelineSpec(),
			TaskRunSpecs: b.buildTaskRunSpecs(),
		},
	}

	return pipelineRun
}

func (b *Builder) buildPipelineSpec() *pipelinev1beta1.PipelineSpec {
	pipelineSpec := &pipelinev1beta1.PipelineSpec{Description: b.workflow.Spec.Description,
		Tasks: b.buildPipelineTasks(),
	}

	return pipelineSpec
}

func (b *Builder) buildPipelineTasks() []pipelinev1beta1.PipelineTask {
	pipelineTasks := make([]pipelinev1beta1.PipelineTask, 0)
	for taskName, task := range b.workflow.Spec.Tasks {
		pipelineTasks = append(pipelineTasks, b.buildPipelineTask(taskName, task))
	}
	return pipelineTasks
}

func (b *Builder) buildPipelineTask(taskName string, task *workflowsv1alpha1.Task) pipelinev1beta1.PipelineTask {
	pipelineTask := pipelinev1beta1.PipelineTask{Name: taskName}

	if task.Use != "" {
		pipelineTask.TaskRef = &pipelinev1beta1.TaskRef{Name: task.Use}
	} else {
		pipelineTask.TaskSpec = b.buildEmbededTask(task)
	}

	pipelineTask.Retries = task.Retries

	pipelineTask.Timeout = task.Timeout

	return pipelineTask
}

func (b *Builder) buildEmbededTask(task *workflowsv1alpha1.Task) *pipelinev1beta1.EmbeddedTask {
	embededTask := &pipelinev1beta1.EmbeddedTask{}

	if task.Env != nil || task.Resources != nil {
		embededTask.StepTemplate = b.buildStepTemplate(task)
	}
	embededTask.Steps = make([]pipelinev1beta1.Step, 0)

	for _, step := range task.Steps {
		embededTask.Steps = append(embededTask.Steps, b.buildStep(step))
	}

	return embededTask
}

func (b *Builder) buildStepTemplate(task *workflowsv1alpha1.Task) *corev1.Container {
	stepTemplate := &corev1.Container{}

	if task.Env != nil {
		stepTemplate.Env = b.buildEnv(task.Env)
	}

	if task.Resources != nil {
		// Assume that requests and limits have the same values.
		stepTemplate.Resources = corev1.ResourceRequirements{Requests: task.Resources,
			Limits: task.Resources,
		}
	}

	return stepTemplate
}

func (b *Builder) buildStep(embeddedStep workflowsv1alpha1.EmbeddedStep) pipelinev1beta1.Step {
	step := pipelinev1beta1.Step{}

	if embeddedStep.Image != "" {
		step.Image = embeddedStep.Image
	} else {
		step.Image = defaultImage
	}

	step.ImagePullPolicy = "Always"

	if embeddedStep.Name != "" {
		step.Name = embeddedStep.Name
	}

	step.Script = fmt.Sprintf(`#!/usr/bin/env sh
set -o errexit
set -o nounset
%s`, embeddedStep.Run)

	if embeddedStep.Env != nil {
		step.Env = b.buildEnv(embeddedStep.Env)
	}

	return step
}

func (b *Builder) buildEnv(env map[string]string) []corev1.EnvVar {
	envVars := make([]corev1.EnvVar, 0)

	for name, value := range env {
		envVars = append(envVars, corev1.EnvVar{Name: name,
			Value: value,
		})
	}

	return envVars
}

func (b *Builder) buildTaskRunSpecs() []pipelinev1beta1.PipelineTaskRunSpec {
	taskRunSpecs := make([]pipelinev1beta1.PipelineTaskRunSpec, 0)
	for taskName, task := range b.workflow.Spec.Tasks {
		var taskRunSpec *pipelinev1beta1.PipelineTaskRunSpec

		if task.ServiceAccount != "" {
			if taskRunSpec == nil {
				taskRunSpec = &pipelinev1beta1.PipelineTaskRunSpec{PipelineTaskName: taskName}
			}
			taskRunSpec.TaskServiceAccountName = task.ServiceAccount
		}

		if task.PodTemplate != nil {
			if taskRunSpec == nil {
				taskRunSpec = &pipelinev1beta1.PipelineTaskRunSpec{PipelineTaskName: taskName}
			}
			taskRunSpec.TaskPodTemplate = task.PodTemplate
		}
		if taskRunSpec != nil {
			taskRunSpecs = append(taskRunSpecs, *taskRunSpec)
		}
	}

	return taskRunSpecs
}
