package hooklistener

import (
	"context"
	"errors"
	"fmt"
	"testing"

	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"

	corev1 "k8s.io/api/core/v1"

	kubeclientset "k8s.io/client-go/kubernetes/fake"

	"k8s.io/apimachinery/pkg/runtime"

	"github.com/golang/mock/gomock"
	"github.com/nubank/workflows/pkg/apis/config"
	workflowsv1alpha1 "github.com/nubank/workflows/pkg/apis/workflows/v1alpha1"
	workflowsclientset "github.com/nubank/workflows/pkg/client/clientset/versioned/fake"
	"github.com/nubank/workflows/pkg/github"
	githubmocks "github.com/nubank/workflows/pkg/github/mocks"
	tektonclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/fake"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	k8stesting "k8s.io/client-go/testing"
	"knative.dev/pkg/logging"
)

func TestReturn404WhenTheWorkflowDoesntExist(t *testing.T) {
	handler := &EventHandler{workflowsClientSet: workflowsclientset.NewSimpleClientset(&workflowsv1alpha1.Workflow{ObjectMeta: metav1.ObjectMeta{Name: "anything",
		Namespace: "dev",
	},
	}),
	}

	ctx := logging.WithLogger(context.Background(), zap.NewNop().Sugar())

	namespacedName := types.NamespacedName{Namespace: "dev", Name: "test-1"}
	event := &github.Event{}

	response := handler.triggerWorkflow(ctx, namespacedName, event)

	wantStatus := 404
	wantMessage := "Workflow dev/test-1 not found"

	gotStatus := response.Status
	gotMessage := response.Payload.Message

	if wantStatus != gotStatus {
		t.Errorf("Want status %d, but got %d", wantStatus, gotStatus)
	}

	if wantMessage != gotMessage {
		t.Errorf("Want message %s, but got %s", wantMessage, gotMessage)
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

	handler := &EventHandler{workflowsClientSet: client}

	ctx := logging.WithLogger(context.Background(), zap.NewNop().Sugar())

	namespacedName := types.NamespacedName{Namespace: "dev", Name: "test-1"}
	event := &github.Event{}

	response := handler.triggerWorkflow(ctx, namespacedName, event)

	wantStatus := 500
	wantMessage := "An internal error has occurred while reading workflow dev/test-1"

	gotStatus := response.Status
	gotMessage := response.Payload.Message

	if wantStatus != gotStatus {
		t.Errorf("Want status %d, but got %d", wantStatus, gotStatus)
	}

	if wantMessage != gotMessage {
		t.Errorf("Want message %s, but got %s", wantMessage, gotMessage)
	}
}

func TestReturns500WhenTheWebhookSecretCannotBeLoaded(t *testing.T) {
	handler := &EventHandler{workflowsClientSet: workflowsclientset.NewSimpleClientset(&workflowsv1alpha1.Workflow{ObjectMeta: metav1.ObjectMeta{Name: "test-1",
		Namespace: "dev",
	},
	}),
		kubeClientSet: kubeclientset.NewSimpleClientset(),
	}

	ctx := logging.WithLogger(context.Background(), zap.NewNop().Sugar())

	namespacedName := types.NamespacedName{Namespace: "dev", Name: "test-1"}
	event := &github.Event{}

	response := handler.triggerWorkflow(ctx, namespacedName, event)

	wantStatus := 500
	wantMessage := "An internal error has occurred while verifying the request signature"

	gotStatus := response.Status
	gotMessage := response.Payload.Message

	if wantStatus != gotStatus {
		t.Errorf("Want status %d, but got %d", wantStatus, gotStatus)
	}

	if wantMessage != gotMessage {
		t.Errorf("Want message %s, but got %s", wantMessage, gotMessage)
	}
}

func TestReturns403WhenSignaturesDoNotMatch(t *testing.T) {
	handler := &EventHandler{workflowsClientSet: workflowsclientset.NewSimpleClientset(&workflowsv1alpha1.Workflow{ObjectMeta: metav1.ObjectMeta{Name: "test-1",
		Namespace: "dev",
	},
	}),
		kubeClientSet: kubeclientset.NewSimpleClientset(&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "test-1-webhook-secret",
			Namespace: "dev",
		},
			Data: map[string][]byte{
				"secret-token": []byte("secret"),
			},
		}),
	}

	ctx := logging.WithLogger(context.Background(), zap.NewNop().Sugar())

	namespacedName := types.NamespacedName{Namespace: "dev", Name: "test-1"}
	event := &github.Event{Body: []byte(`{
    "ref": "refs/heads/dev"
}`),
		// This digest was calculated with the key other-secret.
		HMACSignature: []byte("sha256=d8a72707bd05f566becba60815c77f1e2adddddfceed668ca4844489d12ded07"),
	}

	response := handler.triggerWorkflow(ctx, namespacedName, event)

	wantStatus := 403
	wantMessage := "Access denied: HMAC signatures don't match. The request signature we calculated does not match the provided signature."

	gotStatus := response.Status
	gotMessage := response.Payload.Message

	if wantStatus != gotStatus {
		t.Errorf("Want status %d, but got %d", wantStatus, gotStatus)
	}

	if wantMessage != gotMessage {
		t.Errorf("Want message %s, but got %s", wantMessage, gotMessage)
	}
}

