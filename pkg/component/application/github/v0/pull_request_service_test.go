package github

import (
	"context"
	"fmt"

	"github.com/google/go-github/v62/github"
)

type MockPullRequestService struct {
	Client *github.Client
}

func (m *MockPullRequestService) List(ctx context.Context, owner, repo string, opts *github.PullRequestListOptions) ([]*github.PullRequest, *github.Response, error) {
	switch middleWare(owner) {
	case 403:
		return nil, nil, fmt.Errorf("403 API rate limit exceeded")
	case 404:
		return nil, nil, fmt.Errorf("404 Not Found")
	case 201:
		return []*github.PullRequest{}, nil, nil
	}

	resp := &github.Response{}
	prs := []*github.PullRequest{}
	diffURL := fmt.Sprintf("%v/%v/%v/pull/%v.diff", fakeHost, owner, repo, 1)
	commitsURL := fmt.Sprintf("%v/%v/%v/pull/%v/commits", fakeHost, owner, repo, 1)
	prs = append(prs, &github.PullRequest{
		ID:      github.Int64(1),
		Number:  github.Int(1),
		Title:   github.String("This is a fake PR"),
		Body:    github.String("PR Body"),
		DiffURL: github.String(diffURL),
		Head: &github.PullRequestBranch{
			SHA: github.String("headSHA"),
		},
		Base: &github.PullRequestBranch{
			SHA: github.String("baseSHA"),
		},
		Comments:       github.Int(0),
		Commits:        github.Int(1),
		ReviewComments: github.Int(2),
		State:          github.String("open"),
		CommitsURL:     github.String(commitsURL),
	})
	return prs, resp, nil
}
func (m *MockPullRequestService) Get(ctx context.Context, owner, repo string, number int) (*github.PullRequest, *github.Response, error) {
	switch middleWare(owner) {
	case 403:
		return nil, nil, fmt.Errorf("403 API rate limit exceeded")
	case 404:
		return nil, nil, fmt.Errorf("404 Not Found")
	case 201:
		return nil, nil, nil
	}
	resp := &github.Response{}
	diffURL := fmt.Sprintf("%v/%v/%v/pull/%v.diff", fakeHost, owner, repo, number)
	commitsURL := fmt.Sprintf("%v/%v/%v/pull/%v/commits", fakeHost, owner, repo, number)
	prs := &github.PullRequest{
		ID:      github.Int64(1),
		Number:  github.Int(number),
		Title:   github.String("This is a fake PR"),
		Body:    github.String("PR Body"),
		DiffURL: github.String(diffURL),
		Head: &github.PullRequestBranch{
			SHA: github.String("headSHA"),
		},
		Base: &github.PullRequestBranch{
			SHA: github.String("baseSHA"),
		},
		Comments:       github.Int(0),
		Commits:        github.Int(1),
		ReviewComments: github.Int(2),
		State:          github.String("open"),
		CommitsURL:     github.String(commitsURL),
	}
	return prs, resp, nil
}
func (m *MockPullRequestService) ListComments(ctx context.Context, owner, repo string, number int, opts *github.PullRequestListCommentsOptions) ([]*github.PullRequestComment, *github.Response, error) {
	switch middleWare(owner) {
	case 403:
		return nil, nil, fmt.Errorf("403 API rate limit exceeded")
	case 404:
		return nil, nil, fmt.Errorf("404 Not Found")
	case 201:
		return []*github.PullRequestComment{}, nil, nil
	}
	resp := &github.Response{}
	comments := []*github.PullRequestComment{}
	comments = append(comments, &github.PullRequestComment{
		ID:   github.Int64(1),
		Body: github.String("This is a fake comment"),
	})
	return comments, resp, nil
}
func (m *MockPullRequestService) CreateComment(ctx context.Context, owner, repo string, number int, comment *github.PullRequestComment) (*github.PullRequestComment, *github.Response, error) {
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
	if comment.StartLine != nil && *comment.Line <= *comment.StartLine {
		return nil, nil, fmt.Errorf("422 Unprocessable Entity")
	}

	resp := &github.Response{}
	comment.ID = github.Int64(1)
	return comment, resp, nil
}
func (m *MockPullRequestService) ListCommits(ctx context.Context, owner, repo string, number int, opts *github.ListOptions) ([]*github.RepositoryCommit, *github.Response, error) {
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
