package filter

import (
	"testing"

	workflowsv1alpha1 "github.com/nubank/workflows/pkg/apis/workflows/v1alpha1"
	"github.com/nubank/workflows/pkg/github"
)

func TestEvents(t *testing.T) {
	workflow := &workflowsv1alpha1.Workflow{
		Spec: workflowsv1alpha1.WorkflowSpec{
			Events: []string{"push", "pull_request"},
		},
	}

	tests := []struct {
		eventName   string
		wantMessage string
		wantResult  bool
	}{
		{"push", verified, true},
		{"pull_request", verified, true},
		{"release", "release event doesn't match rules [push pull_request]", false},
	}

	for _, test := range tests {
		gotMessage, gotResult := events(workflow, &github.Event{Name: test.eventName})
		if test.wantMessage != gotMessage {
			t.Errorf("Want message %s, got %s", test.wantMessage, gotMessage)
		}

		if test.wantResult != gotResult {
			t.Errorf("Want result %t, got %t", test.wantResult, gotResult)
		}
	}
}

func TestRepository(t *testing.T) {
	workflow := &workflowsv1alpha1.Workflow{
		Spec: workflowsv1alpha1.WorkflowSpec{
			Repository: &workflowsv1alpha1.Repository{
				Owner: "my-org",
				Name:  "my-repo",
			},
		},
	}

	tests := []struct {
		repository  string
		wantMessage string
		wantResult  bool
	}{
		{"my-org/my-repo", verified, true},
		{"my-org/other-repo", "event's repository my-org/other-repo doesn't match workflow's repository my-org/my-repo", false},
		{"other-org/my-repo", "event's repository other-org/my-repo doesn't match workflow's repository my-org/my-repo", false},
	}

	for _, test := range tests {
		gotMessage, gotResult := repository(workflow, &github.Event{Repository: test.repository})
		if test.wantMessage != gotMessage {
			t.Errorf("Want message %s, got %s", test.wantMessage, gotMessage)
		}

		if test.wantResult != gotResult {
			t.Errorf("Want result %t, got %t", test.wantResult, gotResult)
		}
	}
}

func TestBranches(t *testing.T) {
	workflow := &workflowsv1alpha1.Workflow{
		Spec: workflowsv1alpha1.WorkflowSpec{
			Branches: []string{"main",
				"staging*",
			},
		},
	}

	tests := []struct {
		branch      string
		eventName   string
		wantMessage string
		wantResult  bool
	}{
		{"main", "push", verified, true},
		{"staging", "pull_request", verified, true},
		{"staging-john-patch1", "push", verified, true},
		{"dev", "pull_request", "event's branch dev doesn't match rules [main staging*]", false},
		{"", "release", "release event isn't supported", true},
	}

	for _, test := range tests {
		gotMessage, gotResult := branches(workflow, &github.Event{Name: test.eventName, Branch: test.branch})
		if test.wantMessage != gotMessage {
			t.Errorf("Want message %s, got %s", test.wantMessage, gotMessage)
		}

		if test.wantResult != gotResult {
			t.Errorf("Want result %t, got %t", test.wantResult, gotResult)
		}
	}
}

func TestPaths(t *testing.T) {
	workflow := &workflowsv1alpha1.Workflow{
		Spec: workflowsv1alpha1.WorkflowSpec{
			Paths: []string{"**/*.go",
				"**/*.sh",
			},
		},
	}

	tests := []struct {
		files       []string
		eventName   string
		wantMessage string
		wantResult  bool
	}{
		{[]string{"pkg/x/y.go"}, "push", verified, true},
		{[]string{"pkg/x/y.go", "README.md"}, "push", verified, true},
		{[]string{"scripts/build.sh", "README.md"}, "pull_request", verified, true},
		{[]string{"README.md"}, "push", "event's modified files don't match rules [**/*.go **/*.sh]", false},
		{[]string{"README.md"}, "pull_request", "event's modified files don't match rules [**/*.go **/*.sh]", false},
		{[]string{}, "release", "release event isn't supported", true},
	}

	for _, test := range tests {
		gotMessage, gotResult := paths(workflow, &github.Event{Name: test.eventName, ModifiedFiles: test.files})
		if test.wantMessage != gotMessage {
			t.Errorf("Want message %s, got %s", test.wantMessage, gotMessage)
		}

		if test.wantResult != gotResult {
			t.Errorf("Want result %t, got %t", test.wantResult, gotResult)
		}
	}
}

func TestVerifyRules(t *testing.T) {
	workflow := &workflowsv1alpha1.Workflow{
		Spec: workflowsv1alpha1.WorkflowSpec{
			Repository: &workflowsv1alpha1.Repository{Owner: "my-org",
				Name: "my-repo",
			},
			Events:   []string{"push"},
			Branches: []string{"main"},
			Paths:    []string{"**/*.go"},
		},
	}

	tests := []struct {
		eventName   string
		repo        string
		branch      string
		files       []string
		wantMessage string
		wantResult  bool
	}{
		{"push", "my-org/my-repo", "main", []string{"pkg/x/y.go"}, accepted, true},
		{"pull_request", "my-org/my-repo", "main", []string{"pkg/x/y.go"}, "Workflow has been rejected because pull_request event doesn't match rules [push]", false},
		{"push", "my-org/other-repo", "main", []string{"pkg/x/y.go"}, "Workflow has been rejected because event's repository my-org/other-repo doesn't match workflow's repository my-org/my-repo", false},
		{"push", "my-org/my-repo", "dev", []string{"pkg/x/y.go"}, "Workflow has been rejected because event's branch dev doesn't match rules [main]", false},
		{"push", "my-org/my-repo", "main", []string{"README.md"}, "Workflow has been rejected because event's modified files don't match rules [**/*.go]", false},
	}

	for _, test := range tests {
		event := &github.Event{Name: test.eventName,
			Repository:    test.repo,
			Branch:        test.branch,
			ModifiedFiles: test.files,
		}
		gotMessage, gotResult := VerifyRules(workflow, event)
		if test.wantMessage != gotMessage {
			t.Errorf("Want message %s, got %s", test.wantMessage, gotMessage)
		}

		if test.wantResult != gotResult {
			t.Errorf("Want result %t, got %t", test.wantResult, gotResult)
		}
	}
}
