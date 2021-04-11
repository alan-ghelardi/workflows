package hooklistener

import (
	"context"
	"io/ioutil"
	"net/http/httptest"
	"testing"
)

func TestTestVariousResponseConstructors(t *testing.T) {
	tests := []struct {
		in     func(string) *Response
		status int
	}{
		{in: Accepted, status: 202},
		{in: BadRequest, status: 400},
		{in: Created, status: 201},
		{in: Forbidden, status: 403},
		{in: InternalServerError, status: 500},
		{in: NotFound, status: 404},
		{in: OK, status: 200},
	}

	for _, test := range tests {
		constructor := test.in
		message := "Lorem Ipsum"
		response := constructor(message)

		if test.status != response.Status {
			t.Errorf("Want status %d, but got %d", test.status, response.Status)
		}

		if message != response.Payload.Message {
			t.Errorf("Want message %s, but got %s", message, response.Payload.Message)
		}
	}
}

func TestWritingResponse(t *testing.T) {
	fakeResponseWriter := httptest.NewRecorder()
	Response := OK("Lorem Ipsum")
	Response.write(context.Background(), fakeResponseWriter)
	result := fakeResponseWriter.Result()

	wantStatus := 200
	gotStatus := result.StatusCode
	if wantStatus != gotStatus {
		t.Errorf("Want status %d, but got %d", wantStatus, gotStatus)
	}

	wantContentType := "application/json"
	gotContentType := result.Header.Get("Content-Type")
	if wantContentType != gotContentType {
		t.Errorf("Want Content-Type header %s, but got %s", wantContentType, gotContentType)
	}

	body, err := ioutil.ReadAll(result.Body)
	if err != nil {
		t.Fatalf("Error reading response body: %v", err)
	}

	wantPayload := "{\"message\":\"Lorem Ipsum\"}\n"
	gotPayload := string(body)
	if wantPayload != gotPayload {
		t.Errorf("Want payload %s, but got %s", wantPayload, gotPayload)
	}
}
