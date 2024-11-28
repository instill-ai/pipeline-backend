package github

import (
	"testing"

	"github.com/google/go-github/v62/github"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func TestComponent_ListReviewCommentsTask(t *testing.T) {
	testCases := []TaskCase[listReviewCommentsInput, listReviewCommentsOutput]{
		{
			_type: "ok",
			name:  "list review comments",
			input: listReviewCommentsInput{
				RepoInfo: RepoInfo{
					Owner:      "non-paginated",
					Repository: "test-repo",
				},
				PRNumber:  1,
				Sort:      "created",
				Direction: "asc",
				Since:     "2021-01-01",
			},
			wantOutput: listReviewCommentsOutput{
				ReviewComments: []ReviewComment{
					{
						PullRequestComment: github.PullRequestComment{
							Body: github.String("This is a fake comment"),
							ID:   github.Int64(1),
						},
					},
				},
				Response: &Response{},
			},
		},
		{
			_type: "ok",
			name:  "non-paginated review comments",
			input: listReviewCommentsInput{
				RepoInfo: RepoInfo{
					Owner:      "non-paginated",
					Repository: "test-repo",
				},
				PRNumber:  1,
				Sort:      "created",
				Direction: "asc",
				Since:     "2021-01-01",
			},
			wantOutput: listReviewCommentsOutput{
				ReviewComments: []ReviewComment{
					{
						PullRequestComment: github.PullRequestComment{
							Body: github.String("This is a fake comment"),
							ID:   github.Int64(1),
						},
					},
				},
				Response: &Response{},
			},
		},
		{
			_type: "ok",
			name:  "paginated review comments",
			input: listReviewCommentsInput{
				RepoInfo: RepoInfo{
					Owner:      "paginated",
					Repository: "test-repo",
				},
				PRNumber:  1,
				Sort:      "created",
				Direction: "asc",
				Since:     "2021-01-01",
				PageOptions: PageOptions{
					Page:    2,
					PerPage: 2,
				},
			},
			wantOutput: listReviewCommentsOutput{
				ReviewComments: []ReviewComment{
					{
						PullRequestComment: github.PullRequestComment{
							Body: github.String("This is a fake comment #3"),
							ID:   github.Int64(3),
						},
					},
					{
						PullRequestComment: github.PullRequestComment{
							Body: github.String("This is a fake comment #4"),
							ID:   github.Int64(4),
						},
					},
				},
				Response: &Response{
					NextPage:      3,
					PrevPage:      1,
					FirstPage:     1,
					LastPage:      5,
					NextPageToken: "page_3",
					Cursor:        "cursor_2",
					Before:        "before_2",
					After:         "after_2",
				},
			},
		},
		{
			_type: "ok",
			name:  "paginated review comments last page",
			input: listReviewCommentsInput{
				RepoInfo: RepoInfo{
					Owner:      "paginated",
					Repository: "test-repo",
				},
				PRNumber:  1,
				Sort:      "created",
				Direction: "asc",
				Since:     "2021-01-01",
				PageOptions: PageOptions{
					Page:    5,
					PerPage: 2,
				},
			},
			wantOutput: listReviewCommentsOutput{
				ReviewComments: []ReviewComment{
					{
						PullRequestComment: github.PullRequestComment{
							Body: github.String("This is a fake comment #9"),
							ID:   github.Int64(9),
						},
					},
					{
						PullRequestComment: github.PullRequestComment{
							Body: github.String("This is a fake comment #10"),
							ID:   github.Int64(10),
						},
					},
				},
				Response: &Response{
					NextPage:  0,
					PrevPage:  4,
					FirstPage: 1,
					LastPage:  5,
					Cursor:    "cursor_5",
					Before:    "before_5",
					After:     "after_5",
				},
			},
		},
		{
			_type: "nok",
			name:  "403 API rate limit exceeded",
			input: listReviewCommentsInput{
				RepoInfo: RepoInfo{
					Owner:      "rate_limit",
					Repository: "test-repo",
				},
				PRNumber:  1,
				Sort:      "created",
				Direction: "asc",
				Since:     "2021-01-01",
			},
			wantErr: `403 API rate limit exceeded`,
		},
		{
			_type: "nok",
			name:  "404 Not Found",
			input: listReviewCommentsInput{
				RepoInfo: RepoInfo{
					Owner:      "not_found",
					Repository: "test-repo",
				},
				PRNumber:  1,
				Sort:      "created",
				Direction: "asc",
				Since:     "2021-01-01",
			},
			wantErr: `404 Not Found`,
		},
		{
			_type: "nok",
			name:  "invalid time format",
			input: listReviewCommentsInput{
				RepoInfo: RepoInfo{
					Owner:      "not_found",
					Repository: "test-repo",
				},
				PRNumber:  1,
				Sort:      "created",
				Direction: "asc",
				Since:     "2021-0100:00:00Z",
			},
			wantErr: `^parse since time:.*cannot parse.*$`,
		},
	}

	e := &execution{
		client: *MockGithubClient,
		ComponentExecution: base.ComponentExecution{
			Task:            taskListReviewComments,
			Component:       Init(base.Component{Logger: zap.NewNop()}),
			SystemVariables: nil,
			Setup: func() *structpb.Struct {
				setup, err := structpb.NewStruct(map[string]any{
					"token": token,
				})
				if err != nil {
					t.Fatalf("failed to create setup: %v", err)
				}
				return setup
			}(),
		},
	}

	e.execute = e.client.listReviewComments

	taskTesting(testCases, e, t)
}
