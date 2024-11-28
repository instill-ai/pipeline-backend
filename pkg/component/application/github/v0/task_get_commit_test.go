package github

import (
	"testing"

	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func TestComponent_GetCommitTask(t *testing.T) {
	testCases := []TaskCase[getCommitInput, getCommitOutput]{
		{
			_type: "ok",
			name:  "get commit",
			input: getCommitInput{
				RepoInfo: RepoInfo{
					Owner:      "test_owner",
					Repository: "test-repo",
				},
				SHA: "commitSHA",
			},
			wantOutput: getCommitOutput{
				Commit: Commit{
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
		},
		{
			_type: "nok",
			name:  "403 API rate limit exceeded",
			input: getCommitInput{
				RepoInfo: RepoInfo{
					Owner:      "rate_limit",
					Repository: "test-repo",
				},
				SHA: "commitSHA",
			},
			wantErr: `403 API rate limit exceeded`,
		},
	}

	e := &execution{
		client: *MockGithubClient,
		ComponentExecution: base.ComponentExecution{
			Task:            taskGetCommit,
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

	e.execute = e.client.getCommit

	taskTesting(testCases, e, t)
}
