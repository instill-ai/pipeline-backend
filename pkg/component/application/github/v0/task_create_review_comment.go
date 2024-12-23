package github

import (
	"context"
	"fmt"

	"github.com/google/go-github/v62/github"
	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

// CreateReviewComment creates a review comment for a given pull request.
//
// * This only works for public repositories.
func (client *Client) createReviewComment(ctx context.Context, job *base.Job) error {
	var input createReviewCommentInput
	if err := job.Input.ReadData(ctx, &input); err != nil {
		return fmt.Errorf("reading input data: %w", err)
	}

	owner, repository, err := parseTargetRepo(input)
	if err != nil {
		return err
	}
	number := input.PRNumber
	commentReqs := &input.Comment
	commentReqs.Position = commentReqs.Line // Position is deprecated, use Line instead

	if *commentReqs.Line == *commentReqs.StartLine {
		commentReqs.StartLine = nil // If it's a one line comment, don't send start-line
	}
	req := &github.PullRequestComment{
		Body:              commentReqs.Body,
		Path:              commentReqs.Path,
		Position:          commentReqs.Position,
		Line:              commentReqs.Line,
		StartLine:         commentReqs.StartLine,
		Side:              commentReqs.Side,
		StartSide:         commentReqs.StartSide,
		CommitID:          commentReqs.CommitID,
		OriginalCommitID:  commentReqs.OriginalCommitID,
		SubjectType:       commentReqs.SubjectType,
		AuthorAssociation: commentReqs.AuthorAssociation,
		URL:               commentReqs.URL,
		HTMLURL:           commentReqs.HTMLURL,
		PullRequestURL:    commentReqs.PullRequestURL,
	}
	comment, _, err := client.PullRequests.CreateComment(ctx, owner, repository, number, req)
	if err != nil {
		return addErrMsgToClientError(err)
	}

	var output createReviewCommentOutput
	output.ReviewComment = extractReviewCommentInformation(comment)
	if err := job.Output.WriteData(ctx, output); err != nil {
		return err
	}
	return nil
}
