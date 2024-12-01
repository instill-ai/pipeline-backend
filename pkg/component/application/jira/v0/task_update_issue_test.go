package jira

import "testing"

func TestComponent_UpdateIssueTask(t *testing.T) {
	testCases := []TaskCase[updateIssueInput, updateIssueOutput]{
		{
			_type: "ok",
			name:  "update issue",
			input: updateIssueInput{
				IssueKey: "TST-1",
				Update: update{
					UpdateType: "Custom Update",
					UpdateFields: []updateField{
						{
							FieldName: "summary",
							Action:    "set",
							Value:     "Test issue 1 updated",
						},
						{
							FieldName: "description",
							Action:    "set",
							Value:     "Test description 1 updated",
						},
					},
				},
			},
			wantOutput: updateIssueOutput{
				Issue{
					ID:  "1",
					Key: "TST-1",
					Fields: map[string]interface{}{
						"summary":     "Test issue 1 updated",
						"description": "Test description 1 updated",
						"status": map[string]interface{}{
							"name": "To Do",
						},
						"issuetype": map[string]interface{}{
							"name": "Task",
						},
					},
					Self:        "https://test.atlassian.net/rest/agile/1.0/issue/1",
					Summary:     "Test issue 1 updated",
					Status:      "To Do",
					Description: "Test description 1 updated",
					IssueType:   "Task",
				},
			},
		},
		{
			_type: "ok",
			name:  "move issue to epic",
			input: updateIssueInput{
				IssueKey: "KAN-5",
				Update: update{
					UpdateType: "Move Issue to Epic",
					EpicKey:    "KAN-4",
				},
			},
			wantOutput: updateIssueOutput{
				Issue{
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
						"parent": map[string]interface{}{
							"key": "KAN-4",
						},
					},
					Self:        "https://test.atlassian.net/rest/agile/1.0/issue/5",
					Summary:     "Test issue 5",
					Status:      "Done",
					Description: "Test description 5",
					IssueType:   "Task",
				},
			},
		},
		{
			_type: "nok",
			name:  "400 - Bad Request",
			input: updateIssueInput{
				IssueKey: "INVALID",
				Update: update{
					UpdateType:   "Custom Update",
					UpdateFields: []updateField{},
				},
			},
			wantErr: "updating issue: unsuccessful HTTP response",
		},
	}
	taskTesting(testCases, taskUpdateIssue, t)
}
