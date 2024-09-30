package asana

import (
	"testing"
)

func TestGoal(t *testing.T) {
	// to avoid data race in tests
	testGetGoal(t)
	testUpdateGoal(t)
	testCreateGoal(t)
	testDeleteGoal(t)
}
func testGetGoal(t *testing.T) {
	testcases := []taskCase[GetGoalInput, GoalTaskOutput]{
		{
			_type: "ok",
			name:  "Get goal",
			input: GetGoalInput{
				Action: "get",
				ID:     "1234",
			},
			wantResp: GoalTaskOutput{
				Goal: Goal{
					GID:       "1234",
					Name:      "Test Goal",
					Owner:     User{GID: "123", Name: "Admin User"},
					Notes:     "Test Notes",
					HTMLNotes: "Test HTML Notes",
					DueOn:     "2021-01-01",
					StartOn:   "2021-01-01",
					Liked:     true,
					Likes: []Like{
						{
							LikeGID:  "123",
							UserGID:  "123",
							UserName: "Admin User",
						},
					},
				},
			},
		},
		{
			_type: "nok",
			name:  "Get goal - 404 Not Found",
			input: GetGoalInput{
				Action: "get",
				ID:     "12345",
			},
			wantErr: `unsuccessful HTTP response.*`,
		},
	}
	taskTesting(testcases, TaskAsanaGoal, t)
}
func testUpdateGoal(t *testing.T) {
	testcases := []taskCase[UpdateGoalInput, GoalTaskOutput]{
		{
			_type: "ok",
			name:  "Update goal",
			input: UpdateGoalInput{
				Action: "update",
				ID:     "1234",
				Notes:  "Modified Notes",
				DueOn:  "2021-01-02",
				Liked:  true,
			},
			wantResp: GoalTaskOutput{
				Goal: Goal{
					GID:       "1234",
					Name:      "Test Goal",
					Owner:     User{GID: "123", Name: "Admin User"},
					Notes:     "Modified Notes",
					HTMLNotes: "Test HTML Notes",
					DueOn:     "2021-01-02",
					StartOn:   "2021-01-01",
					Liked:     true,
					Likes: []Like{
						{
							LikeGID:  "123",
							UserGID:  "123",
							UserName: "Admin User",
						},
					},
				},
			},
		},
		{
			_type: "nok",
			name:  "Update goal - 404 Not Found",
			input: UpdateGoalInput{
				Action: "update",
				ID:     "12345",
			},
			wantErr: `unsuccessful HTTP response.*`,
		},
	}
	taskTesting(testcases, TaskAsanaGoal, t)
}
func testCreateGoal(t *testing.T) {
	testcases := []taskCase[CreateGoalInput, GoalTaskOutput]{
		{
			_type: "ok",
			name:  "Create goal",
			input: CreateGoalInput{
				Action:     "create",
				Name:       "Test Goal",
				Notes:      "Modified Notes",
				DueOn:      "2021-01-02",
				StartOn:    "2021-01-01",
				Liked:      true,
				Workspace:  "1308068054504137",
				TimePeriod: "1308068168028231",
				Owner:      "1308068131615343",
			},
			wantResp: GoalTaskOutput{
				Goal: Goal{
					GID:       "123456789",
					Name:      "Test Goal",
					Owner:     User{GID: "123", Name: "Admin User"},
					Notes:     "Modified Notes",
					HTMLNotes: "Test HTML Notes",
					DueOn:     "2021-01-02",
					StartOn:   "2021-01-01",
					Liked:     true,
					Likes: []Like{
						{
							LikeGID:  "123",
							UserGID:  "123",
							UserName: "Admin User",
						},
					},
				},
			},
		},
		{
			_type: "nok",
			name:  "Create goal - 400 Bad Request",
			input: CreateGoalInput{
				Action: "create",
			},
			wantErr: `unsuccessful HTTP response.*`,
		},
	}
	taskTesting(testcases, TaskAsanaGoal, t)
}

func testDeleteGoal(t *testing.T) {
	testcases := []taskCase[DeleteGoalInput, GoalTaskOutput]{
		{
			_type: "ok",
			name:  "Delete goal",
			input: DeleteGoalInput{
				Action: "delete",
				ID:     "1234567890",
			},
			wantResp: GoalTaskOutput{
				Goal: Goal{
					Likes: []Like{},
				},
			},
		},
		{
			_type: "nok",
			name:  "Delete goal - 404 Not Found",
			input: DeleteGoalInput{
				Action: "delete",
				ID:     "12345",
			},
			wantErr: `unsuccessful HTTP response.*`,
		},
	}
	taskTesting(testcases, TaskAsanaGoal, t)
}
