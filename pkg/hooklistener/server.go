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
	"github.com/nubank/workflows/pkg/github"
	tektonclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/configmap/informer"
	"knative.dev/pkg/logging"
)

// New creates a HTTP server to handle events delivered by Github Webhooks.
func New(ctx context.Context) *http.Server {
	handler := newEventHandlerOrDie(ctx)
	routes := initRoutes(handler)
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

// newEventHandlerOrDie returns a new EventHandler by initializing all required
// dependencies (Kubernetes client sets, config map watchers and Github
// clients).
// It panics if any of those dependencies fails to be created.
func newEventHandlerOrDie(ctx context.Context) *EventHandler {
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(fmt.Errorf("Error creating Kubernetes config: %w", err))
	}

	kubeClient := kubernetes.NewForConfigOrDie(config)
	configStore := newConfigStoreOrDie(ctx, kubeClient)
	tektonClient := tektonclientset.NewForConfigOrDie(config)
	workflowsClient := workflowsclientset.NewForConfigOrDie(config)
	workflowReader := github.NewWorkflowReader(github.NewClientOrDie())
	return &EventHandler{
		configStore:        configStore,
		kubeClientSet:      kubeClient,
		tektonClientSet:    tektonClient,
		workflowsClientSet: workflowsClient,
		workflowReader:     workflowReader,
	}
}

// newConfigStoreOrDie creates a Store filled with the initial state of
// configurations.
func newConfigStoreOrDie(ctx context.Context, kubeClient kubernetes.Interface) *config.Store {
	logger := logging.FromContext(ctx).Named("configs")
	watcher := newConfigMapWatcher(kubeClient)
	configStore := config.NewStore(logger)
	configStore.WatchConfigs(watcher)
	if err := watcher.Start(ctx.Done()); err != nil {
		logger.Fatal("Error starting config map watcher", zap.Error(err))
	}
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
