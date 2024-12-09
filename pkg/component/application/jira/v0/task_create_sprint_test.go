package jira

import "testing"

func TestComponent_CreateSprintTask(t *testing.T) {
	testCases := []TaskCase[createSprintInput, createSprintOutput]{
		{
			_type: "ok",
			name:  "create sprint",
			input: createSprintInput{
				Name:      "Test Sprint",
				Goal:      "Sprint goal",
				StartDate: "2021-01-01T00:00:00.000Z",
				EndDate:   "2021-01-15T00:00:00.000Z",
				BoardName: "TST",
			},
			wantOutput: createSprintOutput{
				ID:            2,
				Self:          "https://test.atlassian.net/rest/agile/1.0/sprint/1",
				State:         "active",
				Name:          "Test Sprint",
				StartDate:     "2021-01-01T00:00:00.000Z",
				EndDate:       "2021-01-15T00:00:00.000Z",
				CompleteDate:  "",
				OriginBoardID: 3,
				Goal:          "Sprint goal",
			},
		},
		{
			_type: "nok",
			name:  "400 - Bad Request",
			input: createSprintInput{
				Name:      "Test Sprint",
				BoardName: "INVALID",
			},
			wantErr: "end date is required",
		},
		{
			_type: "nok",
			name:  "400 - Bad Request",
			input: createSprintInput{
				Name:      "Test Sprint",
				BoardName: "INVALID",
				EndDate:   "2021-01-15T00:00:00.000Z",
			},
			wantErr: "board not found",
		},
	}
	taskTesting(testCases, taskCreateSprint, t)
}
