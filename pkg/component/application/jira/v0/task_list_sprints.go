package jira

import (
	"context"
	"fmt"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/x/errmsg"
)

func (c *client) listSprints(ctx context.Context, job *base.Job) error {
	var input listSprintsInput
	if err := job.Input.ReadData(ctx, &input); err != nil {
		return fmt.Errorf("reading input data: %w", err)
	}
	apiEndpoint := fmt.Sprintf("rest/agile/1.0/board/%d/sprint", input.BoardID)

	req := c.R().SetResult(listSprintsResp{})
	err := addQueryOptions(req, input)
	if err != nil {
		return fmt.Errorf("adding query options: %w", err)
	}

	resp, err := req.Get(apiEndpoint)

	if err != nil {
		return fmt.Errorf("getting sprints: %w", err)
	}

	sprints, ok := resp.Result().(*listSprintsResp)
	if !ok {
		return errmsg.AddMessage(
			fmt.Errorf("failed to convert %v to `List Sprint` Output", resp.Result()),
			fmt.Sprintf("failed to convert %v to `List Sprint` Output", resp.Result()),
		)
	}

	var output listSprintsOutput
	for _, sprint := range sprints.Sprints {
		output.Sprints = append(output.Sprints, extractSprintOutput(&sprint))
	}

	output.StartAt = sprints.StartAt
	output.MaxResults = sprints.MaxResults
	output.Total = sprints.Total
	return job.Output.WriteData(ctx, output)
}
