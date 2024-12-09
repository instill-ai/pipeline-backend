package jira

type createIssueInput struct {
	UpdateHistory bool      `instill:"update-history"`
	ProjectKey    string    `instill:"project-key"`
	IssueType     issueType `instill:"issue-type"`
	Summary       string    `instill:"summary"`
	Description   string    `instill:"description"`
}

type createIssueOutput struct {
	Issue
}

type createSprintInput struct {
	BoardName string `instill:"board-name"`
	Name      string `instill:"name"`
	Goal      string `instill:"goal"`
	StartDate string `instill:"start-date"`
	EndDate   string `instill:"end-date"`
}

type createSprintOutput struct {
	ID            int    `instill:"id"`
	Self          string `instill:"self"`
	State         string `instill:"state"`
	Name          string `instill:"name"`
	StartDate     string `instill:"start-date"`
	EndDate       string `instill:"end-date"`
	CompleteDate  string `instill:"complete-date"`
	OriginBoardID int    `instill:"origin-board-id"`
	Goal          string `instill:"goal"`
}

type getIssueInput struct {
	IssueKey      string `instill:"issue-key,omitempty" api:"issueIdOrKey"`
	UpdateHistory bool   `instill:"update-history,omitempty" api:"updateHistory"`
}

type getIssueOutput struct {
	Issue
}

type getSprintInput struct {
	SprintID int `instill:"sprint-id"`
}

type getSprintOutput struct {
	ID            int    `instill:"id"`
	Self          string `instill:"self"`
	State         string `instill:"state"`
	Name          string `instill:"name"`
	StartDate     string `instill:"start-date"`
	EndDate       string `instill:"end-date"`
	CompleteDate  string `instill:"complete-date"`
	OriginBoardID int    `instill:"origin-board-id"`
	Goal          string `instill:"goal"`
}

type listBoardsInput struct {
	ProjectKeyOrID string `instill:"project-key-or-id,omitempty" api:"projectKeyOrId"`
	BoardType      string `instill:"board-type,default=simple" api:"type"`
	Name           string `instill:"name,omitempty" api:"name"`
	StartAt        int    `instill:"start-at,default=0" api:"startAt"`
	MaxResults     int    `instill:"max-results,default=50" api:"maxResults"`
}

type listBoardsOutput struct {
	Boards     []Board `instill:"boards"`
	StartAt    int     `instill:"start-at"`
	MaxResults int     `instill:"max-results"`
	Total      int     `instill:"total"`
	IsLast     bool    `instill:"is-last"`
}

type listIssuesInput struct {
	BoardName  string     `instill:"board-name,omitempty" api:"boardName"`
	MaxResults int        `instill:"max-results,default=50" api:"maxResults"`
	StartAt    int        `instill:"start-at,default=0" api:"startAt"`
	RangeData  issueRange `instill:"range,omitempty"`
}

type listIssuesOutput struct {
	Issues     []Issue `instill:"issues"`
	StartAt    int     `instill:"start-at"`
	MaxResults int     `instill:"max-results"`
	Total      int     `instill:"total"`
}

type listSprintsInput struct {
	BoardID    int `instill:"board-id"`
	StartAt    int `instill:"start-at,default=0" api:"startAt"`
	MaxResults int `instill:"max-results,default=50" api:"maxResults"`
}

type listSprintsOutput struct {
	Sprints    []*getSprintOutput `instill:"sprints"`
	StartAt    int                `instill:"start-at"`
	MaxResults int                `instill:"max-results"`
	Total      int                `instill:"total"`
}

type updateIssueInput struct {
	IssueKey    string `instill:"issue-key"`
	Update      update `instill:"update"`
	NotifyUsers bool   `instill:"notify-users" api:"notifyUsers"`
}

type updateIssueOutput struct {
	Issue
}

type updateSprintInput struct {
	SprintID       int    `instill:"sprint-id"`
	Name           string `instill:"name"`
	Goal           string `instill:"goal"`
	StartDate      string `instill:"start-date"`
	EndDate        string `instill:"end-date"`
	CurrentState   string `instill:"current-state"`
	EnterNextState bool   `instill:"enter-next-state"`
}

type updateSprintOutput struct {
	ID            int    `instill:"id"`
	Self          string `instill:"self"`
	State         string `instill:"state"`
	Name          string `instill:"name"`
	StartDate     string `instill:"start-date"`
	EndDate       string `instill:"end-date"`
	CompleteDate  string `instill:"complete-date"`
	OriginBoardID int    `instill:"origin-board-id"`
	Goal          string `instill:"goal"`
}
