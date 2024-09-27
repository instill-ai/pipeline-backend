package asana

import (
	"testing"
)

func TestPortfolio(t *testing.T) {
	// to avoid data race in tests
	testGetPortfolio(t)
	testUpdatePortfolio(t)
	testCreatePortfolio(t)
	testDeletePortfolio(t)
}

func testGetPortfolio(t *testing.T) {
	testcases := []taskCase[GetPortfolioInput, PortfolioTaskOutput]{
		{
			_type: "ok",
			name:  "Get portfolio",
			input: GetPortfolioInput{
				Action: "get",
				ID:     "1234",
			},
			wantResp: PortfolioTaskOutput{
				Portfolio: Portfolio{
					GID:                 "1234",
					Name:                "Test Portfolio",
					Owner:               User{GID: "123", Name: "Admin User"},
					DueOn:               "2021-01-01",
					StartOn:             "2021-01-01",
					Color:               "red",
					Public:              true,
					CreatedBy:           User{GID: "123", Name: "Admin User"},
					CurrentStatus:       []map[string]interface{}{{"title": "On track"}},
					CustomFields:        []map[string]interface{}{{"field": "value"}},
					CustomFieldSettings: []map[string]interface{}{{"field": "value"}},
				},
			},
		},
		{
			_type: "nok",
			name:  "Get portfolio - 404 Not Found",
			input: GetPortfolioInput{
				Action: "get",
				ID:     "12345",
			},
			wantErr: `unsuccessful HTTP response.*`,
		},
	}
	taskTesting(testcases, TaskAsanaPortfolio, t)
}
func testUpdatePortfolio(t *testing.T) {
	testcases := []taskCase[UpdatePortfolioInput, PortfolioTaskOutput]{
		{
			_type: "ok",
			name:  "Update portfolio",
			input: UpdatePortfolioInput{
				Action: "update",
				ID:     "1234",
				Public: false,
				Color:  "blue",
			},
			wantResp: PortfolioTaskOutput{
				Portfolio: Portfolio{
					GID:                 "1234",
					Name:                "Test Portfolio",
					Owner:               User{GID: "123", Name: "Admin User"},
					DueOn:               "2021-01-01",
					StartOn:             "2021-01-01",
					Color:               "blue",
					Public:              false,
					CreatedBy:           User{GID: "123", Name: "Admin User"},
					CurrentStatus:       []map[string]interface{}{{"title": "On track"}},
					CustomFields:        []map[string]interface{}{{"field": "value"}},
					CustomFieldSettings: []map[string]interface{}{{"field": "value"}},
				},
			},
		},
		{
			_type: "nok",
			name:  "Update portfolio - 404 Not Found",
			input: UpdatePortfolioInput{
				Action: "update",
				ID:     "12345",
			},
			wantErr: `unsuccessful HTTP response.*`,
		},
	}
	taskTesting(testcases, TaskAsanaPortfolio, t)
}
func testCreatePortfolio(t *testing.T) {
	testcases := []taskCase[CreatePortfolioInput, PortfolioTaskOutput]{
		{
			_type: "ok",
			name:  "Create portfolio",
			input: CreatePortfolioInput{
				Action:    "create",
				Name:      "Test Portfolio",
				Color:     "blue",
				Public:    true,
				Workspace: "123",
			},
			wantResp: PortfolioTaskOutput{
				Portfolio: Portfolio{
					GID:                 "123456789",
					Name:                "Test Portfolio",
					Owner:               User{GID: "123", Name: "Admin User"},
					DueOn:               "2021-01-01",
					StartOn:             "2021-01-01",
					Color:               "blue",
					Public:              true,
					CreatedBy:           User{GID: "123", Name: "Admin User"},
					CurrentStatus:       []map[string]interface{}{{"title": "On track"}},
					CustomFields:        []map[string]interface{}{{"field": "value"}},
					CustomFieldSettings: []map[string]interface{}{{"field": "value"}},
				},
			},
		},
		{
			_type: "nok",
			name:  "Create portfolio - 400 Bad Request",
			input: CreatePortfolioInput{
				Action: "create",
			},
			wantErr: `unsuccessful HTTP response.*`,
		},
	}
	taskTesting(testcases, TaskAsanaPortfolio, t)
}

func testDeletePortfolio(t *testing.T) {
	testcases := []taskCase[DeletePortfolioInput, PortfolioTaskOutput]{
		{
			_type: "ok",
			name:  "Delete portfolio",
			input: DeletePortfolioInput{
				Action: "delete",
				ID:     "1234567890",
			},
			wantResp: PortfolioTaskOutput{
				Portfolio: Portfolio{
					CurrentStatus:       []map[string]interface{}{},
					CustomFields:        []map[string]interface{}{},
					CustomFieldSettings: []map[string]interface{}{},
				},
			},
		},
		{
			_type: "nok",
			name:  "Delete portfolio - 404 Not Found",
			input: DeletePortfolioInput{
				Action: "delete",
				ID:     "12345",
			},
			wantErr: `unsuccessful HTTP response.*`,
		},
	}
	taskTesting(testcases, TaskAsanaPortfolio, t)
}
