package github

import (
	"context"

	"fmt"

	"github.com/google/go-github/v62/github"
	"github.com/instill-ai/x/errmsg"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func (client *Client) getPullRequest(ctx context.Context, job *base.Job) error {

	var input getPullRequestInput
	if err := job.Input.ReadData(ctx, &input); err != nil {
		return fmt.Errorf("reading input data: %w", err)
	}

	owner, repository, err := parseTargetRepo(input)
	if err != nil {
		return err
	}
	number := input.PRNumber
	var pullRequest *github.PullRequest
	if number > 0 {
		pr, _, err := client.PullRequests.Get(ctx, owner, repository, int(number))
		if err != nil {
			// err includes the rate limit, 404 not found, etc.
			// if the connection is not authorized, it's easy to get rate limit error in large scale usage.
			return addErrMsgToClientError(err)
		}
		pullRequest = pr
	} else {
		// Get the latest PR
		opts := &github.PullRequestListOptions{
			State:     "all",
			Sort:      "created",
			Direction: "desc",
			ListOptions: github.ListOptions{
				Page:    1,
				PerPage: 1,
			},
		}
		prs, _, err := client.PullRequests.List(ctx, owner, repository, opts)
		if err != nil {
			// err includes the rate limit.
			// if the connection is not authorized, it's easy to get rate limit error in large scale usage.
			return addErrMsgToClientError(err)
		}
		if len(prs) == 0 {
			return errmsg.AddMessage(
				fmt.Errorf("no pull request found"),
				"No pull request found",
			)
		}
		pullRequest = prs[0]
		// Some fields are not included in the list API, so we need to get the PR again.
		pr, _, err := client.PullRequests.Get(ctx, owner, repository, *pullRequest.Number)
		if err != nil {
			// err includes the rate limit, 404 not found, etc.
			// if the connection is not authorized, it's easy to get rate limit error in large scale usage.
			return addErrMsgToClientError(err)
		}
		pullRequest = pr
	}

	var output getPullRequestOutput
	output.PullRequest, err = client.extractPullRequestInformation(ctx, owner, repository, pullRequest, true)
	if err != nil {
		return err
	}
	if err := job.Output.WriteData(ctx, output); err != nil {
		return err
	}

	return nil
}
