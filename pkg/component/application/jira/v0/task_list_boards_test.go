package jira

import "testing"

func TestComponent_ListBoardsTask(t *testing.T) {
	testCases := []TaskCase[listBoardsInput, listBoardsOutput]{
		{
			_type: "ok",
			name:  "list all boards",
			input: listBoardsInput{
				MaxResults: 10,
				StartAt:    0,
				BoardType:  "simple",
			},
			wantOutput: listBoardsOutput{
				Total:      1,
				StartAt:    0,
				MaxResults: 10,
				IsLast:     true,
				Boards: []Board{
					{
						ID:        3,
						Name:      "TST",
						BoardType: "simple",
						Self:      "https://test.atlassian.net/rest/agile/1.0/board/3",
					},
				},
			},
		},
		{
			_type: "ok",
			name:  "get filtered boards",
			input: listBoardsInput{
				MaxResults: 10,
				StartAt:    1,
				BoardType:  "kanban",
			},
			wantOutput: listBoardsOutput{
				Total:      1,
				StartAt:    1,
				MaxResults: 10,
				IsLast:     true,
				Boards:     []Board{},
			},
		},
		{
			_type: "nok",
			name:  "400 - Not Found",
			input: listBoardsInput{
				MaxResults:     10,
				StartAt:        1,
				ProjectKeyOrID: "test",
			},
			wantErr: "listing boards: unsuccessful HTTP response",
		},
	}
	taskTesting(testCases, taskListBoards, t)
}
