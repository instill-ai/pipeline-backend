package github

import "github.com/google/go-github/v62/github"

type createIssueInput struct {
	RepoInfo
	Title     string   `instill:"title"`
	Body      string   `instill:"body"`
	Assignees []string `instill:"assignees"`
	Labels    []string `instill:"labels"`
}

type createIssueOutput struct {
	Issue
}

type createReviewCommentInput struct {
	RepoInfo
	PRNumber int                       `instill:"pr-number"`
	Comment  github.PullRequestComment `instill:"comment"`
}

type createReviewCommentOutput struct {
	ReviewComment
}

type createWebHookInput struct {
	RepoInfo
	HookURL     string   `instill:"hook-url"`
	HookSecret  string   `instill:"hook-secret"`
	Events      []string `instill:"events"`
	Active      bool     `instill:"active"`
	ContentType string   `instill:"content-type"`
}

type createWebHookOutput struct {
	HookInfo
}

type getCommitInput struct {
	RepoInfo
	SHA string `instill:"sha"`
}

type getCommitOutput struct {
	Commit Commit `instill:"commit"`
}

type getIssueInput struct {
	RepoInfo
	IssueNumber int `instill:"issue-number"`
}

type getIssueOutput struct {
	Issue
}

type getPullRequestInput struct {
	RepoInfo
	PRNumber float64 `instill:"pr-number"`
}

type getPullRequestOutput struct {
	PullRequest
}

type listIssuesInput struct {
	RepoInfo
	State         string `instill:"state"`
	Sort          string `instill:"sort"`
	Direction     string `instill:"direction"`
	Since         string `instill:"since"`
	NoPullRequest bool   `instill:"no-pull-request"`
	PageOptions
}

type listIssuesOutput struct {
	Issues []Issue `instill:"issues"`
}

type listPullRequestsInput struct {
	RepoInfo
	PageOptions
	State     string `instill:"state"`
	Sort      string `instill:"sort"`
	Direction string `instill:"direction"`
}

type listPullRequestsOutput struct {
	PullRequests []PullRequest `instill:"pull-requests"`
}

type listReviewCommentsInput struct {
	RepoInfo
	PRNumber  int    `instill:"pr-number"`
	Sort      string `instill:"sort"`
	Direction string `instill:"direction"`
	Since     string `instill:"since"`
	PageOptions
}

type listReviewCommentsOutput struct {
	ReviewComments []ReviewComment `instill:"comments"`
}
