package hooklistener

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/nubank/workflows/pkg/apis/config"
	workflowsclientset "github.com/nubank/workflows/pkg/client/clientset/versioned"
	tektonclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/configmap/informer"
	"knative.dev/pkg/logging"
)

// New creates a HTTP server
func New(ctx context.Context) *http.Server {
	handler := newEventHandler(ctx)
	routes := initRoutes(handler)
	return newServer(ctx, routes)
}

// newServer returns a new HTTP server to start the hook listener API.
func newServer(ctx context.Context, routes *mux.Router) *http.Server {
	server := &http.Server{
		Addr: "127.0.0.1:8080",
		BaseContext: func(listener net.Listener) context.Context {
			return ctx
		},
		Handler:      routes,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
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
	configStore := newConfigStore(ctx, kubeClient)

	return &EventHandler{
		KubeClientSet:      kubeClient,
		TektonClientSet:    tektonClient,
		WorkflowsClientSet: workflowsClient,
		configStore:        configStore,
	}
}

func newConfigStore(ctx context.Context, kubeClient kubernetes.Interface) *config.Store {
	watcher := newConfigMapWatcher(kubeClient)

	logger := logging.FromContext(ctx).Named("configs")
	configStore := config.NewStore(logger)
	configStore.WatchConfigs(watcher)
	return configStore
}

func newConfigMapWatcher(kubeClient kubernetes.Interface) configmap.Watcher {
	const namespaceKey = "SYSTEM_NAMESPACE"
	namespace, ok := os.LookupEnv(namespaceKey)
	if !ok {
		panic(fmt.Errorf("Error creating hook-listener: missing environment variable %s", namespaceKey))
	}

	filter, err := informer.FilterConfigByLabelExists("workflows.workflows.dev/release")
	if err != nil {
		panic(fmt.Errorf("Error creating hook-listener: %w", err))
	}

	return informer.NewInformedWatcher(kubeClient, namespace, *filter)
}
