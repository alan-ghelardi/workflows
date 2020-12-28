package pipelinerun

import (
	"fmt"

	"github.com/google/go-github/v33/github"

	workflowsv1alpha1 "github.com/nubank/workflows/pkg/apis/workflows/v1alpha1"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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
		pipelineTask.TaskSpec = b.buildEmbededTask(taskName, task)
	}

	pipelineTask.Retries = task.Retries

	pipelineTask.Timeout = task.Timeout

	return pipelineTask
}

func (b *Builder) buildEmbededTask(taskName string, task *workflowsv1alpha1.Task) *pipelinev1beta1.EmbeddedTask {
	embededTask := &pipelinev1beta1.EmbeddedTask{}
	return embededTask
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
