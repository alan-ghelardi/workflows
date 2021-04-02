package github

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"regexp"

	"errors"

	"github.com/google/go-github/v33/github"
)

const (

	//	githubDeliveryHeader defines the header that contains a guid identifying the delivery.
	githubDeliveryHeader = "X-GitHub-Delivery"

	// githubHookHeader defines the header that identifies the Webhook that sent the request.
	githubHookHeader = "X-GitHub-Hook-ID"

	//	githubEventHeader defines the header that contains the name of the event delivered.
	githubEventHeader = "X-GitHub-Event"

	// githubSignatureHeader defines the header name for hash signatures sent by Github.
	githubSignatureHeader = "X-Hub-Signature-256"
)

// refsPattern is a regexp used to extract branches from Git references.
var refsPattern = regexp.MustCompile(`^refs/heads/(.*)$`)

// Event represents a Github Webhook event.
type Event struct {
	Body          []byte
	Branch        string
	Data          interface{}
	DeliveryID    string
	HeadCommitSHA string
	HMACSignature []byte
	HookID        string
	Name          string
	Changes       []string
	Repository    string
}

// VerifySignature validates the payload sent by Github Webhooks by calculating
// a hash signature using the provided key and comparing it with the signature
// sent along with the request.
// For further details about the algorithm, please see:
// https://docs.github.com/en/free-pro-team@latest/developers/webhooks-and-events/securing-your-webhooks.
func (e *Event) VerifySignature(webhookSecret []byte) (bool, string) {
	if e.HMACSignature == nil || len(e.HMACSignature) == 0 {
		return false, fmt.Sprintf("Access denied: Github signature header %s is missing", githubSignatureHeader)
	}

	hash := hmac.New(sha256.New, webhookSecret)
	hash.Write(e.Body)
	digest := hash.Sum(nil)

	generatedSignature := make([]byte, hex.EncodedLen(len(digest)))
	hex.Encode(generatedSignature, digest)

	// Drop the prefix sha256= from the HMAC signature sent by Github.
	signature := e.HMACSignature[7:]

	if !hmac.Equal(signature, generatedSignature) {
		return false, "Access denied: HMAC signatures don't match. The request signature we calculated does not match the provided signature."
	}

	return true, "Access permitted: the signature we calculated matches the provided signature."
}

// ParseWebhookEvent creates a new Event object from the supplied HTTP request.
func ParseWebhookEvent(request *http.Request) (*Event, error) {
	eventName := request.Header.Get(githubEventHeader)
	if eventName == "" {
		return nil, errors.New("Request doesn't appear to have been delivered by a Github Webhook")
	}

	body, err := ioutil.ReadAll(request.Body)
	if err != nil {
		return nil, fmt.Errorf("Error reading request body: %w", err)
	}

	event := &Event{
		Body:          body,
		DeliveryID:    request.Header.Get(githubDeliveryHeader),
		HMACSignature: []byte(request.Header.Get(githubSignatureHeader)),
		HookID:        request.Header.Get(githubHookHeader),
		Name:          eventName,
	}

	eventPayload, err := github.ParseWebHook(event.Name, event.Body)
	if err != nil {
		return nil, fmt.Errorf("Error parsing event payload: %w", err)
	}

	event.Data = eventPayload

	event.Repository = getRepoFullName(eventPayload)

	switch event.Name {
	case "push":
		pushEvent := eventPayload.(*github.PushEvent)
		event.Branch = getBranch(*pushEvent.Ref)
		event.HeadCommitSHA = *pushEvent.HeadCommit.ID
		event.Changes = collectChanges(pushEvent)

	case "pull_request":
		pullRequestEvent := eventPayload.(*github.PullRequestEvent)
		event.HeadCommitSHA = *pullRequestEvent.PullRequest.Head.SHA
		event.Branch = getBranch(*pullRequestEvent.PullRequest.Head.Ref)
	}

	return event, nil
}

// getRepoFullName returns the repository's full name (owner/name) using
// reflection or an empty string if the value can't be obtained.
func getRepoFullName(event interface{}) string {
	value := reflect.ValueOf(event).Elem()
	field := value.FieldByName("Repo")
	if !field.IsValid() {
		return ""
	}

	repo := field.Interface()

	if repo == nil {
		return ""
	}

	switch r := repo.(type) {
	case *github.PushEventRepository:
		return *r.FullName

	case *github.Repository:
		return *r.FullName

	default:
		return ""
	}
}

// getBranch returns the name of the branch taken from the Git reference.
func getBranch(reference string) string {
	matches := refsPattern.FindStringSubmatch(reference)

	if matches == nil {
		return ""
	}
	return matches[len(matches)-1]
}

// collectChanges returns all files that have been added, modified or removed in
// commits associated to the push event in question.
func collectChanges(event *github.PushEvent) []string {
	set := make(map[string]bool)
	changes := make([]string, 0)
	addAll := func(files []string) {
		for _, file := range files {
			if !set[file] {
				set[file] = true
				changes = append(changes, file)
			}
		}
	}

	for _, commit := range event.Commits {
		addAll(commit.Added)
		addAll(commit.Modified)
		addAll(commit.Removed)
	}
	return changes
}

// eventKey identifies Event objects in contexts.
type eventKey struct {
}

// WithEvent returns a copy of the supplied context with the Event object added.
func WithEvent(ctx context.Context, event *Event) context.Context {
	return context.WithValue(ctx, eventKey{}, event)
}

// GetEvent returns the Event object stored in the supplied context.
func GetEvent(ctx context.Context) *Event {
	return ctx.Value(eventKey{}).(*Event)
}
