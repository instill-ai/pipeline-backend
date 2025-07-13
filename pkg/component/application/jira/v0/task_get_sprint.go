package jira

import (
	"context"
	"fmt"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"

	errorsx "github.com/instill-ai/x/errors"
)

func (c *client) getSprint(ctx context.Context, job *base.Job) error {
	var input getSprintInput
	if err := job.Input.ReadData(ctx, &input); err != nil {
		return fmt.Errorf("reading input data: %w", err)
	}

	apiEndpoint := fmt.Sprintf("rest/agile/1.0/sprint/%v", input.SprintID)
	req := c.R().SetResult(&Sprint{})
	resp, err := req.Get(apiEndpoint)
	if err != nil {
		return fmt.Errorf("getting sprint: %w", err)
	}

	issue, ok := resp.Result().(*Sprint)
	if !ok {
		return errorsx.AddMessage(
			fmt.Errorf("failed to convert response to `Get Sprint` Output"),
			fmt.Sprintf("failed to convert %v to `Get Sprint` Output", resp.Result()),
		)
	}
	output := *extractSprintOutput(issue)
	return job.Output.WriteData(ctx, output)
}
