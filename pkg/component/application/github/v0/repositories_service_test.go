package github

import (
	"context"
	"fmt"

	"github.com/google/go-github/v62/github"
)

type MockRepositoriesService struct{}

func (m *MockRepositoriesService) GetCommit(ctx context.Context, owner, repo, sha string, opts *github.ListOptions) (*github.RepositoryCommit, *github.Response, error) {
	switch middleWare(owner) {
	case 403:
		return nil, nil, fmt.Errorf("403 API rate limit exceeded")
	case 404:
		return nil, nil, fmt.Errorf("404 Not Found")
	case 422:
		return nil, nil, fmt.Errorf("422 Unprocessable Entity")
	case 201:
		return nil, nil, nil
	}

	resp := &github.Response{}
	commit := &github.RepositoryCommit{
		SHA: github.String("commitSHA"),
		Commit: &github.Commit{
			Message: github.String("This is a fake commit"),
		},
		Stats: &github.CommitStats{
			Additions: github.Int(1),
			Deletions: github.Int(1),
			Total:     github.Int(2),
		},
		Files: []*github.CommitFile{
			{
				Filename:  github.String("filename"),
				Patch:     github.String("patch"),
				Additions: github.Int(1),
				Deletions: github.Int(1),
				Changes:   github.Int(2),
			},
		},
	}
	return commit, resp, nil
}
func (m *MockRepositoriesService) ListCommits(ctx context.Context, owner, repo string, opts *github.CommitsListOptions) ([]*github.RepositoryCommit, *github.Response, error) {
	switch middleWare(owner) {
	case 403:
		return nil, nil, fmt.Errorf("403 API rate limit exceeded")
	case 201:
		return []*github.RepositoryCommit{}, nil, nil
	}
	resp := &github.Response{}
	commits := []*github.RepositoryCommit{}
	commits = append(commits, &github.RepositoryCommit{
		SHA: github.String("commitSHA"),
		Commit: &github.Commit{
			Message: github.String("This is a fake commit"),
		},
		Stats: &github.CommitStats{
			Additions: github.Int(1),
			Deletions: github.Int(1),
			Total:     github.Int(2),
		},
		Files: []*github.CommitFile{
			{
				Filename:  github.String("filename"),
				Patch:     github.String("patch"),
				Additions: github.Int(1),
				Deletions: github.Int(1),
				Changes:   github.Int(2),
			},
		},
	})
	return commits, resp, nil
}

// CreateHook(context.Context, string, string, *github.Hook) (*github.Hook, *github.Response, error)
func (m *MockRepositoriesService) CreateHook(ctx context.Context, owner, repo string, hook *github.Hook) (*github.Hook, *github.Response, error) {
	switch middleWare(owner) {
	case 403:
		return nil, nil, fmt.Errorf("403 API rate limit exceeded")
	case 404:
		return nil, nil, fmt.Errorf("404 Not Found")
	case 201:
		return nil, nil, nil
	}

	resp := &github.Response{}
	hookResp := &github.Hook{
		ID:      github.Int64(1),
		Name:    github.String("hookName"),
		Active:  github.Bool(true),
		URL:     github.String("hook_url"),
		PingURL: github.String("ping_url"),
		TestURL: github.String("test_url"),
		Config: &github.HookConfig{
			URL:         github.String("hook_url"),
			InsecureSSL: github.String("0"),
			ContentType: github.String("json"),
		},
	}
	return hookResp, resp, nil
}
