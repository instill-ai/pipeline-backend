package github

import (
	"context"

	"github.com/google/go-github/v62/github"
)

type IssuesService interface {
	ListByRepo(context.Context, string, string, *github.IssueListByRepoOptions) ([]*github.Issue, *github.Response, error)
	Get(context.Context, string, string, int) (*github.Issue, *github.Response, error)
	Create(context.Context, string, string, *github.IssueRequest) (*github.Issue, *github.Response, error)
	// Edit(context.Context, string, string, int, *github.IssueRequest) (*github.Issue, *github.Response, error)
}

type Issue struct {
	Number        int      `json:"number"`
	Title         string   `json:"title"`
	State         string   `json:"state"`
	Body          string   `json:"body"`
	Assignee      string   `json:"assignee"`
	Assignees     []string `json:"assignees"`
	Labels        []string `json:"labels"`
	IsPullRequest bool     `json:"is-pull-request"`
}

func (client *Client) extractIssue(originalIssue *github.Issue) Issue {
	return Issue{
		Number:        originalIssue.GetNumber(),
		Title:         originalIssue.GetTitle(),
		State:         originalIssue.GetState(),
		Body:          originalIssue.GetBody(),
		Assignee:      originalIssue.GetAssignee().GetName(),
		Assignees:     extractAssignees(originalIssue.Assignees),
		Labels:        extractLabels(originalIssue.Labels),
		IsPullRequest: originalIssue.IsPullRequest(),
	}
}

func extractAssignees(assignees []*github.User) []string {
	assigneeList := make([]string, len(assignees))
	for idx, assignee := range assignees {
		assigneeList[idx] = assignee.GetName()
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

func (client *Client) getIssueFunc(ctx context.Context, owner, repository string, issueNumber int) (*github.Issue, error) {
	issue, _, err := client.Issues.Get(ctx, owner, repository, issueNumber)
	if err != nil {
		return nil, addErrMsgToClientError(err)
	}
	return issue, nil
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
