package runner

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"knative.dev/pkg/logging"

	"github.com/google/go-github/v33/github"
	tektonclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	workflowsclientset "github.com/nubank/workflows/pkg/client/clientset/versioned"
	"github.com/nubank/workflows/pkg/pipelinerun"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

// githubSignatureHeader defines the header name for hash signatures sent by Github.
const githubSignatureHeader = "X-Hub-Signature-256"

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
	Body           []byte
	Event          *github.Event
	NamespacedName types.NamespacedName
	HashSignature  []byte
}

// Response is an internal representation of a HTTP response.
// For convenience the same struct contains the HTTP status and other attributes
// that should be marshaled in the JSON format.
type Response struct {
	Status  int     `json:"-"`
	EventID *string `json:"event_id,omitempty"`
	Message string  `json:"message"`
}

// RunWorkflow takes a request containing information sent by Github Webhooks
// and creates a Tekton PipelineRun object.
func (e *EventSink) RunWorkflow(ctx context.Context, req *Request) Response {
	logger := logging.FromContext(ctx)
	eventID := req.Event.ID

	workflow, err := e.WorkflowsClientSet.WorkflowsV1alpha1().Workflows(req.NamespacedName.Namespace).Get(ctx, req.NamespacedName.Name, metav1.GetOptions{})
	if err != nil {
		logger.Error(err, "Error reading workflow")
		if apierrors.IsNotFound(err) {
			return Response{Status: 404,
				EventID: eventID,
				Message: fmt.Sprintf("Workflow %s not found", req.NamespacedName),
			}
		} else {
			return Response{Status: 500,
				EventID: eventID,
				Message: fmt.Sprintf("An internal error has occurred while reading workflow %s", req.NamespacedName),
			}
		}
	}

	secret, err := e.KubeClientSet.CoreV1().Secrets(workflow.GetNamespace()).Get(ctx, workflow.GetWebhookSecretName(), metav1.GetOptions{})
	if err != nil {
		logger.Error(err, "Error reading Webhook secret")
		return Response{Status: 500,
			EventID: eventID,
			Message: "An internal error has occurred while verifying the request signature",
		}
	}

	webhookSecret, err := e.readWebhookSecret(secret)
	if err != nil {
		logger.Error(err, "Unable to read Webhook secret")
		return Response{Status: 500,
			EventID: eventID,
			Message: "An internal error has occurred while verifying the request signature",
		}
	}

	if valid, message := e.verifySignature(req, webhookSecret); !valid {
		logger.Info(message)
		return Response{Status: 403,
			EventID: eventID,
			Message: message,
		}
	} else {
		logger.Debug(message)
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
			EventID: eventID,
			Message: fmt.Sprintf("An internal error has occurred while creating the PipelineRun for workflow %s", req.NamespacedName),
		}
	}

	logger.Infow("PipelineRun has been successfully created", "tekton.dev/pipeline-run", createdPipelineRun.GetName())

	return Response{Status: 201,
		EventID: eventID,
		Message: fmt.Sprintf("PipelineRun %s has been successfully created", createdPipelineRun.GetName()),
	}
}

// readWebhookSecret returns the decoded representation of the Webhook secret held by the supplied Secret object.
func (e *EventSink) readWebhookSecret(secret *corev1.Secret) ([]byte, error) {
	webhookSecret, exists := secret.Data["secret-token"]
	if !exists {
		return nil, fmt.Errorf("Key secret-token is missing in Secret object %s", types.NamespacedName{Namespace: secret.GetNamespace(), Name: secret.GetName()})
	}

	decodedWebhookSecret := make([]byte, base64.StdEncoding.DecodedLen(len(webhookSecret)))
	bytesWritten, err := base64.StdEncoding.Decode(decodedWebhookSecret, webhookSecret)
	if err != nil {
		return nil, fmt.Errorf("Error decoding Webhook secret from Secret %s: %w", types.NamespacedName{Namespace: secret.GetNamespace(), Name: secret.GetName()}, err)
	}

	return decodedWebhookSecret[:bytesWritten], nil
}

// verifySignature validates the payload sent by Github Webhooks by calculating
// a hash signature using the provided key and comparing it with the signature
// sent along with the request.
// For further details about the algorithm, please see:
// https://docs.github.com/en/free-pro-team@latest/developers/webhooks-and-events/securing-your-webhooks.
func (e *EventSink) verifySignature(req *Request, webhookSecret []byte) (bool, string) {
	if req.HashSignature == nil || len(req.HashSignature) == 0 {
		return false, fmt.Sprintf("Access denied: Github signature header %s is missing", githubSignatureHeader)
	}

	hash := hmac.New(sha256.New, webhookSecret)
	hash.Write(req.Body)
	digest := hash.Sum(nil)

	signature := make([]byte, hex.EncodedLen(len(digest)))
	hex.Encode(signature, digest)
	signature = append([]byte("sha256="), signature...)

	if !hmac.Equal(req.HashSignature, signature) {
		return false, "Access denied: HMAC signatures don't match. The request signature we calculated does not match the provided signature."
	}

	return true, "Access permitted: the signature we calculated match the provided signature."
}
