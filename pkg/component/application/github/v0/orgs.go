package github

import (
	"context"
	"time"

	"github.com/google/go-github/v62/github"
)

// OrganizationsService handles communication with the organization related methods of the GitHub API.
type OrganizationsService interface {
	// Get fetches an organization by name.
	Get(ctx context.Context, org string) (*github.Organization, *github.Response, error)

	// GetByID fetches an organization by ID.
	GetByID(ctx context.Context, id int64) (*github.Organization, *github.Response, error)
}

// Organization represents a GitHub organization.
type Organization struct {
	Login                                          string           `json:"login"`
	ID                                             int64            `json:"id"`
	NodeID                                         string           `json:"node-id"`
	AvatarURL                                      string           `json:"avatar-url"`
	HTMLURL                                        string           `json:"html-url"`
	Name                                           string           `json:"name"`
	Company                                        string           `json:"company"`
	Blog                                           string           `json:"blog"`
	Location                                       string           `json:"location"`
	Email                                          string           `json:"email"`
	TwitterUsername                                string           `json:"twitter-username"`
	Description                                    string           `json:"description"`
	PublicRepos                                    int              `json:"public-repos"`
	PublicGists                                    int              `json:"public-gists"`
	Followers                                      int              `json:"followers"`
	Following                                      int              `json:"following"`
	CreatedAt                                      time.Time        `json:"created-at"`
	UpdatedAt                                      time.Time        `json:"updated-at"`
	TotalPrivateRepos                              int64            `json:"total-private-repos"`
	OwnedPrivateRepos                              int64            `json:"owned-private-repos"`
	PrivateGists                                   int              `json:"private-gists"`
	DiskUsage                                      int              `json:"disk-usage"`
	Collaborators                                  int              `json:"collaborators"`
	BillingEmail                                   string           `json:"billing-email"`
	Type                                           string           `json:"type"`
	Plan                                           OrganizationPlan `json:"plan"`
	TwoFactorRequirementEnabled                    bool             `json:"two-factor-requirement-enabled"`
	IsVerified                                     bool             `json:"is-verified"`
	HasOrganizationProjects                        bool             `json:"has-organization-projects"`
	HasRepositoryProjects                          bool             `json:"has-repository-projects"`
	DefaultRepoPermission                          string           `json:"default-repository-permission"`
	DefaultRepoSettings                            string           `json:"default-repository-settings"`
	MembersCanCreateRepos                          bool             `json:"members-can-create-repositories"`
	MembersCanCreatePublicRepos                    bool             `json:"members-can-create-public-repositories"`
	MembersCanCreatePrivateRepos                   bool             `json:"members-can-create-private-repositories"`
	MembersCanCreateInternalRepos                  bool             `json:"members-can-create-internal-repositories"`
	MembersCanForkPrivateRepos                     bool             `json:"members-can-fork-private-repositories"`
	MembersAllowedRepositoryCreationType           string           `json:"members-allowed-repository-creation-type"`
	MembersCanCreatePages                          bool             `json:"members-can-create-pages"`
	MembersCanCreatePublicPages                    bool             `json:"members-can-create-public-pages"`
	MembersCanCreatePrivatePages                   bool             `json:"members-can-create-private-pages"`
	WebCommitSignoffRequired                       bool             `json:"web-commit-signoff-required"`
	AdvancedSecurityEnabledForNewRepos             bool             `json:"advanced-security-enabled-for-new-repositories"`
	DependabotAlertsEnabledForNewRepos             bool             `json:"dependabot-alerts-enabled-for-new-repositories"`
	DependabotSecurityUpdatesEnabledForNewRepos    bool             `json:"dependabot-security-updates-enabled-for-new-repositories"`
	DependencyGraphEnabledForNewRepos              bool             `json:"dependency-graph-enabled-for-new-repositories"`
	SecretScanningEnabledForNewRepos               bool             `json:"secret-scanning-enabled-for-new-repositories"`
	SecretScanningPushProtectionEnabledForNewRepos bool             `json:"secret-scanning-push-protection-enabled-for-new-repositories"`
	SecretScanningValidityChecksEnabled            bool             `json:"secret-scanning-validity-checks-enabled"`

	// API URLs
	URL              string `json:"url"`
	EventsURL        string `json:"events-url"`
	HooksURL         string `json:"hooks-url"`
	IssuesURL        string `json:"issues-url"`
	MembersURL       string `json:"members-url"`
	PublicMembersURL string `json:"public-members-url"`
	ReposURL         string `json:"repos-url"`
}

// OrganizationPlan represents a GitHub organization's plan
type OrganizationPlan struct {
	Name         string `json:"name"`
	Space        int    `json:"space"`
	PrivateRepos int64  `json:"private-repos"`
	FilledSeats  int    `json:"filled-seats"`
	Seats        int    `json:"seats"`
}

