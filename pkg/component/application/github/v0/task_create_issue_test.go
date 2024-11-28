package github

import (
	"testing"

	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func TestComponent_CreateIssueTask(t *testing.T) {
	testcases := []TaskCase[createIssueInput, createIssueOutput]{
		{
			_type: "ok",
			name:  "get all issues",
			input: createIssueInput{
				RepoInfo: RepoInfo{
					Owner:      "test_owner",
					Repository: "test-repo",
				},
				Title: "This is a fake Issue",
				Body:  "Issue Body",
			},
			wantOutput: createIssueOutput{
				Issue: Issue{
					Number:        1,
					Title:         "This is a fake Issue",
					Body:          "Issue Body",
					State:         "open",
					IsPullRequest: false,
					Assignees:     []string{},
					Labels:        []string{},
					Assignee:      "",
				},
			},
		},
		{
			_type: "nok",
			name:  "403 API rate limit exceeded",
			input: createIssueInput{
				RepoInfo: RepoInfo{
					Owner:      "rate_limit",
					Repository: "test-repo",
				},
				Title: "This is a fake Issue",
				Body:  "Issue Body",
			},
			wantErr: `403 API rate limit exceeded`,
		},
		{
			_type: "nok",
			name:  "404 Not Found",
			input: createIssueInput{
				RepoInfo: RepoInfo{
					Owner:      "not_found",
					Repository: "test-repo",
				},
				Title: "This is a fake Issue",
				Body:  "Issue Body",
			},
			wantErr: `404 Not Found`,
		},
	}

	e := &execution{
		client: *MockGithubClient,
		ComponentExecution: base.ComponentExecution{
			Task:            taskCreateIssue,
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

	e.execute = e.client.createIssue

	taskTesting(testcases, e, t)
}
