package mockasana

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func getTask(res http.ResponseWriter, req *http.Request) {
	var err error
	opt := req.URL.Query()
	optFields := opt.Get("opt_fields")
	taskGID := chi.URLParam(req, "taskGID")

	if optFields == "" {
		http.Error(res, "optFields is not expected to be null", http.StatusBadRequest)
		return
	}
	if taskGID == "" {
		http.Error(res, "taskGID is required", http.StatusBadRequest)
		return
	}

	for _, task := range FakeTask {
		if task.GID == taskGID {
			if err = json.NewEncoder(res).Encode(
				map[string]interface{}{
					"data": task,
				},
			); err != nil {
				http.Error(res, err.Error(), http.StatusInternalServerError)
				return
			}
			return
		}
	}
	http.Error(res, "task not found", http.StatusNotFound)
}

type UpdateTaskReqBody struct {
	Data struct {
		Name            string `json:"name"`
		ResourceSubtype string `json:"resource_subtype"`
		ApprovalStatus  string `json:"approval_status"`
		DueOn           string `json:"due_on"`
		Completed       bool   `json:"completed"`
		Liked           bool   `json:"liked"`
		Notes           string `json:"notes"`
		StartOn         string `json:"start_on"`
		Assignee        string `json:"assignee"`
		Parent          string `json:"parent"`
	} `json:"data"`
}

func updateTask(res http.ResponseWriter, req *http.Request) {
	var err error
	opt := req.URL.Query()
	optFields := opt.Get("opt_fields")
	if optFields == "" {
		http.Error(res, "optFields is not expected to be null", http.StatusBadRequest)
		return
	}
	taskGID := chi.URLParam(req, "taskGID")
	if taskGID == "" {
		http.Error(res, "taskGID is required", http.StatusBadRequest)
		return
	}

	var task UpdateTaskReqBody
	if err = json.NewDecoder(req.Body).Decode(&task); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}
	for i, v := range FakeTask {
		if v.GID == taskGID {
			updateTask := task.Data
			if updateTask.Name != "" {
				FakeTask[i].Name = updateTask.Name
			}
			if updateTask.ResourceSubtype != "" {
				FakeTask[i].ResourceSubtype = updateTask.ResourceSubtype
			}
			if updateTask.ApprovalStatus != "" {
				FakeTask[i].ApprovalStatus = updateTask.ApprovalStatus
			}
			if updateTask.Notes != "" {
				FakeTask[i].Notes = updateTask.Notes
			}
			if updateTask.DueOn != "" {
				FakeTask[i].DueOn = updateTask.DueOn
			}
			if updateTask.StartOn != "" {
				FakeTask[i].StartOn = updateTask.StartOn
			}
			if updateTask.Assignee != "" {
				FakeTask[i].Assignee = User{
					GID:  updateTask.Assignee,
					Name: "Test User",
				}
			}
			if updateTask.Parent != "" {
				FakeTask[i].Parent = TaskParent{
					GID:             updateTask.Parent,
					Name:            "Test Task",
					ResourceSubtype: "default_task",
					CreatedBy: User{
						GID:  "123",
						Name: "Admin User",
					},
				}
			}

			FakeTask[i].Liked = updateTask.Liked
			FakeTask[i].Completed = updateTask.Completed

			if err = json.NewEncoder(res).Encode(
				map[string]interface{}{
					"data": FakeTask[i],
				},
			); err != nil {
				http.Error(res, err.Error(), http.StatusInternalServerError)
				return
			}
			return
		}
	}
	http.Error(res, "task not found", http.StatusNotFound)
}

type CreateTaskReqBody struct {
	Data struct {
		Name            string `json:"name"`
		ResourceSubtype string `json:"resource_subtype"`
		ApprovalStatus  string `json:"approval_status"`
		Completed       bool   `json:"completed"`
		Liked           bool   `json:"liked"`
		Notes           string `json:"notes"`
		DueAt           string `json:"due_at"`
		StartAt         string `json:"start_at"`
		Assignee        string `json:"assignee"`
		Parent          string `json:"parent"`
	} `json:"data"`
}

