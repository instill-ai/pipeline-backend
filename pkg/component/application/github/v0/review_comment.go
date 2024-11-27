package github

import (
	"github.com/google/go-github/v62/github"
)

type ReviewComment struct {
	github.PullRequestComment
}

func extractReviewCommentInformation(originalComment *github.PullRequestComment) ReviewComment {
	return ReviewComment{
		PullRequestComment: *originalComment,
	}
}
