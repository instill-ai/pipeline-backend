package github

import (
	"github.com/google/go-github/v62/github"
)

// ReviewComment is originate from github.PullRequestComment, but uses kebab-case field names
// in the instill tags.
type ReviewComment struct {
	github.PullRequestComment
	ID                  *int64     `instill:"id"`
	NodeID              *string    `instill:"node-id"`
	InReplyTo           *int64     `instill:"in-reply-to-id"`
	Body                *string    `instill:"body"`
	Path                *string    `instill:"path"`
	DiffHunk            *string    `instill:"diff-hunk"`
	PullRequestReviewID *int64     `instill:"pull-request-review-id"`
	Position            *int       `instill:"position"`
	OriginalPosition    *int       `instill:"original-position"`
	StartLine           *int       `instill:"start-line"`
	Line                *int       `instill:"line"`
	OriginalLine        *int       `instill:"original-line"`
	OriginalStartLine   *int       `instill:"original-start-line"`
	Side                *string    `instill:"side"`
	StartSide           *string    `instill:"start-side"`
	CommitID            *string    `instill:"commit-id"`
	OriginalCommitID    *string    `instill:"original-commit-id"`
	User                *User      `instill:"user"`
	Reactions           *Reactions `instill:"reactions"`
	CreatedAt           *Timestamp `instill:"created-at"`
	UpdatedAt           *Timestamp `instill:"updated-at"`
	// AuthorAssociation is the comment author's relationship to the pull request's repository.
	// Possible values are "COLLABORATOR", "CONTRIBUTOR", "FIRST_TIMER", "FIRST_TIME_CONTRIBUTOR", "MEMBER", "OWNER", or "NONE".
	AuthorAssociation *string `instill:"author-association"`
	URL               *string `instill:"url"`
	HTMLURL           *string `instill:"html-url"`
	PullRequestURL    *string `instill:"pull-request-url"`
	// Can be one of: LINE, FILE from https://docs.github.com/rest/pulls/comments#create-a-review-comment-for-a-pull-request
	SubjectType *string `instill:"subject-type"`
}

type Reactions struct {
	TotalCount *int    `instill:"total-count"`
	PlusOne    *int    `instill:"+1"`
	MinusOne   *int    `instill:"-1"`
	Laugh      *int    `instill:"laugh"`
	Confused   *int    `instill:"confused"`
	Heart      *int    `instill:"heart"`
	Hooray     *int    `instill:"hooray"`
	Rocket     *int    `instill:"rocket"`
	Eyes       *int    `instill:"eyes"`
	URL        *string `instill:"url"`
}

func extractReviewCommentInformation(originalComment *github.PullRequestComment) ReviewComment {
	return ReviewComment{
		PullRequestComment: *originalComment,
	}
}
