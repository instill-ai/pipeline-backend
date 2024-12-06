package jira

import (
	"context"
	"fmt"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/x/errmsg"
)

func (c *client) createIssue(ctx context.Context, job *base.Job) error {
	var input createIssueInput
	if err := job.Input.ReadData(ctx, &input); err != nil {
		return fmt.Errorf("reading input data: %w", err)
	}

	apiEndpoint := "rest/api/2/issue"
	req := c.R().SetResult(&createIssueResp{}).SetBody(convertCreateIssueReq(&input))
	err := addQueryOptions(req, map[string]interface{}{"updateHistory": input.UpdateHistory})
	if err != nil {
		return fmt.Errorf("adding query options: %w", err)
	}
	resp, err := req.Post(apiEndpoint)
	if err != nil {
		return fmt.Errorf("creating issue: %w", err)
	}

	createdResult, ok := resp.Result().(*createIssueResp)

	if !ok {
		return errmsg.AddMessage(
			fmt.Errorf("failed to convert response to `Create Issue` Output"),
			fmt.Sprintf("failed to convert %v to `Create Issue` Output", resp.Result()),
		)
	}

	issue, err := getIssue(c.Client, createdResult.Key, input.UpdateHistory)
	if err != nil {
		return fmt.Errorf("getting created issue: %w", err)
	}

	output := createIssueOutput{Issue: *issue}
	return job.Output.WriteData(ctx, output)
}
