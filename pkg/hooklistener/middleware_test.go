package hooklistener

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nubank/workflows/pkg/github"
)

// fakeHandler records the request passed to ServeHTTP method.
type fakeHandler struct {
	request *http.Request
}

func (f *fakeHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	f.request = request
}

func TestParsesGithubEvents(t *testing.T) {
	handler := &fakeHandler{}

	payload := `{
    "head_commit": {
	"id": "32eec86"
    },
    "ref": "refs/heads/main",
    "repository": {
	"full_name": "my-org/my-repo"
    }
}`

	request := &http.Request{
		Header: http.Header{},
		Body:   ioutil.NopCloser(strings.NewReader(payload)),
	}
	request.Header.Set("X-GitHub-Delivery", "123")
	request.Header.Set("X-GitHub-Event", "push")
	request.Header.Set("X-GitHub-Hook-ID", "456")
	request.Header.Set("X-Hub-Signature-256", "sha256=d8a72707")

	eventParser(handler).
		ServeHTTP(httptest.NewRecorder(), request)

	event := github.GetEvent(handler.request.Context())
	if event == nil {
		t.Error("Request's context doesn't contain a valid Github Event object")
	}
}

func TestReturnsABadRequestErrorWhenTheEventCannotBeParsed(t *testing.T) {
	handler := &fakeHandler{}

	payload := "{"

	request := &http.Request{
		Header: http.Header{},
		Body:   ioutil.NopCloser(strings.NewReader(payload)),
	}
	request.Header.Set("X-GitHub-Delivery", "123")
	request.Header.Set("X-GitHub-Event", "push")
	request.Header.Set("X-GitHub-Hook-ID", "456")
	request.Header.Set("X-Hub-Signature-256", "sha256=d8a72707")

	responswWriter := httptest.NewRecorder()

	eventParser(handler).
		ServeHTTP(responswWriter, request)

	wantStatus := 400
	gotStatus := responswWriter.Result().StatusCode
	if wantStatus != gotStatus {
		t.Errorf("Want status %d, but got %d", wantStatus, gotStatus)
	}
}

func TestTraceableResponseWriter(t *testing.T) {
	rw := httptest.NewRecorder()
	trw := &traceableResponseWriter{writer: rw}

	wantStatus := http.StatusOK
	wantContentType := "text/plain"
	wantBody := "Lorem ipsum"

	trw.Header().Set("Content-Type", wantContentType)
	trw.WriteHeader(http.StatusOK)
	trw.Write([]byte(wantBody))

	result := rw.Result()

	gotStatus := result.StatusCode
	if wantStatus != gotStatus {
		t.Errorf("Fail in WriteHeader(): want status %d, but got %d", wantStatus, gotStatus)
	}

	if wantStatus != trw.status {
		t.Errorf("Fail to record status: want %d, but got %d", wantStatus, trw.status)
	}

	gotContentType := result.Header.Get("Content-Type")
	if wantContentType != gotContentType {
		t.Errorf("Fail in Header(): want content type %s, but got %s", wantContentType, gotContentType)
	}

	gotBody, err := ioutil.ReadAll(result.Body)
	if err != nil {
		t.Fatalf("Error reading response body: %v", err)
	}

	if wantBody != string(gotBody) {
		t.Errorf("Fail in Write(): want body %s, but got %s", wantBody, gotBody)
	}
}
