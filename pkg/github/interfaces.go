package github

import (
	"context"

	"github.com/google/go-github/v33/github"
)

type repositoryService interface {
	Get(ctx context.Context, owner, repo string) (*github.Repository, *github.Response, error)
}
