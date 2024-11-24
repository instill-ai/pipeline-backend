package github

import (
	"testing"

	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func TestComponent_ListPullRequestsTask(t *testing.T) {
	testCases := []TaskCase[listPullRequestsInput, listPullRequestsOutput]{
		{
			_type: "ok",
			name:  "list all pull requests",
			input: listPullRequestsInput{
				RepoInfo: RepoInfo{
					Owner:      "test_owner",
					Repository: "test_repo",
				},
				State:     "open",
				Direction: "asc",
				Sort:      "created",
			},
			wantOutput: listPullRequestsOutput{
				PullRequests: []PullRequest{
					{
						Base: "baseSHA",
						Body: "PR Body",
						Commits: []Commit{
							{
								Message: "This is a fake commit",
								SHA:     "commitSHA",
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
		},
		{
			_type: "nok",
			name:  "403 API rate limit exceeded",
			input: listPullRequestsInput{
				RepoInfo: RepoInfo{
					Owner:      "rate_limit",
					Repository: "test_repo",
				},
				State:     "open",
				Direction: "asc",
				Sort:      "created",
			},
			wantErr: `403 API rate limit exceeded`,
		},
		{
			_type: "nok",
			name:  "404 Not Found",
			input: listPullRequestsInput{
				RepoInfo: RepoInfo{
					Owner:      "not_found",
					Repository: "test_repo",
				},
				State:     "open",
				Direction: "asc",
				Sort:      "created",
			},
			wantErr: `404 Not Found`,
		},
		{
			_type: "nok",
			name:  "no PRs found",
			input: listPullRequestsInput{
				RepoInfo: RepoInfo{
					Owner:      "no_pr",
					Repository: "test_repo",
				},
				State:     "open",
				Direction: "asc",
				Sort:      "created",
			},
			wantOutput: listPullRequestsOutput{
				PullRequests: []PullRequest{},
			},
		},
	}

	e := &execution{
		client: *MockGithubClient,
		ComponentExecution: base.ComponentExecution{
			Task:            taskListPullRequests,
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

	e.execute = e.client.listPullRequests

	taskTesting(testCases, e, t)
}