// extractOrganization extracts organization information from a GitHub organization object.
func (client *Client) extractOrganization(org *github.Organization) Organization {
	organization := Organization{
		Login:                              getString(org.Login),
		ID:                                 getInt64(org.ID),
		NodeID:                             getString(org.NodeID),
		URL:                                getString(org.URL),
		ReposURL:                           getString(org.ReposURL),
		EventsURL:                          getString(org.EventsURL),
		HooksURL:                           getString(org.HooksURL),
		IssuesURL:                          getString(org.IssuesURL),
		MembersURL:                         getString(org.MembersURL),
		PublicMembersURL:                   getString(org.PublicMembersURL),
		AvatarURL:                          getString(org.AvatarURL),
		Description:                        getString(org.Description),
		Name:                               getString(org.Name),
		Company:                            getString(org.Company),
		Blog:                               getString(org.Blog),
		Location:                           getString(org.Location),
		Email:                              getString(org.Email),
		TwitterUsername:                    getString(org.TwitterUsername),
		IsVerified:                         getBool(org.IsVerified),
		HasOrganizationProjects:            getBool(org.HasOrganizationProjects),
		HasRepositoryProjects:              getBool(org.HasRepositoryProjects),
		PublicRepos:                        getInt(org.PublicRepos),
		PublicGists:                        getInt(org.PublicGists),
		Followers:                          getInt(org.Followers),
		Following:                          getInt(org.Following),
		HTMLURL:                            getString(org.HTMLURL),
		Type:                               getString(org.Type),
		TotalPrivateRepos:                  getInt64(org.TotalPrivateRepos),
		OwnedPrivateRepos:                  getInt64(org.OwnedPrivateRepos),
		PrivateGists:                       getInt(org.PrivateGists),
		DiskUsage:                          getInt(org.DiskUsage),
		Collaborators:                      getInt(org.Collaborators),
		BillingEmail:                       getString(org.BillingEmail),
		DefaultRepoPermission:              getString(org.DefaultRepoPermission),
		DefaultRepoSettings:                getString(org.DefaultRepoSettings),
		MembersCanCreateRepos:              getBool(org.MembersCanCreateRepos),
		MembersCanCreatePublicRepos:        getBool(org.MembersCanCreatePublicRepos),
		MembersCanCreatePrivateRepos:       getBool(org.MembersCanCreatePrivateRepos),
		MembersCanCreateInternalRepos:      getBool(org.MembersCanCreateInternalRepos),
		MembersCanForkPrivateRepos:         getBool(org.MembersCanForkPrivateRepos),
		MembersCanCreatePages:              getBool(org.MembersCanCreatePages),
		MembersCanCreatePublicPages:        getBool(org.MembersCanCreatePublicPages),
		MembersCanCreatePrivatePages:       getBool(org.MembersCanCreatePrivatePages),
		WebCommitSignoffRequired:           getBool(org.WebCommitSignoffRequired),
		AdvancedSecurityEnabledForNewRepos: getBool(org.AdvancedSecurityEnabledForNewRepos),
		DependabotAlertsEnabledForNewRepos: getBool(org.DependabotAlertsEnabledForNewRepos),
		DependabotSecurityUpdatesEnabledForNewRepos:    getBool(org.DependabotSecurityUpdatesEnabledForNewRepos),
		DependencyGraphEnabledForNewRepos:              getBool(org.DependencyGraphEnabledForNewRepos),
		SecretScanningEnabledForNewRepos:               getBool(org.SecretScanningEnabledForNewRepos),
		SecretScanningPushProtectionEnabledForNewRepos: getBool(org.SecretScanningPushProtectionEnabledForNewRepos),
		SecretScanningValidityChecksEnabled:            getBool(org.SecretScanningValidityChecksEnabled),
		TwoFactorRequirementEnabled:                    getBool(org.TwoFactorRequirementEnabled),
	}

	if org.CreatedAt != nil && org.CreatedAt.Time != (time.Time{}) {
		organization.CreatedAt = org.CreatedAt.Time
	}
	if org.UpdatedAt != nil && org.UpdatedAt.Time != (time.Time{}) {
		organization.UpdatedAt = org.UpdatedAt.Time
	}

	if org.Plan != nil {
		organization.Plan = OrganizationPlan{
			Name:         getString(org.Plan.Name),
			Space:        getInt(org.Plan.Space),
			PrivateRepos: getInt64(org.Plan.PrivateRepos),
			FilledSeats:  getInt(org.Plan.FilledSeats),
			Seats:        getInt(org.Plan.Seats),
		}
	}

	return organization
}
