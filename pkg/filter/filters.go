// package filter implements filtering logic to determine whether workflows must run based on their filtering rules.
package filter

import (
	"fmt"

	"github.com/gobwas/glob"
	workflowsv1alpha1 "github.com/nubank/workflows/pkg/apis/workflows/v1alpha1"
	"github.com/nubank/workflows/pkg/github"
)

const (
	accepted = "Workflow has been accepted"

	verified = "Rule successfully verified"
)

// Filter is a function that takes a workflow and a Github event, verifies
// whether the event satisfies a filtering rule and returns a message and a
// boolean indicating results of this verification.
type Filter func(*workflowsv1alpha1.Workflow, *github.Event) (string, bool)

// events verifies whether events configured in the workflow match the name of
// the incoming Github event.
func events(workflow *workflowsv1alpha1.Workflow, event *github.Event) (string, bool) {
	for _, eventName := range workflow.Spec.Events {
		if eventName == event.Name {
			return verified, true
		}
	}
	return fmt.Sprintf("%s event doesn't match rules %+v", event.Name, workflow.Spec.Events), false
}

// repository verifies whether the repository associated to the workflow matches
// the repository that originated the Github event.
func repository(workflow *workflowsv1alpha1.Workflow, event *github.Event) (string, bool) {
	if event.Repository == workflow.Spec.Repository.String() {
		return verified, true
	}
	return fmt.Sprintf("event's repository %s doesn't match workflow's repository %s", event.Repository, workflow.Spec.Repository), false
}

// branches verifies whether branches configured in the workflow match the
// branch present in the Github event. This filter is only applied on push and
// pull_request events.
func branches(workflow *workflowsv1alpha1.Workflow, event *github.Event) (string, bool) {
	if event.Name != "push" && event.Name != "pull_request" {
		return fmt.Sprintf("%s event isn't supported", event.Name), true
	}

	for _, branch := range workflow.Spec.Branches {
		globPattern, err := glob.Compile(branch)
		if err != nil {
			return err.Error(), false
		}

		if globPattern.Match(event.Branch) {
			return verified, true
		}
	}
	return fmt.Sprintf("event's branch %s doesn't match rules %+v", event.Branch, workflow.Spec.Branches), false
}

// paths verifies whether paths configured in the workflow match modified files
// present in the Github event. This filter is only applied on push and
// pull_request events.
func paths(workflow *workflowsv1alpha1.Workflow, event *github.Event) (string, bool) {
	if event.Name != "push" && event.Name != "pull_request" {
		return fmt.Sprintf("%s event isn't supported", event.Name), true
	}

	for _, path := range workflow.Spec.Paths {
		globPattern, err := glob.Compile(path)
		if err != nil {
			return err.Error(), false
		}

		for _, file := range event.ModifiedFiles {
			if globPattern.Match(file) {
				return verified, true
			}
		}
	}
	return fmt.Sprintf("event's modified files don't match rules %+v", workflow.Spec.Paths), false
}

// filters is a chain of filter funcs.
var filters = []Filter{events,
	repository,
	branches,
	paths,
}

// VerifyRules verifies all filtering rules declared in the supplied workflow,
// by validating them against the incoming Github event.
func VerifyRules(workflow *workflowsv1alpha1.Workflow, event *github.Event) (string, bool) {
	for _, filter := range filters {
		if message, accepted := filter(workflow, event); !accepted {
			return fmt.Sprintf("Workflow has been rejected because %s", message), false
		}
	}
	return accepted, true
}
