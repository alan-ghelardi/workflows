package variables

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

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
