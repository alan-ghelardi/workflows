package hooklistener

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

// write ends the request by writing the response to the server's output stream.
func (r *Response) write(ctx context.Context, writer http.ResponseWriter) {
	writer.Header().Set("content-type", "application/json")
	writer.WriteHeader(r.Status)
	err := json.NewEncoder(writer).Encode(r.Payload)
	if err != nil {
		logger := logging.FromContext(ctx)
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

// Created returns a HTTP 201 response with the supplied message.
func Created(message string) *Response {
	return newResponse(http.StatusCreated, message)
}

// BadRequest returns a HTTP 400 error response with the supplied message.
func BadRequest(message string) *Response {
	return newResponse(http.StatusBadRequest, message)
}

// Forbidden returns a HTTP 403 error response with the supplied message.
func Forbidden(message string) *Response {
	return newResponse(http.StatusForbidden, message)
}

// InternalServerError returns a HTTP 500 error response with the supplied message.
func InternalServerError(message string) *Response {
	return newResponse(http.StatusInternalServerError, message)
}

// NotFound returns a HTTP 404 error response with the supplied message.
func NotFound(message string) *Response {
	return newResponse(http.StatusNotFound, message)
}