func TestReturns200WhenGithubSendAPingEvent(t *testing.T) {
	handler := &EventHandler{workflowsClientSet: workflowsclientset.NewSimpleClientset(&workflowsv1alpha1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-1",
			Namespace: "dev",
		},
		Spec: workflowsv1alpha1.WorkflowSpec{
			Repository: &workflowsv1alpha1.Repository{
				Owner: "my-org",
				Name:  "my-repo",
			},
		},
	}),
		kubeClientSet: kubeclientset.NewSimpleClientset(&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "test-1-webhook-secret",
			Namespace: "dev",
		},
			Data: map[string][]byte{
				"secret-token": []byte("secret"),
			},
		}),
	}

	ctx := logging.WithLogger(context.Background(), zap.NewNop().Sugar())

	namespacedName := types.NamespacedName{Namespace: "dev", Name: "test-1"}
	event := &github.Event{
		Body: []byte(`{
    "ref": "refs/heads/dev"
}`),
		// This digest was calculated with the key secret.
		HMACSignature: []byte("sha256=4ae9df17f8cc696722c87f771f0c60fa7b03d44488ae3e0f712f570c4e7a3888"),
		Name:          "ping",
	}

	response := handler.triggerWorkflow(ctx, namespacedName, event)

	wantStatus := 200
	gotStatus := response.Status

	if wantStatus != gotStatus {
		t.Errorf("Want status %d, but got %d", wantStatus, gotStatus)
	}
}

func TestReturns403WhenFiltersDoNotMatch(t *testing.T) {
	handler := &EventHandler{workflowsClientSet: workflowsclientset.NewSimpleClientset(&workflowsv1alpha1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-1",
			Namespace: "dev",
		},
		Spec: workflowsv1alpha1.WorkflowSpec{
			Repository: &workflowsv1alpha1.Repository{
				Owner: "my-org",
				Name:  "my-repo",
			},
			Events:   []string{"push"},
			Branches: []string{"main"},
		},
	}),
		kubeClientSet: kubeclientset.NewSimpleClientset(&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "test-1-webhook-secret",
			Namespace: "dev",
		},
			Data: map[string][]byte{
				"secret-token": []byte("secret"),
			},
		}),
	}

	ctx := logging.WithLogger(context.Background(), zap.NewNop().Sugar())

	namespacedName := types.NamespacedName{Namespace: "dev", Name: "test-1"}
	event := &github.Event{
		Body: []byte(`{
    "ref": "refs/heads/dev"
}`),
		// This digest was calculated with the key secret.
		HMACSignature: []byte("sha256=4ae9df17f8cc696722c87f771f0c60fa7b03d44488ae3e0f712f570c4e7a3888"),
		Name:          "push",
		Branch:        "john-patch1",
		Repository:    "my-org/my-repo",
	}

	response := handler.triggerWorkflow(ctx, namespacedName, event)

	wantStatus := 403
	wantMessage := "Workflow was rejected because Github event doesn't satisfy filter criteria: branch john-patch1 doesn't match filters [main]"

	gotStatus := response.Status
	gotMessage := response.Payload.Message

	if wantStatus != gotStatus {
		t.Errorf("Want status %d, but got %d", wantStatus, gotStatus)
	}

	if wantMessage != gotMessage {
		t.Errorf("Want message %s, but got %s", wantMessage, gotMessage)
	}
}

