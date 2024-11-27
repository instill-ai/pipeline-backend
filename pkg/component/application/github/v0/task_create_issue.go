package github

import (
	"context"
	"fmt"

	"github.com/google/go-github/v62/github"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func (client *Client) createIssue(ctx context.Context, job *base.Job) error {
	var input createIssueInput
	if err := job.Input.ReadData(ctx, &input); err != nil {
		return fmt.Errorf("reading input data: %w", err)
	}
	owner, repository, err := parseTargetRepo(input)
	if err != nil {
		return err
	}
	issueRequest := &github.IssueRequest{
		Title:     &input.Title,
		Body:      &input.Body,
		Assignees: &input.Assignees,
		Labels:    &input.Labels,
	}
	issue, _, err := client.Issues.Create(ctx, owner, repository, issueRequest)
	if err != nil {
		return addErrMsgToClientError(err)
	}

	var output createIssueOutput
	output.Issue = client.extractIssue(issue)
	if err := job.Output.WriteData(ctx, output); err != nil {
		return err
	}
	return nil
}
