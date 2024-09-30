package jira

import (
	"encoding/json"
	"net/http"
)

type mockCreateSprintRequest struct {
	Name          string `json:"name"`
	Goal          string `json:"goal"`
	StartDate     string `json:"startDate"`
	EndDate       string `json:"endDate"`
	OriginBoardID int    `json:"originBoardId"`
}

func mockCreateSprint(res http.ResponseWriter, req *http.Request) {
	var err error

	if req.Method != http.MethodPost {
		http.Error(res, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	body := mockCreateSprintRequest{}
	err = json.NewDecoder(req.Body).Decode(&body)
	if err != nil {
		http.Error(res, "Bad Request", http.StatusBadRequest)
		return
	}
	var newSprint = FakeSprint{
		ID:            1,
		Self:          "https://test.atlassian.net/rest/agile/1.0/sprint/1",
		State:         "active",
		Name:          body.Name,
		StartDate:     body.StartDate,
		EndDate:       body.EndDate,
		CompleteDate:  "",
		OriginBoardID: body.OriginBoardID,
		Goal:          body.Goal,
	}
	fakeSprints = append(fakeSprints, newSprint)

	res.WriteHeader(http.StatusCreated)
	err = json.NewEncoder(res).Encode(newSprint)
	if err != nil {
		http.Error(res, "Bad Request", http.StatusBadRequest)
		return
	}
}
