package github

import (
	"time"
)

type User struct {
	Login                   *string    `instill:"login"`
	ID                      *int64     `instill:"id"`
	NodeID                  *string    `instill:"node-id"`
	AvatarURL               *string    `instill:"avatar-url"`
	HTMLURL                 *string    `instill:"html-url"`
	GravatarID              *string    `instill:"gravatar-id"`
	Name                    *string    `instill:"name"`
	Company                 *string    `instill:"company"`
	Blog                    *string    `instill:"blog"`
	Location                *string    `instill:"location"`
	Email                   *string    `instill:"email"`
	Hireable                *bool      `instill:"hireable"`
	Bio                     *string    `instill:"bio"`
	TwitterUsername         *string    `instill:"twitter-username"`
	PublicRepos             *int       `instill:"public-repos"`
	PublicGists             *int       `instill:"public-gists"`
	Followers               *int       `instill:"followers"`
	Following               *int       `instill:"following"`
	CreatedAt               *Timestamp `instill:"created-at"`
	UpdatedAt               *Timestamp `instill:"updated-at"`
	SuspendedAt             *Timestamp `instill:"suspended-at"`
	Type                    *string    `instill:"type"`
	SiteAdmin               *bool      `instill:"site-admin"`
	TotalPrivateRepos       *int64     `instill:"total-private-repos"`
	OwnedPrivateRepos       *int64     `instill:"owned-private-repos"`
	PrivateGists            *int       `instill:"private-gists"`
	DiskUsage               *int       `instill:"disk-usage"`
	Collaborators           *int       `instill:"collaborators"`
	TwoFactorAuthentication *bool      `instill:"two-factor-authentication"`
	Plan                    *Plan      `instill:"plan"`
	LdapDn                  *string    `instill:"ldap-dn"`

	// API URLs
	URL               *string `instill:"url"`
	EventsURL         *string `instill:"events-url"`
	FollowingURL      *string `instill:"following-url"`
	FollowersURL      *string `instill:"followers-url"`
	GistsURL          *string `instill:"gists-url"`
	OrganizationsURL  *string `instill:"organizations-url"`
	ReceivedEventsURL *string `instill:"received-events-url"`
	ReposURL          *string `instill:"repos-url"`
	StarredURL        *string `instill:"starred-url"`
	SubscriptionsURL  *string `instill:"subscriptions-url"`

	// TextMatches is only populated from search results that request text matches
	// See: search.go and https://docs.github.com/rest/search/#text-match-metadata
	TextMatches []*TextMatch `instill:"text-matches"`

	// Permissions and RoleName identify the permissions and role that a user has on a given
	// repository. These are only populated when calling Repositories.ListCollaborators.
	Permissions map[string]bool `instill:"permissions"`
	RoleName    *string         `instill:"role-name"`
}

type Timestamp struct {
	time.Time
}

type Plan struct {
	Name          *string `instill:"name"`
	Space         *int    `instill:"space"`
	Collaborators *int    `instill:"collaborators"`
	PrivateRepos  *int64  `instill:"private-repos"`
	FilledSeats   *int    `instill:"filled-seats"`
	Seats         *int    `instill:"seats"`
}

type TextMatch struct {
	ObjectURL  *string  `instill:"object-url"`
	ObjectType *string  `instill:"object-type"`
	Property   *string  `instill:"property"`
	Fragment   *string  `instill:"fragment"`
	Matches    []*Match `instill:"matches"`
}

type Match struct {
	Text    *string `instill:"text"`
	Indices []int   `instill:"indices"`
}
