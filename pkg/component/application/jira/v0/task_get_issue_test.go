package jira

import "testing"

func TestComponent_GetIssueTask(t *testing.T) {
	testCases := []TaskCase[getIssueInput, getIssueOutput]{
		{
			_type: "ok",
			name:  "get issue-Task",
			input: getIssueInput{
				IssueKey:      "TST-1",
				UpdateHistory: true,
			},
			wantOutput: getIssueOutput{
				Issue: Issue{
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
					Self:        "https://test.atlassian.net/rest/agile/1.0/issue/1",
					Summary:     "Test issue 1",
					Status:      "To Do",
					Description: "Test description 1",
					IssueType:   "Task",
				},
			},
		},
		{
			_type: "ok",
			name:  "get issue-Epic",
			input: getIssueInput{
				IssueKey:      "KAN-4",
				UpdateHistory: false,
			},
			wantOutput: getIssueOutput{
				Issue: Issue{
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
					Self:        "https://test.atlassian.net/rest/agile/1.0/issue/4",
					Summary:     "Test issue 4",
					Description: "Test description 4",
					Status:      "Done",
					IssueType:   "Epic",
				},
			},
		},
		{
			_type: "nok",
			name:  "404 - Not Found",
			input: getIssueInput{
				IssueKey:      "100",
				UpdateHistory: true,
			},
			wantErr: `unsuccessful HTTP response.*Jira-Client responded with a 404 status code.*Issue does not exist or you do not have permission to see it.*`,
		},
	}
	taskTesting(testCases, taskGetIssue, t)
}
