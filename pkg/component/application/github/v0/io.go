package github

type createIssueInput struct {
	RepoInfo
	Title     string   `instill:"title"`
	Body      string   `instill:"body"`
	Assignees []string `instill:"assignees"`
	Labels    []string `instill:"labels"`
}

type createIssueOutput struct {
	Issue
}

type createReviewCommentInput struct {
	RepoInfo
	PRNumber int                `instill:"pr-number"`
	Comment  PullRequestComment `instill:"comment"`
}

type createReviewCommentOutput struct {
	ReviewComment
}

type createWebHookInput struct {
	RepoInfo
	HookURL     string   `instill:"hook-url"`
	HookSecret  string   `instill:"hook-secret"`
	Events      []string `instill:"events"`
	Active      bool     `instill:"active"`
	ContentType string   `instill:"content-type"`
}

type createWebHookOutput struct {
	HookInfo
}

type getCommitInput struct {
	RepoInfo
	SHA string `instill:"sha"`
}

type getCommitOutput struct {
	Commit Commit `instill:"commit"`
}

type getIssueInput struct {
	RepoInfo
	IssueNumber int `instill:"issue-number"`
}

type getIssueOutput struct {
	Issue
}

type getOrganizationInput struct {
	OrgName string `instill:"name"`
	OrgID   int64  `instill:"id"`
}

type getOrganizationOutput struct {
	Organization Organization `instill:"organization"`
}

type getPullRequestInput struct {
	RepoInfo
	PRNumber float64 `instill:"pr-number"`
}

type getPullRequestOutput struct {
	PullRequest
}

type getUserInput struct {
	Username string `instill:"name"`
	UserID   int64  `instill:"id"`
}

type getUserOutput struct {
	User User `instill:"user"`
}

type listIssuesInput struct {
	PageOptions
	RepoInfo
	State         string `instill:"state"`
	Sort          string `instill:"sort"`
	Direction     string `instill:"direction"`
	Since         string `instill:"since"`
	NoPullRequest bool   `instill:"no-pull-request"`
}

type listIssuesOutput struct {
	Issues   []Issue   `instill:"issues"`
	Response *Response `instill:"response"`
}

type listPullRequestsInput struct {
	RepoInfo
	PageOptions
	State     string `instill:"state,default=open"`
	Sort      string `instill:"sort,default=created"`
	Direction string `instill:"direction,default=desc"`
}

type listPullRequestsOutput struct {
	PullRequests []PullRequest `instill:"pull-requests"`
	Response     *Response     `instill:"response"`
}

type listReviewCommentsInput struct {
	PageOptions
	RepoInfo
	PRNumber  int    `instill:"pr-number"`
	Sort      string `instill:"sort"`
	Direction string `instill:"direction"`
	Since     string `instill:"since"`
}

type listReviewCommentsOutput struct {
	ReviewComments []ReviewComment `instill:"comments"`
	Response       *Response       `instill:"response"`
}
