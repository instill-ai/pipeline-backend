package github

import (
	"context"
	"fmt"

	"github.com/google/go-github/v62/github"

	"github.com/instill-ai/x/errmsg"
)

// RepositoriesService is a wrapper around the github.RepositoriesService
type RepositoriesService interface {
	GetCommit(ctx context.Context, owner string, repository string, sha string, opts *github.ListOptions) (*github.RepositoryCommit, *github.Response, error)
	ListHooks(ctx context.Context, owner string, repository string, opts *github.ListOptions) ([]*github.Hook, *github.Response, error)
	GetHook(ctx context.Context, owner string, repository string, id int64) (*github.Hook, *github.Response, error)
	CreateHook(ctx context.Context, owner string, repository string, hook *github.Hook) (*github.Hook, *github.Response, error)
	DeleteHook(ctx context.Context, owner string, repository string, id int64) (*github.Response, error)
	EditHook(ctx context.Context, owner string, repository string, id int64, hook *github.Hook) (*github.Hook, *github.Response, error)
}

// RepoInfoInterface is an interface for the RepoInfo struct
type RepoInfoInterface interface {
	getOwner() (string, error)
	getRepository() (string, error)
}

// RepoInfo is a struct that contains the owner and repository of a GitHub repository
type RepoInfo struct {
	Owner      string `instill:"owner"`
	Repository string `instill:"repository"`
}

func (info RepoInfo) getOwner() (string, error) {
	if info.Owner == "" {
		return "", errmsg.AddMessage(
			fmt.Errorf("owner not provided"),
			"Owner not provided.",
		)
	}
	return info.Owner, nil
}

func (info RepoInfo) getRepository() (string, error) {
	if info.Repository == "" {
		return "", errmsg.AddMessage(
			fmt.Errorf("repository not provided"),
			"Repository not provided.",
		)
	}
	return info.Repository, nil
}
