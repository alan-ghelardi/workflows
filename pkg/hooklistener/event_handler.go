package hooklistener

import (
	"context"
	"fmt"

	"github.com/nubank/workflows/pkg/apis/config"
	"github.com/nubank/workflows/pkg/filters"
	"github.com/nubank/workflows/pkg/github"
	"github.com/nubank/workflows/pkg/secrets"
	"go.uber.org/zap"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"knative.dev/pkg/logging"

	workflowsv1alpha1 "github.com/nubank/workflows/pkg/apis/workflows/v1alpha1"
	workflowsclientset "github.com/nubank/workflows/pkg/client/clientset/versioned"
	"github.com/nubank/workflows/pkg/pipelinerun"
	tektonclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// EventHandler handles incoming events from Github Webhooks by coordinating the
// execution of Tekton PipelineRuns.
type EventHandler struct {

	// configStore holds a collection of configurations required by the
	// EventHandler.
	configStore *config.Store

	// kubeClientSet allows us to talk to the k8s for core APIs.
	kubeClientSet kubernetes.Interface

	// tektonClientSet allows us to configure pipeline objects.
	tektonClientSet tektonclientset.Interface

	// workflowsClientSet allows us to retrieve workflow objects from the Kubernetes cluster.
	workflowsClientSet workflowsclientset.Interface

	// workflowReader allows us to read workflows declared directly in
	// Github repositories.
	workflowReader github.WorkflowReader
}

// triggerWorkflow takes the event delivered by a Github Webhook and creates a
// Tekton PipelineRun.
func (e *EventHandler) triggerWorkflow(ctx context.Context, namespacedName types.NamespacedName, event *github.Event) *Response {
	logger := logging.FromContext(ctx)

	workflow, err := e.workflowsClientSet.WorkflowsV1alpha1().Workflows(namespacedName.Namespace).Get(ctx, namespacedName.Name, metav1.GetOptions{})
	if err != nil {
		logger.Error("Error reading workflow", zap.Error(err))
		if apierrors.IsNotFound(err) {
			return NotFound(fmt.Sprintf("Workflow %s not found", namespacedName))
		} else {
			return InternalServerError(fmt.Sprintf("An internal error has occurred while reading workflow %s", namespacedName))
		}
	}

	webhookSecret, err := e.kubeClientSet.CoreV1().Secrets(workflow.GetNamespace()).Get(ctx, workflow.GetWebhookSecretName(), metav1.GetOptions{})
	if err != nil {
		logger.Error("Error reading Webhook secret", zap.Error(err))
		return InternalServerError("An internal error has occurred while verifying the request signature")
	}

	webhookSecretToken, err := secrets.GetSecretToken(webhookSecret)
	if err != nil {
		logger.Error("Unable to read Webhook secret", zap.Error(err))
		return InternalServerError("An internal error has occurred while verifying the request signature")
	}

	if valid, message := event.VerifySignature(webhookSecretToken); !valid {
		return Forbidden(message)
	}

	if event.Name == "ping" {
		// Respond to the ping event sent by Github to check the Webhook validity.
		return OK("Webhook is all set!")
	}

	if w, err := e.getWorkflowFromRepository(ctx, workflow, event); err != nil {
		logger.Errorw("Error getting workflow from repository", zap.Error(err))
		return InternalServerError("An internal error has occurred while trying to read the workflow's configuration from the repository")
	} else if w != nil {
		workflow = w
	} else {
		logger.Info("Defaulting to the workflow's configuration read from the cluster")
	}

	if ok, message := filters.CanTrigger(workflow, event); !ok {
		logger.Info(message)
		return Accepted(message)
	}

	defaults := config.Get(ctx).Defaults
	pipelineRun := pipelinerun.NewBuilder(workflow, event).WithDefaults(defaults).Build()
	createdPipelineRun, err := e.tektonClientSet.TektonV1beta1().PipelineRuns(workflow.GetNamespace()).Create(ctx, pipelineRun, metav1.CreateOptions{})

	if err != nil {
		logger.Error("Error creating PipelineRun object", zap.Error(err))
		return InternalServerError(fmt.Sprintf("An internal error has occurred while creating the PipelineRun for workflow %s", namespacedName))
	}

	logger.Infow("PipelineRun has been successfully created", "tekton.dev/pipeline-run", createdPipelineRun.GetName())
	return Created(fmt.Sprintf("PipelineRun %s has been successfully created", createdPipelineRun.GetName()))
}

func (e *EventHandler) getWorkflowFromRepository(ctx context.Context, workflow *workflowsv1alpha1.Workflow, event *github.Event) (*workflowsv1alpha1.Workflow, error) {
	logger := logging.FromContext(ctx)

	if event.HeadCommitSHA == "" {
		logger.Info("Ignoring any workflow config possibly declared in the repository because the head commit is unknown")
		return nil, nil
	}

	defaults := config.Get(ctx).Defaults
	filePath := fmt.Sprintf("%s/%s.yaml", defaults.WorkflowsDir, workflow.GetName())
	w, err := e.workflowReader.GetWorkflowContent(ctx, workflow, filePath, event.HeadCommitSHA)
	if err != nil {
		if github.IsNotFound(err) {
			logger.Infof("Couldn't find the workflow's configuration at %s", filePath)
			return nil, nil
		} else {
			return nil, err
		}
	}

	logger.Infof("Successfully read workflow's configuration from %s", filePath)

	// Apply the same default values as those set by the admission controller.
	w.SetDefaults(ctx)

	return w, nil
}
