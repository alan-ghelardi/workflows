package hooklistener

import (
	"context"
	"fmt"

	"github.com/nubank/workflows/pkg/apis/config"
	"github.com/nubank/workflows/pkg/filter"
	"github.com/nubank/workflows/pkg/github"
	"go.uber.org/zap"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"knative.dev/pkg/logging"

	workflowsclientset "github.com/nubank/workflows/pkg/client/clientset/versioned"
	"github.com/nubank/workflows/pkg/pipelinerun"
	"github.com/nubank/workflows/pkg/secret"
	tektonclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// EventHandler handles incoming events from Github Webhooks by coordinating the
// execution of Tekton PipelineRuns.
type EventHandler struct {

	// KubeClientSet allows us to talk to the k8s for core APIs.
	KubeClientSet kubernetes.Interface

	// TektonClientSet allows us to configure pipeline objects.
	TektonClientSet tektonclientset.Interface

	// WorkflowsClientSet allows us to retrieve workflow objects from the Kubernetes cluster.
	WorkflowsClientSet workflowsclientset.Interface

	// WorkflowRetriever allows us to retrieve workflows declared directly
	// in Github repositories.
	WorkflowRetriever *github.WorkflowRetriever
}

// triggerWorkflow takes the event delivered by a Github Webhook and creates a
// Tekton PipelineRun.
func (e *EventHandler) triggerWorkflow(ctx context.Context, namespacedName types.NamespacedName, event *github.Event) *Response {
	logger := logging.FromContext(ctx)

	workflow, err := e.WorkflowsClientSet.WorkflowsV1alpha1().Workflows(namespacedName.Namespace).Get(ctx, namespacedName.Name, metav1.GetOptions{})
	if err != nil {
		logger.Error("Error reading workflow", zap.Error(err))
		if apierrors.IsNotFound(err) {
			return NotFound(fmt.Sprintf("Workflow %s not found", namespacedName))
		} else {
			return InternalServerError(fmt.Sprintf("An internal error has occurred while reading workflow %s", namespacedName))
		}
	}

	webhookSecret, err := e.KubeClientSet.CoreV1().Secrets(workflow.GetNamespace()).Get(ctx, workflow.GetWebhookSecretName(), metav1.GetOptions{})
	if err != nil {
		logger.Error("Error reading Webhook secret", zap.Error(err))
		return InternalServerError("An internal error has occurred while verifying the request signature")
	}

	webhookSecretToken, err := secret.GetSecretToken(webhookSecret)
	if err != nil {
		logger.Error("Unable to read Webhook secret", zap.Error(err))
		return InternalServerError("An internal error has occurred while verifying the request signature")
	}

	if valid, message := event.VerifySignature(webhookSecretToken); !valid {
		return Forbidden(message)
	}

	if workflowAccepted, message := filter.VerifyCriteria(workflow, event); !workflowAccepted {
		logger.Info(message)
		return Forbidden(message)
	}

	defaults := config.Get(ctx).Defaults
	pipelineRun := pipelinerun.NewBuilder(workflow, event).WithDefaults(defaults).Build()
	createdPipelineRun, err := e.TektonClientSet.TektonV1beta1().PipelineRuns(workflow.GetNamespace()).Create(ctx, pipelineRun, metav1.CreateOptions{})

	if err != nil {
		logger.Error("Error creating PipelineRun object", zap.Error(err))
		return InternalServerError(fmt.Sprintf("An internal error has occurred while creating the PipelineRun for workflow %s", namespacedName))
	}

	logger.Infow("PipelineRun has been successfully created", "tekton.dev/pipeline-run", createdPipelineRun.GetName())
	return Created(fmt.Sprintf("PipelineRun %s has been successfully created", createdPipelineRun.GetName()))
}
