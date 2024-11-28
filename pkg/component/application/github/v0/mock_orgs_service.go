package github

import (
	"context"
	"fmt"

	"github.com/google/go-github/v62/github"
)

// MockOrganizationsService is a mock implementation of the OrganizationService interface.
type MockOrganizationsService struct{}

// Get is a mock implementation of the Get method for the OrganizationService.
func (m *MockOrganizationsService) Get(ctx context.Context, org string) (*github.Organization, *github.Response, error) {
	switch middleWare(org) {
	case 403:
		return nil, nil, fmt.Errorf("403 API rate limit exceeded")
	case 404:
		return nil, nil, fmt.Errorf("404 Not Found")
	}

	resp := &github.Response{}
	organization := &github.Organization{
		Login:                       github.String("test-org"),
		ID:                          github.Int64(1),
		NodeID:                      github.String("node1"),
		URL:                         github.String("https://api.github.com/orgs/test-org"),
		ReposURL:                    github.String("https://api.github.com/orgs/test-org/repos"),
		EventsURL:                   github.String("https://api.github.com/orgs/test-org/events"),
		HooksURL:                    github.String("https://api.github.com/orgs/test-org/hooks"),
		IssuesURL:                   github.String("https://api.github.com/orgs/test-org/issues"),
		MembersURL:                  github.String("https://api.github.com/orgs/test-org/members{/member}"),
		PublicMembersURL:            github.String("https://api.github.com/orgs/test-org/public_members{/member}"),
		AvatarURL:                   github.String("https://github.com/images/error/octocat_happy.gif"),
		Description:                 github.String("A great organization"),
		Name:                        github.String("test-org"),
		Company:                     github.String("GitHub"),
		Blog:                        github.String("https://github.com/blog"),
		Location:                    github.String("San Francisco"),
		Email:                       github.String("octocat@github.com"),
		TwitterUsername:             github.String("github"),
		IsVerified:                  github.Bool(true),
		HasOrganizationProjects:     github.Bool(true),
		HasRepositoryProjects:       github.Bool(true),
		PublicRepos:                 github.Int(2),
		PublicGists:                 github.Int(1),
		Followers:                   github.Int(20),
		Following:                   github.Int(0),
		HTMLURL:                     github.String("https://github.com/octocat"),
		Type:                        github.String("Organization"),
		CreatedAt:                   &github.Timestamp{},
		UpdatedAt:                   &github.Timestamp{},
		TotalPrivateRepos:           github.Int64(50),
		OwnedPrivateRepos:           github.Int64(45),
		PrivateGists:                github.Int(10),
		DiskUsage:                   github.Int(50000),
		Collaborators:               github.Int(25),
		BillingEmail:                github.String("billing@test-org.com"),
		TwoFactorRequirementEnabled: github.Bool(true),
	}
	return organization, resp, nil
}

// GetByID is a mock implementation of the GetByID method for the OrganizationService.
func (m *MockOrganizationsService) GetByID(ctx context.Context, id int64) (*github.Organization, *github.Response, error) {
	if id == 403 {
		return nil, nil, fmt.Errorf("403 API rate limit exceeded")
	}
	if id == 404 {
		return nil, nil, fmt.Errorf("404 Not Found")
	}

	return m.Get(ctx, "test-org")
}
