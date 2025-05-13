package asana

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

type TaskTaskOutput struct {
	Task
}

type TaskTaskResp struct {
	Data struct {
		GID             string          `json:"gid"`
		Name            string          `json:"name"`
		Notes           string          `json:"notes"`
		HTMLNotes       string          `json:"html_notes"`
		Projects        []SimpleProject `json:"projects"`
		DueOn           string          `json:"due_on"`
		StartOn         string          `json:"start_on"`
		Liked           bool            `json:"liked"`
		Likes           []RawLike       `json:"likes"`
		ApprovalStatus  string          `json:"approval_status" api:"approval_status"`
		ResourceSubtype string          `json:"resource_subtype"`
		Completed       bool            `json:"completed" api:"completed"`
		Assignee        User            `json:"assignee" api:"assignee"`
		Parent          TaskParent      `json:"parent" api:"parent"`
	} `json:"data"`
}

func taskResp2Output(resp *TaskTaskResp) TaskTaskOutput {
	out := TaskTaskOutput{
		Task: Task{
			GID:             resp.Data.GID,
			Name:            resp.Data.Name,
			Notes:           resp.Data.Notes,
			HTMLNotes:       resp.Data.HTMLNotes,
			Projects:        resp.Data.Projects,
			DueOn:           resp.Data.DueOn,
			StartOn:         resp.Data.StartOn,
			Liked:           resp.Data.Liked,
			Likes:           []Like{},
			ApprovalStatus:  resp.Data.ApprovalStatus,
			ResourceSubtype: resp.Data.ResourceSubtype,
			Completed:       resp.Data.Completed,
			Assignee:        resp.Data.Assignee.GID,
			Parent:          resp.Data.Parent.GID,
		},
	}
	if out.Projects == nil {
		out.Projects = []SimpleProject{}
	}
	for _, like := range resp.Data.Likes {
		out.Likes = append(out.Likes, Like{
			LikeGID:  like.LikeGID,
			UserGID:  like.User.GID,
			UserName: like.User.Name,
		})
	}
	return out
}

type GetTaskInput struct {
	Action string `json:"action"`
	ID     string `json:"task-gid"`
}

func (c *Client) GetTask(ctx context.Context, props *structpb.Struct) (*structpb.Struct, error) {
	var input GetTaskInput
	if err := base.ConvertFromStructpb(props, &input); err != nil {
		return nil, err
	}

	apiEndpoint := fmt.Sprintf("/tasks/%s", input.ID)
	req := c.Client.R().SetResult(&TaskTaskResp{})

	wantOptFields := parseWantOptionFields(Task{})
	if err := addQueryOptions(req, map[string]interface{}{"opt_fields": wantOptFields}); err != nil {
		return nil, err
	}
	resp, err := req.Get(apiEndpoint)
	if err != nil {
		return nil, err
	}

	task := resp.Result().(*TaskTaskResp)
	out := taskResp2Output(task)

	return base.ConvertToStructpb(out)
}

type UpdateTaskInput struct {
	Action          string `json:"action"`
	ID              string `json:"task-gid"`
	Name            string `json:"name"`
	ResourceSubtype string `json:"resource-subtype"`
	ApprovalStatus  string `json:"approval-status"`
	Completed       bool   `json:"completed"`
	Liked           bool   `json:"liked"`
	Notes           string `json:"notes"`
	Assignee        string `json:"assignee"`
	Parent          string `json:"parent"`
}

type UpdateTaskReq struct {
	Name            string `json:"name,omitempty"`
	ResourceSubtype string `json:"resource_subtype,omitempty"`
	ApprovalStatus  string `json:"approval_status,omitempty"`
	Completed       bool   `json:"completed"`
	Liked           bool   `json:"liked"`
	Notes           string `json:"notes,omitempty"`
	Assignee        string `json:"assignee,omitempty"`
	Parent          string `json:"parent,omitempty"`
}

func (c *Client) UpdateTask(ctx context.Context, props *structpb.Struct) (*structpb.Struct, error) {
	var input UpdateTaskInput
	if err := base.ConvertFromStructpb(props, &input); err != nil {
		return nil, err
	}

	apiEndpoint := fmt.Sprintf("/tasks/%s", input.ID)
	req := c.Client.R().SetResult(&TaskTaskResp{})
	rawBody, _ := json.Marshal(map[string]interface{}{
		"data": &UpdateTaskReq{
			Name:            input.Name,
			ResourceSubtype: input.ResourceSubtype,
			ApprovalStatus:  input.ApprovalStatus,
			Completed:       input.Completed,
			Liked:           input.Liked,
			Notes:           input.Notes,
			Assignee:        input.Assignee,
			Parent:          input.Parent,
		},
	})
	var jsonBody map[string]map[string]interface{}
	_ = json.Unmarshal(rawBody, &jsonBody)
	if input.ApprovalStatus != "" || input.ResourceSubtype != "" {
		delete(jsonBody["data"], "completed")
	}
	bytebody, _ := json.Marshal(jsonBody)
	body := string(bytebody)
	req.SetBody(body)

	wantOptFields := parseWantOptionFields(Task{})
	if err := addQueryOptions(req, map[string]interface{}{"opt_fields": wantOptFields}); err != nil {
		return nil, err
	}

	resp, err := req.Put(apiEndpoint)

	if err != nil {
		return nil, err
	}
	task := resp.Result().(*TaskTaskResp)
	out := taskResp2Output(task)
	return base.ConvertToStructpb(out)
}

