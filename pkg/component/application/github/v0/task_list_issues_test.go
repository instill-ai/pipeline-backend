package github

import (
	"testing"

	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func TestComponent_ListIssuesTask(t *testing.T) {
	testCases := []TaskCase[listIssuesInput, listIssuesOutput]{
		{
			_type: "ok",
			name:  "list all issues",
			input: listIssuesInput{
				RepoInfo: RepoInfo{
					Owner:      "non-paginated",
					Repository: "test-repo",
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
				Response: &Response{},
			},
		},
		{
			_type: "ok",
			name:  "paginated issues",
			input: listIssuesInput{
				PageOptions: PageOptions{
					Page:    2,
					PerPage: 2,
				},
				RepoInfo: RepoInfo{
					Owner:      "paginated",
					Repository: "test-repo",
				},
				State:     "open",
				Direction: "asc",
				Sort:      "created",
				Since:     "2021-01-01",
			},
			wantOutput: listIssuesOutput{
				Issues: []Issue{
					{
						Number:    3,
						Title:     "This is a fake Issue #3",
						State:     "open",
						Body:      "Issue Body #3",
						Assignee:  "assignee3",
						Assignees: []string{"assignee3_1", "assignee3_2"},
						Labels:    []string{"label3_1", "label3_2"},
					},
					{
						Number:    4,
						Title:     "This is a fake Issue #4",
						State:     "open",
						Body:      "Issue Body #4",
						Assignee:  "assignee4",
						Assignees: []string{"assignee4_1", "assignee4_2"},
						Labels:    []string{"label4_1", "label4_2"},
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
			name:  "paginated issues last page",
			input: listIssuesInput{
				PageOptions: PageOptions{
					Page:    5,
					PerPage: 2,
				},
				RepoInfo: RepoInfo{
					Owner:      "paginated",
					Repository: "test-repo",
				},
				State:     "open",
				Direction: "asc",
				Sort:      "created",
				Since:     "2021-01-01",
			},
			wantOutput: listIssuesOutput{
				Issues: []Issue{
					{
						Number:    9,
						Title:     "This is a fake Issue #9",
						State:     "open",
						Body:      "Issue Body #9",
						Assignee:  "assignee9",
						Assignees: []string{"assignee9_1", "assignee9_2"},
						Labels:    []string{"label9_1", "label9_2"},
					},
					{
						Number:    10,
						Title:     "This is a fake Issue #10",
						State:     "open",
						Body:      "Issue Body #10",
						Assignee:  "assignee10",
						Assignees: []string{"assignee10_1", "assignee10_2"},
						Labels:    []string{"label10_1", "label10_2"},
					},
				},
				Response: &Response{
					NextPage:      0,
					PrevPage:      4,
					FirstPage:     1,
					LastPage:      5,
					NextPageToken: "",
					Cursor:        "cursor_5",
					Before:        "before_5",
					After:         "after_5",
				},
			},
		},
		{
			_type: "nok",
			name:  "403 API rate limit exceeded",
			input: listIssuesInput{
				RepoInfo: RepoInfo{
					Owner:      "rate_limit",
					Repository: "test-repo",
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
					Repository: "test-repo",
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
					Repository: "test-repo",
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

	taskTesting(testCases, e, t)
}
