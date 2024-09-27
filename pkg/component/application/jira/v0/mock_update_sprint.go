package jira

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type MockUpdateSprintRequset struct {
	Name          string `json:"name,omitempty"`
	State         string `json:"state,omitempty"`
	OriginBoardID int    `json:"originBoardId,omitempty"`
	Goal          string `json:"goal,omitempty"`
	StartDate     string `json:"startDate,omitempty"`
	EndDate       string `json:"endDate,omitempty"`
	CompleteDate  string `json:"completeDate,omitempty"`
}
type MockUpdateSprintResp struct {
	FakeSprint
}

// UpdateSprint updates an issue in Jira.
func mockUpdateSprint(res http.ResponseWriter, req *http.Request) {
	var request MockUpdateSprintRequset

	err := json.NewDecoder(req.Body).Decode(&request)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}
	sprintID, err := strconv.Atoi(chi.URLParam(req, "sprintId"))
	if err != nil {
		http.Error(res, "sprint id is required", http.StatusBadRequest)
		return
	}

	mockSprint := Sprint{
		ID:            sprintID,
		Name:          request.Name,
		State:         request.State,
		OriginBoardID: request.OriginBoardID,
		Goal:          request.Goal,
		StartDate:     request.StartDate,
		EndDate:       request.EndDate,
		CompleteDate:  request.CompleteDate,
	}
	var resp MockUpdateSprintResp
	for i, s := range fakeSprints {
		if s.ID == mockSprint.ID {
			if mockSprint.Name != "" {
				fakeSprints[i].Name = mockSprint.Name
			}
			if mockSprint.Goal != "" {
				fakeSprints[i].Goal = mockSprint.Goal
			}
			if mockSprint.StartDate != "" {
				fakeSprints[i].StartDate = mockSprint.StartDate
			}
			if mockSprint.EndDate != "" {
				fakeSprints[i].EndDate = mockSprint.EndDate
			}
			if mockSprint.CompleteDate != "" {
				fakeSprints[i].CompleteDate = mockSprint.CompleteDate
			}
			if mockSprint.State != "" {
				fakeSprints[i].State = mockSprint.State
			}
			if mockSprint.OriginBoardID != 0 {
				fakeSprints[i].OriginBoardID = mockSprint.OriginBoardID
			}
			resp = MockUpdateSprintResp{
				FakeSprint: FakeSprint{
					ID:            fakeSprints[i].ID,
					Self:          fakeSprints[i].Self,
					Name:          fakeSprints[i].Name,
					State:         fakeSprints[i].State,
					OriginBoardID: fakeSprints[i].OriginBoardID,
					Goal:          fakeSprints[i].Goal,
					StartDate:     fakeSprints[i].StartDate,
					EndDate:       fakeSprints[i].EndDate,
					CompleteDate:  fakeSprints[i].CompleteDate,
				},
			}
			resp.FakeSprint.getSelf()
			break
		}
	}

	if resp.ID == 0 {
		http.Error(res, "sprint not found", http.StatusNotFound)
		return
	}
	err = json.NewEncoder(res).Encode(resp)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
}
