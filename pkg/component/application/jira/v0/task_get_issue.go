package jira

import (
	"context"
	"fmt"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func (c *client) getIssue(ctx context.Context, job *base.Job) error {
	var input getIssueInput
	if err := job.Input.ReadData(ctx, &input); err != nil {
		return fmt.Errorf("reading input data: %w", err)
	}

	issue, err := getIssue(c.Client, input.IssueKey, input.UpdateHistory)
	if err != nil {
		return err
	}

	output := getIssueOutput{Issue: *issue}
	return job.Output.WriteData(ctx, output)
}
