package jira

import (
	"context"
	"fmt"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func (c *client) listBoards(ctx context.Context, job *base.Job) error {
	var input listBoardsInput
	if err := job.Input.ReadData(ctx, &input); err != nil {
		return fmt.Errorf("reading input data: %w", err)
	}

	resp, err := listBoards(c, &input)
	if err != nil {
		return fmt.Errorf("listing boards: %w", err)
	}
	var output listBoardsOutput
	output.Boards = append(output.Boards, resp.Boards...)
	if output.Boards == nil {
		output.Boards = []Board{}
	}
	output.StartAt = resp.StartAt
	output.MaxResults = resp.MaxResults
	output.IsLast = resp.IsLast
	output.Total = resp.Total
	return job.Output.WriteData(ctx, output)
}
