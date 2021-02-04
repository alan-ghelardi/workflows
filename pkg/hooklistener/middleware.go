package hooklistener

import (
	"net/http"
	"time"

	"go.uber.org/zap"
	"knative.dev/pkg/logging"

	"github.com/nubank/workflows/pkg/github"
)

// eventParser is a middleware function that attempts to parse request bodies
// into Github events.
// It passes a request infused with the parsed event to the next handler or
// returns a bad request error if the request body can't be coerced into a valid
// Github event.
func eventParser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		ctx := request.Context()
		logger := logging.FromContext(ctx)
		event, err := github.ParseWebhookEvent(request)
		if err != nil {
			logger.Errorw("Unable to process incoming request", zap.Error(err))
			BadRequest(err.Error()).
				write(ctx, writer)
		} else {
			// Populate the logging context with relevant
			// information about the event being handled.
			logger = logger.With(zap.String("github/delivery-id", event.DeliveryID),
				zap.String("github/hook-id", event.HookID),
				zap.String("github/event", event.Name),
				zap.String("github/repository", event.Repository))
			ctx = logging.WithLogger(ctx, logger)

			next.ServeHTTP(writer, request.WithContext(github.WithEvent(ctx, event)))
		}
	})
}

// traceableResponseWriter wraps a http.ResponseWriter by allowing us to capture
// the final status code sent in the response.
type traceableResponseWriter struct {
	writer http.ResponseWriter
	status int
}

// Header implements http.ResponseWriter/Header.
func (t *traceableResponseWriter) Header() http.Header {
	return t.writer.Header()
}

// Write implements http.ResponseWriter/Write.
func (t *traceableResponseWriter) Write(content []byte) (int, error) {
	return t.writer.Write(content)
}

// WriteHeader implements http.ResponseWriter/WriteHeader.
func (t *traceableResponseWriter) WriteHeader(status int) {
	t.status = status
	t.writer.WriteHeader(status)
}

// tracer is a middleware function that augments the logger present in the
// context with relevant information to improve traceability.
func tracer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		ctx := request.Context()
		logger := logging.FromContext(ctx)

		logger = logger.With(zap.String("method", request.Method),
			zap.String("path", request.URL.Path),
			zap.String("user-agent", request.UserAgent()))

		startTime := time.Now()

		logger.Info("Handling event")
		trw := &traceableResponseWriter{writer: writer}
		next.ServeHTTP(trw, request.WithContext(logging.WithLogger(ctx, logger)))

		timeTaken := time.Since(startTime)
		logger.Infow("Request completed", zap.Int("status", trw.status),
			zap.Duration("time-taken", timeTaken))
	})
}
