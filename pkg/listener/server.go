package listener

import (
	"context"
	"net"
	"net/http"
	"time"

	"k8s.io/apimachinery/pkg/types"

	"github.com/gorilla/mux"
	"github.com/nubank/workflows/pkg/github"
)

// repositoryEventHandler returns a handler func that calls the provided
// EventListener object.
func repositoryEventHandler(listener *EventListener) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		ctx := request.Context()
		vars := mux.Vars(request)
		namespacedName := types.NamespacedName{
			Namespace: vars["namespace"],
			Name:      vars["name"],
		}
		event := github.GetEvent(ctx)
		Response := listener.RunWorkflow(ctx, namespacedName, event)
		Response.write(ctx, writer)
	})
}

// Routes configures routes exposed by the event listener API.
func Routes(listener *EventListener) *mux.Router {
	router := mux.NewRouter().StrictSlash(true)

	// Simple liveness check
	router.Methods("GET").Path("/live").HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		OK("Event listener is alive").
			write(request.Context(), writer)
	})

	api := router.PathPrefix("/api/v1alpha1").Subrouter()
	api.Use(tracer)
	api.Use(eventParser)
	api.Methods("POST").Path("/namespaces/{namespace}/workflows/{name}").Handler(repositoryEventHandler(listener))

	return router
}

// NewServer returns a new HTTP server to start the event listener API.
func NewServer(ctx context.Context, routes *mux.Router) *http.Server {
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
