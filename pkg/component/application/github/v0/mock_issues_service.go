package github

import (
	"context"
	"fmt"

	"github.com/google/go-github/v62/github"
)

// MockIssuesService is a mock implementation of the IssuesService interface.
type MockIssuesService struct{}

// ListByRepo is a mock implementation of the ListByRepo method for the IssuesService.
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

	// Generate paginated issues
	for i := 1; i <= 10; i++ {
		issues = append(issues, &github.Issue{
			Number: github.Int(i),
			Title:  github.String(fmt.Sprintf("This is a fake Issue #%d", i)),
			State:  github.String("open"),
			Body:   github.String(fmt.Sprintf("Issue Body #%d", i)),
			Assignee: &github.User{
				Login: github.String(fmt.Sprintf("assignee%d", i)),
			},
			Assignees: []*github.User{
				{Login: github.String(fmt.Sprintf("assignee%d_1", i))},
				{Login: github.String(fmt.Sprintf("assignee%d_2", i))},
			},
			Labels: []*github.Label{
				{Name: github.String(fmt.Sprintf("label%d_1", i))},
				{Name: github.String(fmt.Sprintf("label%d_2", i))},
			},
		})
	}

	// Handle pagination if owner is "paginated"
	if owner == "paginated" {
		if opt == nil {
			opt = &github.IssueListByRepoOptions{
				ListOptions: github.ListOptions{
					Page:    1,
					PerPage: 2, // Default page size for tests
				},
			}
		}
		if opt.Page == 0 {
			opt.Page = 1
		}
		if opt.PerPage == 0 {
			opt.PerPage = 2 // Default page size for tests
		}

		start := (opt.Page - 1) * opt.PerPage
		end := start + opt.PerPage
		if start >= len(issues) {
			return []*github.Issue{}, resp, nil
		}
		if end > len(issues) {
			end = len(issues)
		}

		// Calculate total pages correctly
		totalPages := (len(issues) + opt.PerPage - 1) / opt.PerPage

		// Set pagination information
		resp.FirstPage = 1
		resp.LastPage = totalPages

		if opt.Page < totalPages {
			resp.NextPage = opt.Page + 1
			resp.NextPageToken = fmt.Sprintf("page_%d", resp.NextPage)
		}

		if opt.Page > 1 {
			resp.PrevPage = opt.Page - 1
		}

		resp.Cursor = fmt.Sprintf("cursor_%d", opt.Page)
		resp.Before = fmt.Sprintf("before_%d", opt.Page)
		resp.After = fmt.Sprintf("after_%d", opt.Page)

		return issues[start:end], resp, nil
	}

	// For non-paginated cases, return a single issue
	issues = []*github.Issue{
		{
			ID:     github.Int64(1),
			Number: github.Int(1),
			Title:  github.String("This is a fake Issue"),
			Body:   github.String("Issue Body"),
			State:  github.String("open"),
			Assignee: &github.User{
				Login: github.String("assignee"),
			},
			Assignees: []*github.User{
				{Login: github.String("assignee1")},
				{Login: github.String("assignee2")},
			},
			Labels: []*github.Label{
				{Name: github.String("label1")},
				{Name: github.String("label2")},
			},
			PullRequestLinks: nil,
		},
	}
	return issues, resp, nil
}

// Get is a mock implementation of the Get method for the IssuesService.
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
			Login: github.String("assignee"),
		},
		Assignees: []*github.User{
			{
				Login: github.String("assignee1"),
			},
			{
				Login: github.String("assignee2"),
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

// Create is a mock implementation of the Create method for the IssuesService.
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