func TestReturns500WhenThePipelineRunCannotBeCreated(t *testing.T) {
	tektonClient := tektonclientset.NewSimpleClientset()
	tektonClient.PrependReactor("create", "pipelineruns", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, &pipelinev1beta1.PipelineRun{}, errors.New("Error creating pipelinerun")
	})

	handler := &EventHandler{workflowsClientSet: workflowsclientset.NewSimpleClientset(&workflowsv1alpha1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-1",
			Namespace: "dev",
		},
		Spec: workflowsv1alpha1.WorkflowSpec{
			Repository: &workflowsv1alpha1.Repository{
				Owner: "my-org",
				Name:  "my-repo",
			},
			Events:   []string{"push"},
			Branches: []string{"main"},
		},
	}),
		kubeClientSet: kubeclientset.NewSimpleClientset(&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "test-1-webhook-secret",
			Namespace: "dev",
		},
			Data: map[string][]byte{
				"secret-token": []byte("secret"),
			},
		}),
		tektonClientSet: tektonClient,
	}

	ctx := logging.WithLogger(context.Background(), zap.NewNop().Sugar())
	ctx = config.WithConfig(ctx, &config.Config{
		Defaults: &config.Defaults{},
	})

	namespacedName := types.NamespacedName{Namespace: "dev", Name: "test-1"}
	event := &github.Event{Body: []byte(`{
    "ref": "refs/heads/dev"
}`),
		// This digest was calculated with the key secret.
		HMACSignature: []byte("sha256=4ae9df17f8cc696722c87f771f0c60fa7b03d44488ae3e0f712f570c4e7a3888"),
		Name:          "push",
		Branch:        "main",
		Repository:    "my-org/my-repo",
	}

	response := handler.triggerWorkflow(ctx, namespacedName, event)

	wantStatus := 500
	wantMessage := "An internal error has occurred while creating the PipelineRun for workflow dev/test-1"

	gotStatus := response.Status
	gotMessage := response.Payload.Message

	if wantStatus != gotStatus {
		t.Errorf("Want status %d, but got %d", wantStatus, gotStatus)
	}

	if wantMessage != gotMessage {
		t.Errorf("Want message %s, but got %s", wantMessage, gotMessage)
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

	handler := &EventHandler{workflowsClientSet: workflowsclientset.NewSimpleClientset(&workflowsv1alpha1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-1",
			Namespace: "dev",
		},
		Spec: workflowsv1alpha1.WorkflowSpec{
			Repository: &workflowsv1alpha1.Repository{
				Owner: "my-org",
				Name:  "my-repo",
			},
			Events:   []string{"push"},
			Branches: []string{"main"},
		},
	}),
		kubeClientSet: kubeclientset.NewSimpleClientset(&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "test-1-webhook-secret",
			Namespace: "dev",
		},
			Data: map[string][]byte{
				"secret-token": []byte("secret"),
			},
		}),
		tektonClientSet: tektonClient,
	}

	ctx := logging.WithLogger(context.Background(), zap.NewNop().Sugar())
	ctx = config.WithConfig(ctx, &config.Config{
		Defaults: &config.Defaults{},
	})

	namespacedName := types.NamespacedName{Namespace: "dev", Name: "test-1"}
	event := &github.Event{
		Body: []byte(`{
    "ref": "refs/heads/dev"
}`),
		// This digest was calculated with the key secret.
		HMACSignature: []byte("sha256=4ae9df17f8cc696722c87f771f0c60fa7b03d44488ae3e0f712f570c4e7a3888"),
		Name:          "push",
		Branch:        "main",
		Repository:    "my-org/my-repo",
	}

	response := handler.triggerWorkflow(ctx, namespacedName, event)

	wantStatus := 201
	wantMessage := "PipelineRun test-1-run-123 has been successfully created"

	gotStatus := response.Status
	gotMessage := response.Payload.Message

	if wantStatus != gotStatus {
		t.Errorf("Want status %d, but got %d", wantStatus, gotStatus)
	}

	if wantMessage != gotMessage {
		t.Errorf("Want message %s, but got %s", wantMessage, gotMessage)
	}
}

