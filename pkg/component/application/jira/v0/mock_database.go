package jira

import (
	"fmt"
)

type FakeBoard struct {
	Board
}

func (f *FakeBoard) getSelf() string {
	if f.Self == "" {
		f.Self = fmt.Sprintf("https://test.atlassian.net/rest/agile/1.0/board/%d", f.ID)
	}
	return f.Self
}

var fakeBoards = []FakeBoard{
	{
		Board: Board{
			ID:        1,
			Name:      "KAN",
			BoardType: "kanban",
		},
	},
	{
		Board: Board{
			ID:        2,
			Name:      "SCR",
			BoardType: "scrum",
		},
	},
	{
		Board: Board{
			ID:        3,
			Name:      "TST",
			BoardType: "simple",
		},
	},
}

type FakeIssue struct {
	ID     string                 `json:"id"`
	Key    string                 `json:"key"`
	Self   string                 `json:"self"`
	Fields map[string]interface{} `json:"fields"`
}

func (f *FakeIssue) getSelf() string {
	if f.Self == "" {
		f.Self = fmt.Sprintf("https://test.atlassian.net/rest/agile/1.0/issue/%s", f.ID)
	}
	return f.Self
}

var fakeIssues = []FakeIssue{
	{
		ID:  "1",
		Key: "TST-1",
		Fields: map[string]interface{}{
			"summary":     "Test issue 1",
			"description": "Test description 1",
			"status": map[string]interface{}{
				"name": "To Do",
			},
			"issuetype": map[string]interface{}{
				"name": "Task",
			},
		},
	},
	{
		ID:  "2",
		Key: "TST-2",
		Fields: map[string]interface{}{
			"summary":     "Test issue 2",
			"description": "Test description 2",
			"status": map[string]interface{}{
				"name": "In Progress",
			},
			"issuetype": map[string]interface{}{
				"name": "Task",
			},
		},
	},
	{
		ID:  "3",
		Key: "TST-3",
		Fields: map[string]interface{}{
			"summary":     "Test issue 3",
			"description": "Test description 3",
			"status": map[string]interface{}{
				"name": "Done",
			},
			"issuetype": map[string]interface{}{
				"name": "Task",
			},
		},
	},
	{
		ID:  "4",
		Key: "KAN-4",
		Fields: map[string]interface{}{
			"summary":     "Test issue 4",
			"description": "Test description 4",
			"status": map[string]interface{}{
				"name": "Done",
			},
			"issuetype": map[string]interface{}{
				"name": "Epic",
			},
		},
	},
	{
		ID:  "5",
		Key: "KAN-5",
		Fields: map[string]interface{}{
			"summary":     "Test issue 5",
			"description": "Test description 5",
			"status": map[string]interface{}{
				"name": "Done",
			},
			"issuetype": map[string]interface{}{
				"name": "Task",
			},
		},
	},
}

type FakeSprint struct {
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

func (f *FakeSprint) getSelf() string {
	if f.Self == "" {
		f.Self = fmt.Sprintf("https://test.atlassian.net/rest/agile/1.0/sprint/%d", f.ID)
	}
	return f.Self
}

var fakeSprints = []FakeSprint{
	{
		ID:            1,
		State:         "active",
		Name:          "Sprint 1",
		StartDate:     "2021-01-01T00:00:00.000Z",
		EndDate:       "2021-01-15T00:00:00.000Z",
		CompleteDate:  "2021-01-15T00:00:00.000Z",
		OriginBoardID: 1,
		Goal:          "Sprint goal",
	},
}
