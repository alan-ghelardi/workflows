package variables

import (
	"regexp"

	workflowsv1alpha1 "github.com/nubank/workflows/pkg/apis/workflows/v1alpha1"
	"github.com/nubank/workflows/pkg/github"
)

var (
	expressionPattern = regexp.MustCompile(`\$\(((event\s*\{[^\}]+\})|(workflow\.[^\)]+))\)`)

	eventPattern = regexp.MustCompile(`event\s*(\{[^\}]+\})`)
)

type Replacements struct {
	specialVariables map[string]string
	event            *github.Event
}

// MakeReplacements ...
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

// Expand ...
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

// applyReplacement ...
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
