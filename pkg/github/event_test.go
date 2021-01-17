package github

import (
	"context"
	"io/ioutil"
	"net/http"
	"sort"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestReturnsAnErrorWhenTheEventNameIsmissing(t *testing.T) {
	_, err := ParseWebhookEvent(&http.Request{})

	if err == nil {
		t.Error("Want error, but got a well-formed Event object")
	}

	want := "Request doesn't appear to have been delivered by a Github Webhook"
	got := err.Error()

	if got != want {
		t.Errorf("Want message %s, but got %s", want, got)
	}
}

func TestReturnsAnErrorWhenPayloadCannotBeParsed(t *testing.T) {
	// Malformed JSON
	payload := ioutil.NopCloser(strings.NewReader(`{
"ref": "refs/heads/dev
}
`))
	request := &http.Request{
		Body:   payload,
		Header: http.Header{},
	}
	request.Header.Set("X-GitHub-Event", "push")

	_, err := ParseWebhookEvent(request)

	if err == nil {
		t.Error("Want error, but got a well formed Event object")
	}

	want := "Error parsing event payload"
	got := err.Error()

	if !strings.Contains(got, want) {
		t.Errorf("Want error message containing %s, but got %s", want, got)
	}
}

func TestParsesThePushEventProperly(t *testing.T) {
	payload := `{
    "commits": [
	{
	    "added": [
		"pkg/foo/foo.go",
		"pkg/foo/foo_test.go"
	    ],
	    "modified": [
		"README.md"
	    ]
	},
	{
	    "removed": [
		"build/build.sh"
	    ],
	    "added": [
		"Makefile"
	    ],
	    "modified": [
		"pkg/foo/foo.go",
		"pkg/foo/foo_test.go"
	    ]
	}
    ],
    "head_commit": {
	"id": "32eec86"
    },
    "ref": "refs/heads/main",
    "repository": {
	"full_name": "my-org/my-repo"
    }
}`

	request := &http.Request{
		Header: http.Header{},
		Body:   ioutil.NopCloser(strings.NewReader(payload)),
	}
	request.Header.Set("X-GitHub-Delivery", "123")
	request.Header.Set("X-GitHub-Event", "push")
	request.Header.Set("X-GitHub-Hook-ID", "456")
	request.Header.Set("X-Hub-Signature-256", "sha256=d8a72707")

	event, err := ParseWebhookEvent(request)

	if err != nil {
		t.Errorf("Want a well-formed event, but got error: %s", err)
		t.FailNow()
	}

	if string(event.Body) != payload {
		t.Errorf("event.Body: unexpected value %s", event.Body)
	}

	wantBranch := "main"
	gotBranch := event.Branch
	if wantBranch != gotBranch {
		t.Errorf("event.Branch: want %s, but got %s", wantBranch, gotBranch)
	}

	wantDeliveryID := "123"
	gotDeliveryID := event.DeliveryID
	if wantDeliveryID != gotDeliveryID {
		t.Errorf("event.DeliveryID: want %s, but got %s", wantDeliveryID, gotDeliveryID)
	}

	wantHeadCommitSHA := "32eec86"
	gotHeadCommitSHA := event.HeadCommitSHA
	if wantHeadCommitSHA != gotHeadCommitSHA {
		t.Errorf("event.HeadCommitSHA: want %s, but got %s", wantHeadCommitSHA, gotHeadCommitSHA)
	}

	wantHMACSignature := "sha256=d8a72707"
	gotHMACSignature := string(event.HMACSignature)

	if wantHMACSignature != gotHMACSignature {
		t.Errorf("event.HMACSignature: want %s, but got %s", wantHMACSignature, gotHMACSignature)
	}

	wantHookID := "456"
	gotHookID := event.HookID
	if wantHookID != gotHookID {
		t.Errorf("event.HookID: want %s, but got %s", wantHookID, gotHookID)
	}

	wantEventName := "push"
	gotEventName := event.Name
	if wantEventName != gotEventName {
		t.Errorf("event.Name: want %s, but got %s", wantEventName, gotEventName)
	}

	wantRepository := "my-org/my-repo"
	gotRepository := event.Repository
	if wantRepository != gotRepository {
		t.Errorf("event.Repository: want %s, but got %s", wantRepository, gotRepository)
	}

	wantChanges := []string{
		"pkg/foo/foo.go",
		"pkg/foo/foo_test.go",
		"build/build.sh",
		"README.md",
		"Makefile",
	}
	gotChanges := event.Changes

	sort.Strings(wantChanges)
	sort.Strings(gotChanges)

	if diff := cmp.Diff(wantChanges, gotChanges); diff != "" {
		t.Errorf("Mismatch (-want +got):\n%s", diff)
	}
}

func TestParsesThePullRequestEventProperly(t *testing.T) {
	payload := `{
    "pull_request": {
	"head": {
	    "ref": "refs/heads/dev",
	    "sha": "32eec86"
	}
    },
    "repository": {
	"full_name": "my-org/my-repo"
    }
}`

	request := &http.Request{
		Header: http.Header{},
		Body:   ioutil.NopCloser(strings.NewReader(payload)),
	}
	request.Header.Set("X-GitHub-Event", "pull_request")

	event, err := ParseWebhookEvent(request)
	if err != nil {
		t.Errorf("Want a well-formed event, but got error: %s", err)
		t.FailNow()
	}

	wantBranch := "dev"
	gotBranch := event.Branch
	if wantBranch != gotBranch {
		t.Errorf("event.Branch: want %s, but got %s", wantBranch, gotBranch)
	}

	wantHeadCommitSHA := "32eec86"
	gotHeadCommitSHA := event.HeadCommitSHA
	if wantHeadCommitSHA != gotHeadCommitSHA {
		t.Errorf("event.HeadCommitSHA: want %s, but got %s", wantHeadCommitSHA, gotHeadCommitSHA)
	}

	wantEventName := "pull_request"
	gotEventName := event.Name
	if wantEventName != gotEventName {
		t.Errorf("event.Name: want %s, but got %s", wantEventName, gotEventName)
	}

	wantRepository := "my-org/my-repo"
	gotRepository := event.Repository
	if wantRepository != gotRepository {
		t.Errorf("event.Repository: want %s, but got %s", wantRepository, gotRepository)
	}
}

func TestGetBranch(t *testing.T) {
	tests := []struct {
		ref    string
		branch string
	}{
		{"refs/heads/main", "main"},
		{"refs/heads/issue10/fix-foo", "issue10/fix-foo"},
		{"refs/heads/issue11/foo/bar", "issue11/foo/bar"},
	}

	for _, test := range tests {
		gotBranch := getBranch(test.ref)
		if test.branch != gotBranch {
			t.Errorf("Want branch %s, but got %s", test.branch, gotBranch)
		}
	}
}

func TestDeniesRequestsIfSignatureIsMissing(t *testing.T) {
	events := []*Event{
		{HMACSignature: nil},
		{HMACSignature: []byte{}}}

	wantMessage := "Access denied: Github signature header X-Hub-Signature-256 is missing"

	for _, event := range events {
		valid, message := event.VerifySignature([]byte{})

		if valid {
			t.Error("Want an invalid signature result, but got a valid one")
		}

		if wantMessage != message {
			t.Errorf("Want message %s, but got %s", wantMessage, message)
		}
	}
}

func TestAcceptsRequestsWhenSignaturesMatch(t *testing.T) {
	event := &Event{Body: []byte(`{
    "ref": "refs/heads/dev"
}`),
		HMACSignature: []byte("sha256=4ae9df17f8cc696722c87f771f0c60fa7b03d44488ae3e0f712f570c4e7a3888"),
	}

	webhookSecret := []byte("secret")

	if valid, _ := event.VerifySignature(webhookSecret); !valid {
		t.Errorf("Want a valid result for the signature validation, but got an invalid one")
	}
}

func TestRejectsRequestsWhenSignaturesDoNotMatch(t *testing.T) {
	event := &Event{Body: []byte(`{
    "ref": "refs/heads/dev"
}`),
		// This digest was calculated with the key other-secret.
		HMACSignature: []byte("sha256=d8a72707bd05f566becba60815c77f1e2adddddfceed668ca4844489d12ded07"),
	}

	webhookSecret := []byte("secret")

	if valid, _ := event.VerifySignature(webhookSecret); valid {
		t.Errorf("Want a invalid result for the signature validation, but got a valid one")
	}
}

func TestContextInfusedWithEvent(t *testing.T) {
	ctx := context.Background()
	wantEvent := &Event{Name: "push"}
	gotEvent := GetEvent(WithEvent(ctx, wantEvent))

	if wantEvent != gotEvent {
		t.Errorf("Want event %+v, got %+v", wantEvent, gotEvent)
	}
}
