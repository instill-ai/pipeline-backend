package github

import (
	"testing"

	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/google/go-github/v62/github"
	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func TestComponent_GetUserTask(t *testing.T) {
	testCases := []TaskCase[getUserInput, getUserOutput]{
		{
			_type: "ok",
			name:  "get user by username",
			input: getUserInput{
				Username: "test-user",
			},
			wantOutput: getUserOutput{
				User: User{
					Login:                   github.String("test-user"),
					ID:                      github.Int64(1),
					NodeID:                  github.String("node1"),
					AvatarURL:               github.String("https://avatar.url"),
					HTMLURL:                 github.String("https://github.com/test-user"),
					GravatarID:              github.String(""),
					Name:                    github.String("Test User"),
					Company:                 github.String("Test Company"),
					Blog:                    github.String("https://blog.com"),
					Location:                github.String("Test Location"),
					Email:                   github.String("test@example.com"),
					Hireable:                github.Bool(true),
					Bio:                     github.String("Test Bio"),
					TwitterUsername:         github.String("testuser"),
					PublicRepos:             github.Int(10),
					PublicGists:             github.Int(5),
					Followers:               github.Int(100),
					Following:               github.Int(50),
					CreatedAt:               &github.Timestamp{},
					UpdatedAt:               &github.Timestamp{},
					SuspendedAt:             &github.Timestamp{},
					Type:                    github.String("User"),
					SiteAdmin:               github.Bool(false),
					TotalPrivateRepos:       github.Int64(2),
					OwnedPrivateRepos:       github.Int64(2),
					PrivateGists:            github.Int(1),
					DiskUsage:               github.Int(1000),
					Collaborators:           github.Int(3),
					TwoFactorAuthentication: github.Bool(true),
					Plan: &github.Plan{
						Name:          github.String("pro"),
						Space:         github.Int(100),
						Collaborators: github.Int(10),
						PrivateRepos:  github.Int64(50),
					},
					LdapDn:            github.String(""),
					URL:               github.String("https://api.github.com/users/test-user"),
					EventsURL:         github.String("https://api.github.com/users/test-user/events{/privacy}"),
					FollowingURL:      github.String("https://api.github.com/users/test-user/following{/other_user}"),
					FollowersURL:      github.String("https://api.github.com/users/test-user/followers"),
					GistsURL:          github.String("https://api.github.com/users/test-user/gists{/gist_id}"),
					OrganizationsURL:  github.String("https://api.github.com/users/test-user/orgs"),
					ReceivedEventsURL: github.String("https://api.github.com/users/test-user/received_events"),
					ReposURL:          github.String("https://api.github.com/users/test-user/repos"),
					StarredURL:        github.String("https://api.github.com/users/test-user/starred{/owner}{/repo}"),
					SubscriptionsURL:  github.String("https://api.github.com/users/test-user/subscriptions"),
					TextMatches:       nil,
					Permissions:       nil,
					RoleName:          nil,
				},
			},
		},
		{
			_type: "ok",
			name:  "get user by ID",
			input: getUserInput{
				UserID: 1,
			},
			wantOutput: getUserOutput{
				User: User{
					Login:                   github.String("test-user"),
					ID:                      github.Int64(1),
					NodeID:                  github.String("node1"),
					AvatarURL:               github.String("https://avatar.url"),
					HTMLURL:                 github.String("https://github.com/test-user"),
					GravatarID:              github.String(""),
					Name:                    github.String("Test User"),
					Company:                 github.String("Test Company"),
					Blog:                    github.String("https://blog.com"),
					Location:                github.String("Test Location"),
					Email:                   github.String("test@example.com"),
					Hireable:                github.Bool(true),
					Bio:                     github.String("Test Bio"),
					TwitterUsername:         github.String("testuser"),
					PublicRepos:             github.Int(10),
					PublicGists:             github.Int(5),
					Followers:               github.Int(100),
					Following:               github.Int(50),
					CreatedAt:               &github.Timestamp{},
					UpdatedAt:               &github.Timestamp{},
					SuspendedAt:             &github.Timestamp{},
					Type:                    github.String("User"),
					SiteAdmin:               github.Bool(false),
					TotalPrivateRepos:       github.Int64(2),
					OwnedPrivateRepos:       github.Int64(2),
					PrivateGists:            github.Int(1),
					DiskUsage:               github.Int(1000),
					Collaborators:           github.Int(3),
					TwoFactorAuthentication: github.Bool(true),
					Plan: &github.Plan{
						Name:          github.String("pro"),
						Space:         github.Int(100),
						Collaborators: github.Int(10),
						PrivateRepos:  github.Int64(50),
					},
					LdapDn:            github.String(""),
					URL:               github.String("https://api.github.com/users/test-user"),
					EventsURL:         github.String("https://api.github.com/users/test-user/events{/privacy}"),
					FollowingURL:      github.String("https://api.github.com/users/test-user/following{/other_user}"),
					FollowersURL:      github.String("https://api.github.com/users/test-user/followers"),
					GistsURL:          github.String("https://api.github.com/users/test-user/gists{/gist_id}"),
					OrganizationsURL:  github.String("https://api.github.com/users/test-user/orgs"),
					ReceivedEventsURL: github.String("https://api.github.com/users/test-user/received_events"),
					ReposURL:          github.String("https://api.github.com/users/test-user/repos"),
					StarredURL:        github.String("https://api.github.com/users/test-user/starred{/owner}{/repo}"),
					SubscriptionsURL:  github.String("https://api.github.com/users/test-user/subscriptions"),
					TextMatches:       nil,
					Permissions:       nil,
					RoleName:          nil,
				},
			},
		},
		{
			_type: "nok",
			name:  "403 API rate limit exceeded",
			input: getUserInput{
				Username: "rate_limit",
			},
			wantErr: `403 API rate limit exceeded`,
		},
		{
			_type: "nok",
			name:  "404 Not Found",
			input: getUserInput{
				Username: "not_found",
			},
			wantErr: `404 Not Found`,
		},
	}

	e := &execution{
		client: *MockGithubClient,
		ComponentExecution: base.ComponentExecution{
			Task:            taskGetUser,
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

	e.execute = e.client.getUser

	taskTesting(testCases, e, t)
}
