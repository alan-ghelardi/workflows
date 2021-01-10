// package filter implements filtering logic to determine whether workflows must run based on their filtering rules.
package filter

import (
	"fmt"

	"github.com/gobwas/glob"
	workflowsv1alpha1 "github.com/nubank/workflows/pkg/apis/workflows/v1alpha1"
	"github.com/nubank/workflows/pkg/github"
)

const (
	workflowAccepted = "Workflow accepted"

	filterSucceeded = "filter succeeded"
)

// Filter is a function that takes a workflow and a Github event and returns a
// boolean value indicating whether the event satisfies filters declared in the
// workflow along with a message explaining the result.
type Filter func(*workflowsv1alpha1.Workflow, *github.Event) (bool, string)

// events verifies whether events configured in the workflow match the name of
// the incoming Github event.
func events(workflow *workflowsv1alpha1.Workflow, event *github.Event) (bool, string) {
	for _, eventName := range workflow.Spec.Events {
		if eventName == event.Name {
			return true, filterSucceeded
		}
	}
	return false, fmt.Sprintf("%s event doesn't match filters %+v", event.Name, workflow.Spec.Events)
}

// repository verifies whether the repository associated to the workflow matches
// the repository that originated the Github event.
func repository(workflow *workflowsv1alpha1.Workflow, event *github.Event) (bool, string) {
	if event.Repository == workflow.Spec.Repository.String() {
		return true, filterSucceeded
	}
	return false, fmt.Sprintf("repository %s doesn't match workflow's repository %s", event.Repository, workflow.Spec.Repository)
}

// branches verifies whether branches configured in the workflow match the
// branch present in the Github event. This filter is only applied on push and
// pull_request events.
func branches(workflow *workflowsv1alpha1.Workflow, event *github.Event) (bool, string) {
	if event.Name != "push" && event.Name != "pull_request" {
		return true, fmt.Sprintf("skipped because %s event isn't supported", event.Name)
	}

	for _, branch := range workflow.Spec.Branches {
		globPattern, err := glob.Compile(branch)
		if err != nil {
			return false, err.Error()
		}

		if globPattern.Match(event.Branch) {
			return true, filterSucceeded
		}
	}
	return false, fmt.Sprintf("branch %s doesn't match filters %+v", event.Branch, workflow.Spec.Branches)
}

// paths verifies whether paths configured in the workflow match modified files
// present in the Github event. This filter is only applied on push and
// pull_request events.
func paths(workflow *workflowsv1alpha1.Workflow, event *github.Event) (bool, string) {
	if event.Name != "push" && event.Name != "pull_request" {
		return true, fmt.Sprintf("skipped because %s event isn't supported", event.Name)
	}

	if len(workflow.Spec.Paths) == 0 {
		return true, "skipped because there are no configured paths"
	}

	for _, path := range workflow.Spec.Paths {
		globPattern, err := glob.Compile(path)
		if err != nil {
			return false, err.Error()
		}

		for _, file := range event.Changes {
			if globPattern.Match(file) {
				return true, filterSucceeded
			}
		}
	}
	return false, fmt.Sprintf("modified files don't match filters %+v", workflow.Spec.Paths)
}

// filters is a chain of filter funcs.
var filters = []Filter{events,
	repository,
	branches,
	paths,
}

// VerifyCriteria verifies all filter criteria declared in the supplied
// workflow, by validating them against the incoming Github event.
func VerifyCriteria(workflow *workflowsv1alpha1.Workflow, event *github.Event) (bool, string) {
	for _, filter := range filters {
		if ok, message := filter(workflow, event); !ok {
			return false, fmt.Sprintf("Workflow was rejected because Github event doesn't satisfy filter criteria: %s", message)
		}
	}
	return true, workflowAccepted
}
