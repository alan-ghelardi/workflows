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
			"workflow.name": workflow.GetName(),
		},
		event: event,
	}

	return replacements
}

// Expand ...
func Expand(text string, replacements *Replacements) string {
	return expressionPattern.ReplaceAllStringFunc(text, func(match string) string {
		submatches := expressionPattern.FindStringSubmatch(text)
		if submatches == nil {
			return match
		}

		key := submatches[len(submatches)-1]
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
		jsonPathExpr := matches[len(matches)-1]
		result, _ := query(replacements.event, jsonPathExpr)
		return result, true
	}

	if value, exists := replacements.specialVariables[key]; exists {
		return value, true
	}
	return "", false
}
