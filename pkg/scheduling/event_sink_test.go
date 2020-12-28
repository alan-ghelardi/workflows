package scheduling

import (
	"context"
	"errors"
	"testing"

	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"

	corev1 "k8s.io/api/core/v1"

	kubeclientset "k8s.io/client-go/kubernetes/fake"

	"k8s.io/apimachinery/pkg/runtime"

	"github.com/google/go-github/v33/github"
	workflowsv1alpha1 "github.com/nubank/workflows/pkg/apis/workflows/v1alpha1"
	workflowsclientset "github.com/nubank/workflows/pkg/client/clientset/versioned/fake"
	tektonclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/fake"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	k8stesting "k8s.io/client-go/testing"
	"knative.dev/pkg/logging"
)

func TestReturn404WhenTheWorkflowDoesntExist(t *testing.T) {
	sink := &EventSink{WorkflowsClientSet: workflowsclientset.NewSimpleClientset(&workflowsv1alpha1.Workflow{ObjectMeta: metav1.ObjectMeta{Name: "anything",
		Namespace: "dev",
	},
	}),
	}

	ctx := logging.WithLogger(context.Background(), zap.NewNop().Sugar())

	request := &Request{NamespacedName: types.NamespacedName{Namespace: "dev", Name: "test-1"},
		Event: &github.Event{ID: github.String("1")},
	}

	response := sink.RunWorkflow(ctx, request)

	wantedStatus := 404
	wantedMessage := "Workflow dev/test-1 not found"

	gotStatus := response.Status
	gotMessage := response.Message

	if wantedStatus != gotStatus {
		t.Errorf("Wanted status %d, but got %d", wantedStatus, gotStatus)
	}

	if wantedMessage != gotMessage {
		t.Errorf("Wanted message %s, but got %s", wantedMessage, gotMessage)
	}
}

func TestReturns500WhenTheWorkflowCannotBeLoaded(t *testing.T) {
	client := workflowsclientset.NewSimpleClientset(&workflowsv1alpha1.Workflow{ObjectMeta: metav1.ObjectMeta{Name: "test-1",
		Namespace: "dev",
	},
	})

	client.PrependReactor("get", "workflows", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, &workflowsv1alpha1.Workflow{}, errors.New("Error creating workflow")
	})

	sink := &EventSink{WorkflowsClientSet: client}

	ctx := logging.WithLogger(context.Background(), zap.NewNop().Sugar())

	request := &Request{NamespacedName: types.NamespacedName{Namespace: "dev", Name: "test-1"},
		Event: &github.Event{ID: github.String("1")},
	}

	response := sink.RunWorkflow(ctx, request)

	wantedStatus := 500
	wantedMessage := "An internal error has occurred while reading workflow dev/test-1"

	gotStatus := response.Status
	gotMessage := response.Message

	if wantedStatus != gotStatus {
		t.Errorf("Wanted status %d, but got %d", wantedStatus, gotStatus)
	}

	if wantedMessage != gotMessage {
		t.Errorf("Wanted message %s, but got %s", wantedMessage, gotMessage)
	}
}

func TestReturns500WhenTheWebhookSecretCannotBeLoaded(t *testing.T) {
	sink := &EventSink{WorkflowsClientSet: workflowsclientset.NewSimpleClientset(&workflowsv1alpha1.Workflow{ObjectMeta: metav1.ObjectMeta{Name: "test-1",
		Namespace: "dev",
	},
	}),
		KubeClientSet: kubeclientset.NewSimpleClientset(),
	}

	ctx := logging.WithLogger(context.Background(), zap.NewNop().Sugar())

	request := &Request{NamespacedName: types.NamespacedName{Namespace: "dev", Name: "test-1"},
		Event: &github.Event{ID: github.String("1")},
	}

	response := sink.RunWorkflow(ctx, request)

	wantedStatus := 500
	wantedMessage := "An internal error has occurred while verifying the request signature"

	gotStatus := response.Status
	gotMessage := response.Message

	if wantedStatus != gotStatus {
		t.Errorf("Wanted status %d, but got %d", wantedStatus, gotStatus)
	}

	if wantedMessage != gotMessage {
		t.Errorf("Wanted message %s, but got %s", wantedMessage, gotMessage)
	}
}

