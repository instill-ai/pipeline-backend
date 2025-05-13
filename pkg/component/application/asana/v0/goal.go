package asana

import (
	"context"
	"encoding/json"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

type GoalTaskOutput struct {
	Goal
}

type GoalTaskResp struct {
	Data struct {
		GID       string    `json:"gid"`
		Name      string    `json:"name"`
		Owner     User      `json:"owner"`
		Notes     string    `json:"notes"`
		HTMLNotes string    `json:"html_notes"`
		DueOn     string    `json:"due_on"`
		StartOn   string    `json:"start_on"`
		Liked     bool      `json:"liked"`
		Likes     []RawLike `json:"likes"`
	} `json:"data"`
}

func goalResp2Output(resp *GoalTaskResp) GoalTaskOutput {
	out := GoalTaskOutput{
		Goal: Goal{
			GID:       resp.Data.GID,
			Name:      resp.Data.Name,
			Owner:     resp.Data.Owner,
			Notes:     resp.Data.Notes,
			HTMLNotes: resp.Data.HTMLNotes,
			DueOn:     resp.Data.DueOn,
			StartOn:   resp.Data.StartOn,
			Liked:     resp.Data.Liked,
			Likes:     []Like{},
		},
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

type GetGoalInput struct {
	Action string `json:"action"`
	ID     string `json:"goal-gid"`
}

func (c *Client) GetGoal(ctx context.Context, props *structpb.Struct) (*structpb.Struct, error) {
	var input GetGoalInput
	if err := base.ConvertFromStructpb(props, &input); err != nil {
		return nil, err
	}
	apiEndpoint := "/goals/" + input.ID
	req := c.Client.R().SetResult(&GoalTaskResp{})

	wantOptFields := parseWantOptionFields(Goal{})
	if err := addQueryOptions(req, map[string]interface{}{"opt_fields": wantOptFields}); err != nil {
		return nil, err
	}

	resp, err := req.Get(apiEndpoint)
	if err != nil {
		return nil, err
	}
	goal := resp.Result().(*GoalTaskResp)
	out := goalResp2Output(goal)
	return base.ConvertToStructpb(out)
}

type UpdateGoalInput struct {
	Action  string `json:"action"`
	ID      string `json:"goal-gid"`
	Name    string `json:"name"`
	Notes   string `json:"notes"`
	DueOn   string `json:"due-on"`
	StartOn string `json:"start-on"`
	Liked   bool   `json:"liked"`
	Status  string `json:"status"`
}

type UpdateGoalReq struct {
	Name    string `json:"name,omitempty"`
	Notes   string `json:"notes,omitempty"`
	DueOn   string `json:"due_on,omitempty"`
	StartOn string `json:"start_on,omitempty"`
	Liked   bool   `json:"liked"`
	Status  string `json:"status,omitempty"`
}

func (c *Client) UpdateGoal(ctx context.Context, props *structpb.Struct) (*structpb.Struct, error) {
	var input UpdateGoalInput
	if err := base.ConvertFromStructpb(props, &input); err != nil {
		return nil, err
	}

	apiEndpoint := "/goals/" + input.ID

	body, _ := json.Marshal(map[string]interface{}{
		"data": &UpdateGoalReq{
			Name:    input.Name,
			Notes:   input.Notes,
			DueOn:   input.DueOn,
			StartOn: input.StartOn,
			Liked:   input.Liked,
			Status:  input.Status,
		},
	})
	req := c.Client.R().SetResult(&GoalTaskResp{}).SetBody(string(body))

	wantOptFields := parseWantOptionFields(Goal{})
	if err := addQueryOptions(req, map[string]interface{}{"opt_fields": wantOptFields}); err != nil {
		return nil, err
	}
	resp, err := req.Put(apiEndpoint)

	if err != nil {
		return nil, err
	}
	goal := resp.Result().(*GoalTaskResp)
	out := goalResp2Output(goal)
	return base.ConvertToStructpb(out)
}

type CreateGoalInput struct {
	Action     string `json:"action"`
	Name       string `json:"name"`
	Notes      string `json:"notes"`
	DueOn      string `json:"due-on"`
	StartOn    string `json:"start-on"`
	Liked      bool   `json:"liked"`
	Workspace  string `json:"workspace"`
	TimePeriod string `json:"time-period"`
	Owner      string `json:"owner"`
}
type CreateGoalReq struct {
	Name       string `json:"name"`
	Notes      string `json:"notes"`
	DueOn      string `json:"due_on"`
	StartOn    string `json:"start_on"`
	Liked      bool   `json:"liked"`
	Workspace  string `json:"workspace"`
	TimePeriod string `json:"time_period"`
	Owner      string `json:"owner"`
}

func (c *Client) CreateGoal(ctx context.Context, props *structpb.Struct) (*structpb.Struct, error) {
	var input CreateGoalInput
	if err := base.ConvertFromStructpb(props, &input); err != nil {
		return nil, err
	}

	apiEndpoint := "/goals"
	req := c.Client.R().SetResult(&GoalTaskResp{}).SetBody(
		map[string]interface{}{
			"data": &CreateGoalReq{
				Name:       input.Name,
				Notes:      input.Notes,
				DueOn:      input.DueOn,
				StartOn:    input.StartOn,
				Liked:      input.Liked,
				Workspace:  input.Workspace,
				TimePeriod: input.TimePeriod,
				Owner:      input.Owner,
			},
		})
	wantOptFields := parseWantOptionFields(Goal{})
	if err := addQueryOptions(req, map[string]interface{}{"opt_fields": wantOptFields}); err != nil {
		return nil, err
	}
	resp, err := req.Post(apiEndpoint)
	if err != nil {
		return nil, err
	}
	goal := resp.Result().(*GoalTaskResp)
	out := goalResp2Output(goal)
	return base.ConvertToStructpb(out)
}

type DeleteGoalInput struct {
	Action string `json:"action"`
	ID     string `json:"goal-gid"`
}

func (c *Client) DeleteGoal(ctx context.Context, props *structpb.Struct) (*structpb.Struct, error) {
	var input DeleteGoalInput
	if err := base.ConvertFromStructpb(props, &input); err != nil {
		return nil, err
	}

	apiEndpoint := "/goals/" + input.ID
	req := c.R()

	_, err := req.Delete(apiEndpoint)
	if err != nil {
		return nil, err
	}
	goal := GoalTaskResp{}
	out := goalResp2Output(&goal)
	return base.ConvertToStructpb(out)
}
