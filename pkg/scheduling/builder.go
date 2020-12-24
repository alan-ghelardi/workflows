package scheduling

import (
	"fmt"

	"github.com/google/go-github/v33/github"

	workflowsv1alpha1 "github.com/nubank/workflows/pkg/apis/workflows/v1alpha1"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BuildPipelineRun
func buildPipelineRun(workflow *workflowsv1alpha1.Workflow, event *github.Event) *pipelinev1beta1.PipelineRun {
	pipelineRun := &pipelinev1beta1.PipelineRun{ObjectMeta: metav1.ObjectMeta{GenerateName: fmt.Sprintf("%s-run", workflow.GetName()),
		Namespace: workflow.GetNamespace()},
		Spec: pipelinev1beta1.PipelineRunSpec{PipelineSpec: buildPipelineSpec(workflow, event),
			TaskRunSpecs: buildTaskRunSpecs(workflow),
		},
	}

	return pipelineRun
}

// buildPipelineSpec ...
func buildPipelineSpec(workflow *workflowsv1alpha1.Workflow, event *github.Event) *pipelinev1beta1.PipelineSpec {
	pipelineSpec := &pipelinev1beta1.PipelineSpec{Description: workflow.Spec.Description,
		Tasks: buildPipelineTasks(workflow, event),
	}

	return pipelineSpec
}

// buildPipelineTasks ...
func buildPipelineTasks(workflow *workflowsv1alpha1.Workflow, event *github.Event) []pipelinev1beta1.PipelineTask {
	pipelineTasks := make([]pipelinev1beta1.PipelineTask, 0)
	for _, task := range workflow.Spec.Tasks {
		pipelineTasks = append(pipelineTasks, buildPipelineTask(&task, event))
	}
	return pipelineTasks
}

func buildPipelineTask(task *workflowsv1alpha1.Task, event *github.Event) pipelinev1beta1.PipelineTask {
	pipelineTask := pipelinev1beta1.PipelineTask{Name: task.Name}
	if task.TaskRef != "" {
		pipelineTask.TaskRef = &pipelinev1beta1.TaskRef{Name: task.TaskRef}
	} else {
		pipelineTask.TaskSpec = buildEmbededTask(task, event)
	}

	pipelineTask.Retries = task.Retries

	pipelineTask.Timeout = task.Timeout

	return pipelineTask
}

// buildEmbededTask ...
func buildEmbededTask(task *workflowsv1alpha1.Task, event *github.Event) *pipelinev1beta1.EmbeddedTask {
	embededTask := &pipelinev1beta1.EmbeddedTask{}
	return embededTask
}

// buildTaskRunSpecs ...
func buildTaskRunSpecs(workflow *workflowsv1alpha1.Workflow) []pipelinev1beta1.PipelineTaskRunSpec {
	taskRunSpecs := make([]pipelinev1beta1.PipelineTaskRunSpec, 0)
	for _, task := range workflow.Spec.Tasks {
		var taskRunSpec *pipelinev1beta1.PipelineTaskRunSpec

		if task.ServiceAccountName != "" {
			if taskRunSpec == nil {
				taskRunSpec = &pipelinev1beta1.PipelineTaskRunSpec{PipelineTaskName: task.Name}
			}
			taskRunSpec.TaskServiceAccountName = task.ServiceAccountName
		}

		if task.PodTemplate != nil {
			if taskRunSpec == nil {
				taskRunSpec = &pipelinev1beta1.PipelineTaskRunSpec{PipelineTaskName: task.Name}
			}
			taskRunSpec.TaskPodTemplate = task.PodTemplate
		}
		if taskRunSpec != nil {
			taskRunSpecs = append(taskRunSpecs, *taskRunSpec)
		}
	}

	return taskRunSpecs
}
