package github

import (
	"testing"

	"github.com/google/go-github/v62/github"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func TestComponent_CreateWebHook(t *testing.T) {
	testCases := []TaskCase[createWebHookInput, createWebHookOutput]{
		{
			_type: "ok",
			name:  "create webhook",
			input: createWebHookInput{
				RepoInfo: RepoInfo{
					Owner:      "test_owner",
					Repository: "test_repo",
				},
				Events:      []string{"push"},
				Active:      *github.Bool(true),
				HookSecret:  "hook_secret",
				ContentType: "json",
			},
			wantOutput: createWebHookOutput{
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
			input: createWebHookInput{
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
			input: createWebHookInput{
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
	e := &execution{
		client: *MockGithubClient,
		ComponentExecution: base.ComponentExecution{
			Task:            taskCreateWebhook,
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

	e.execute = e.client.createWebhook

	taskTesting(testCases, e, t)
}
