package runner

import (
	"context"
	"fmt"

	"github.com/nubank/workflows/pkg/filter"
	"github.com/nubank/workflows/pkg/github"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"knative.dev/pkg/logging"

	tektonclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	workflowsclientset "github.com/nubank/workflows/pkg/client/clientset/versioned"
	"github.com/nubank/workflows/pkg/pipelinerun"
	"github.com/nubank/workflows/pkg/secret"
	"k8s.io/client-go/kubernetes"
)

// EventSink handles incoming events from Github Webhooks by coordinating the
// execution of Tekton PipelineRuns.
type EventSink struct {

	// KubeClientSet allows us to talk to the k8s for core APIs.
	KubeClientSet kubernetes.Interface

	// TektonClientSet allows us to configure pipeline objects.
	TektonClientSet tektonclientset.Interface

	// WorkflowsClientSet allows us to retrieve workflow objects.
	WorkflowsClientSet workflowsclientset.Interface
}

// Request is an internal representation of HTTP requests, containing only
// relevant information.
type Request struct {
	Event          *github.Event
	NamespacedName types.NamespacedName
}

// Response is an internal representation of a HTTP response.
// For convenience the same struct contains the HTTP status and other attributes
// that should be marshaled into JSON format.
type Response struct {
	Status  int    `json:"-"`
	Message string `json:"message"`
}

// RunWorkflow takes a request containing information sent by Github Webhooks
// and creates a Tekton PipelineRun object.
func (e *EventSink) RunWorkflow(ctx context.Context, req *Request) Response {
	logger := logging.FromContext(ctx)

	workflow, err := e.WorkflowsClientSet.WorkflowsV1alpha1().Workflows(req.NamespacedName.Namespace).Get(ctx, req.NamespacedName.Name, metav1.GetOptions{})
	if err != nil {
		logger.Error(err, "Error reading workflow")
		if apierrors.IsNotFound(err) {
			return Response{
				Status:  404,
				Message: fmt.Sprintf("Workflow %s not found", req.NamespacedName),
			}
		} else {
			return Response{
				Status:  500,
				Message: fmt.Sprintf("An internal error has occurred while reading workflow %s", req.NamespacedName),
			}
		}
	}

	webhookSecret, err := e.KubeClientSet.CoreV1().Secrets(workflow.GetNamespace()).Get(ctx, workflow.GetWebhookSecretName(), metav1.GetOptions{})
	if err != nil {
		logger.Error(err, "Error reading Webhook secret")
		return Response{
			Status:  500,
			Message: "An internal error has occurred while verifying the request signature",
		}
	}

	webhookSecretToken, err := secret.GetSecretToken(webhookSecret)
	if err != nil {
		logger.Error(err, "Unable to read Webhook secret")
		return Response{
			Status:  500,
			Message: "An internal error has occurred while verifying the request signature",
		}
	}

	if valid, message := req.Event.VerifySignature(webhookSecretToken); !valid {
		logger.Info(message)
		return Response{
			Status:  403,
			Message: message,
		}
	}

	if workflowAccepted, message := filter.VerifyCriteria(workflow, req.Event); !workflowAccepted {
		logger.Info(message)
		return Response{
			Status:  422,
			Message: message,
		}
	}

	pipelineRun := pipelinerun.From(workflow).
		And(req.Event).
		Build()
	createdPipelineRun, err := e.TektonClientSet.TektonV1beta1().
		PipelineRuns(workflow.GetNamespace()).
		Create(ctx, pipelineRun, metav1.CreateOptions{})

	if err != nil {
		logger.Error(err, "Error creating PipelineRun object")
		return Response{Status: 500,
			Message: fmt.Sprintf("An internal error has occurred while creating the PipelineRun for workflow %s", req.NamespacedName),
		}
	}

	logger.Infow("PipelineRun has been successfully created", "tekton.dev/pipeline-run", createdPipelineRun.GetName())

	return Response{Status: 201,
		Message: fmt.Sprintf("PipelineRun %s has been successfully created", createdPipelineRun.GetName()),
	}
}
