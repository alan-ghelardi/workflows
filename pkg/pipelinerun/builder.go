package pipelinerun

import (
	"fmt"

	"github.com/nubank/workflows/pkg/apis/config"
	workflowsv1alpha1 "github.com/nubank/workflows/pkg/apis/workflows/v1alpha1"
	"github.com/nubank/workflows/pkg/github"
	"github.com/nubank/workflows/pkg/variables"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Builder builds Tekton PipelineRun objects.
type Builder struct {
	builtInSteps map[BuiltInStep]bool
	defaults     *config.Defaults
	event        *github.Event
	replacements *variables.Replacements
	workflow     *workflowsv1alpha1.Workflow
}

// BuiltInStep is the interface that built-in, convenience steps provided by
// workflows must implement.
type BuiltInStep interface {
	// BuildStep converts an EmbeddedStep to a Tekton Step object.
	BuildStep(embeddedStep workflowsv1alpha1.EmbeddedStep) pipelinev1beta1.Step

	// PostEmbeddedTaskCreation is a hook called after the supplied embedded
	// task is created.
	PostEmbeddedTaskCreation(embeddedTask *pipelinev1beta1.EmbeddedTask)

	// PostPipelineTaskCreation is a hook called after the supplied pipeline
	// task is created.
	PostPipelineTaskCreation(pipelineTask *pipelinev1beta1.PipelineTask)

	// PostPipelineSpecCreation is a hook called after the supplied pipeline
	// spec is created.
	PostPipelineSpecCreation(pipelineSpec *pipelinev1beta1.PipelineSpec)

	// PostPipelineRunCreation is a hook called after the supplied pipeline
	// run is created.
	PostPipelineRunCreation(pipelineRun *pipelinev1beta1.PipelineRun)
}

// NewBuilder returns a new Builder object that builds Tekton PipelineRuns from
// the provided workflow and event.
func NewBuilder(workflow *workflowsv1alpha1.Workflow, event *github.Event) *Builder {
	return &Builder{
		builtInSteps: make(map[BuiltInStep]bool),
		defaults:     &config.Defaults{},
		event:        event,
		replacements: variables.MakeReplacements(workflow, event),
		workflow:     workflow,
	}
}

// WithDefaults returns the same Builder with the provided default configuration.
func (b *Builder) WithDefaults(defaults *config.Defaults) *Builder {
	b.defaults = defaults
	return b
}

// Build returns a new PipelineRun object.
func (b *Builder) Build() *pipelinev1beta1.PipelineRun {
	pipelineRun := &pipelinev1beta1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-run-", b.workflow.GetName()),
			Namespace:    b.workflow.GetNamespace(),
			Labels: map[string]string{
				"workflows.dev/workflow": b.workflow.GetName(),
			},
			Annotations: make(map[string]string),
		},
		Spec: pipelinev1beta1.PipelineRunSpec{
			PipelineSpec: b.buildPipelineSpec(),
			TaskRunSpecs: b.buildTaskRunSpecs(),
		},
	}

	b.copyLabelsAndAnnotations(pipelineRun)
	b.addDefaultLabelsAndAnnotations(pipelineRun)

	// Let built-in steps to modify the PipelineRun resource.
	for builtInStep := range b.builtInSteps {
		builtInStep.PostPipelineRunCreation(pipelineRun)
	}

	return pipelineRun
}

func (b *Builder) buildPipelineSpec() *pipelinev1beta1.PipelineSpec {
	pipelineSpec := &pipelinev1beta1.PipelineSpec{
		Description: b.workflow.Spec.Description,
		Tasks:       b.buildPipelineTasks(),
	}

	// Let built-in steps to modify the PipelineSpec
	for builtInStep := range b.builtInSteps {
		builtInStep.PostPipelineSpecCreation(pipelineSpec)
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

	if task.Use != nil {
		pipelineTask.TaskRef = task.Use
	} else {
		pipelineTask.TaskSpec = b.buildEmbededTask(task)
	}

	pipelineTask.RunAfter = task.Require

	pipelineTask.Retries = task.Retries

	pipelineTask.Timeout = task.Timeout

	// Let built-in steps to modify the pipeline task.
	for builtInStep := range b.builtInSteps {
		builtInStep.PostPipelineTaskCreation(&pipelineTask)
	}

	return pipelineTask
}

func (b *Builder) buildEmbededTask(task *workflowsv1alpha1.Task) *pipelinev1beta1.EmbeddedTask {
	embededTask := &pipelinev1beta1.EmbeddedTask{}

	if task.Env != nil || task.Resources != nil {
		embededTask.StepTemplate = b.buildStepTemplate(task)
	}
	embededTask.Steps = make([]pipelinev1beta1.Step, 0)

	for _, embeddedStep := range task.Steps {
		var step pipelinev1beta1.Step

		if embeddedStep.Use != "" {
			step = b.invokeBuiltInAction(embeddedStep)
		} else {
			step = b.buildStep(embeddedStep)
		}

		embededTask.Steps = append(embededTask.Steps, step)
	}

	// Let built-in steps to modify the embedded task
	for builtInStep := range b.builtInSteps {
		builtInStep.PostEmbeddedTaskCreation(embededTask)
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
	script := fmt.Sprintf(`#!/usr/bin/env sh
set -eu
%s`, embeddedStep.Run)

	step := pipelinev1beta1.Step{
		Container: corev1.Container{
			Name:            embeddedStep.Name,
			Image:           embeddedStep.Image,
			ImagePullPolicy: corev1.PullAlways,
			WorkingDir:      embeddedStep.WorkingDir,
		},
		Script: variables.Expand(script, b.replacements),
	}

	if embeddedStep.Env != nil {
		step.Env = b.buildEnv(embeddedStep.Env)
	}

	return step
}

func (b *Builder) invokeBuiltInAction(embeddedStep workflowsv1alpha1.EmbeddedStep) pipelinev1beta1.Step {
	var builtInStep BuiltInStep

	switch embeddedStep.Use {
	case workflowsv1alpha1.CheckoutStep:
		builtInStep = &Checkout{
			event:    b.event,
			workflow: b.workflow,
		}

	default:
		panic(fmt.Errorf("Unrecognized built-in step %s", embeddedStep.Use))
	}

	b.builtInSteps[builtInStep] = true

	return builtInStep.BuildStep(embeddedStep)
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

// copyLabelsAndAnnotations copies all labels and annotations in the workflow to
// the supplied pipelineRun.
// All variables declared in annotations will be substituted by values taken
// from the replacement context.
func (b *Builder) copyLabelsAndAnnotations(pipelineRun *pipelinev1beta1.PipelineRun) {
	for key, value := range b.workflow.Labels {
		pipelineRun.Labels[key] = value
	}

	for key, value := range b.workflow.Annotations {
		pipelineRun.Annotations[key] = variables.Expand(value, b.replacements)
	}
}

// addDefaultLabelsAndAnnotations adds labels and annotations declared in the
// default configuration to the PipelineRun.
// All variables declared in labels and/or annotations will be substituted by
// values taken from the replacement context.
func (b *Builder) addDefaultLabelsAndAnnotations(pipelineRun *pipelinev1beta1.PipelineRun) {
	for key, value := range b.defaults.Labels {
		if _, ok := pipelineRun.Labels[key]; !ok {
			pipelineRun.Labels[key] = variables.Expand(value, b.replacements)
		}
	}

	for key, value := range b.defaults.Annotations {
		if _, ok := pipelineRun.Annotations[key]; !ok {
			pipelineRun.Annotations[key] = variables.Expand(value, b.replacements)
		}
	}
}
