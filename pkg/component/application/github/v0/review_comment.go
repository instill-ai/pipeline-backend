package github

import (
	"context"
	"time"

	"github.com/google/go-github/v62/github"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

type ReviewComment struct {
	github.PullRequestComment
}

func extractReviewCommentInformation(originalComment *github.PullRequestComment) ReviewComment {
	return ReviewComment{
		PullRequestComment: *originalComment,
	}
}

type ListReviewCommentsInput struct {
	RepoInfo
	PrNumber  int    `json:"pr-number"`
	Sort      string `json:"sort"`
	Direction string `json:"direction"`
	Since     string `json:"since"`
	PageOptions
}

type ListReviewCommentsResp struct {
	ReviewComments []ReviewComment `json:"comments"`
}

// ListReviewComments retrieves all review comments for a given pull request.
// Specifying a pull request number of 0 will return all comments on all pull requests for the repository.
//
// * This only works for public repositories.
func (githubClient *Client) listReviewCommentsTask(ctx context.Context, props *structpb.Struct) (*structpb.Struct, error) {
	var inputStruct ListReviewCommentsInput
	err := base.ConvertFromStructpb(props, &inputStruct)
	if err != nil {
		return nil, err
	}

	owner, repository, err := parseTargetRepo(inputStruct)
	if err != nil {
		return nil, err
	}
	// from format like `2006-01-02` parse it into UTC time
	// The time will be 2006-01-02 00:00:00 +0000 UTC exactly
	sinceTime, err := time.Parse(time.DateOnly, inputStruct.Since)
	if err != nil {
		return nil, err
	}
	opts := &github.PullRequestListCommentsOptions{
		Sort:      inputStruct.Sort,
		Direction: inputStruct.Direction,
		Since:     sinceTime,
		ListOptions: github.ListOptions{
			Page:    inputStruct.Page,
			PerPage: min(inputStruct.PerPage, 100), // GitHub API only allows 100 per page
		},
	}
	number := inputStruct.PrNumber
	comments, _, err := githubClient.PullRequests.ListComments(ctx, owner, repository, number, opts)
	if err != nil {
		return nil, addErrMsgToClientError(err)
	}

	reviewComments := make([]ReviewComment, len(comments))
	for i, comment := range comments {
		reviewComments[i] = extractReviewCommentInformation(comment)
	}
	var reviewCommentsResp ListReviewCommentsResp
	reviewCommentsResp.ReviewComments = reviewComments
	out, err := base.ConvertToStructpb(reviewCommentsResp)
	if err != nil {
		return nil, err
	}
	return out, nil
}

type CreateReviewCommentInput struct {
	RepoInfo
	PrNumber int                       `json:"pr-number"`
	Comment  github.PullRequestComment `json:"comment"`
}

type CreateReviewCommentResp struct {
	ReviewComment
}

// CreateReviewComment creates a review comment for a given pull request.
//
// * This only works for public repositories.
func (githubClient *Client) createReviewCommentTask(ctx context.Context, props *structpb.Struct) (*structpb.Struct, error) {
	var commentInput CreateReviewCommentInput
	err := base.ConvertFromStructpb(props, &commentInput)
	if err != nil {
		return nil, err
	}

	owner, repository, err := parseTargetRepo(commentInput)
	if err != nil {
		return nil, err
	}
	number := commentInput.PrNumber
	commentReqs := &commentInput.Comment
	commentReqs.Position = commentReqs.Line // Position is deprecated, use Line instead

	if *commentReqs.Line == *commentReqs.StartLine {
		commentReqs.StartLine = nil // If it's a one line comment, don't send start-line
	}
	comment, _, err := githubClient.PullRequests.CreateComment(ctx, owner, repository, number, commentReqs)
	if err != nil {
		return nil, addErrMsgToClientError(err)
	}

	reviewComment := extractReviewCommentInformation(comment)
	var reviewCommentResp CreateReviewCommentResp
	reviewCommentResp.ReviewComment = reviewComment
	out, err := base.ConvertToStructpb(reviewCommentResp)
	if err != nil {
		return nil, err
	}
	return out, nil
}
