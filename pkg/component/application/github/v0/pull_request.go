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
	ID                int64    `instill:"id"`
	Number            int      `instill:"number"`
	State             string   `instill:"state"`
	Title             string   `instill:"title"`
	Body              string   `instill:"body"`
	DiffURL           string   `instill:"diff-url"`
	CommitsURL        string   `instill:"commits-url"`
	Commits           []Commit `instill:"commits"`
	Head              string   `instill:"head"`
	Base              string   `instill:"base"`
	CommentsNum       int      `instill:"comments-num"`
	CommitsNum        int      `instill:"commits-num"`
	ReviewCommentsNum int      `instill:"review-comments-num"`
}

type PullRequestComment struct {
	ID                  *int64     `instill:"id"`
	NodeID              *string    `instill:"node-id"`
	InReplyTo           *int64     `instill:"in-reply-to-id"`
	Body                *string    `instill:"body"`
	Path                *string    `instill:"path"`
	DiffHunk            *string    `instill:"diff-hunk"`
	PullRequestReviewID *int64     `instill:"pull-request-review-id"`
	Position            *int       `instill:"position"`
	OriginalPosition    *int       `instill:"original-position"`
	StartLine           *int       `instill:"start-line"`
	Line                *int       `instill:"line"`
	OriginalLine        *int       `instill:"original-line"`
	OriginalStartLine   *int       `instill:"original-start-line"`
	Side                *string    `instill:"side"`
	StartSide           *string    `instill:"start-side"`
	CommitID            *string    `instill:"commit-id"`
	OriginalCommitID    *string    `instill:"original-commit-id"`
	User                *User      `instill:"user"`
	Reactions           *Reactions `instill:"reactions"`
	CreatedAt           *Timestamp `instill:"created-at"`
	UpdatedAt           *Timestamp `instill:"updated-at"`
	// AuthorAssociation is the comment author's relationship to the pull request's repository.
	// Possible values are "COLLABORATOR", "CONTRIBUTOR", "FIRST_TIMER", "FIRST_TIME_CONTRIBUTOR", "MEMBER", "OWNER", or "NONE".
	AuthorAssociation *string `instill:"author-association"`
	URL               *string `instill:"url"`
	HTMLURL           *string `instill:"html-url"`
	PullRequestURL    *string `instill:"pull-request-url"`
	// Can be one of: LINE, FILE from https://docs.github.com/rest/pulls/comments#create-a-review-comment-for-a-pull-request
	SubjectType *string `instill:"subject-type"`
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
	State     string `instill:"state"`
	Sort      string `instill:"sort"`
	Direction string `instill:"direction"`
	PageOptions
}
type ListPullRequestsResp struct {
	PullRequests []PullRequest `instill:"pull-requests"`
}
