package hooklistener

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	workflowsclientset "github.com/nubank/workflows/pkg/client/clientset/versioned"
	tektonclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// New creates a HTTP server
func New(ctx context.Context) *http.Server {
	handler := newEventHandler(ctx)
	routes := initRoutes(ctx, handler)
	return newServer(ctx, routes)
}

// newServer returns a new HTTP server to start the hook listener API.
func newServer(ctx context.Context, routes *mux.Router) *http.Server {
	server := &http.Server{
		Addr: ":8080",
		BaseContext: func(listener net.Listener) context.Context {
			return ctx
		},
		Handler:      routes,
		IdleTimeout:  30 * time.Second,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 40 * time.Second,
	}

	return server
}

// newEventHandler returns a new EventHandler initializing all required client sets.
// It panics if a in-cluster config can't be obtained or if any client set fails
// to be created.
func newEventHandler(ctx context.Context) *EventHandler {
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(fmt.Errorf("Error creating in-cluster config: %w", err))
	}

	kubeClient := kubernetes.NewForConfigOrDie(config)
	tektonClient := tektonclientset.NewForConfigOrDie(config)
	workflowsClient := workflowsclientset.NewForConfigOrDie(config)
	return &EventHandler{
		KubeClientSet:      kubeClient,
		TektonClientSet:    tektonClient,
		WorkflowsClientSet: workflowsClient,
	}
}
