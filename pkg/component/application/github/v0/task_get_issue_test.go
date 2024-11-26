package github

import (
	"testing"

	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func TestComponent_GetIssueTask(t *testing.T) {
	testCases := []TaskCase[getIssueInput, getIssueOutput]{
		{
			_type: "ok",
			name:  "get all issues",
			input: getIssueInput{
				RepoInfo: RepoInfo{
					Owner:      "test_owner",
					Repository: "test_repo",
				},
				IssueNumber: 1,
			},
			wantOutput: getIssueOutput{
				Issue: Issue{
					Number:        1,
					Title:         "This is a fake Issue",
					Body:          "Issue Body",
					State:         "open",
					Assignee:      "assignee",
					Assignees:     []string{"assignee1", "assignee2"},
					Labels:        []string{"label1", "label2"},
					IsPullRequest: false,
				},
			},
		},
		{
			_type: "nok",
			name:  "403 API rate limit exceeded",
			input: getIssueInput{
				RepoInfo: RepoInfo{
					Owner:      "rate_limit",
					Repository: "test_repo",
				},
				IssueNumber: 1,
			},
			wantErr: `403 API rate limit exceeded`,
		},
		{
			_type: "nok",
			name:  "404 Not Found",
			input: getIssueInput{
				RepoInfo: RepoInfo{
					Owner:      "not_found",
					Repository: "test_repo",
				},
				IssueNumber: 1,
			},
			wantErr: `404 Not Found`,
		},
	}

	e := &execution{
		client: *MockGithubClient,
		ComponentExecution: base.ComponentExecution{
			Task:            taskGetIssue,
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

	e.execute = e.client.getIssue

	taskTesting(testCases, e, t)
}