func createTask(res http.ResponseWriter, req *http.Request) {
	var err error
	opt := req.URL.Query()
	optFields := opt.Get("opt_fields")
	if optFields == "" {
		http.Error(res, "optFields is not expected to be null", http.StatusBadRequest)
		return
	}

	var task CreateTaskReqBody

	if err = json.NewDecoder(req.Body).Decode(&task); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}
	newID := "123456789"
	newTaskInfo := task.Data
	if newTaskInfo.Name == "" {
		http.Error(res, "Name is required", http.StatusBadRequest)
		return
	}

	newTask := RawTask{
		GID:             newID,
		Name:            newTaskInfo.Name,
		Notes:           newTaskInfo.Notes,
		HTMLNotes:       "Test HTML Notes",
		Projects:        []SimpleProject{},
		DueOn:           newTaskInfo.DueAt,
		StartOn:         newTaskInfo.DueAt,
		Liked:           newTaskInfo.Liked,
		Likes:           FakeLike,
		ApprovalStatus:  newTaskInfo.ApprovalStatus,
		ResourceSubtype: newTaskInfo.ResourceSubtype,
		Completed:       newTaskInfo.Completed,
		Assignee: User{
			GID:  newTaskInfo.Assignee,
			Name: "Test User",
		},
		Parent: TaskParent{
			GID:             newTaskInfo.Parent,
			Name:            "Test Task",
			ResourceSubtype: "default_task",
			CreatedBy: User{
				GID:  "123",
				Name: "Admin User",
			},
		},
	}
	FakeTask = append(FakeTask, newTask)

	if err = json.NewEncoder(res).Encode(
		map[string]interface{}{
			"data": newTask,
		},
	); err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
}
func deleteTask(res http.ResponseWriter, req *http.Request) {
	var err error
	taskGID := chi.URLParam(req, "taskGID")
	if taskGID == "" {
		http.Error(res, "taskGID is required", http.StatusBadRequest)
		return
	}

	for i, v := range FakeTask {
		if v.GID == taskGID {
			FakeTask = append(FakeTask[:i], FakeTask[i+1:]...)
			if err = json.NewEncoder(res).Encode(
				map[string]interface{}{
					"data": RawTask{},
				},
			); err != nil {
				http.Error(res, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}

	http.Error(res, "task not found", http.StatusNotFound)
}

type duplicateTaskReqBody struct {
	Name    string `json:"name"`
	Include string `json:"include"`
}

func duplicateTask(res http.ResponseWriter, req *http.Request) {
	var err error
	opt := req.URL.Query()
	optFields := opt.Get("opt_fields")
	if optFields == "" {
		http.Error(res, "optFields is not expected to be null", http.StatusBadRequest)
		return
	}

	taskGID := chi.URLParam(req, "taskGID")
	if taskGID == "" {
		http.Error(res, "taskGID is required", http.StatusBadRequest)
		return
	}
	var reqBody map[string]duplicateTaskReqBody
	if err = json.NewDecoder(req.Body).Decode(&reqBody); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}
	body := reqBody["data"]
	newID := "4321"

	if body.Include != "assignee,attachments,dates,dependencies,followers,notes,parent,projects,subtasks,tags" {
		http.Error(res, "Include Not correct", http.StatusBadRequest)
		return
	}

	for _, v := range FakeTask {
		if v.GID == taskGID {
			newTask := v
			newTask.GID = newID
			newTask.Name = body.Name

			FakeTask = append(FakeTask, newTask)
			if err = json.NewEncoder(res).Encode(
				map[string]interface{}{
					"data": newTask,
				},
			); err != nil {
				http.Error(res, err.Error(), http.StatusInternalServerError)
				return
			}
			return
		}
	}
	http.Error(res, "task not found", http.StatusNotFound)
}

type setParentReqBody struct {
	Parent string `json:"parent"`
}

func setParentTask(res http.ResponseWriter, req *http.Request) {
	var err error
	opt := req.URL.Query()
	optFields := opt.Get("opt_fields")
	if optFields == "" {
		http.Error(res, "optFields is not expected to be null", http.StatusBadRequest)
		return
	}

	taskGID := chi.URLParam(req, "taskGID")
	if taskGID == "" {
		http.Error(res, "taskGID is required", http.StatusBadRequest)
		return
	}
	var reqBody map[string]setParentReqBody
	if err = json.NewDecoder(req.Body).Decode(&reqBody); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}
	body := reqBody["data"]

	for _, v := range FakeTask {
		if v.GID == taskGID {
			newTask := v
			newTask.Parent = TaskParent{
				GID:             body.Parent,
				Name:            "Test Task",
				ResourceSubtype: "default_task",
				CreatedBy: User{
					GID:  "123",
					Name: "Admin User",
				},
			}

			if err = json.NewEncoder(res).Encode(
				map[string]interface{}{
					"data": newTask,
				},
			); err != nil {
				http.Error(res, err.Error(), http.StatusInternalServerError)
				return
			}
			return
		}
	}
	http.Error(res, "task not found", http.StatusNotFound)
}

type editTagReqBody struct {
	Tag string `json:"tag"`
}

func taskAddTag(res http.ResponseWriter, req *http.Request) {
	var err error

	taskGID := chi.URLParam(req, "taskGID")
	if taskGID == "" {
		http.Error(res, "taskGID is required", http.StatusBadRequest)
		return
	}
	var reqBody map[string]editTagReqBody
	if err = json.NewDecoder(req.Body).Decode(&reqBody); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}
	body := reqBody["data"]

	for _, v := range FakeTask {
		if v.GID == taskGID {
			if err = json.NewEncoder(res).Encode(
				map[string]interface{}{
					"data": fmt.Sprintf("Add tag %s", body.Tag),
				},
			); err != nil {
				http.Error(res, err.Error(), http.StatusInternalServerError)
				return
			}
			return
		}
	}
	http.Error(res, "task not found", http.StatusNotFound)
}
func taskRemoveTag(res http.ResponseWriter, req *http.Request) {
	var err error

	taskGID := chi.URLParam(req, "taskGID")
	if taskGID == "" {
		http.Error(res, "taskGID is required", http.StatusBadRequest)
		return
	}
	var reqBody map[string]editTagReqBody
	if err = json.NewDecoder(req.Body).Decode(&reqBody); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}
	body := reqBody["data"]

	for _, v := range FakeTask {
		if v.GID == taskGID {
			if err = json.NewEncoder(res).Encode(
				map[string]interface{}{
					"data": fmt.Sprintf("Remove tag %s", body.Tag),
				},
			); err != nil {
				http.Error(res, err.Error(), http.StatusInternalServerError)
				return
			}
			return
		}
	}
	http.Error(res, "task not found", http.StatusNotFound)
}

