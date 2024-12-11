package jira

import (
	"strings"
	"testing"
)

func TestComponent_ListIssuesTask(t *testing.T) {
	testCases := []TaskCase[listIssuesInput, listIssuesOutput]{
		{
			_type: "ok",
			name:  "All",
			input: listIssuesInput{
				BoardName:  "KAN",
				MaxResults: 10,
				StartAt:    0,
				RangeData: issueRange{
					Range: "All",
				},
			},
			wantOutput: listIssuesOutput{
				Total:      2,
				StartAt:    0,
				MaxResults: 10,
				Issues: []Issue{
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
						IssueType:   "Epic",
						Self:        "https://test.atlassian.net/rest/agile/1.0/issue/4",
						Description: "Test description 4",
						Status:      "Done",
						Summary:     "Test issue 4",
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
						IssueType:   "Task",
						Self:        "https://test.atlassian.net/rest/agile/1.0/issue/5",
						Description: "Test description 5",
						Status:      "Done",
						Summary:     "Test issue 5",
					},
				},
			},
		},
		{
			_type: "ok",
			name:  "Epics only",
			input: listIssuesInput{
				BoardName:  "KAN",
				MaxResults: 10,
				StartAt:    0,
				RangeData: issueRange{
					Range: "Epics only",
				},
			},
			wantOutput: listIssuesOutput{
				Total:      2,
				StartAt:    0,
				MaxResults: 10,
				Issues: []Issue{
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
						IssueType:   "Epic",
						Self:        "https://test.atlassian.net/rest/agile/1.0/issue/4",
						Description: "Test description 4",
						Status:      "Done",
						Summary:     "Test issue 4",
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
						IssueType:   "Epic",
						Self:        "https://test.atlassian.net/rest/agile/1.0/issue/5",
						Description: "Test description 5",
						Status:      "Done",
						Summary:     "Test issue 5",
					},
				},
			},
		},
		{
			_type: "ok",
			name:  "In backlog only",
			input: listIssuesInput{
				BoardName:  "KAN",
				MaxResults: 10,
				StartAt:    0,
				RangeData: issueRange{
					Range: "In backlog only",
				},
			},
			wantOutput: listIssuesOutput{
				Total:      2,
				StartAt:    0,
				MaxResults: 10,
				Issues: []Issue{
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
						IssueType:   "Epic",
						Self:        "https://test.atlassian.net/rest/agile/1.0/issue/4",
						Description: "Test description 4",
						Status:      "Done",
						Summary:     "Test issue 4",
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
						IssueType:   "Task",
						Self:        "https://test.atlassian.net/rest/agile/1.0/issue/5",
						Description: "Test description 5",
						Status:      "Done",
						Summary:     "Test issue 5",
					},
				},
			},
		},
		{
			_type: "ok",
			name:  "Issues without epic assigned",
			input: listIssuesInput{
				BoardName:  "KAN",
				MaxResults: 10,
				StartAt:    0,
				RangeData: issueRange{
					Range: "Issues without epic assigned",
				},
			},
			wantOutput: listIssuesOutput{
				Total:      2,
				StartAt:    0,
				MaxResults: 10,
				Issues: []Issue{
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
						IssueType:   "Epic",
						Self:        "https://test.atlassian.net/rest/agile/1.0/issue/4",
						Description: "Test description 4",
						Status:      "Done",
						Summary:     "Test issue 4",
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
						IssueType:   "Task",
						Self:        "https://test.atlassian.net/rest/agile/1.0/issue/5",
						Description: "Test description 5",
						Status:      "Done",
						Summary:     "Test issue 5",
					},
				},
			},
		},
		{
			_type: "ok",
			name:  "Issues of an epic",
			input: listIssuesInput{
				BoardName:  "KAN",
				MaxResults: 10,
				StartAt:    0,
				RangeData: issueRange{
					Range:   "Issues of an epic",
					EpicKey: "KAN-4",
				},
			},
			wantOutput: listIssuesOutput{
				Total:      0,
				StartAt:    0,
				MaxResults: 10,
				Issues:     []Issue{},
			},
		},
		{
			_type: "ok",
			name:  "Issues of an epic(long query)",
			input: listIssuesInput{
				BoardName:  "KAN",
				MaxResults: 10,
				StartAt:    0,
				RangeData: issueRange{
					Range:   "Issues of an epic",
					EpicKey: "KAN-4" + strings.Repeat("-0", 100),
				},
			},
			wantOutput: listIssuesOutput{
				Total:      0,
				StartAt:    0,
				MaxResults: 10,
				Issues:     []Issue{},
			},
		},
		{
			_type: "ok",
			name:  "Issues of a sprint",
			input: listIssuesInput{
				BoardName:  "KAN",
				MaxResults: 10,
				StartAt:    0,
				RangeData: issueRange{
					Range:      "Issues of a sprint",
					SprintName: "KAN Sprint 1",
				},
			},
			wantOutput: listIssuesOutput{
				Total:      0,
				StartAt:    0,
				MaxResults: 10,
				Issues:     []Issue{},
			},
		},
		{
			_type: "ok",
			name:  "Standard Issues",
			input: listIssuesInput{
				BoardName:  "TST",
				MaxResults: 10,
				StartAt:    0,
				RangeData: issueRange{
					Range: "Standard Issues",
				},
			},
			wantOutput: listIssuesOutput{
				Total:      0,
				StartAt:    0,
				MaxResults: 10,
				Issues:     []Issue{},
			},
		},
		{
			_type: "ok",
			name:  "JQL",
			input: listIssuesInput{
				BoardName:  "TST",
				MaxResults: 10,
				StartAt:    0,
				RangeData: issueRange{
					Range: "JQL query",
					JQL:   "project = TST",
				},
			},
			wantOutput: listIssuesOutput{
				Total:      0,
				StartAt:    0,
				MaxResults: 10,
				Issues:     []Issue{},
			},
		},
		{
			_type: "nok",
			name:  "invalid range",
			input: listIssuesInput{
				BoardName:  "TST",
				MaxResults: 10,
				StartAt:    0,
				RangeData: issueRange{
					Range: "invalid",
				},
			},
			wantErr: "invalid range",
		},
	}
	taskTesting(testCases, taskListIssues, t)
}
