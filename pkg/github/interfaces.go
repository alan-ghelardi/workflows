package github

import (
	"context"

	"github.com/google/go-github/v33/github"
)

type keysService interface {
	GetKey(ctx context.Context, owner string, repo string, id int64) (*github.Key, *github.Response, error)
	CreateKey(ctx context.Context, owner string, repo string, key *github.Key) (*github.Key, *github.Response, error)
	DeleteKey(ctx context.Context, owner string, repo string, id int64) (*github.Response, error)
}

type repositoryService interface {
	Get(ctx context.Context, owner, repo string) (*github.Repository, *github.Response, error)
}
