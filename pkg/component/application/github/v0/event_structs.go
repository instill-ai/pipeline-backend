package github

type githubEventStarCreatedConfig struct {
	Repository string `instill:"repository"`
}

type rawGithubEvent struct {
	Action string `instill:"action"`
}

type rawGithubStarCreated struct {
	Action     string     `instill:"action"`
	StarredAt  string     `instill:"starred_at"`
	Repository repository `instill:"repository"`
	Sender     user       `instill:"sender"`
}

type repository struct {
	ID              int64    `instill:"id"`
	NodeID          string   `instill:"node_id"`
	Name            string   `instill:"name"`
	FullName        string   `instill:"full_name"`
	Private         bool     `instill:"private"`
	Owner           user     `instill:"owner"`
	HTMLURL         string   `instill:"html_url"`
	Description     *string  `instill:"description"`
	Fork            bool     `instill:"fork"`
	URL             string   `instill:"url"`
	CreatedAt       string   `instill:"created_at"`
	UpdatedAt       string   `instill:"updated_at"`
	PushedAt        string   `instill:"pushed_at"`
	Homepage        *string  `instill:"homepage"`
	Size            int      `instill:"size"`
	StargazersCount int      `instill:"stargazers_count"`
	WatchersCount   int      `instill:"watchers_count"`
	Language        *string  `instill:"language"`
	HasIssues       bool     `instill:"has_issues"`
	HasProjects     bool     `instill:"has_projects"`
	HasDownloads    bool     `instill:"has_downloads"`
	HasWiki         bool     `instill:"has_wiki"`
	HasPages        bool     `instill:"has_pages"`
	ForksCount      int      `instill:"forks_count"`
	Archived        bool     `instill:"archived"`
	OpenIssuesCount int      `instill:"open_issues_count"`
	License         *license `instill:"license"`
	Forks           int      `instill:"forks"`
	OpenIssues      int      `instill:"open_issues"`
	Watchers        int      `instill:"watchers"`
	DefaultBranch   string   `instill:"default_branch"`
	IsTemplate      bool     `instill:"is_template"`
	Topics          []string `instill:"topics"`
	Visibility      string   `instill:"visibility"`
}

type user struct {
	Login             string `instill:"login"`
	ID                int64  `instill:"id"`
	NodeID            string `instill:"node_id"`
	AvatarURL         string `instill:"avatar_url"`
	GravatarID        string `instill:"gravatar_id"`
	URL               string `instill:"url"`
	HTMLURL           string `instill:"html_url"`
	FollowersURL      string `instill:"followers_url"`
	FollowingURL      string `instill:"following_url"`
	GistsURL          string `instill:"gists_url"`
	StarredURL        string `instill:"starred_url"`
	SubscriptionsURL  string `instill:"subscriptions_url"`
	OrganizationsURL  string `instill:"organizations_url"`
	ReposURL          string `instill:"repos_url"`
	EventsURL         string `instill:"events_url"`
	ReceivedEventsURL string `instill:"received_events_url"`
	Type              string `instill:"type"`
	SiteAdmin         bool   `instill:"site_admin"`
}

type license struct {
	Key    string  `instill:"key"`
	Name   string  `instill:"name"`
	SPDXID string  `instill:"spdx_id"`
	URL    *string `instill:"url"`
	NodeID string  `instill:"node_id"`
}
