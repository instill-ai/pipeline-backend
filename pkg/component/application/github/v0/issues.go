package github

import (
	"context"

	"github.com/google/go-github/v62/github"
)

// IssuesService is the interface for the GitHub issues service.
type IssuesService interface {
	ListByRepo(context.Context, string, string, *github.IssueListByRepoOptions) ([]*github.Issue, *github.Response, error)
	Get(context.Context, string, string, int) (*github.Issue, *github.Response, error)
	Create(context.Context, string, string, *github.IssueRequest) (*github.Issue, *github.Response, error)
}

// Issue is the GitHub issue object.
type Issue struct {
	Number        int      `instill:"number"`
	Title         string   `instill:"title"`
	State         string   `instill:"state"`
	Body          string   `instill:"body"`
	Assignee      string   `instill:"assignee"`
	Assignees     []string `instill:"assignees"`
	Labels        []string `instill:"labels"`
	IsPullRequest bool     `instill:"is-pull-request"`
}

func (client *Client) extractIssue(originalIssue *github.Issue) Issue {
	return Issue{
		Number:        originalIssue.GetNumber(),
		Title:         originalIssue.GetTitle(),
		State:         originalIssue.GetState(),
		Body:          originalIssue.GetBody(),
		Assignee:      originalIssue.GetAssignee().GetLogin(),
		Assignees:     extractAssignees(originalIssue.Assignees),
		Labels:        extractLabels(originalIssue.Labels),
		IsPullRequest: originalIssue.IsPullRequest(),
	}
}

func extractAssignees(assignees []*github.User) []string {
	assigneeList := make([]string, len(assignees))
	for idx, assignee := range assignees {
		assigneeList[idx] = assignee.GetLogin()
	}
	return assigneeList
}

func extractLabels(labels []*github.Label) []string {
	labelList := make([]string, len(labels))
	for idx, label := range labels {
		labelList[idx] = label.GetName()
	}
	return labelList
}

func filterOutPullRequests(issues []Issue) []Issue {
	filteredIssues := make([]Issue, 0)
	for _, issue := range issues {
		if !issue.IsPullRequest {
			filteredIssues = append(filteredIssues, issue)
		}
	}
	return filteredIssues
}
