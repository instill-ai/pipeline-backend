package jira

import "testing"

func TestComponent_UpdateSprintTask(t *testing.T) {
	testCases := []TaskCase[updateSprintInput, updateSprintOutput]{
		{
			_type: "ok",
			name:  "update sprint",
			input: updateSprintInput{
				SprintID:       1,
				Name:           "Test Sprint updated",
				Goal:           "Sprint goal updated",
				StartDate:      "2021-01-01T00:00:00.000Z",
				EndDate:        "2021-01-15T00:00:00.000Z",
				CurrentState:   "active",
				EnterNextState: false,
			},
			wantOutput: updateSprintOutput{
				ID:            1,
				Self:          "https://test.atlassian.net/rest/agile/1.0/sprint/1",
				State:         "active",
				Name:          "Test Sprint updated",
				StartDate:     "2021-01-01T00:00:00.000Z",
				EndDate:       "2021-01-15T00:00:00.000Z",
				CompleteDate:  "2021-01-15T00:00:00.000Z",
				OriginBoardID: 1,
				Goal:          "Sprint goal updated",
			},
		},
		{
			_type: "ok",
			name:  "future to active",
			input: updateSprintInput{
				SprintID:       1,
				Name:           "Test Sprint updated",
				Goal:           "Sprint goal updated",
				StartDate:      "2021-01-01T00:00:00.000Z",
				EndDate:        "2021-01-15T00:00:00.000Z",
				CurrentState:   "future",
				EnterNextState: true,
			},
			wantOutput: updateSprintOutput{
				ID:            1,
				Self:          "https://test.atlassian.net/rest/agile/1.0/sprint/1",
				State:         "active",
				Name:          "Test Sprint updated",
				StartDate:     "2021-01-01T00:00:00.000Z",
				EndDate:       "2021-01-15T00:00:00.000Z",
				CompleteDate:  "2021-01-15T00:00:00.000Z",
				OriginBoardID: 1,
				Goal:          "Sprint goal updated",
			},
		},
		{
			_type: "ok",
			name:  "active to closed",
			input: updateSprintInput{
				SprintID:       1,
				Name:           "Test Sprint updated",
				Goal:           "Sprint goal updated",
				StartDate:      "2021-01-01T00:00:00.000Z",
				EndDate:        "2021-01-15T00:00:00.000Z",
				CurrentState:   "active",
				EnterNextState: true,
			},
			wantOutput: updateSprintOutput{
				ID:            1,
				Self:          "https://test.atlassian.net/rest/agile/1.0/sprint/1",
				State:         "closed",
				Name:          "Test Sprint updated",
				StartDate:     "2021-01-01T00:00:00.000Z",
				EndDate:       "2021-01-15T00:00:00.000Z",
				CompleteDate:  "2021-01-15T00:00:00.000Z",
				OriginBoardID: 1,
				Goal:          "Sprint goal updated",
			},
		},
		{
			_type: "nok",
			name:  "400 - Bad Request",
			input: updateSprintInput{
				SprintID: -1,
			},
			wantErr: "end date is required",
		},
	}
	taskTesting(testCases, taskUpdateSprint, t)
}
