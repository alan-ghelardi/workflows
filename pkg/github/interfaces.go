package github

import (
	"context"

	"github.com/google/go-github/v33/github"
)

// Interfaces declared in this file represent various Github services that our
// operator interact with. Their methods follow the signatures of corresponding
// methods on the go-github API in order to allow us to mock these calls in unit
// tests.

type contentsService interface {
	GetContents(ctx context.Context, owner, repo, path string, opts *github.RepositoryContentGetOptions) (fileContent *github.RepositoryContent, directoryContent []*github.RepositoryContent, resp *github.Response, err error)
}

type keysService interface {
	GetKey(ctx context.Context, owner string, repo string, id int64) (*github.Key, *github.Response, error)
	CreateKey(ctx context.Context, owner string, repo string, key *github.Key) (*github.Key, *github.Response, error)
	DeleteKey(ctx context.Context, owner string, repo string, id int64) (*github.Response, error)
}

type hooksService interface {
	GetHook(ctx context.Context, owner, repo string, id int64) (*github.Hook, *github.Response, error)
	CreateHook(ctx context.Context, owner, repo string, hook *github.Hook) (*github.Hook, *github.Response, error)
	EditHook(ctx context.Context, owner, repo string, id int64, hook *github.Hook) (*github.Hook, *github.Response, error)
	DeleteHook(ctx context.Context, owner, repo string, id int64) (*github.Response, error)
}

type repositoriesService interface {
	Get(ctx context.Context, owner, repo string) (*github.Repository, *github.Response, error)
}
