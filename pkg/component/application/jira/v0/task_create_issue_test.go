package jira

import "testing"

func TestComponent_CreateIssueTask(t *testing.T) {
	testCases := []TaskCase[createIssueInput, createIssueOutput]{
		{
			_type: "ok",
			name:  "create issue",
			input: createIssueInput{
				ProjectKey: "CRI",
				IssueType: issueType{
					IssueType: "Task",
				},
				Summary:     "Test issue 1",
				Description: "Test description 1",
			},
			wantOutput: createIssueOutput{
				Issue{
					ID:  "30000",
					Key: "CRI-1",
					Fields: map[string]interface{}{
						"summary":     "Test issue 1",
						"description": "Test description 1",
						"issuetype": map[string]interface{}{
							"name": "Task",
						},
						"project": map[string]interface{}{
							"key": "CRI",
						},
					},
					Self:        "https://test.atlassian.net/rest/agile/1.0/issue/30000",
					Summary:     "Test issue 1",
					Description: "Test description 1",
					IssueType:   "Task",
				},
			},
		},
		{
			_type: "nok",
			name:  "400 - Bad Request",
			input: createIssueInput{
				ProjectKey: "INVALID",
			},
			wantErr: "creating issue: unsuccessful HTTP response",
		},
	}
	taskTesting(testCases, taskCreateIssue, t)
}