type editFollowerReqBody struct {
	Followers []string `json:"followers"`
}

func taskAddFollowers(res http.ResponseWriter, req *http.Request) {
	var err error
	opt := req.URL.Query()
	optFields := opt.Get("opt_fields")
	if optFields == "" {
		http.Error(res, "optFields is not expected to be null", http.StatusBadRequest)
		return
	}

	taskGID := chi.URLParam(req, "taskGID")
	if taskGID == "" {
		http.Error(res, "taskGID is required", http.StatusBadRequest)
		return
	}
	var reqBody map[string]editFollowerReqBody
	if err = json.NewDecoder(req.Body).Decode(&reqBody); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}
	body := reqBody["data"]

	for _, v := range FakeTask {
		if v.GID == taskGID {
			if len(body.Followers) == 2 && body.Followers[0] == "1234" && body.Followers[1] == "test@instill.tech" {
				if err = json.NewEncoder(res).Encode(
					map[string]interface{}{
						"data": v,
					},
				); err != nil {
					http.Error(res, err.Error(), http.StatusInternalServerError)
					return
				}
			} else {
				http.Error(res, "follower GID incorrect", http.StatusBadRequest)
				return
			}
			return
		}
	}
	http.Error(res, "task not found", http.StatusNotFound)
}

func taskRemoveFollowers(res http.ResponseWriter, req *http.Request) {
	var err error
	opt := req.URL.Query()
	optFields := opt.Get("opt_fields")
	if optFields == "" {
		http.Error(res, "optFields is not expected to be null", http.StatusBadRequest)
		return
	}

	taskGID := chi.URLParam(req, "taskGID")
	if taskGID == "" {
		http.Error(res, "taskGID is required", http.StatusBadRequest)
		return
	}
	var reqBody map[string]editFollowerReqBody
	if err = json.NewDecoder(req.Body).Decode(&reqBody); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}
	body := reqBody["data"]

	for _, v := range FakeTask {
		if v.GID == taskGID {
			if len(body.Followers) == 1 && body.Followers[0] == "1234" {
				if err = json.NewEncoder(res).Encode(
					map[string]interface{}{
						"data": v,
					},
				); err != nil {
					http.Error(res, err.Error(), http.StatusInternalServerError)
					return
				}
			} else {
				http.Error(res, "follower GID incorrect", http.StatusBadRequest)
				return
			}
			return
		}
	}
	http.Error(res, "task not found", http.StatusNotFound)
}

type editProjectReqBody struct {
	Project string `json:"project"`
}

func taskAddProject(res http.ResponseWriter, req *http.Request) {
	var err error

	taskGID := chi.URLParam(req, "taskGID")
	if taskGID == "" {
		http.Error(res, "taskGID is required", http.StatusBadRequest)
		return
	}
	var reqBody map[string]editProjectReqBody
	if err = json.NewDecoder(req.Body).Decode(&reqBody); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}
	body := reqBody["data"]

	for _, v := range FakeTask {
		if v.GID == taskGID {
			if err = json.NewEncoder(res).Encode(
				map[string]interface{}{
					"data": fmt.Sprintf("Add editProjectReqBody %s", body.Project),
				},
			); err != nil {
				http.Error(res, err.Error(), http.StatusInternalServerError)
				return
			}
			return
		}
	}
	http.Error(res, "task not found", http.StatusNotFound)
}
func taskRemoveProject(res http.ResponseWriter, req *http.Request) {
	var err error

	taskGID := chi.URLParam(req, "taskGID")
	if taskGID == "" {
		http.Error(res, "taskGID is required", http.StatusBadRequest)
		return
	}
	var reqBody map[string]editProjectReqBody
	if err = json.NewDecoder(req.Body).Decode(&reqBody); err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}
	body := reqBody["data"]

	for _, v := range FakeTask {
		if v.GID == taskGID {
			if err = json.NewEncoder(res).Encode(
				map[string]interface{}{
					"data": fmt.Sprintf("Remove editProjectReqBody %s", body.Project),
				},
			); err != nil {
				http.Error(res, err.Error(), http.StatusInternalServerError)
				return
			}
			return
		}
	}
	http.Error(res, "task not found", http.StatusNotFound)
}
