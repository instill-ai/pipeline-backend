package github

import (
	"context"
	"fmt"

	"github.com/google/go-github/v62/github"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func (client *Client) listPullRequests(ctx context.Context, job *base.Job) error {

	var input listPullRequestsInput
	if err := job.Input.ReadData(ctx, &input); err != nil {
		return fmt.Errorf("reading input data: %w", err)
	}

	owner, repository, err := parseTargetRepo(input)
	if err != nil {
		return err
	}

	opts := &github.PullRequestListOptions{
		State:     input.State,
		Sort:      input.Sort,
		Direction: input.Direction,
		ListOptions: github.ListOptions{
			Page:    input.Page,
			PerPage: min(input.PerPage, 100), // GitHub API only allows 100 per page
		},
	}
	prs, resp, err := client.PullRequests.List(ctx, owner, repository, opts)
	if err != nil {
		return addErrMsgToClientError(err)
	}
	PullRequests := make([]PullRequest, len(prs))
	for idx, pr := range prs {
		PullRequests[idx], err = client.extractPullRequestInformation(ctx, owner, repository, pr, false)
		if err != nil {
			return err
		}
	}

	output := listPullRequestsOutput{
		PullRequests: PullRequests,
		Response:     client.extractResponse(resp),
	}
	if err := job.Output.WriteData(ctx, output); err != nil {
		return err
	}

	return nil
}
