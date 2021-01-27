package testutils

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	workflowsv1alpha1 "github.com/nubank/workflows/pkg/apis/workflows/v1alpha1"
	"github.com/nubank/workflows/pkg/github"

	"github.com/ghodss/yaml"
)

// ReadWorkflow reads the corresponding workflow from the testdata directory.
func ReadWorkflow(fileName string) (*workflowsv1alpha1.Workflow, error) {
	file, err := ioutil.ReadFile(fmt.Sprintf("testdata/%s", fileName))
	if err != nil {
		return nil, err
	}

	var workflow workflowsv1alpha1.Workflow
	if err := yaml.Unmarshal(file, &workflow); err != nil {
		return nil, fmt.Errorf("Error unmarshaling workflow from file %s: %w", fileName, err)
	}

	return &workflow, nil
}

// ReadEvent reads the event from the testdata directory.
func ReadEvent(fileName string) (*github.Event, error) {
	payload, err := ioutil.ReadFile(fmt.Sprintf("testdata/%s", fileName))
	if err != nil {
		return nil, err
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
