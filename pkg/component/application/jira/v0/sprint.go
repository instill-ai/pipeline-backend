package jira

import (
	_ "embed"
)

// Sprint is the Jira sprint object.
type Sprint struct {
	ID            int    `json:"id"`
	Self          string `json:"self"`
	State         string `json:"state"`
	Name          string `json:"name"`
	StartDate     string `json:"startDate"`
	EndDate       string `json:"endDate"`
	CompleteDate  string `json:"completeDate"`
	OriginBoardID int    `json:"originBoardId"`
	Goal          string `json:"goal"`
}

func extractSprintOutput(sprint *Sprint) *getSprintOutput {
	return &getSprintOutput{
		ID:            sprint.ID,
		Self:          sprint.Self,
		State:         sprint.State,
		Name:          sprint.Name,
		StartDate:     sprint.StartDate,
		EndDate:       sprint.EndDate,
		CompleteDate:  sprint.CompleteDate,
		OriginBoardID: sprint.OriginBoardID,
		Goal:          sprint.Goal,
	}
}

type listSprintsResp struct {
	Sprints    []Sprint `json:"sprints"`
	StartAt    int      `json:"startAt"`
	MaxResults int      `json:"maxResults"`
	Total      int      `json:"total"`
}

type createSprintReq struct {
	Name          string `json:"name"`
	Goal          string `json:"goal"`
	StartDate     string `json:"startDate"`
	EndDate       string `json:"endDate"`
	OriginBoardID int    `json:"originBoardId"`
}

type createSprintResp struct {
	Sprint
}

type updateSprintReq struct {
	Name      string `json:"name,omitempty"`
	Goal      string `json:"goal,omitempty"`
	StartDate string `json:"startDate,omitempty"`
	EndDate   string `json:"endDate,omitempty"`
	State     string `json:"state,omitempty"`
}
