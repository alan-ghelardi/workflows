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
		{"push", filterSucceeded, true},
		{"pull_request", filterSucceeded, true},
		{"release", "release event doesn't match filters [push pull_request]", false},
	}

	for _, test := range tests {
		gotResult, gotMessage := events(workflow, &github.Event{Name: test.eventName})
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
		{"my-org/my-repo", filterSucceeded, true},
		{"my-org/other-repo", "repository my-org/other-repo doesn't match workflow's repository my-org/my-repo", false},
		{"other-org/my-repo", "repository other-org/my-repo doesn't match workflow's repository my-org/my-repo", false},
	}

	for _, test := range tests {
		gotResult, gotMessage := repository(workflow, &github.Event{Repository: test.repository})
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
		{"main", "push", filterSucceeded, true},
		{"staging", "pull_request", filterSucceeded, true},
		{"staging-john-patch1", "push", filterSucceeded, true},
		{"dev", "pull_request", "branch dev doesn't match filters [main staging*]", false},
		{"", "release", "skipped because release event isn't supported", true},
	}

	for _, test := range tests {
		gotResult, gotMessage := branches(workflow, &github.Event{Name: test.eventName, Branch: test.branch})
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
		{[]string{"pkg/x/y.go"}, "push", filterSucceeded, true},
		{[]string{"pkg/x/y.go", "README.md"}, "push", filterSucceeded, true},
		{[]string{"scripts/build.sh", "README.md"}, "pull_request", filterSucceeded, true},
		{[]string{"README.md"}, "push", "modified files don't match filters [**/*.go **/*.sh]", false},
		{[]string{"README.md"}, "pull_request", "modified files don't match filters [**/*.go **/*.sh]", false},
		{[]string{}, "release", "skipped because release event isn't supported", true},
	}

	for _, test := range tests {
		gotResult, gotMessage := paths(workflow, &github.Event{Name: test.eventName, ModifiedFiles: test.files})
		if test.wantMessage != gotMessage {
			t.Errorf("Want message %s, got %s", test.wantMessage, gotMessage)
		}

		if test.wantResult != gotResult {
			t.Errorf("Want result %t, got %t", test.wantResult, gotResult)
		}
	}
}

func TestVerifyFilterCriteria(t *testing.T) {
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
		{"push", "my-org/my-repo", "main", []string{"pkg/x/y.go"}, workflowAccepted, true},
		{"pull_request", "my-org/my-repo", "main", []string{"pkg/x/y.go"}, "Workflow was rejected because Github event doesn't satisfy filter criteria: pull_request event doesn't match filters [push]", false},
		{"push", "my-org/other-repo", "main", []string{"pkg/x/y.go"}, "Workflow was rejected because Github event doesn't satisfy filter criteria: repository my-org/other-repo doesn't match workflow's repository my-org/my-repo", false},
		{"push", "my-org/my-repo", "dev", []string{"pkg/x/y.go"}, "Workflow was rejected because Github event doesn't satisfy filter criteria: branch dev doesn't match filters [main]", false},
		{"push", "my-org/my-repo", "main", []string{"README.md"}, "Workflow was rejected because Github event doesn't satisfy filter criteria: modified files don't match filters [**/*.go]", false},
	}

	for _, test := range tests {
		event := &github.Event{Name: test.eventName,
			Repository:    test.repo,
			Branch:        test.branch,
			ModifiedFiles: test.files,
		}
		gotResult, gotMessage := VerifyCriteria(workflow, event)
		if test.wantMessage != gotMessage {
			t.Errorf("Want message %s, got %s", test.wantMessage, gotMessage)
		}

		if test.wantResult != gotResult {
			t.Errorf("Want result %t, got %t", test.wantResult, gotResult)
		}
	}
}
