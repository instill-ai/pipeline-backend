package github

import (
	"context"

	"github.com/google/go-github/v62/github"
)

// UsersService handles communication with the user related methods of the GitHub API.
type UsersService interface {
	// Get fetches a user by username. If username is empty, fetches the authenticated user.
	Get(ctx context.Context, username string) (*github.User, *github.Response, error)

	// GetByID fetches a user by their numeric ID.
	GetByID(ctx context.Context, id int64) (*github.User, *github.Response, error)
}

// User represents a GitHub user (can be either private or public)
type User struct {
	Login                   *string           `json:"login,omitempty"`
	ID                      *int64            `json:"id,omitempty"`
	NodeID                  *string           `json:"node-id,omitempty"`
	AvatarURL               *string           `json:"avatar-url,omitempty"`
	HTMLURL                 *string           `json:"html-url,omitempty"`
	GravatarID              *string           `json:"gravatar-id,omitempty"`
	Name                    *string           `json:"name,omitempty"`
	Company                 *string           `json:"company,omitempty"`
	Blog                    *string           `json:"blog,omitempty"`
	Location                *string           `json:"location,omitempty"`
	Email                   *string           `json:"email,omitempty"`
	Hireable                *bool             `json:"hireable,omitempty"`
	Bio                     *string           `json:"bio,omitempty"`
	TwitterUsername         *string           `json:"twitter-username,omitempty"`
	PublicRepos             *int              `json:"public-repos,omitempty"`
	PublicGists             *int              `json:"public-gists,omitempty"`
	Followers               *int              `json:"followers,omitempty"`
	Following               *int              `json:"following,omitempty"`
	CreatedAt               *github.Timestamp `json:"created-at,omitempty"`
	UpdatedAt               *github.Timestamp `json:"updated-at,omitempty"`
	SuspendedAt             *github.Timestamp `json:"suspended-at,omitempty"`
	Type                    *string           `json:"type,omitempty"`
	SiteAdmin               *bool             `json:"site-admin,omitempty"`
	TotalPrivateRepos       *int64            `json:"total-private-repos,omitempty"`
	OwnedPrivateRepos       *int64            `json:"owned-private-repos,omitempty"`
	PrivateGists            *int              `json:"private-gists,omitempty"`
	DiskUsage               *int              `json:"disk-usage,omitempty"`
	Collaborators           *int              `json:"collaborators,omitempty"`
	TwoFactorAuthentication *bool             `json:"two-factor-authentication,omitempty"`
	Plan                    *github.Plan      `json:"plan,omitempty"`
	LdapDn                  *string           `json:"ldap-dn,omitempty"`

	// API URLs
	URL               *string `json:"url,omitempty"`
	EventsURL         *string `json:"events_url,omitempty"`
	FollowingURL      *string `json:"following-url,omitempty"`
	FollowersURL      *string `json:"followers-url,omitempty"`
	GistsURL          *string `json:"gists-url,omitempty"`
	OrganizationsURL  *string `json:"organizations-url,omitempty"`
	ReceivedEventsURL *string `json:"received-events-url,omitempty"`
	ReposURL          *string `json:"repos-url,omitempty"`
	StarredURL        *string `json:"starred-url,omitempty"`
	SubscriptionsURL  *string `json:"subscriptions-url,omitempty"`

	TextMatches []*github.TextMatch `json:"text-matches,omitempty"`

	Permissions map[string]bool `json:"permissions,omitempty"`
	RoleName    *string         `json:"role-name,omitempty"`
}

func (client *Client) extractUser(user *github.User) User {
	return User{
		Login:                   github.String(user.GetLogin()),
		ID:                      github.Int64(user.GetID()),
		NodeID:                  github.String(user.GetNodeID()),
		AvatarURL:               github.String(user.GetAvatarURL()),
		HTMLURL:                 github.String(user.GetHTMLURL()),
		GravatarID:              github.String(user.GetGravatarID()),
		Name:                    github.String(user.GetName()),
		Company:                 github.String(user.GetCompany()),
		Blog:                    github.String(user.GetBlog()),
		Location:                github.String(user.GetLocation()),
		Email:                   github.String(user.GetEmail()),
		Hireable:                github.Bool(user.GetHireable()),
		Bio:                     github.String(user.GetBio()),
		TwitterUsername:         github.String(user.GetTwitterUsername()),
		PublicRepos:             github.Int(user.GetPublicRepos()),
		PublicGists:             github.Int(user.GetPublicGists()),
		Followers:               github.Int(user.GetFollowers()),
		Following:               github.Int(user.GetFollowing()),
		CreatedAt:               &github.Timestamp{Time: user.GetCreatedAt().Time},
		UpdatedAt:               &github.Timestamp{Time: user.GetUpdatedAt().Time},
		SuspendedAt:             &github.Timestamp{Time: user.GetSuspendedAt().Time},
		Type:                    github.String(user.GetType()),
		SiteAdmin:               github.Bool(user.GetSiteAdmin()),
		TotalPrivateRepos:       github.Int64(user.GetTotalPrivateRepos()),
		OwnedPrivateRepos:       github.Int64(user.GetOwnedPrivateRepos()),
		PrivateGists:            github.Int(user.GetPrivateGists()),
		DiskUsage:               github.Int(user.GetDiskUsage()),
		Collaborators:           github.Int(user.GetCollaborators()),
		TwoFactorAuthentication: github.Bool(user.GetTwoFactorAuthentication()),
		Plan:                    user.Plan,
		LdapDn:                  github.String(user.GetLdapDn()),
		URL:                     github.String(user.GetURL()),
		EventsURL:               github.String(user.GetEventsURL()),
		FollowingURL:            github.String(user.GetFollowingURL()),
		FollowersURL:            github.String(user.GetFollowersURL()),
		GistsURL:                github.String(user.GetGistsURL()),
		OrganizationsURL:        github.String(user.GetOrganizationsURL()),
		ReceivedEventsURL:       github.String(user.GetReceivedEventsURL()),
		ReposURL:                github.String(user.GetReposURL()),
		StarredURL:              github.String(user.GetStarredURL()),
		SubscriptionsURL:        github.String(user.GetSubscriptionsURL()),
		TextMatches:             nil,
		Permissions:             nil,
		RoleName:                nil,
	}
}
