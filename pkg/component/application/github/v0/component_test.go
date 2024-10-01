package github

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/go-github/v62/github"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

var MockGithubClient = &Client{
	Repositories: &MockRepositoriesService{},
	PullRequests: &MockPullRequestService{},
	Issues:       &MockIssuesService{},
}

var fakeHost = "https://fake-github.com"

const (
	token = "testkey"
)

type TaskCase[inType any, outType any] struct {
	_type    string
	name     string
	input    inType
	wantResp outType
	wantErr  string
}

func TestComponent_ListPullRequestsTask(t *testing.T) {
	testcases := []TaskCase[ListPullRequestsInput, ListPullRequestsResp]{
		{
			_type: "ok",
			name:  "get all pull requests",
			input: ListPullRequestsInput{
				RepoInfo: RepoInfo{
					Owner:      "test_owner",
					Repository: "test_repo",
				},
				State:     "open",
				Direction: "asc",
				Sort:      "created",
			},
			wantResp: ListPullRequestsResp{
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
			input: ListPullRequestsInput{
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
			input: ListPullRequestsInput{
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
			input: ListPullRequestsInput{
				RepoInfo: RepoInfo{
					Owner:      "no_pr",
					Repository: "test_repo",
				},
				State:     "open",
				Direction: "asc",
				Sort:      "created",
			},
			wantResp: ListPullRequestsResp{
				PullRequests: []PullRequest{},
			},
		},
	}
	taskTesting(testcases, taskListPRs, t)
}

func TestComponent_GetPullRequestTask(t *testing.T) {
	testcases := []TaskCase[GetPullRequestInput, GetPullRequestResp]{
		{
			_type: "ok",
			name:  "get pull request",
			input: GetPullRequestInput{
				RepoInfo: RepoInfo{
					Owner:      "test_owner",
					Repository: "test_repo",
				},
				PrNumber: 1,
			},
			wantResp: GetPullRequestResp{
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
			input: GetPullRequestInput{
				RepoInfo: RepoInfo{
					Owner:      "test_owner",
					Repository: "test_repo",
				},
				PrNumber: 0,
			},
			wantResp: GetPullRequestResp{
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
			input: GetPullRequestInput{
				RepoInfo: RepoInfo{
					Owner:      "rate_limit",
					Repository: "test_repo",
				},
				PrNumber: 1,
			},
			wantErr: `403 API rate limit exceeded`,
		},
		{
			_type: "nok",
			name:  "404 Not Found",
			input: GetPullRequestInput{
				RepoInfo: RepoInfo{
					Owner:      "not_found",
					Repository: "test_repo",
				},
				PrNumber: 1,
			},
			wantErr: `404 Not Found`,
		},
	}
	taskTesting(testcases, taskGetPR, t)
}

func TestComponent_ListReviewCommentsTask(t *testing.T) {
	testcases := []TaskCase[ListReviewCommentsInput, ListReviewCommentsResp]{
		{
			_type: "ok",
			name:  "get review comments",
			input: ListReviewCommentsInput{
				RepoInfo: RepoInfo{
					Owner:      "test_owner",
					Repository: "test_repo",
				},
				PrNumber:  1,
				Sort:      "created",
				Direction: "asc",
				Since:     "2021-01-01T00:00:00Z",
			},
			wantResp: ListReviewCommentsResp{
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
			input: ListReviewCommentsInput{
				RepoInfo: RepoInfo{
					Owner:      "rate_limit",
					Repository: "test_repo",
				},
				PrNumber:  1,
				Sort:      "created",
				Direction: "asc",
				Since:     "2021-01-01T00:00:00Z",
			},
			wantErr: `403 API rate limit exceeded`,
		},
		{
			_type: "nok",
			name:  "404 Not Found",
			input: ListReviewCommentsInput{
				RepoInfo: RepoInfo{
					Owner:      "not_found",
					Repository: "test_repo",
				},
				PrNumber:  1,
				Sort:      "created",
				Direction: "asc",
				Since:     "2021-01-01T00:00:00Z",
			},
			wantErr: `404 Not Found`,
		},
		{
			_type: "nok",
			name:  "invalid time format",
			input: ListReviewCommentsInput{
				RepoInfo: RepoInfo{
					Owner:      "not_found",
					Repository: "test_repo",
				},
				PrNumber:  1,
				Sort:      "created",
				Direction: "asc",
				Since:     "2021-0100:00:00Z",
			},
			wantErr: `invalid time format`,
		},
	}
	taskTesting(testcases, taskGetReviewComments, t)
}

func TestComponent_CreateReviewCommentTask(t *testing.T) {
	testcases := []TaskCase[CreateReviewCommentInput, CreateReviewCommentResp]{
		{
			_type: "ok",
			name:  "create review comment",
			input: CreateReviewCommentInput{
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
			wantResp: CreateReviewCommentResp{
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
			name:  "create oneline review comment",
			input: CreateReviewCommentInput{
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
			wantResp: CreateReviewCommentResp{
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
			input: CreateReviewCommentInput{
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
			input: CreateReviewCommentInput{
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
			input: CreateReviewCommentInput{
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
			input: CreateReviewCommentInput{
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
	taskTesting(testcases, taskCreateReviewComment, t)
}

func TestComponent_GetCommitTask(t *testing.T) {
	testcases := []TaskCase[GetCommitInput, GetCommitResp]{
		{
			_type: "ok",
			name:  "get commit",
			input: GetCommitInput{
				RepoInfo: RepoInfo{
					Owner:      "test_owner",
					Repository: "test_repo",
				},
				SHA: "commitSHA",
			},
			wantResp: GetCommitResp{
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
			input: GetCommitInput{
				RepoInfo: RepoInfo{
					Owner:      "rate_limit",
					Repository: "test_repo",
				},
				SHA: "commitSHA",
			},
			wantErr: `403 API rate limit exceeded`,
		},
	}
	taskTesting(testcases, taskGetCommit, t)
}

func TestComponent_ListIssuesTask(t *testing.T) {
	testcases := []TaskCase[ListIssuesInput, ListIssuesResp]{
		{
			_type: "ok",
			name:  "get all issues",
			input: ListIssuesInput{
				RepoInfo: RepoInfo{
					Owner:      "test_owner",
					Repository: "test_repo",
				},
				State:         "open",
				Direction:     "asc",
				Sort:          "created",
				Since:         "2021-01-01T00:00:00Z",
				NoPullRequest: true,
			},
			wantResp: ListIssuesResp{
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
			input: ListIssuesInput{
				RepoInfo: RepoInfo{
					Owner:      "rate_limit",
					Repository: "test_repo",
				},
				State:         "open",
				Direction:     "asc",
				Sort:          "created",
				Since:         "2021-01-01T00:00:00Z",
				NoPullRequest: true,
			},
			wantErr: `403 API rate limit exceeded`,
		},
		{
			_type: "nok",
			name:  "404 Not Found",
			input: ListIssuesInput{
				RepoInfo: RepoInfo{
					Owner:      "not_found",
					Repository: "test_repo",
				},
				State:         "open",
				Direction:     "asc",
				Sort:          "created",
				Since:         "2021-01-01T00:00:00Z",
				NoPullRequest: true,
			},
			wantErr: `404 Not Found`,
		},
		{
			_type: "nok",
			name:  "invalid time format",
			input: ListIssuesInput{
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
			wantErr: `invalid time format`,
		},
	}
	taskTesting(testcases, taskListIssues, t)
}
func TestComponent_GetIssueTask(t *testing.T) {
	testcases := []TaskCase[GetIssueInput, GetIssueResp]{
		{
			_type: "ok",
			name:  "get all issues",
			input: GetIssueInput{
				RepoInfo: RepoInfo{
					Owner:      "test_owner",
					Repository: "test_repo",
				},
				IssueNumber: 1,
			},
			wantResp: GetIssueResp{
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
			input: GetIssueInput{
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
			input: GetIssueInput{
				RepoInfo: RepoInfo{
					Owner:      "not_found",
					Repository: "test_repo",
				},
				IssueNumber: 1,
			},
			wantErr: `404 Not Found`,
		},
	}
	taskTesting(testcases, taskGetIssue, t)
}
func TestComponent_CreateIssueTask(t *testing.T) {
	testcases := []TaskCase[CreateIssueInput, CreateIssueResp]{
		{
			_type: "ok",
			name:  "get all issues",
			input: CreateIssueInput{
				RepoInfo: RepoInfo{
					Owner:      "test_owner",
					Repository: "test_repo",
				},
				Title: "This is a fake Issue",
				Body:  "Issue Body",
			},
			wantResp: CreateIssueResp{
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
			input: CreateIssueInput{
				RepoInfo: RepoInfo{
					Owner:      "rate_limit",
					Repository: "test_repo",
				},
				Title: "This is a fake Issue",
				Body:  "Issue Body",
			},
			wantErr: `403 API rate limit exceeded`,
		},
		{
			_type: "nok",
			name:  "404 Not Found",
			input: CreateIssueInput{
				RepoInfo: RepoInfo{
					Owner:      "not_found",
					Repository: "test_repo",
				},
				Title: "This is a fake Issue",
				Body:  "Issue Body",
			},
			wantErr: `404 Not Found`,
		},
	}
	taskTesting(testcases, taskCreateIssue, t)
}

func TestComponent_CreateWebHook(t *testing.T) {
	testcases := []TaskCase[CreateWebHookInput, CreateWebHookResp]{
		{
			_type: "ok",
			name:  "create webhook",
			input: CreateWebHookInput{
				RepoInfo: RepoInfo{
					Owner:      "test_owner",
					Repository: "test_repo",
				},
				Events:      []string{"push"},
				Active:      *github.Bool(true),
				HookSecret:  "hook_secret",
				ContentType: "json",
			},
			wantResp: CreateWebHookResp{
				HookInfo: HookInfo{
					ID:      1,
					URL:     "hook_url",
					PingURL: "ping_url",
					TestURL: "test_url",
					Config: HookConfig{
						URL:         "hook_url",
						InsecureSSL: "0",
						ContentType: "json",
					},
				},
			},
		},
		{
			_type: "nok",
			name:  "403 API rate limit exceeded",
			input: CreateWebHookInput{
				RepoInfo: RepoInfo{
					Owner:      "rate_limit",
					Repository: "test_repo",
				},
				Events:      []string{"push"},
				Active:      *github.Bool(true),
				HookSecret:  "hook_secret",
				ContentType: "json",
			},
			wantErr: `403 API rate limit exceeded`,
		},
		{
			_type: "nok",
			name:  "404 Not Found",
			input: CreateWebHookInput{
				RepoInfo: RepoInfo{
					Owner:      "not_found",
					Repository: "test_repo",
				},
				Events:      []string{"push"},
				Active:      *github.Bool(true),
				HookSecret:  "hook_secret",
				ContentType: "json",
			},
			wantErr: `404 Not Found`,
		},
	}
	taskTesting(testcases, taskCreateWebhook, t)
}

func taskTesting[inType any, outType any](testcases []TaskCase[inType, outType], task string, t *testing.T) {
	c := qt.New(t)
	ctx := context.Background()
	bc := base.Component{Logger: zap.NewNop()}
	component := Init(bc)

	for _, tc := range testcases {
		c.Run(tc._type+`-`+tc.name, func(c *qt.C) {

			setup, err := structpb.NewStruct(map[string]any{
				"token": token,
			})
			c.Assert(err, qt.IsNil)

			e := &execution{
				ComponentExecution: base.ComponentExecution{Component: component, SystemVariables: nil, Setup: setup, Task: task},
				client:             *MockGithubClient,
			}
			switch task {
			case taskListPRs:
				e.execute = e.client.listPullRequestsTask
			case taskGetPR:
				e.execute = e.client.getPullRequestTask
			case taskGetReviewComments:
				e.execute = e.client.listReviewCommentsTask
			case taskCreateReviewComment:
				e.execute = e.client.createReviewCommentTask
			case taskGetCommit:
				e.execute = e.client.getCommitTask
			case taskListIssues:
				e.execute = e.client.listIssuesTask
			case taskGetIssue:
				e.execute = e.client.getIssueTask
			case taskCreateIssue:
				e.execute = e.client.createIssueTask
			case taskCreateWebhook:
				e.execute = e.client.createWebhookTask
			default:
				c.Fatalf("not supported testing task: %s", task)
			}

			pbIn, err := base.ConvertToStructpb(tc.input)
			c.Assert(err, qt.IsNil)

			ir, ow, eh, job := base.GenerateMockJob(c)
			ir.ReadMock.Return(pbIn, nil)
			ow.WriteMock.Optional().Set(func(ctx context.Context, output *structpb.Struct) (err error) {
				wantJSON, err := json.Marshal(tc.wantResp)
				c.Assert(err, qt.IsNil)
				c.Check(wantJSON, qt.JSONEquals, output.AsMap())
				return nil
			})
			eh.ErrorMock.Optional().Set(func(ctx context.Context, err error) {
				c.Assert(err, qt.ErrorMatches, tc.wantErr)
			})
			err = e.Execute(ctx, []*base.Job{job})
			c.Assert(err, qt.IsNil)

		})
	}
}
