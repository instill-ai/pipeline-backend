package github

import (
	"testing"
	"time"

	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func TestComponent_GetOrganizationTask(t *testing.T) {
	testCases := []TaskCase[getOrganizationInput, getOrganizationOutput]{
		{
			_type: "ok",
			name:  "get organization by name",
			input: getOrganizationInput{
				OrgName: "test-org",
			},
			wantOutput: getOrganizationOutput{
				Organization: Organization{
					Login:                       "test-org",
					ID:                          1,
					NodeID:                      "node1",
					URL:                         "https://api.github.com/orgs/test-org",
					ReposURL:                    "https://api.github.com/orgs/test-org/repos",
					EventsURL:                   "https://api.github.com/orgs/test-org/events",
					HooksURL:                    "https://api.github.com/orgs/test-org/hooks",
					IssuesURL:                   "https://api.github.com/orgs/test-org/issues",
					MembersURL:                  "https://api.github.com/orgs/test-org/members{/member}",
					PublicMembersURL:            "https://api.github.com/orgs/test-org/public_members{/member}",
					AvatarURL:                   "https://github.com/images/error/octocat_happy.gif",
					Description:                 "A great organization",
					Name:                        "test-org",
					Company:                     "GitHub",
					Blog:                        "https://github.com/blog",
					Location:                    "San Francisco",
					Email:                       "octocat@github.com",
					TwitterUsername:             "github",
					IsVerified:                  true,
					HasOrganizationProjects:     true,
					HasRepositoryProjects:       true,
					PublicRepos:                 2,
					PublicGists:                 1,
					Followers:                   20,
					Following:                   0,
					HTMLURL:                     "https://github.com/octocat",
					Type:                        "Organization",
					CreatedAt:                   time.Time{},
					UpdatedAt:                   time.Time{},
					TotalPrivateRepos:           50,
					OwnedPrivateRepos:           45,
					PrivateGists:                10,
					DiskUsage:                   50000,
					Collaborators:               25,
					BillingEmail:                "billing@test-org.com",
					TwoFactorRequirementEnabled: true,
				},
			},
		},
		{
			_type: "ok",
			name:  "get organization by ID",
			input: getOrganizationInput{
				OrgID: 1,
			},
			wantOutput: getOrganizationOutput{
				Organization: Organization{
					Login:                       "test-org",
					ID:                          1,
					NodeID:                      "node1",
					URL:                         "https://api.github.com/orgs/test-org",
					ReposURL:                    "https://api.github.com/orgs/test-org/repos",
					EventsURL:                   "https://api.github.com/orgs/test-org/events",
					HooksURL:                    "https://api.github.com/orgs/test-org/hooks",
					IssuesURL:                   "https://api.github.com/orgs/test-org/issues",
					MembersURL:                  "https://api.github.com/orgs/test-org/members{/member}",
					PublicMembersURL:            "https://api.github.com/orgs/test-org/public_members{/member}",
					AvatarURL:                   "https://github.com/images/error/octocat_happy.gif",
					Description:                 "A great organization",
					Name:                        "test-org",
					Company:                     "GitHub",
					Blog:                        "https://github.com/blog",
					Location:                    "San Francisco",
					Email:                       "octocat@github.com",
					TwitterUsername:             "github",
					IsVerified:                  true,
					HasOrganizationProjects:     true,
					HasRepositoryProjects:       true,
					PublicRepos:                 2,
					PublicGists:                 1,
					Followers:                   20,
					Following:                   0,
					HTMLURL:                     "https://github.com/octocat",
					Type:                        "Organization",
					CreatedAt:                   time.Time{},
					UpdatedAt:                   time.Time{},
					TotalPrivateRepos:           50,
					OwnedPrivateRepos:           45,
					PrivateGists:                10,
					DiskUsage:                   50000,
					Collaborators:               25,
					BillingEmail:                "billing@test-org.com",
					TwoFactorRequirementEnabled: true,
				},
			},
		},
		{
			_type: "nok",
			name:  "403 API rate limit exceeded",
			input: getOrganizationInput{
				OrgName: "rate_limit",
			},
			wantErr: `403 API rate limit exceeded`,
		},
		{
			_type: "nok",
			name:  "404 Not Found",
			input: getOrganizationInput{
				OrgName: "not_found",
			},
			wantErr: `404 Not Found`,
		},
	}

	e := &execution{
		client: *MockGithubClient,
		ComponentExecution: base.ComponentExecution{
			Task:            taskGetOrganization,
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

	e.execute = e.client.getOrganization

	taskTesting(testCases, e, t)
}
