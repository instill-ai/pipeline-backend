package github

import (
	"context"
	"fmt"
	"time"

	"github.com/google/go-github/v62/github"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

// ListReviewComments retrieves all review comments for a given pull request.
// Specifying a pull request number of 0 will return all comments on all pull requests for the repository.
//
// * This only works for public repositories.
func (client *Client) listReviewComments(ctx context.Context, job *base.Job) error {
	var input listReviewCommentsInput
	if err := job.Input.ReadData(ctx, &input); err != nil {
		return fmt.Errorf("reading input data: %w", err)
	}

	owner, repository, err := parseTargetRepo(input)
	if err != nil {
		return err
	}
	// from format like `2006-01-02` parse it into UTC time
	// The time will be 2006-01-02 00:00:00 +0000 UTC exactly
	sinceTime, err := time.Parse(time.DateOnly, input.Since)
	if err != nil {
		return fmt.Errorf("parse since time: %w", err)
	}
	opts := &github.PullRequestListCommentsOptions{
		Sort:      input.Sort,
		Direction: input.Direction,
		Since:     sinceTime,
		ListOptions: github.ListOptions{
			Page:    input.Page,
			PerPage: min(input.PerPage, 100), // GitHub API only allows 100 per page
		},
	}
	number := input.PrNumber
	comments, _, err := client.PullRequests.ListComments(ctx, owner, repository, number, opts)
	if err != nil {
		return addErrMsgToClientError(err)
	}

	reviewComments := make([]ReviewComment, len(comments))
	for i, comment := range comments {
		reviewComments[i] = extractReviewCommentInformation(comment)
	}
	var output listReviewCommentsOutput
	output.ReviewComments = reviewComments
	if err := job.Output.WriteData(ctx, output); err != nil {
		return err
	}
	return nil
}
