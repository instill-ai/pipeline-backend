package jira

import "testing"

func TestComponent_ListSprintsTask(t *testing.T) {
	testCases := []TaskCase[listSprintsInput, listSprintsOutput]{
		{
			_type: "ok",
			name:  "list all sprints",
			input: listSprintsInput{
				BoardID:    1,
				StartAt:    0,
				MaxResults: 10,
			},
			wantOutput: listSprintsOutput{
				Total:      1,
				StartAt:    0,
				MaxResults: 10,
				Sprints: []*getSprintOutput{
					{
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
			},
		},
		{
			_type: "nok",
			name:  "400 - Bad Request",
			input: listSprintsInput{
				BoardID:    -1,
				StartAt:    0,
				MaxResults: 10,
			},
			wantErr: "getting sprints: unsuccessful HTTP response",
		},
	}
	taskTesting(testCases, taskListSprints, t)
}