type CreateTaskInput struct {
	Action          string `json:"action"`
	Name            string `json:"name"`
	Notes           string `json:"notes"`
	ResourceSubtype string `json:"resource-subtype"`
	ApprovalStatus  string `json:"approval-status"`
	Completed       bool   `json:"completed"`
	Liked           bool   `json:"liked"`
	Assignee        string `json:"assignee"`
	Parent          string `json:"parent"`
	StartAt         string `json:"start-at"`
	DueAt           string `json:"due-at"`
	Workspace       string `json:"workspace"`
}

type CreateTaskReq struct {
	Name            string `json:"name"`
	Notes           string `json:"notes,omitempty"`
	ResourceSubtype string `json:"resource_subtype,omitempty"`
	ApprovalStatus  string `json:"approval_status,omitempty"`
	Completed       bool   `json:"completed"`
	Liked           bool   `json:"liked"`
	Assignee        string `json:"assignee,omitempty"`
	Parent          string `json:"parent,omitempty"`
	StartAt         string `json:"start_at,omitempty"`
	DueAt           string `json:"due_at,omitempty"`
	Workspace       string `json:"workspace"`
}

func (c *Client) CreateTask(ctx context.Context, props *structpb.Struct) (*structpb.Struct, error) {
	var input CreateTaskInput
	if err := base.ConvertFromStructpb(props, &input); err != nil {
		return nil, err
	}

	apiEndpoint := "/tasks"
	req := c.Client.R().SetResult(&TaskTaskResp{}).SetBody(
		map[string]interface{}{
			"data": &CreateTaskReq{
				Name:            input.Name,
				Notes:           input.Notes,
				ResourceSubtype: input.ResourceSubtype,
				ApprovalStatus:  input.ApprovalStatus,
				Completed:       input.Completed,
				Liked:           input.Liked,
				Assignee:        input.Assignee,
				Parent:          input.Parent,
				StartAt:         input.StartAt,
				DueAt:           input.DueAt,
				Workspace:       input.Workspace,
			},
		})
	wantOptFields := parseWantOptionFields(Task{})
	if err := addQueryOptions(req, map[string]interface{}{"opt_fields": wantOptFields}); err != nil {
		return nil, err
	}

	resp, err := req.Post(apiEndpoint)
	if err != nil {
		return nil, err
	}
	task := resp.Result().(*TaskTaskResp)
	out := taskResp2Output(task)
	return base.ConvertToStructpb(out)
}

type DeleteTaskInput struct {
	Action string `json:"action"`
	ID     string `json:"task-gid"`
}

func (c *Client) DeleteTask(ctx context.Context, props *structpb.Struct) (*structpb.Struct, error) {
	var input DeleteTaskInput
	if err := base.ConvertFromStructpb(props, &input); err != nil {
		return nil, err
	}

	apiEndpoint := fmt.Sprintf("/tasks/%s", input.ID)
	req := c.R()

	_, err := req.Delete(apiEndpoint)
	if err != nil {
		return nil, err
	}
	out := taskResp2Output(&TaskTaskResp{})
	return base.ConvertToStructpb(out)

}

type DuplicateTaskInput struct {
	Action string `json:"action"`
	ID     string `json:"task-gid"`
	Name   string `json:"name"`
}

type DuplicateTaskReq struct {
	Name    string `json:"name"`
	Include string `json:"include"`
}

func (c *Client) DuplicateTask(ctx context.Context, props *structpb.Struct) (*structpb.Struct, error) {
	var input DuplicateTaskInput
	if err := base.ConvertFromStructpb(props, &input); err != nil {
		return nil, err
	}

	apiEndpoint := fmt.Sprintf("/tasks/%s/duplicate", input.ID)
	req := c.Client.R().SetResult(&TaskTaskResp{}).SetBody(
		map[string]interface{}{
			"data": &DuplicateTaskReq{
				Name: input.Name,
				// include all fields, see https://developers.asana.com/reference/duplicatetask
				Include: "assignee,attachments,dates,dependencies,followers,notes,parent,projects,subtasks,tags",
			},
		},
	)

	wantOptFields := parseWantOptionFields(Task{})
	if err := addQueryOptions(req, map[string]interface{}{"opt_fields": wantOptFields}); err != nil {
		return nil, err
	}

	resp, err := req.Post(apiEndpoint)
	if err != nil {
		return nil, err
	}
	task := resp.Result().(*TaskTaskResp)
	getJobProps, _ := base.ConvertToStructpb(map[string]interface{}{
		"action":  "get",
		"job-gid": task.Data.GID,
	})
	var jobInfoMap Job
	if jobInfo, err := c.GetJob(ctx, getJobProps); err != nil {
		return nil, err
	} else {
		_ = base.ConvertFromStructpb(jobInfo, &jobInfoMap)
	}
	getProps, _ := base.ConvertToStructpb(map[string]interface{}{
		"action":   "get",
		"task-gid": jobInfoMap.NewTask.GID,
	})

	return c.GetTask(ctx, getProps)
}

