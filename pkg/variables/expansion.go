package variables

import (
	"regexp"

	workflowsv1alpha1 "github.com/nubank/workflows/pkg/apis/workflows/v1alpha1"
	"github.com/nubank/workflows/pkg/github"
)

var (

	// Regex to match expressions containing variables such as
	// $(workflow.name) or $(event {.pusher.name}).
	expressionPattern = regexp.MustCompile(`\$\(((event\s*\{[^\}]+\})|(workflow\.[^\)]+))\)`)

	// Regex that matches event expressions by allowing us to capture JSON
	// path expressions enclosed between curly braces.
	eventPattern = regexp.MustCompile(`event\s*(\{[^\}]+\})`)
)

// Replacements represents a substitution context for variables declared in
// expressions.
type Replacements struct {
	specialVariables map[string]string
	event            *github.Event
}

// MakeReplacements returns a new Replacements object containing information
// about the workflow as well as the supplied Github event.
func MakeReplacements(workflow *workflowsv1alpha1.Workflow, event *github.Event) *Replacements {
	replacements := &Replacements{
		specialVariables: map[string]string{
			"workflow.name":        workflow.GetName(),
			"workflow.repo.owner":  workflow.Spec.Repository.Owner,
			"workflow.repo.name":   workflow.Spec.Repository.Name,
			"workflow.head-commit": event.HeadCommitSHA,
		},
		event: event,
	}

	return replacements
}

// Expand attempts to substitute all variables declared in the supplied text by
// evaluating them against the provided replacement context.
func Expand(text string, replacements *Replacements) string {
	return expressionPattern.ReplaceAllStringFunc(text, func(match string) string {
		submatches := expressionPattern.FindStringSubmatch(match)
		if submatches == nil {
			return match
		}

		key := submatches[1]

		value, ok := applyReplacement(key, replacements)
		if !ok {
			return match
		}
		return value
	})
}

func applyReplacement(key string, replacements *Replacements) (string, bool) {
	if matches := eventPattern.FindStringSubmatch(key); matches != nil {
		jsonPathExpr := matches[1]
		result, err := query(replacements.event, jsonPathExpr)
		if err != nil {
			return "", false
		}
		return result, true
	}

	if value, exists := replacements.specialVariables[key]; exists {
		return value, true
	}
	return "", false
}
