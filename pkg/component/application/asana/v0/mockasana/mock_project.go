package mockasana

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func getProject(res http.ResponseWriter, req *http.Request) {
	var err error
	opt := req.URL.Query()
	optFields := opt.Get("opt_fields")
	projectGID := chi.URLParam(req, "projectGID")

	if optFields == "" {
		http.Error(res, "optFields is not expected to be null", http.StatusBadRequest)
		return
	}
	if projectGID == "" {
		http.Error(res, "projectGID is required", http.StatusBadRequest)
		return
	}

	for _, project := range FakeProject {
		if project.GID == projectGID {
			if err = json.NewEncoder(res).Encode(
				map[string]interface{}{
					"data": project,
				},
			); err != nil {
				http.Error(res, err.Error(), http.StatusInternalServerError)
				return
			}
			return
		}
	}
	http.Error(res, "project not found", http.StatusNotFound)
}

func updateProject(res http.ResponseWriter, req *http.Request) {
	var err error
	opt := req.URL.Query()
	optFields := opt.Get("opt_fields")
	if optFields == "" {
		http.Error(res, "optFields is not expected to be null", http.StatusBadRequest)
		return
	}
	projectGID := chi.URLParam(req, "projectGID")
	if projectGID == "" {
		http.Error(res, "projectGID is required", http.StatusBadRequest)
		return
	}

	var project map[string]RawProject

	if err = json.NewDecoder(req.Body).Decode(&project); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}
	for i, v := range FakeProject {
		if v.GID == projectGID {
			updateProject := project["data"]
			if updateProject.Name != "" {
				FakeProject[i].Name = updateProject.Name
			}
			if updateProject.Notes != "" {
				FakeProject[i].Notes = updateProject.Notes
			}
			if updateProject.DueOn != "" {
				FakeProject[i].DueOn = updateProject.DueOn
			}
			if updateProject.StartOn != "" {
				FakeProject[i].StartOn = updateProject.StartOn
			}
			if updateProject.Color != "" {
				FakeProject[i].Color = updateProject.Color
			}
			if updateProject.PrivacySetting != "" {
				FakeProject[i].PrivacySetting = updateProject.PrivacySetting
			}
			FakeProject[i].Archived = updateProject.Archived

			if err = json.NewEncoder(res).Encode(
				map[string]interface{}{
					"data": FakeProject[i],
				},
			); err != nil {
				http.Error(res, err.Error(), http.StatusInternalServerError)
				return
			}
			return
		}
	}
	http.Error(res, "project not found", http.StatusNotFound)
}

func createProject(res http.ResponseWriter, req *http.Request) {
	var err error
	opt := req.URL.Query()
	optFields := opt.Get("opt_fields")
	if optFields == "" {
		http.Error(res, "optFields is not expected to be null", http.StatusBadRequest)
		return
	}

	var project map[string]RawProject

	if err = json.NewDecoder(req.Body).Decode(&project); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}
	newID := "123456789"
	newProject := project["data"]
	if newProject.Name == "" {
		http.Error(res, "Name is required", http.StatusBadRequest)
		return
	}
	newProject.GID = newID
	newProject.Owner = FakeUser[0]
	newProject.HTMLNotes = "Test HTML Notes"
	newProject.Completed = false
	FakeProject = append(FakeProject, newProject)
	newProject.CompletedBy = FakeUser[0]
	newProject.CurrentStatus = map[string]string{
		"status": "on_track",
	}
	newProject.CustomFields = map[string]string{
		"field": "value",
	}
	newProject.CustomFieldSettings = map[string]string{
		"field": "value",
	}
	if err = json.NewEncoder(res).Encode(
		map[string]interface{}{
			"data": newProject,
		},
	); err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
}

func deleteProject(res http.ResponseWriter, req *http.Request) {
	var err error
	projectGID := chi.URLParam(req, "projectGID")
	if projectGID == "" {
		http.Error(res, "projectGID is required", http.StatusBadRequest)
		return
	}

	for i, v := range FakeProject {
		if v.GID == projectGID {
			FakeProject = append(FakeProject[:i], FakeProject[i+1:]...)
			if err = json.NewEncoder(res).Encode(
				map[string]interface{}{
					"data": RawProject{},
				},
			); err != nil {
				http.Error(res, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}

	http.Error(res, "project not found", http.StatusNotFound)
}

type duplicateProjectReqBody struct {
	Name          string        `json:"name"`
	Team          string        `json:"team"`
	Include       string        `json:"include"`
	ScheduleDates ScheduleDates `json:"schedule_dates"`
}

func duplicateProject(res http.ResponseWriter, req *http.Request) {
	var err error
	opt := req.URL.Query()
	optFields := opt.Get("opt_fields")
	if optFields == "" {
		http.Error(res, "optFields is not expected to be null", http.StatusBadRequest)
		return
	}

	projectGID := chi.URLParam(req, "projectGID")
	if projectGID == "" {
		http.Error(res, "projectGID is required", http.StatusBadRequest)
		return
	}
	var reqBody map[string]duplicateProjectReqBody
	if err = json.NewDecoder(req.Body).Decode(&reqBody); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}
	body := reqBody["data"]
	newID := "4321"

	if body.Include != "allocations,forms,members,notes,task_assignee,task_attachments,task_dates,task_dependencies,task_followers,task_notes,task_projects,task_subtasks,task_tags" {
		http.Error(res, "Include Not correct", http.StatusBadRequest)
		return
	}

	for _, v := range FakeProject {
		if v.GID == projectGID {
			newProject := v
			newProject.GID = newID
			newProject.Name = body.Name
			FakeProject = append(FakeProject, newProject)
			if err = json.NewEncoder(res).Encode(
				map[string]interface{}{
					"data": newProject,
				},
			); err != nil {
				http.Error(res, err.Error(), http.StatusInternalServerError)
				return
			}
			return
		}
	}
	http.Error(res, "project not found", http.StatusNotFound)
}
