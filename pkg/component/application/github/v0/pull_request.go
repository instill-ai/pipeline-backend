package github

import (
	"context"

	"github.com/google/go-github/v62/github"
)

type PullRequestService interface {
	List(context.Context, string, string, *github.PullRequestListOptions) ([]*github.PullRequest, *github.Response, error)
	Get(context.Context, string, string, int) (*github.PullRequest, *github.Response, error)
	ListComments(context.Context, string, string, int, *github.PullRequestListCommentsOptions) ([]*github.PullRequestComment, *github.Response, error)
	CreateComment(context.Context, string, string, int, *github.PullRequestComment) (*github.PullRequestComment, *github.Response, error)
	ListCommits(context.Context, string, string, int, *github.ListOptions) ([]*github.RepositoryCommit, *github.Response, error)
}

type PullRequest struct {
	ID                int64    `json:"id"`
	Number            int      `json:"number"`
	State             string   `json:"state"`
	Title             string   `json:"title"`
	Body              string   `json:"body"`
	DiffURL           string   `json:"diff-url,omitempty"`
	CommitsURL        string   `json:"commits-url,omitempty"`
	Commits           []Commit `json:"commits"`
	Head              string   `json:"head"`
	Base              string   `json:"base"`
	CommentsNum       int      `json:"comments-num"`
	CommitsNum        int      `json:"commits-num"`
	ReviewCommentsNum int      `json:"review-comments-num"`
}

func (client *Client) extractPullRequestInformation(ctx context.Context, owner string, repository string, originalPr *github.PullRequest, needCommitDetails bool) (PullRequest, error) {
	resp := PullRequest{
		ID:                originalPr.GetID(),
		Number:            originalPr.GetNumber(),
		State:             originalPr.GetState(),
		Title:             originalPr.GetTitle(),
		Body:              originalPr.GetBody(),
		DiffURL:           originalPr.GetDiffURL(),
		Head:              originalPr.GetHead().GetSHA(),
		Base:              originalPr.GetBase().GetSHA(),
		CommentsNum:       originalPr.GetComments(),
		CommitsNum:        originalPr.GetCommits(),
		ReviewCommentsNum: originalPr.GetReviewComments(),
	}
	if originalPr.GetCommitsURL() != "" {
		commits, _, err := client.PullRequests.ListCommits(ctx, owner, repository, resp.Number, nil)
		if err != nil {
			return PullRequest{}, addErrMsgToClientError(err)
		}
		resp.Commits = make([]Commit, len(commits))
		for idx, commit := range commits {
			resp.Commits[idx] = client.extractCommitInformation(ctx, owner, repository, commit, needCommitDetails)
		}
	}
	return resp, nil
}

type ListPullRequestsInput struct {
	RepoInfo
	State     string `json:"state"`
	Sort      string `json:"sort"`
	Direction string `json:"direction"`
	PageOptions
}
type ListPullRequestsResp struct {
	PullRequests []PullRequest `json:"pull-requests"`
}
