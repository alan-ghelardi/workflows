package listener

import (
	"net/http"

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
		logger := logging.FromContext(ctx).
			With(zap.String("user-agent", request.UserAgent()))

		event, err := github.ParseWebhookEvent(request)
		if err != nil {
			logger.Errorw("Unable to process incoming request", zap.Error(err))
			BadRequest(err.Error()).
				write(ctx, writer)
		}
		next.ServeHTTP(writer, request.WithContext(github.WithEvent(ctx, event)))
	})
}

// tracer is a middleware function that augments the logger present in the
// context with relevant information to improve traceability.
func tracer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		ctx := request.Context()
		logger := logging.FromContext(ctx)
		event := github.GetEvent(ctx)

		logger.With(zap.String("github/delivery-id", event.DeliveryID),
			zap.String("github/hook-id", event.HookID),
			zap.String("github/event", event.Name),
			zap.String("github/repository", event.Repository),
			zap.String("user-agent", request.UserAgent()))

		next.ServeHTTP(writer, request.WithContext(logging.WithLogger(ctx, logger)))
	})
}
