package jira

import "testing"

func TestComponent_GetSprintTask(t *testing.T) {
	testCases := []TaskCase[getSprintInput, getSprintOutput]{
		{
			_type: "ok",
			name:  "get sprint",
			input: getSprintInput{
				SprintID: 1,
			},
			wantOutput: getSprintOutput{
				ID:            1,
				Self:          "https://test.atlassian.net/rest/agile/1.0/sprint/1",
				State:         "active",
				Name:          "Sprint 1",
				StartDate:     "2021-01-01T00:00:00.000Z",
				EndDate:       "2021-01-15T00:00:00.000Z",
				CompleteDate:  "2021-01-15T00:00:00.000Z",
				OriginBoardID: 1,
				Goal:          "Sprint goal",
			},
		},
		{
			_type: "nok",
			name:  "400 - Bad Request",
			input: getSprintInput{
				SprintID: -1,
			},
			wantErr: "getting sprint: unsuccessful HTTP response",
		},
		{
			_type: "nok",
			name:  "404 - Not Found",
			input: getSprintInput{
				SprintID: 2,
			},
			wantErr: "getting sprint: unsuccessful HTTP response",
		},
	}
	taskTesting(testCases, taskGetSprint, t)
}
