package github

import (
	"context"
	"fmt"

	"github.com/google/go-github/v62/github"
)

type MockIssuesService struct{}

func (m *MockIssuesService) ListByRepo(ctx context.Context, owner, repo string, opt *github.IssueListByRepoOptions) ([]*github.Issue, *github.Response, error) {
	switch middleWare(owner) {
	case 403:
		return nil, nil, fmt.Errorf("403 API rate limit exceeded")
	case 404:
		return nil, nil, fmt.Errorf("404 Not Found")
	case 201:
		return []*github.Issue{}, nil, nil
	}

	resp := &github.Response{}
	issues := []*github.Issue{}
	issues = append(issues, &github.Issue{
		ID:     github.Int64(1),
		Number: github.Int(1),
		Title:  github.String("This is a fake Issue"),
		Body:   github.String("Issue Body"),
		State:  github.String("open"),
		Assignee: &github.User{
			Name: github.String("assignee"),
		},
		Assignees: []*github.User{
			{
				Name: github.String("assignee1"),
			},
			{
				Name: github.String("assignee2"),
			},
		},
		Labels: []*github.Label{
			{
				Name: github.String("label1"),
			},
			{
				Name: github.String("label2"),
			},
		},
		PullRequestLinks: nil,
	})
	return issues, resp, nil
}
func (m *MockIssuesService) Get(ctx context.Context, owner, repo string, number int) (*github.Issue, *github.Response, error) {
	switch middleWare(owner) {
	case 403:
		return nil, nil, fmt.Errorf("403 API rate limit exceeded")
	case 404:
		return nil, nil, fmt.Errorf("404 Not Found")
	case 201:
		return &github.Issue{}, nil, nil
	}
	resp := &github.Response{}
	issue := &github.Issue{
		ID:     github.Int64(1),
		Number: github.Int(1),
		Title:  github.String("This is a fake Issue"),
		Body:   github.String("Issue Body"),
		State:  github.String("open"),
		Assignee: &github.User{
			Name: github.String("assignee"),
		},
		Assignees: []*github.User{
			{
				Name: github.String("assignee1"),
			},
			{
				Name: github.String("assignee2"),
			},
		},
		Labels: []*github.Label{
			{
				Name: github.String("label1"),
			},
			{
				Name: github.String("label2"),
			},
		},
		PullRequestLinks: nil,
	}
	return issue, resp, nil
}
func (m *MockIssuesService) Create(ctx context.Context, owner, repo string, issue *github.IssueRequest) (*github.Issue, *github.Response, error) {
	switch middleWare(owner) {
	case 403:
		return nil, nil, fmt.Errorf("403 API rate limit exceeded")
	case 404:
		return nil, nil, fmt.Errorf("404 Not Found")
	case 201:
		return &github.Issue{}, nil, nil
	}
	resp := &github.Response{}

	newIssue := &github.Issue{
		ID:               github.Int64(1),
		Number:           github.Int(1),
		Title:            issue.Title,
		Body:             issue.Body,
		State:            github.String("open"),
		PullRequestLinks: nil,
	}
	return newIssue, resp, nil
}