func TestReturns403WhenSignaturesDoNotMatch(t *testing.T) {
	sink := &EventSink{WorkflowsClientSet: workflowsclientset.NewSimpleClientset(&workflowsv1alpha1.Workflow{ObjectMeta: metav1.ObjectMeta{Name: "test-1",
		Namespace: "dev",
	},
	}),
		KubeClientSet: kubeclientset.NewSimpleClientset(&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "test-1-webhook-secret",
			Namespace: "dev",
		},
			Data: map[string][]byte{
				// The word secret in base64 format
				"secret-token": []byte("c2VjcmV0"),
			},
		}),
	}

	ctx := logging.WithLogger(context.Background(), zap.NewNop().Sugar())

	request := &Request{Body: []byte(`{
    "ref": "refs/heads/dev"
}`),
		// This digest was calculated with the key other-secret.
		HashSignature:  []byte("sha256=d8a72707bd05f566becba60815c77f1e2adddddfceed668ca4844489d12ded07"),
		NamespacedName: types.NamespacedName{Namespace: "dev", Name: "test-1"},
		Event:          &github.Event{ID: github.String("1")},
	}

	response := sink.RunWorkflow(ctx, request)

	wantedStatus := 403
	wantedMessage := "Access denied: HMAC signatures don't match. The request signature we calculated does not match the provided signature."

	gotStatus := response.Status
	gotMessage := response.Message

	if wantedStatus != gotStatus {
		t.Errorf("Wanted status %d, but got %d", wantedStatus, gotStatus)
	}

	if wantedMessage != gotMessage {
		t.Errorf("Wanted message %s, but got %s", wantedMessage, gotMessage)
	}
}

func TestReturns500WhenThePipelineRunCannotBeCreated(t *testing.T) {
	tektonClient := tektonclientset.NewSimpleClientset()
	tektonClient.PrependReactor("create", "pipelineruns", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, &pipelinev1beta1.PipelineRun{}, errors.New("Error creating pipelinerun")
	})

	sink := &EventSink{WorkflowsClientSet: workflowsclientset.NewSimpleClientset(&workflowsv1alpha1.Workflow{ObjectMeta: metav1.ObjectMeta{Name: "test-1",
		Namespace: "dev",
	},
	}),
		KubeClientSet: kubeclientset.NewSimpleClientset(&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "test-1-webhook-secret",
			Namespace: "dev",
		},
			Data: map[string][]byte{
				// The word secret in base64 format
				"secret-token": []byte("c2VjcmV0"),
			},
		}),
		TektonClientSet: tektonClient,
	}

	ctx := logging.WithLogger(context.Background(), zap.NewNop().Sugar())

	request := &Request{Body: []byte(`{
    "ref": "refs/heads/dev"
}`),
		// This digest was calculated with the key secret.
		HashSignature:  []byte("sha256=4ae9df17f8cc696722c87f771f0c60fa7b03d44488ae3e0f712f570c4e7a3888"),
		NamespacedName: types.NamespacedName{Namespace: "dev", Name: "test-1"},
		Event:          &github.Event{ID: github.String("1")},
	}

	response := sink.RunWorkflow(ctx, request)

	wantedStatus := 500
	wantedMessage := "An internal error has occurred while creating the PipelineRun for workflow dev/test-1"

	gotStatus := response.Status
	gotMessage := response.Message

	if wantedStatus != gotStatus {
		t.Errorf("Wanted status %d, but got %d", wantedStatus, gotStatus)
	}

	if wantedMessage != gotMessage {
		t.Errorf("Wanted message %s, but got %s", wantedMessage, gotMessage)
	}
}

