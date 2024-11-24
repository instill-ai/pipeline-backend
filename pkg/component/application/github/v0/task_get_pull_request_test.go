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
				PrNumber: 1,
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
				PrNumber: 0,
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
				PrNumber: 1,
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
				PrNumber: 1,
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

// func TestGetPullRequest(t *testing.T) {
// 	c := qt.New(t)

// 	testCases := []struct {
// 		name          string
// 		input         getPullRequestInput
// 		expectedResp  getPullRequestOutput
// 		expectedError string
// 	}{
// 		{
// 			name: "ok - get specific pull request",
// 			input: getPullRequestInput{
// 				RepoInfo: RepoInfo{
// 					Owner:      "test_owner",
// 					Repository: "test_repo",
// 				},
// 				PrNumber: 1,
// 			},
// 			expectedResp: getPullRequestOutput{
// 				PullRequest: PullRequest{
// 					Base: "baseSHA",
// 					Body: "PR Body",
// 					Commits: []Commit{
// 						{
// 							Message: "This is a fake commit",
// 							SHA:     "commitSHA",
// 							Stats: &CommitStats{
// 								Additions: 1,
// 								Deletions: 1,
// 								Changes:   2,
// 							},
// 							Files: []CommitFile{
// 								{
// 									Filename: "filename",
// 									Patch:    "patch",
// 									CommitStats: CommitStats{
// 										Additions: 1,
// 										Deletions: 1,
// 										Changes:   2,
// 									},
// 								},
// 							},
// 						},
// 					},
// 					DiffURL:           "https://fake-github.com/test_owner/test_repo/pull/1.diff",
// 					Head:              "headSHA",
// 					ID:                1,
// 					Number:            1,
// 					CommentsNum:       0,
// 					CommitsNum:        1,
// 					ReviewCommentsNum: 2,
// 					State:             "open",
// 					Title:             "This is a fake PR",
// 				},
// 			},
// 		},
// 		{
// 			name: "ok - get latest pull request",
// 			input: getPullRequestInput{
// 				RepoInfo: RepoInfo{
// 					Owner:      "test_owner",
// 					Repository: "test_repo",
// 				},
// 				PrNumber: 0,
// 			},
// 			expectedResp: getPullRequestOutput{
// 				PullRequest: PullRequest{
// 					Base: "baseSHA",
// 					Body: "PR Body",
// 					Commits: []Commit{
// 						{
// 							Message: "This is a fake commit",
// 							SHA:     "commitSHA",
// 							Stats: &CommitStats{
// 								Additions: 1,
// 								Deletions: 1,
// 								Changes:   2,
// 							},
// 							Files: []CommitFile{
// 								{
// 									Filename: "filename",
// 									Patch:    "patch",
// 									CommitStats: CommitStats{
// 										Additions: 1,
// 										Deletions: 1,
// 										Changes:   2,
// 									},
// 								},
// 							},
// 						},
// 					},
// 					DiffURL:           "https://fake-github.com/test_owner/test_repo/pull/1.diff",
// 					Head:              "headSHA",
// 					ID:                1,
// 					Number:            1,
// 					CommentsNum:       0,
// 					CommitsNum:        1,
// 					ReviewCommentsNum: 2,
// 					State:             "open",
// 					Title:             "This is a fake PR",
// 				},
// 			},
// 		},
// 		{
// 			name: "nok - rate limit exceeded",
// 			input: getPullRequestInput{
// 				RepoInfo: RepoInfo{
// 					Owner:      "rate_limit",
// 					Repository: "test_repo",
// 				},
// 				PrNumber: 1,
// 			},
// 			expectedError: "403 API rate limit exceeded",
// 		},
// 		{
// 			name: "nok - repository not found",
// 			input: getPullRequestInput{
// 				RepoInfo: RepoInfo{
// 					Owner:      "not_found",
// 					Repository: "test_repo",
// 				},
// 				PrNumber: 1,
// 			},
// 			expectedError: "404 Not Found",
// 		},
// 		{
// 			name: "nok - no pull requests found",
// 			input: getPullRequestInput{
// 				RepoInfo: RepoInfo{
// 					Owner:      "no_pr",
// 					Repository: "test_repo",
// 				},
// 				PrNumber: 0,
// 			},
// 			expectedError: "no pull request found",
// 		},
// 	}

// 	for _, tc := range testCases {
// 		c.Run(tc.name, func(c *qt.C) {
// 			component := Init(base.Component{Logger: zap.NewNop()})
// 			c.Assert(component, qt.IsNotNil)

// 			setup, err := structpb.NewStruct(map[string]any{
// 				"token": token,
// 			})
// 			c.Assert(err, qt.IsNil)

// 			e := &execution{
// 				ComponentExecution: base.ComponentExecution{Component: component, SystemVariables: nil, Setup: setup, Task: taskGetPR},
// 				client:             *MockGithubClient,
// 			}

// 			e.execute = e.client.getPullRequest

// 			ir, ow, eh, job := mock.GenerateMockJob(c)

// 			ir.ReadDataMock.Set(func(ctx context.Context, input any) error {
// 				switch input := input.(type) {
// 				case *getPullRequestInput:
// 					*input = tc.input
// 				}
// 				return nil
// 			})

// 			var capturedOutput any
// 			ow.WriteDataMock.Set(func(ctx context.Context, output any) error {
// 				capturedOutput = output
// 				output, ok := capturedOutput.(getPullRequestOutput)
// 				c.Assert(ok, qt.IsTrue)
// 				c.Assert(output, qt.DeepEquals, tc.expectedResp)
// 				return nil
// 			})

// 			eh.ErrorMock.Set(func(ctx context.Context, err error) {
// 				c.Assert(err, qt.ErrorMatches, tc.expectedError)
// 			})

// 			if tc.expectedError != "" {
// 				ow.WriteDataMock.Optional()
// 			} else {
// 				eh.ErrorMock.Optional()
// 			}

// 			err = e.Execute(context.Background(), []*base.Job{job})

// 			if tc.expectedError == "" {
// 				c.Assert(err, qt.IsNil)
// 			}
// 		})
// 	}
// }
