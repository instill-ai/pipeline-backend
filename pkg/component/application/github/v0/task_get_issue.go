package github

import (
	"context"
	"fmt"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func (client *Client) getIssue(ctx context.Context, job *base.Job) error {
	var input getIssueInput
	if err := job.Input.ReadData(ctx, &input); err != nil {
		return fmt.Errorf("reading input data: %w", err)
	}
	owner, repository, err := parseTargetRepo(input)
	if err != nil {
		return err
	}

	issueNumber := input.IssueNumber
	issue, err := client.getIssueFunc(ctx, owner, repository, issueNumber)
	if err != nil {
		return err
	}

	var output getIssueOutput
	output.Issue = client.extractIssue(issue)
	if err := job.Output.WriteData(ctx, output); err != nil {
		return err
	}
	return nil
}
