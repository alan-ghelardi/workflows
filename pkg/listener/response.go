package listener

import (
	"context"
	"encoding/json"
	"net/http"

	"go.uber.org/zap"
	"knative.dev/pkg/logging"
)

// Response is an internal representation of a HTTP response.
type Response struct {
	Status  int
	Payload ResponsePayload
}

// ResponsePayload is the payload returned in the HTTP response.
type ResponsePayload struct {
	Message string `json:"message"`
}

// write
func (r *Response) write(ctx context.Context, writer http.ResponseWriter) {
	logger := logging.FromContext(ctx)

	writer.Header().Set("content-type", "application/json")
	writer.WriteHeader(r.Status)
	err := json.NewEncoder(writer).Encode(r.Payload)
	if err != nil {
		logger.Errorw("Error writing response payload", zap.Error(err))
	}
}

// newResponse returns a new Response object with the supplied status code and message.
func newResponse(status int, message string) *Response {
	return &Response{
		Status: status,
		Payload: ResponsePayload{
			Message: message,
		},
	}
}

// OK returns a HTTP 200 response with the supplied message.
func OK(message string) *Response {
	return newResponse(http.StatusOK, message)
}

// BadRequest returns a HTTP 400 error response with the supplied message.
func BadRequest(message string) *Response {
	return newResponse(http.StatusBadRequest, message)
}