type TaskSetParentInput struct {
	Action string `json:"action"`
	ID     string `json:"task-gid"`
	Parent string `json:"parent"`
}
type TaskSetParentReq struct {
	Parent string `json:"parent"`
}

func (c *Client) TaskSetParent(ctx context.Context, props *structpb.Struct) (*structpb.Struct, error) {
	var input TaskSetParentInput
	if err := base.ConvertFromStructpb(props, &input); err != nil {
		return nil, err
	}

	apiEndpoint := fmt.Sprintf("/tasks/%s/setParent", input.ID)
	req := c.Client.R().SetResult(&TaskTaskResp{}).SetBody(
		map[string]interface{}{
			"data": &TaskSetParentReq{
				Parent: input.Parent,
			},
		},
	)

	wantOptFields := parseWantOptionFields(Task{})
	if err := addQueryOptions(req, map[string]interface{}{"opt_fields": wantOptFields}); err != nil {
		return nil, err
	}

	resp, err := req.Post(apiEndpoint)
	if err != nil {
		return nil, err
	}
	task := resp.Result().(*TaskTaskResp)
	getProps, _ := base.ConvertToStructpb(map[string]interface{}{
		"action":   "get",
		"task-gid": task.Data.GID,
	})
	return c.GetTask(ctx, getProps)
}

type TaskEditTagInput struct {
	Action     string `json:"action"`
	ID         string `json:"task-gid"`
	TagID      string `json:"tag-gid"`
	EditOption string `json:"edit-option"`
}
type TaskEditTagReq struct {
	Tag string `json:"tag"`
}

func (c *Client) TaskEditTag(ctx context.Context, props *structpb.Struct) (*structpb.Struct, error) {
	var input TaskEditTagInput
	if err := base.ConvertFromStructpb(props, &input); err != nil {
		return nil, err
	}

	apiEndpoint := fmt.Sprintf("/tasks/%s/", input.ID)
	switch input.EditOption {
	case "add":
		apiEndpoint += "addTag"
	case "remove":
		apiEndpoint += "removeTag"
	}

	req := c.Client.R().SetBody(
		map[string]interface{}{
			"data": &TaskEditTagReq{
				Tag: input.TagID,
			},
		},
	)
	_, err := req.Post(apiEndpoint)
	if err != nil {
		return nil, err
	}
	return c.GetTask(ctx, props)
}

type TaskEditFollowerInput struct {
	Action     string `json:"action"`
	ID         string `json:"task-gid"`
	Followers  string `json:"followers"`
	EditOption string `json:"edit-option"`
}
type TaskEditFollowerReq struct {
	Followers []string `json:"followers"`
}

func (c *Client) TaskEditFollower(ctx context.Context, props *structpb.Struct) (*structpb.Struct, error) {
	var input TaskEditFollowerInput
	if err := base.ConvertFromStructpb(props, &input); err != nil {
		return nil, err
	}

	apiEndpoint := fmt.Sprintf("/tasks/%s/", input.ID)
	switch input.EditOption {
	case "add":
		apiEndpoint += "addFollowers"
	case "remove":
		apiEndpoint += "removeFollowers"
	}
	followers := strings.Split(input.Followers, ",")
	req := c.Client.R().SetResult(&TaskTaskResp{}).SetBody(
		map[string]interface{}{
			"data": &TaskEditFollowerReq{
				Followers: followers,
			},
		},
	)
	wantOptFields := parseWantOptionFields(Task{})
	if err := addQueryOptions(req, map[string]interface{}{"opt_fields": wantOptFields}); err != nil {
		return nil, err
	}

	resp, err := req.Post(apiEndpoint)
	if err != nil {
		return nil, err
	}
	task := resp.Result().(*TaskTaskResp)
	out := taskResp2Output(task)
	return base.ConvertToStructpb(out)
}

type TaskEditProjectInput struct {
	Action     string `json:"action"`
	ID         string `json:"task-gid"`
	ProjectID  string `json:"project-gid"`
	EditOption string `json:"edit-option"`
}
type TaskEditProjectReq struct {
	ProjectID string `json:"project"`
}

func (c *Client) TaskEditProject(ctx context.Context, props *structpb.Struct) (*structpb.Struct, error) {

	var input TaskEditProjectInput
	if err := base.ConvertFromStructpb(props, &input); err != nil {
		return nil, err
	}

	apiEndpoint := fmt.Sprintf("/tasks/%s/", input.ID)
	switch input.EditOption {
	case "add":
		apiEndpoint += "addProject"
	case "remove":
		apiEndpoint += "removeProject"
	}
	body, _ := json.Marshal(map[string]interface{}{
		"data": &TaskEditProjectReq{
			ProjectID: input.ProjectID,
		},
	})
	req := c.Client.R().SetBody(string(body))
	_, err := req.Post(apiEndpoint)
	if err != nil {
		return nil, err
	}
	return c.GetTask(ctx, props)
}
