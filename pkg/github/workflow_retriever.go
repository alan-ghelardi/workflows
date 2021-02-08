package github

import (
	"context"
	"fmt"

	"github.com/ghodss/yaml"
	"github.com/google/go-github/v33/github"
	workflowsv1alpha1 "github.com/nubank/workflows/pkg/apis/workflows/v1alpha1"
)

// WorkflowRetriever defines the interface for retrieving workflows stored in
// Github repositories alongside the code that they build, test, release, etc.
type WorkflowRetriever interface {
	GetWorkflowContent(ctx context.Context, workflow *workflowsv1alpha1.Workflow, filePath, ref string) (*workflowsv1alpha1.Workflow, error)
}

// defaultWorkflowRetriever is the default implementation of WorkflowRetriever
// interface.
type defaultWorkflowRetriever struct {
	service *github.Client
}

// GetWorkflowContent implements WorkflowRetriever.GetWorkflowContent.
func (d *defaultWorkflowRetriever) GetWorkflowContent(ctx context.Context, workflow *workflowsv1alpha1.Workflow, filePath, ref string) (*workflowsv1alpha1.Workflow, error) {
	content, _, response, err := d.service.Repositories.GetContents(ctx,
		workflow.Spec.Repository.Owner,
		workflow.Spec.Repository.Name,
		filePath,
		&github.RepositoryContentGetOptions{Ref: ref})

	if response != nil && response.StatusCode == 404 {
		return nil, &NotFoundError{msg: fmt.Sprintf("Unable to find workflow %s", workflow.GetName())}
	}

	if err != nil {
		return nil, err
	}

	raw, err := content.GetContent()
	if err != nil {
		return nil, err
	}

	var w workflowsv1alpha1.Workflow
	if err := yaml.Unmarshal([]byte(raw), &w); err != nil {
		return nil, err
	}

	return &w, nil
}
