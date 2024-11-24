package github

import (
	"testing"

	"github.com/google/go-github/v62/github"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func TestComponent_CreateReviewCommentTask(t *testing.T) {
	testCases := []TaskCase[createReviewCommentInput, createReviewCommentOutput]{
		{
			_type: "ok",
			name:  "create review comment",
			input: createReviewCommentInput{
				RepoInfo: RepoInfo{
					Owner:      "test_owner",
					Repository: "test_repo",
				},
				PrNumber: 1,
				Comment: github.PullRequestComment{
					Body:        github.String("This is a fake comment"),
					Line:        github.Int(2),
					StartLine:   github.Int(1),
					Side:        github.String("side"),
					StartSide:   github.String("side"),
					SubjectType: github.String("line"),
				},
			},
			wantOutput: createReviewCommentOutput{
				ReviewComment: ReviewComment{
					PullRequestComment: github.PullRequestComment{
						Body:        github.String("This is a fake comment"),
						ID:          github.Int64(1),
						Line:        github.Int(2),
						Position:    github.Int(2),
						StartLine:   github.Int(1),
						Side:        github.String("side"),
						StartSide:   github.String("side"),
						SubjectType: github.String("line"),
					},
				},
			},
		},
		{
			_type: "ok",
			name:  "create one line review comment",
			input: createReviewCommentInput{
				RepoInfo: RepoInfo{
					Owner:      "test_owner",
					Repository: "test_repo",
				},
				PrNumber: 1,
				Comment: github.PullRequestComment{
					Body:        github.String("This is a fake comment"),
					Line:        github.Int(1),
					StartLine:   github.Int(1),
					Side:        github.String("side"),
					StartSide:   github.String("side"),
					SubjectType: github.String("line"),
				},
			},
			wantOutput: createReviewCommentOutput{
				ReviewComment: ReviewComment{
					PullRequestComment: github.PullRequestComment{
						Body:        github.String("This is a fake comment"),
						ID:          github.Int64(1),
						Line:        github.Int(1),
						Position:    github.Int(1),
						Side:        github.String("side"),
						StartSide:   github.String("side"),
						SubjectType: github.String("line"),
					},
				},
			},
		},
		{
			_type: "nok",
			name:  "403 API rate limit exceeded",
			input: createReviewCommentInput{
				RepoInfo: RepoInfo{
					Owner:      "rate_limit",
					Repository: "test_repo",
				},
				PrNumber: 1,
				Comment: github.PullRequestComment{
					Body:        github.String("This is a fake comment"),
					Line:        github.Int(2),
					StartLine:   github.Int(1),
					Side:        github.String("RIGHT"),
					StartSide:   github.String("RIGHT"),
					SubjectType: github.String("line"),
				},
			},
			wantErr: `403 API rate limit exceeded`,
		},
		{
			_type: "nok",
			name:  "404 Not Found",
			input: createReviewCommentInput{
				RepoInfo: RepoInfo{
					Owner:      "not_found",
					Repository: "test_repo",
				},
				PrNumber: 1,
				Comment: github.PullRequestComment{
					Body:        github.String("This is a fake comment"),
					Line:        github.Int(2),
					StartLine:   github.Int(1),
					Side:        github.String("RIGHT"),
					StartSide:   github.String("RIGHT"),
					SubjectType: github.String("line"),
				},
			},
			wantErr: `404 Not Found`,
		},
		{
			_type: "nok",
			name:  "422 Unprocessable Entity",
			input: createReviewCommentInput{
				RepoInfo: RepoInfo{
					Owner:      "unprocessable_entity",
					Repository: "test_repo",
				},
				PrNumber: 1,
				Comment: github.PullRequestComment{
					Body:        github.String("This is a fake comment"),
					Line:        github.Int(2),
					StartLine:   github.Int(1),
					Side:        github.String("RIGHT"),
					StartSide:   github.String("RIGHT"),
					SubjectType: github.String("line"),
				},
			},
			wantErr: `422 Unprocessable Entity`,
		},
		{
			_type: "nok",
			name:  "422 Unprocessable Entity",
			input: createReviewCommentInput{
				RepoInfo: RepoInfo{
					Owner:      "test_owner",
					Repository: "test_repo",
				},
				PrNumber: 1,
				Comment: github.PullRequestComment{
					Body:        github.String("This is a fake comment"),
					Line:        github.Int(1),
					StartLine:   github.Int(2),
					Side:        github.String("RIGHT"),
					StartSide:   github.String("RIGHT"),
					SubjectType: github.String("line"),
				},
			},
			wantErr: `422 Unprocessable Entity`,
		},
	}

	e := &execution{
		client: *MockGithubClient,
		ComponentExecution: base.ComponentExecution{
			Task:            taskCreateReviewComment,
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

	e.execute = e.client.createReviewComment

	taskTesting(testCases, e, t)
}
