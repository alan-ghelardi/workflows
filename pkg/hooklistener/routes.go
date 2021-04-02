package hooklistener

import (
	"net/http"

	"k8s.io/apimachinery/pkg/types"

	"github.com/gorilla/mux"
	"github.com/nubank/workflows/pkg/github"
)

// repositoryEventHandler returns a handler func that calls the provided
// EventHandler object.
func repositoryEventHandler(handler *EventHandler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		ctx := handler.configStore.ToContext(request.Context())
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
func initRoutes(handler *EventHandler) *mux.Router {
	router := mux.NewRouter().StrictSlash(true)

	// Simple readiness/liveness check.
	router.Methods("GET").Path("/health").HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		OK("Event listener is alive").
			write(request.Context(), writer)
	})

	api := router.PathPrefix("/api/v1alpha1").Subrouter()
	api.Use(tracer)
	api.Use(eventParser)
	api.Methods("POST").Path("/namespaces/{namespace}/workflows/{name}/hooks").Handler(repositoryEventHandler(handler))

	return router
}