func TestPipelineRunCreationWithWorkflowReadFromRepo(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	workflowReader := githubmocks.NewMockWorkflowReader(mockCtrl)

	tektonClient := tektonclientset.NewSimpleClientset()
	tektonClient.PrependReactor("create", "pipelineruns", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, &pipelinev1beta1.PipelineRun{ObjectMeta: metav1.ObjectMeta{Name: "test-1-run-123",
			Namespace: "dev",
		},
		}, nil
	})

	workflow := &workflowsv1alpha1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-1",
			Namespace: "dev",
		},
		Spec: workflowsv1alpha1.WorkflowSpec{
			Repository: &workflowsv1alpha1.Repository{
				Owner: "my-org",
				Name:  "my-repo",
			},
			Events:   []string{"push"},
			Branches: []string{"main"},
		},
	}

	// Simulate a change made by users in the repository version of this
	// workflow.
	workflowFromRepo := workflow.DeepCopy()
	workflowFromRepo.Spec.Events = []string{"pull_request"}

	handler := &EventHandler{workflowsClientSet: workflowsclientset.NewSimpleClientset(workflow),
		kubeClientSet: kubeclientset.NewSimpleClientset(&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "test-1-webhook-secret",
			Namespace: "dev",
		},
			Data: map[string][]byte{
				"secret-token": []byte("secret"),
			},
		}),
		tektonClientSet: tektonClient,
		workflowReader:  workflowReader,
	}

	ctx := logging.WithLogger(context.Background(), zap.NewNop().Sugar())
	ctx = config.WithConfig(ctx, &config.Config{
		Defaults: &config.Defaults{WorkflowsDir: ".tektoncd/workflows"},
	})

	namespacedName := types.NamespacedName{Namespace: "dev", Name: "test-1"}
	event := &github.Event{
		Body: []byte(`{
    "ref": "refs/heads/dev"
}`),
		HeadCommitSHA: "abc123",
		// This digest was calculated with the key secret.
		HMACSignature: []byte("sha256=4ae9df17f8cc696722c87f771f0c60fa7b03d44488ae3e0f712f570c4e7a3888"),
		// Notice that the event matches the event expected by the repository version of the workflow.
		Name:       "pull_request",
		Branch:     "main",
		Repository: "my-org/my-repo",
	}

	// Mock setup
	workflowReader.EXPECT().
		GetWorkflowContent(gomock.Eq(ctx), gomock.Eq(workflow), gomock.Eq(".tektoncd/workflows/test-1.yaml"), gomock.Eq("abc123")).
		Return(workflowFromRepo, nil)

	response := handler.triggerWorkflow(ctx, namespacedName, event)

	wantStatus := 201
	gotStatus := response.Status
	if wantStatus != gotStatus {
		t.Errorf("Want status %d, but got %d", wantStatus, gotStatus)
	}
}

func TestReturns500WhenTheWorkflowConfigCannotBeRead(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	workflowReader := githubmocks.NewMockWorkflowReader(mockCtrl)

	workflow := &workflowsv1alpha1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-1",
			Namespace: "dev",
		},
		Spec: workflowsv1alpha1.WorkflowSpec{
			Repository: &workflowsv1alpha1.Repository{
				Owner: "my-org",
				Name:  "my-repo",
			},
			Events:   []string{"push"},
			Branches: []string{"main"},
		},
	}

	handler := &EventHandler{
		workflowsClientSet: workflowsclientset.NewSimpleClientset(workflow),
		kubeClientSet: kubeclientset.NewSimpleClientset(&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "test-1-webhook-secret",
			Namespace: "dev",
		},
			Data: map[string][]byte{
				"secret-token": []byte("secret"),
			},
		}),
		workflowReader: workflowReader,
	}

	ctx := logging.WithLogger(context.Background(), zap.NewNop().Sugar())
	ctx = config.WithConfig(ctx, &config.Config{
		Defaults: &config.Defaults{WorkflowsDir: ".tektoncd/workflows"},
	})

	namespacedName := types.NamespacedName{Namespace: "dev", Name: "test-1"}
	event := &github.Event{
		Body: []byte(`{
    "ref": "refs/heads/dev"
}`),
		HeadCommitSHA: "abc123",
		// This digest was calculated with the key secret.
		HMACSignature: []byte("sha256=4ae9df17f8cc696722c87f771f0c60fa7b03d44488ae3e0f712f570c4e7a3888"),
		Name:          "push",
		Branch:        "main",
		Repository:    "my-org/my-repo",
	}

	// Mock setup
	workflowReader.EXPECT().
		GetWorkflowContent(gomock.Eq(ctx), gomock.Eq(workflow), gomock.Eq(".tektoncd/workflows/test-1.yaml"), gomock.Eq("abc123")).
		Return(nil, fmt.Errorf("Boom!"))

	response := handler.triggerWorkflow(ctx, namespacedName, event)

	wantStatus := 500
	gotStatus := response.Status
	if wantStatus != gotStatus {
		t.Errorf("Want status %d, but got %d", wantStatus, gotStatus)
	}

	wantMessage := "An internal error has occurred while trying to read the workflow's configuration from the repository"
	gotMessage := response.Payload.Message
	if wantMessage != gotMessage {
		t.Errorf("Want message %s, but got %s", wantMessage, gotMessage)
	}
}
