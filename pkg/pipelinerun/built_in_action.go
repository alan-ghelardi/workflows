package pipelinerun

import (
	workflowsv1alpha1 "github.com/nubank/workflows/pkg/apis/workflows/v1alpha1"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

type BuiltInAction interface {
	BuildStep(embeddedStep workflowsv1alpha1.EmbeddedStep) pipelinev1beta1.Step

	ModifyEmbeddedTask(embeddedTask *pipelinev1beta1.EmbeddedTask)

	ModifyPipelineTask(pipelineTask *pipelinev1beta1.PipelineTask)

	ModifyPipelineSpec(pipelineSpec *pipelinev1beta1.PipelineSpec)

	ModifyPipelineRun(pipelineRun *pipelinev1beta1.PipelineRun)
}
