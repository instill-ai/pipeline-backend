package github

import (
	"testing"

	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func TestComponent_ListIssuesTask(t *testing.T) {
	testcases := []TaskCase[listIssuesInput, listIssuesOutput]{
		{
			_type: "ok",
			name:  "get all issues",
			input: listIssuesInput{
				RepoInfo: RepoInfo{
					Owner:      "test_owner",
					Repository: "test_repo",
				},
				State:         "open",
				Direction:     "asc",
				Sort:          "created",
				Since:         "2021-01-01",
				NoPullRequest: true,
			},
			wantOutput: listIssuesOutput{
				Issues: []Issue{
					{
						Number:    1,
						Title:     "This is a fake Issue",
						Body:      "Issue Body",
						State:     "open",
						Assignee:  "assignee",
						Assignees: []string{"assignee1", "assignee2"},
						Labels:    []string{"label1", "label2"},
					},
				},
			},
		},
		{
			_type: "nok",
			name:  "403 API rate limit exceeded",
			input: listIssuesInput{
				RepoInfo: RepoInfo{
					Owner:      "rate_limit",
					Repository: "test_repo",
				},
				State:         "open",
				Direction:     "asc",
				Sort:          "created",
				Since:         "2021-01-01",
				NoPullRequest: true,
			},
			wantErr: `403 API rate limit exceeded`,
		},
		{
			_type: "nok",
			name:  "404 Not Found",
			input: listIssuesInput{
				RepoInfo: RepoInfo{
					Owner:      "not_found",
					Repository: "test_repo",
				},
				State:         "open",
				Direction:     "asc",
				Sort:          "created",
				Since:         "2021-01-01",
				NoPullRequest: true,
			},
			wantErr: `404 Not Found`,
		},
		{
			_type: "nok",
			name:  "invalid time format",
			input: listIssuesInput{
				RepoInfo: RepoInfo{
					Owner:      "not_found",
					Repository: "test_repo",
				},
				State:         "open",
				Direction:     "asc",
				Sort:          "created",
				Since:         "2021-0Z",
				NoPullRequest: true,
			},
			wantErr: `^parse since time:.*cannot parse.*$`,
		},
	}

	e := &execution{
		client: *MockGithubClient,
		ComponentExecution: base.ComponentExecution{
			Task:            taskListIssues,
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

	e.execute = e.client.listIssues

	taskTesting(testcases, e, t)
}
