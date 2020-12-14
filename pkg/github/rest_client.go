package github

import (
	"context"

	"github.com/google/go-github/v33/github"
	"golang.org/x/oauth2"
)

// NotFoundError is returned when certain Github resources aren't found.
type NotFoundError struct {
	msg string
}

// Error satisfies the error interface.
func (n *NotFoundError) Error() string {
	return n.msg
}

// IsNotFound returns true if the supplied error is of the type NotFoundError
// otherwise it returns false.
func IsNotFound(e error) bool {
	switch e.(type) {
	case *NotFoundError:
		return true
	}
	return false
}

const token = ""

// NewClient returns a client to talk to Github APIs.
func NewClient() *github.Client {
	ctx := context.Background()
	tokenSource := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token})
	tokenClient := oauth2.NewClient(ctx, tokenSource)
	return github.NewClient(tokenClient)
}
