package listener

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

// HandleRepositoryEvent
func HandleRepositoryEvent(writer http.ResponseWriter, request *http.Request) {
	OK("Hello!").write(request.Context(), writer)
}

// Routes configures routes exposed by the event listener API.
func Routes() *mux.Router {
	router := mux.NewRouter().StrictSlash(true)

	// Simple liveness check
	router.HandleFunc("/live", func(writer http.ResponseWriter, request *http.Request) {
		OK("Event listener is alive").
			write(request.Context(), writer)
	})

	api := router.Path("/api/v1alpha1").Subrouter()
	api.Use(eventParser)
	api.Use(tracer)

	api.HandleFunc("/namespaces/{namespace}/workflow/{name}", HandleRepositoryEvent)

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
