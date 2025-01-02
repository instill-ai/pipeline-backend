package github

import (
	"context"
	"fmt"
	"time"

	"github.com/google/go-github/v62/github"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func (client *Client) listIssues(ctx context.Context, job *base.Job) error {
	var input listIssuesInput
	if err := job.Input.ReadData(ctx, &input); err != nil {
		return fmt.Errorf("reading input data: %w", err)
	}

	owner, repository, err := parseTargetRepo(input)
	if err != nil {
		return err
	}

	opts := &github.IssueListByRepoOptions{
		State:     input.State,
		Sort:      input.Sort,
		Direction: input.Direction,
		ListOptions: github.ListOptions{
			Page:    input.Page,
			PerPage: min(input.PerPage, 100), // GitHub API only allows 100 per page
		},
	}
	// from format like `2006-01-02` parse it into UTC time
	// The time will be 2006-01-02 00:00:00 +0000 UTC exactly
	if input.Since != "" {
		sinceTime, err := time.Parse(time.DateOnly, input.Since)
		if err != nil {
			return fmt.Errorf("parse since time: %w", err)
		}
		opts.Since = sinceTime
	}
	if opts.Mentioned == "none" {
		opts.Mentioned = ""
	}

	issues, resp, err := client.Issues.ListByRepo(ctx, owner, repository, opts)
	if err != nil {
		return addErrMsgToClientError(err)
	}

	issueList := make([]Issue, len(issues))
	for idx, issue := range issues {
		issueList[idx] = client.extractIssue(issue)
	}

	// filter out pull requests if no-pull-request is true
	if input.NoPullRequest {
		issueList = filterOutPullRequests(issueList)
	}

	output := listIssuesOutput{
		Issues:   issueList,
		Response: client.extractResponse(resp),
	}
	if err := job.Output.WriteData(ctx, output); err != nil {
		return err
	}
	return nil
}
