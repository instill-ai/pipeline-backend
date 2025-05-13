package jira

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type mockUpdateSprintReq struct {
	Name          string `json:"name,omitempty"`
	State         string `json:"state,omitempty"`
	OriginBoardID int    `json:"originBoardId,omitempty"`
	Goal          string `json:"goal,omitempty"`
	StartDate     string `json:"startDate,omitempty"`
	EndDate       string `json:"endDate,omitempty"`
	CompleteDate  string `json:"completeDate,omitempty"`
}
type mockUpdateSprintResp struct {
	fakeSprint
}

// UpdateSprint updates an issue in Jira.
func mockUpdateSprint(res http.ResponseWriter, req *http.Request) {
	var updateSprintReq mockUpdateSprintReq

	err := json.NewDecoder(req.Body).Decode(&updateSprintReq)
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
		Name:          updateSprintReq.Name,
		State:         updateSprintReq.State,
		OriginBoardID: updateSprintReq.OriginBoardID,
		Goal:          updateSprintReq.Goal,
		StartDate:     updateSprintReq.StartDate,
		EndDate:       updateSprintReq.EndDate,
		CompleteDate:  updateSprintReq.CompleteDate,
	}
	var resp mockUpdateSprintResp
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
			resp = mockUpdateSprintResp{
				fakeSprint: fakeSprint{
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
			resp.getSelf()
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
