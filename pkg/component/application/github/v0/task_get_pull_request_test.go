package github

import (
	"testing"

	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func TestComponent_GetPullRequestTask(t *testing.T) {
	testCases := []TaskCase[getPullRequestInput, getPullRequestOutput]{
		{
			_type: "ok",
			name:  "get pull request",
			input: getPullRequestInput{
				RepoInfo: RepoInfo{
					Owner:      "test_owner",
					Repository: "test_repo",
				},
				PRNumber: 1,
			},
			wantOutput: getPullRequestOutput{
				PullRequest: PullRequest{
					Base: "baseSHA",
					Body: "PR Body",
					Commits: []Commit{
						{
							Message: "This is a fake commit",
							SHA:     "commitSHA",
							Stats: &CommitStats{
								Additions: 1,
								Deletions: 1,
								Changes:   2,
							},
							Files: []CommitFile{
								{
									Filename: "filename",
									Patch:    "patch",
									CommitStats: CommitStats{
										Additions: 1,
										Deletions: 1,
										Changes:   2,
									},
								},
							},
						},
					},
					DiffURL:           "https://fake-github.com/test_owner/test_repo/pull/1.diff",
					Head:              "headSHA",
					ID:                1,
					Number:            1,
					CommentsNum:       0,
					CommitsNum:        1,
					ReviewCommentsNum: 2,
					State:             "open",
					Title:             "This is a fake PR",
				},
			},
		},
		{
			_type: "ok",
			name:  "get latest pull request",
			input: getPullRequestInput{
				RepoInfo: RepoInfo{
					Owner:      "test_owner",
					Repository: "test_repo",
				},
				PRNumber: 0,
			},
			wantOutput: getPullRequestOutput{
				PullRequest: PullRequest{
					Base: "baseSHA",
					Body: "PR Body",
					Commits: []Commit{
						{
							Message: "This is a fake commit",
							SHA:     "commitSHA",
							Stats: &CommitStats{
								Additions: 1,
								Deletions: 1,
								Changes:   2,
							},
							Files: []CommitFile{
								{
									Filename: "filename",
									Patch:    "patch",
									CommitStats: CommitStats{
										Additions: 1,
										Deletions: 1,
										Changes:   2,
									},
								},
							},
						},
					},
					DiffURL:           "https://fake-github.com/test_owner/test_repo/pull/1.diff",
					Head:              "headSHA",
					ID:                1,
					Number:            1,
					CommentsNum:       0,
					CommitsNum:        1,
					ReviewCommentsNum: 2,
					State:             "open",
					Title:             "This is a fake PR",
				},
			},
		},
		{
			_type: "nok",
			name:  "403 API rate limit exceeded",
			input: getPullRequestInput{
				RepoInfo: RepoInfo{
					Owner:      "rate_limit",
					Repository: "test_repo",
				},
				PRNumber: 1,
			},
			wantErr: `403 API rate limit exceeded`,
		},
		{
			_type: "nok",
			name:  "404 Not Found",
			input: getPullRequestInput{
				RepoInfo: RepoInfo{
					Owner:      "not_found",
					Repository: "test_repo",
				},
				PRNumber: 1,
			},
			wantErr: `404 Not Found`,
		},
	}

	e := &execution{
		client: *MockGithubClient,
		ComponentExecution: base.ComponentExecution{
			Task:            taskGetPullRequest,
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

	e.execute = e.client.getPullRequest

	taskTesting(testCases, e, t)
}
