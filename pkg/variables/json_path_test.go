package variables

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/nubank/workflows/pkg/github"
)

// readEvent reads the event from the testdata directory.
func readEvent() (*github.Event, error) {
	fileName := "testdata/push-event.json"
	payload, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, fmt.Errorf("Error reading %s: %w", fileName, err)
	}

	request := &http.Request{
		Header: http.Header{},
		Body:   ioutil.NopCloser(strings.NewReader(string(payload))),
	}
	request.Header.Set("X-GitHub-Event", "push")

	event, err := github.ParseWebhookEvent(request)
	if err != nil {
		return nil, fmt.Errorf("Error parsing event: %s", err)
	}

	return event, nil
}

func TestQuery(t *testing.T) {
	event, err := readEvent()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		expr   string
		result string
	}{
		{"text", "text"},
		{"{.head_commit.id}", "833568e"},
		{"{.head_commit}", `{"id":"833568e"}`},
		{"{.commits..modified}", `[["README.md"],["CHANGELOG.md"]]`},
		{"{.forced}", "false"},
		{"{.base_ref}", null},
		{"{.missing_key}", "[]"},
		{"{.commits..missing_key}", "[]"},
	}

	for _, test := range tests {
		gotResult, err := query(event, test.expr)
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(test.result, gotResult); diff != "" {
			t.Errorf("Mismatch (-want +got):\n%s", diff)
		}
	}
}
