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
					Owner:      "non-paginated",
					Repository: "test-repo",
				},
				State:     "open",
				Direction: "asc",
				Sort:      "created",
			},
			wantOutput: listPullRequestsOutput{
				PullRequests: []PullRequest{
					{
						Base:              "baseSHA",
						Body:              "PR Body",
						DiffURL:           "https://fake-github.com/non-paginated/test-repo/pull/1.diff",
						Head:              "headSHA",
						ID:                1,
						Number:            1,
						CommentsNum:       0,
						CommitsNum:        1,
						ReviewCommentsNum: 2,
						State:             "open",
						Title:             "This is a fake PR",
						Commits: []Commit{
							{
								Message: "This is a fake commit",
								SHA:     "commitSHA",
							},
						},
					},
				},
				Response: &Response{},
			},
		},
		{
			_type: "ok",
			name:  "paginated pull requests",
			input: listPullRequestsInput{
				RepoInfo: RepoInfo{
					Owner:      "paginated",
					Repository: "test-repo",
				},
				State:     "open",
				Direction: "asc",
				Sort:      "created",
				PageOptions: PageOptions{
					Page:    2,
					PerPage: 2,
				},
			},
			wantOutput: listPullRequestsOutput{
				PullRequests: []PullRequest{
					{
						Base:              "baseSHA3",
						Body:              "PR Body #3",
						DiffURL:           "https://fake-github.com/paginated/test-repo/pull/3.diff",
						Head:              "headSHA3",
						ID:                3,
						Number:            3,
						CommentsNum:       3,
						CommitsNum:        3,
						ReviewCommentsNum: 6,
						State:             "open",
						Title:             "This is a fake PR #3",
						Commits: []Commit{
							{
								Message: "This is a fake commit",
								SHA:     "commitSHA",
							},
						},
					},
					{
						Base:              "baseSHA4",
						Body:              "PR Body #4",
						DiffURL:           "https://fake-github.com/paginated/test-repo/pull/4.diff",
						Head:              "headSHA4",
						ID:                4,
						Number:            4,
						CommentsNum:       4,
						CommitsNum:        4,
						ReviewCommentsNum: 8,
						State:             "open",
						Title:             "This is a fake PR #4",
						Commits: []Commit{
							{
								Message: "This is a fake commit",
								SHA:     "commitSHA",
							},
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
			name:  "paginated pull requests last page",
			input: listPullRequestsInput{
				RepoInfo: RepoInfo{
					Owner:      "paginated",
					Repository: "test-repo",
				},
				State:     "open",
				Direction: "asc",
				Sort:      "created",
				PageOptions: PageOptions{
					Page:    5,
					PerPage: 2,
				},
			},
			wantOutput: listPullRequestsOutput{
				PullRequests: []PullRequest{
					{
						Base:              "baseSHA9",
						Body:              "PR Body #9",
						DiffURL:           "https://fake-github.com/paginated/test-repo/pull/9.diff",
						Head:              "headSHA9",
						ID:                9,
						Number:            9,
						CommentsNum:       9,
						CommitsNum:        9,
						ReviewCommentsNum: 18,
						State:             "open",
						Title:             "This is a fake PR #9",
						Commits: []Commit{
							{
								Message: "This is a fake commit",
								SHA:     "commitSHA",
							},
						},
					},
					{
						Base:              "baseSHA10",
						Body:              "PR Body #10",
						DiffURL:           "https://fake-github.com/paginated/test-repo/pull/10.diff",
						Head:              "headSHA10",
						ID:                10,
						Number:            10,
						CommentsNum:       10,
						CommitsNum:        10,
						ReviewCommentsNum: 20,
						State:             "open",
						Title:             "This is a fake PR #10",
						Commits: []Commit{
							{
								Message: "This is a fake commit",
								SHA:     "commitSHA",
							},
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
			input: listPullRequestsInput{
				RepoInfo: RepoInfo{
					Owner:      "rate_limit",
					Repository: "test-repo",
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
					Repository: "test-repo",
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
					Repository: "test-repo",
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
