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
			name:  "get review comments",
			input: listReviewCommentsInput{
				RepoInfo: RepoInfo{
					Owner:      "test_owner",
					Repository: "test_repo",
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
			},
		},
		{
			_type: "nok",
			name:  "403 API rate limit exceeded",
			input: listReviewCommentsInput{
				RepoInfo: RepoInfo{
					Owner:      "rate_limit",
					Repository: "test_repo",
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
					Repository: "test_repo",
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
					Repository: "test_repo",
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
