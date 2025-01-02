package github

import (
	"context"
	"fmt"

	"github.com/google/go-github/v62/github"
)

// MockPullRequestsService is a mock implementation of the PullRequestService interface.
type MockPullRequestsService struct {
	Client *github.Client
}

// List is a mock implementation of the List method for the PullRequestService.
func (m *MockPullRequestsService) List(ctx context.Context, owner, repo string, opts *github.PullRequestListOptions) ([]*github.PullRequest, *github.Response, error) {
	switch middleWare(owner) {
	case 403:
		return nil, nil, fmt.Errorf("403 API rate limit exceeded")
	case 404:
		return nil, nil, fmt.Errorf("404 Not Found")
	case 201:
		return []*github.PullRequest{}, nil, nil
	}

	resp := &github.Response{}

	// Return single PR for non-paginated case
	if owner == "non-paginated" {
		diffURL := fmt.Sprintf("%v/%v/%v/pull/%v.diff", fakeHost, owner, repo, 1)
		commitsURL := fmt.Sprintf("%v/%v/%v/pull/%v/commits", fakeHost, owner, repo, 1)
		return []*github.PullRequest{
			{
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
			},
		}, resp, nil
	}

	// Generate paginated PRs for "paginated" owner
	prs := []*github.PullRequest{}
	for i := 1; i <= 10; i++ {
		diffURL := fmt.Sprintf("%v/%v/%v/pull/%v.diff", fakeHost, owner, repo, i)
		commitsURL := fmt.Sprintf("%v/%v/%v/pull/%v/commits", fakeHost, owner, repo, i)
		prs = append(prs, &github.PullRequest{
			ID:      github.Int64(int64(i)),
			Number:  github.Int(i),
			Title:   github.String(fmt.Sprintf("This is a fake PR #%d", i)),
			Body:    github.String(fmt.Sprintf("PR Body #%d", i)),
			DiffURL: github.String(diffURL),
			Head: &github.PullRequestBranch{
				SHA: github.String(fmt.Sprintf("headSHA%d", i)),
			},
			Base: &github.PullRequestBranch{
				SHA: github.String(fmt.Sprintf("baseSHA%d", i)),
			},
			Comments:       github.Int(i),
			Commits:        github.Int(i),
			ReviewComments: github.Int(i * 2),
			State:          github.String("open"),
			CommitsURL:     github.String(commitsURL),
		})
	}

	// Handle pagination if owner is "paginated"
	if owner == "paginated" {
		if opts == nil {
			opts = &github.PullRequestListOptions{
				ListOptions: github.ListOptions{
					Page:    1,
					PerPage: 30, // GitHub default
				},
			}
		}
		if opts.Page == 0 {
			opts.Page = 1
		}
		if opts.PerPage == 0 {
			opts.PerPage = 30 // GitHub default
		}

		start := (opts.Page - 1) * opts.PerPage
		end := start + opts.PerPage
		if start >= len(prs) {
			// Return empty slice when starting beyond available data
			return []*github.PullRequest{}, resp, nil
		}
		if end > len(prs) {
			end = len(prs)
		}

		// Calculate total pages
		totalPages := (len(prs) + opts.PerPage - 1) / opts.PerPage

		// Set pagination information
		resp.FirstPage = 1
		resp.LastPage = totalPages

		if opts.Page < totalPages {
			resp.NextPage = opts.Page + 1
			resp.NextPageToken = fmt.Sprintf("page_%d", resp.NextPage)
		}

		if opts.Page > 1 {
			resp.PrevPage = opts.Page - 1
		}

		resp.Cursor = fmt.Sprintf("cursor_%d", opts.Page)
		resp.Before = fmt.Sprintf("before_%d", opts.Page)
		resp.After = fmt.Sprintf("after_%d", opts.Page)

		return prs[start:end], resp, nil
	}

	return prs, resp, nil
}

// Get is a mock implementation of the Get method for the PullRequestService.
func (m *MockPullRequestsService) Get(ctx context.Context, owner, repo string, number int) (*github.PullRequest, *github.Response, error) {
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

// ListComments is a mock implementation of the ListComments method for the PullRequestService.
func (m *MockPullRequestsService) ListComments(ctx context.Context, owner, repo string, number int, opts *github.PullRequestListCommentsOptions) ([]*github.PullRequestComment, *github.Response, error) {
	switch middleWare(owner) {
	case 403:
		return nil, nil, fmt.Errorf("403 API rate limit exceeded")
	case 404:
		return nil, nil, fmt.Errorf("404 Not Found")
	case 201:
		return []*github.PullRequestComment{}, nil, nil
	}
	resp := &github.Response{}

	// Return single comment for non-paginated case
	if owner == "non-paginated" {
		return []*github.PullRequestComment{
			{
				ID:   github.Int64(1),
				Body: github.String("This is a fake comment"),
			},
		}, resp, nil
	}

	// Generate paginated comments for "paginated" owner
	comments := []*github.PullRequestComment{}
	for i := 1; i <= 10; i++ {
		comments = append(comments, &github.PullRequestComment{
			ID:   github.Int64(int64(i)),
			Body: github.String(fmt.Sprintf("This is a fake comment #%d", i)),
		})
	}

	// Handle pagination if owner is "paginated"
	if owner == "paginated" {
		if opts == nil {
			opts = &github.PullRequestListCommentsOptions{
				ListOptions: github.ListOptions{
					Page:    1,
					PerPage: 30, // GitHub default
				},
			}
		}
		if opts.Page == 0 {
			opts.Page = 1
		}
		if opts.PerPage == 0 {
			opts.PerPage = 30 // GitHub default
		}

		start := (opts.Page - 1) * opts.PerPage
		end := start + opts.PerPage
		if start >= len(comments) {
			// Return empty slice when starting beyond available data
			return []*github.PullRequestComment{}, resp, nil
		}
		if end > len(comments) {
			end = len(comments)
		}

		// Calculate total pages
		totalPages := (len(comments) + opts.PerPage - 1) / opts.PerPage

		// Set pagination information
		resp.FirstPage = 1
		resp.LastPage = totalPages

		if opts.Page < totalPages {
			resp.NextPage = opts.Page + 1
			resp.NextPageToken = fmt.Sprintf("page_%d", resp.NextPage)
		}

		if opts.Page > 1 {
			resp.PrevPage = opts.Page - 1
		}

		resp.Cursor = fmt.Sprintf("cursor_%d", opts.Page)
		resp.Before = fmt.Sprintf("before_%d", opts.Page)
		resp.After = fmt.Sprintf("after_%d", opts.Page)

		return comments[start:end], resp, nil
	}

	return comments, resp, nil
}

// CreateComment is a mock implementation of the CreateComment method for the PullRequestService.
func (m *MockPullRequestsService) CreateComment(ctx context.Context, owner, repo string, number int, comment *github.PullRequestComment) (*github.PullRequestComment, *github.Response, error) {
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

// ListCommits is a mock implementation of the ListCommits method for the PullRequestService.
func (m *MockPullRequestsService) ListCommits(ctx context.Context, owner, repo string, number int, opts *github.ListOptions) ([]*github.RepositoryCommit, *github.Response, error) {
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
