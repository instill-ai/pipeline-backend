package github

import (
	"context"
	"fmt"

	"github.com/google/go-github/v62/github"
)

// MockUsersService is a mock implementation of the UsersService interface.
type MockUsersService struct{}

// Get is a mock implementation of the Get method for the UsersService.
func (m *MockUsersService) Get(ctx context.Context, username string) (*github.User, *github.Response, error) {
	switch middleWare(username) {
	case 403:
		return nil, nil, fmt.Errorf("403 API rate limit exceeded")
	case 404:
		return nil, nil, fmt.Errorf("404 Not Found")
	}

	resp := &github.Response{}
	user := &github.User{
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
		CreatedAt:               nil,
		UpdatedAt:               nil,
		SuspendedAt:             nil,
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
		TextMatches:       []*github.TextMatch{},
		Permissions: map[string]bool{
			"admin": true,
			"push":  true,
			"pull":  true,
		},
		RoleName: github.String("admin"),
	}
	return user, resp, nil
}

// GetByID is a mock implementation of the GetByID method for the UsersService.
func (m *MockUsersService) GetByID(ctx context.Context, id int64) (*github.User, *github.Response, error) {
	if id == 403 {
		return nil, nil, fmt.Errorf("403 API rate limit exceeded")
	}
	if id == 404 {
		return nil, nil, fmt.Errorf("404 Not Found")
	}

	return m.Get(ctx, "test-user")
}
