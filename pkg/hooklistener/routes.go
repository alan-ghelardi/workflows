package hooklistener

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/configmap/informer"
	"knative.dev/pkg/logging"

	"github.com/gorilla/mux"
	"github.com/nubank/workflows/pkg/apis/config"
	"github.com/nubank/workflows/pkg/github"
)

// repositoryEventHandler returns a handler func that calls the provided
// EventHandler object.
func repositoryEventHandler(handler *EventHandler, configStore *config.Store) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		ctx := request.Context()
		ctx = configStore.ToContext(ctx)
		vars := mux.Vars(request)
		namespacedName := types.NamespacedName{
			Namespace: vars["namespace"],
			Name:      vars["name"],
		}
		event := github.GetEvent(ctx)
		Response := handler.triggerWorkflow(ctx, namespacedName, event)
		Response.write(ctx, writer)
	})
}

// initRoutes configures routes exposed by the hook listener API.
func initRoutes(ctx context.Context, handler *EventHandler) *mux.Router {
	configStore := newConfigStore(ctx, handler.KubeClientSet)
	router := mux.NewRouter().StrictSlash(true)

	// Simple readiness/liveness check.
	router.Methods("GET").Path("/health").HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		OK("Event listener is alive").
			write(request.Context(), writer)
	})

	api := router.PathPrefix("/api/v1alpha1").Subrouter()
	api.Use(tracer)
	api.Use(eventParser)
	api.Methods("POST").Path("/namespaces/{namespace}/workflows/{name}/hooks").Handler(repositoryEventHandler(handler, configStore))

	return router
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