func TestReturns201WhenThePipelineRunIsCreated(t *testing.T) {
	tektonClient := tektonclientset.NewSimpleClientset()
		tektonClient.PrependReactor("create", "pipelineruns", func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, &pipelinev1beta1.PipelineRun{ObjectMeta: metav1.ObjectMeta{Name: "test-1-run-123",
				Namespace: "dev",
			},
			}, nil
	})

	sink := &EventSink{WorkflowsClientSet: workflowsclientset.NewSimpleClientset(&workflowsv1alpha1.Workflow{ObjectMeta: metav1.ObjectMeta{Name: "test-1",
		Namespace: "dev",
	},
	}),
		KubeClientSet: kubeclientset.NewSimpleClientset(&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "test-1-webhook-secret",
			Namespace: "dev",
		},
			Data: map[string][]byte{
				// The word secret in base64 format
				"secret-token": []byte("c2VjcmV0"),
			},
		}),
		TektonClientSet: tektonClient,
	}

	ctx := logging.WithLogger(context.Background(), zap.NewNop().Sugar())

	request := &Request{Body: []byte(`{
    "ref": "refs/heads/dev"
}`),
		// This digest was calculated with the key secret.
		HashSignature:  []byte("sha256=4ae9df17f8cc696722c87f771f0c60fa7b03d44488ae3e0f712f570c4e7a3888"),
		NamespacedName: types.NamespacedName{Namespace: "dev", Name: "test-1"},
		Event:          &github.Event{ID: github.String("1")},
	}

	response := sink.RunWorkflow(ctx, request)

	wantedStatus := 201
	wantedMessage := "PipelineRun test-1-run-123 has been successfully created"

	gotStatus := response.Status
	gotMessage := response.Message

	if wantedStatus != gotStatus {
		t.Errorf("Wanted status %d, but got %d", wantedStatus, gotStatus)
	}

	if wantedMessage != gotMessage {
		t.Errorf("Wanted message %s, but got %s", wantedMessage, gotMessage)
	}
}

func TestDeniesRequestsIfSignatureIsMissing(t *testing.T) {
	sink := &EventSink{}

	requests := []Request{{HashSignature: nil},
		{HashSignature: []byte{}}}

	wantedMessage := "Access denied: Github signature header X-Hub-Signature-256 is missing"

	for _, req := range requests {
		valid, message := sink.verifySignature(&req, []byte{})

		if valid {
			t.Error("Wanted an invalid signature result, but got a valid one")
		}

		if wantedMessage != message {
			t.Errorf("Wanted message %s, but got %s", wantedMessage, message)
		}
	}
}

func TestAcceptsRequestsWhenSignaturesMatch(t *testing.T) {
	sink := &EventSink{}

	request := &Request{Body: []byte(`{
    "ref": "refs/heads/dev"
}`),
		HashSignature: []byte("sha256=4ae9df17f8cc696722c87f771f0c60fa7b03d44488ae3e0f712f570c4e7a3888"),
	}

	webhookSecret := []byte("secret")

	if valid, _ := sink.verifySignature(request, webhookSecret); !valid {
		t.Errorf("Wanted a valid result for the signature validation, but got an invalid one")
	}
}

func TestRejectsRequestsWhenSignaturesDoNotMatch(t *testing.T) {
	sink := &EventSink{}

	request := &Request{Body: []byte(`{
    "ref": "refs/heads/dev"
}`),
		// This digest was calculated with the key other-secret.
		HashSignature: []byte("sha256=d8a72707bd05f566becba60815c77f1e2adddddfceed668ca4844489d12ded07"),
	}

	webhookSecret := []byte("secret")

	if valid, _ := sink.verifySignature(request, webhookSecret); valid {
		t.Errorf("Wanted a invalid result for the signature validation, but got a valid one")
	}
}
