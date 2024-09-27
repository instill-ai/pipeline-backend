package mockasana

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func getGoal(res http.ResponseWriter, req *http.Request) {
	var err error
	opt := req.URL.Query()
	optFields := opt.Get("opt_fields")
	goalGID := chi.URLParam(req, "goalGID")

	if optFields == "" {
		http.Error(res, "optFields is not expected to be null", http.StatusBadRequest)
		return
	}
	if goalGID == "" {
		http.Error(res, "goalGID is required", http.StatusBadRequest)
		return
	}

	for _, goal := range FakeGoal {
		if goal.GID == goalGID {
			if err = json.NewEncoder(res).Encode(
				map[string]interface{}{
					"data": goal,
				},
			); err != nil {
				http.Error(res, err.Error(), http.StatusInternalServerError)
				return
			}
			return
		}
	}
	http.Error(res, "goal not found", http.StatusNotFound)
}

func updateGoal(res http.ResponseWriter, req *http.Request) {
	var err error
	opt := req.URL.Query()
	optFields := opt.Get("opt_fields")
	if optFields == "" {
		http.Error(res, "optFields is not expected to be null", http.StatusBadRequest)
		return
	}
	goalGID := chi.URLParam(req, "goalGID")
	if goalGID == "" {
		http.Error(res, "goalGID is required", http.StatusBadRequest)
		return
	}

	var goal map[string]RawGoal

	if err = json.NewDecoder(req.Body).Decode(&goal); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}
	for i, v := range FakeGoal {
		if v.GID == goalGID {
			updateGoal := goal["data"]
			if updateGoal.Name != "" {
				FakeGoal[i].Name = updateGoal.Name
			}
			if updateGoal.Notes != "" {
				FakeGoal[i].Notes = updateGoal.Notes
			}
			if updateGoal.DueOn != "" {
				FakeGoal[i].DueOn = updateGoal.DueOn
			}
			if updateGoal.StartOn != "" {
				FakeGoal[i].StartOn = updateGoal.StartOn
			}
			FakeGoal[i].Liked = updateGoal.Liked

			if err = json.NewEncoder(res).Encode(
				map[string]interface{}{
					"data": FakeGoal[i],
				},
			); err != nil {
				http.Error(res, err.Error(), http.StatusInternalServerError)
				return
			}
			return
		}
	}
	http.Error(res, "goal not found", http.StatusNotFound)
}

func createGoal(res http.ResponseWriter, req *http.Request) {
	var err error
	opt := req.URL.Query()
	optFields := opt.Get("opt_fields")
	if optFields == "" {
		http.Error(res, "optFields is not expected to be null", http.StatusBadRequest)
		return
	}

	var goal map[string]RawGoal

	if err = json.NewDecoder(req.Body).Decode(&goal); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}
	newID := "123456789"
	newGoal := goal["data"]
	if newGoal.Name == "" {
		http.Error(res, "Name is required", http.StatusBadRequest)
		return
	}
	newGoal.GID = newID
	newGoal.Owner = FakeUser[0]
	newGoal.Likes = FakeLike
	newGoal.HTMLNotes = "Test HTML Notes"
	FakeGoal = append(FakeGoal, newGoal)

	if err = json.NewEncoder(res).Encode(
		map[string]interface{}{
			"data": newGoal,
		},
	); err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
}
func deleteGoal(res http.ResponseWriter, req *http.Request) {
	var err error
	goalGID := chi.URLParam(req, "goalGID")
	if goalGID == "" {
		http.Error(res, "goalGID is required", http.StatusBadRequest)
		return
	}

	for i, v := range FakeGoal {
		if v.GID == goalGID {
			FakeGoal = append(FakeGoal[:i], FakeGoal[i+1:]...)
			if err = json.NewEncoder(res).Encode(
				map[string]interface{}{
					"data": RawGoal{},
				},
			); err != nil {
				http.Error(res, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}

	http.Error(res, "goal not found", http.StatusNotFound)
}
