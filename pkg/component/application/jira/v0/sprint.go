package jira

import (
	"context"
	"fmt"
	"time"

	_ "embed"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/x/errmsg"
)

type Sprint struct {
	ID            int    `json:"id"`
	Self          string `json:"self"`
	State         string `json:"state"`
	Name          string `json:"name"`
	StartDate     string `json:"startDate"`
	EndDate       string `json:"endDate"`
	CompleteDate  string `json:"completeDate"`
	OriginBoardID int    `json:"originBoardId"`
	Goal          string `json:"goal"`
}

type GetSprintInput struct {
	SprintID int `json:"sprint-id"`
}
type GetSprintOutput struct {
	ID            int    `json:"id"`
	Self          string `json:"self"`
	State         string `json:"state"`
	Name          string `json:"name"`
	StartDate     string `json:"start-date"`
	EndDate       string `json:"end-date"`
	CompleteDate  string `json:"complete-date"`
	OriginBoardID int    `json:"origin-board-id"`
	Goal          string `json:"goal"`
}

func extractSprintOutput(sprint *Sprint) *GetSprintOutput {
	return &GetSprintOutput{
		ID:            sprint.ID,
		Self:          sprint.Self,
		State:         sprint.State,
		Name:          sprint.Name,
		StartDate:     sprint.StartDate,
		EndDate:       sprint.EndDate,
		CompleteDate:  sprint.CompleteDate,
		OriginBoardID: sprint.OriginBoardID,
		Goal:          sprint.Goal,
	}
}
func (jiraClient *Client) getSprintTask(_ context.Context, props *structpb.Struct) (*structpb.Struct, error) {
	var opt GetSprintInput
	if err := base.ConvertFromStructpb(props, &opt); err != nil {
		return nil, err
	}

	apiEndpoint := fmt.Sprintf("rest/agile/1.0/sprint/%v", opt.SprintID)
	req := jiraClient.Client.R().SetResult(&Sprint{})
	resp, err := req.Get(apiEndpoint)

	if err != nil {
		return nil, fmt.Errorf(
			err.Error(), errmsg.Message(err),
		)
	}

	issue, ok := resp.Result().(*Sprint)
	if !ok {
		return nil, errmsg.AddMessage(
			fmt.Errorf("failed to convert response to `Get Sprint` Output"),
			fmt.Sprintf("failed to convert %v to `Get Sprint` Output", resp.Result()),
		)
	}
	out := extractSprintOutput(issue)
	return base.ConvertToStructpb(out)
}

type ListSprintInput struct {
	BoardID    int `json:"board-id"`
	StartAt    int `json:"start-at" api:"startAt"`
	MaxResults int `json:"max-results" api:"maxResults"`
}

type ListSprintsResp struct {
	Values     []Sprint `json:"values"`
	StartAt    int      `json:"startAt"`
	MaxResults int      `json:"maxResults"`
	Total      int      `json:"total"`
}
type ListSprintsOutput struct {
	Sprints    []*GetSprintOutput `json:"sprints"`
	StartAt    int                `json:"start-at"`
	MaxResults int                `json:"max-results"`
	Total      int                `json:"total"`
}

func (jiraClient *Client) listSprintsTask(ctx context.Context, props *structpb.Struct) (*structpb.Struct, error) {
	var opt ListSprintInput
	if err := base.ConvertFromStructpb(props, &opt); err != nil {
		return nil, err
	}
	apiEndpoint := fmt.Sprintf("rest/agile/1.0/board/%d/sprint", opt.BoardID)

	req := jiraClient.Client.R().SetResult(&ListSprintsResp{})
	opt.BoardID = 0
	err := addQueryOptions(req, opt)
	if err != nil {
		return nil, err
	}

	resp, err := req.Get(apiEndpoint)

	if err != nil {
		return nil, fmt.Errorf(
			err.Error(), errmsg.Message(err),
		)
	}

	issues, ok := resp.Result().(*ListSprintsResp)
	if !ok {
		return nil, errmsg.AddMessage(
			fmt.Errorf("failed to convert response to `List Sprint` Output"),
			fmt.Sprintf("failed to convert %v to `List Sprint` Output", resp.Result()),
		)
	}
	var out ListSprintsOutput
	for _, issue := range issues.Values {
		out.Sprints = append(out.Sprints, extractSprintOutput(&issue))
	}
	out.StartAt = issues.StartAt
	out.MaxResults = issues.MaxResults
	out.Total = issues.Total
	return base.ConvertToStructpb(out)
}

type CreateSprintInput struct {
	BoardName string `json:"board-name"`
	Name      string `json:"name"`
	Goal      string `json:"goal"`
	StartDate string `json:"start-date"`
	EndDate   string `json:"end-date"`
}

type CreateSprintRequest struct {
	Name          string `json:"name"`
	Goal          string `json:"goal"`
	StartDate     string `json:"startDate"`
	EndDate       string `json:"endDate"`
	OriginBoardID int    `json:"originBoardId"`
}

type CreateSprintResp struct {
	Sprint
}

type CreateSprintOutput struct {
	ID            int    `json:"id"`
	Self          string `json:"self"`
	State         string `json:"state"`
	Name          string `json:"name"`
	StartDate     string `json:"start-date"`
	EndDate       string `json:"end-date"`
	CompleteDate  string `json:"complete-date"`
	OriginBoardID int    `json:"origin-board-id"`
	Goal          string `json:"goal"`
}

func (jiraClient *Client) createSprintTask(ctx context.Context, props *structpb.Struct) (*structpb.Struct, error) {
	var opt CreateSprintInput
	if err := base.ConvertFromStructpb(props, &opt); err != nil {
		return nil, err
	}
	apiBaseURL := "rest/agile/1.0/sprint"

	// Validate timestamp format RFC3339
	if _, err := time.Parse(time.RFC3339, opt.StartDate); err != nil {
		if opt.StartDate == "" {
			opt.StartDate = time.Now().Format(time.RFC3339)
		} else if _, err := time.Parse(time.RFC3339, opt.StartDate+"T00:00:00Z"); err == nil {
			opt.StartDate = opt.StartDate + "T00:00:00.000Z"
		} else {
			return nil, errmsg.AddMessage(
				err,
				fmt.Sprintf("invalid start date format: %s", opt.StartDate),
			)
		}
	}
	if _, err := time.Parse(time.RFC3339, opt.EndDate); err != nil {
		if opt.EndDate == "" {
			return nil, errmsg.AddMessage(
				fmt.Errorf("end date is required"),
				"end date is required",
			)
		} else if _, err := time.Parse(time.RFC3339, opt.EndDate+"T00:00:00Z"); err == nil {
			opt.EndDate = opt.EndDate + "T00:00:00.000Z"
		} else {
			return nil, errmsg.AddMessage(
				err,
				fmt.Sprintf("invalid end date format: %s", opt.EndDate),
			)
		}
	}
	boardName := opt.BoardName
	boards, err := jiraClient.listBoards(ctx, &ListBoardsInput{Name: boardName})
	if err != nil {
		return nil, err
	}

	if len(boards.Values) == 0 {
		return nil, errmsg.AddMessage(
			fmt.Errorf("board not found"),
			fmt.Sprintf("board with name %s not found", opt.BoardName),
		)
	} else if len(boards.Values) > 1 {
		return nil, errmsg.AddMessage(
			fmt.Errorf("multiple boards found"),
			fmt.Sprintf("multiple boards are found with the partial name \"%s\". Please provide a more specific name", opt.BoardName),
		)
	}
	board := boards.Values[0]
	boardID := board.ID

	req := jiraClient.Client.R().SetResult(&CreateSprintResp{}).SetBody(&CreateSprintRequest{
		Name:          opt.Name,
		Goal:          opt.Goal,
		StartDate:     opt.StartDate,
		EndDate:       opt.EndDate,
		OriginBoardID: boardID,
	})

	resp, err := req.Post(apiBaseURL)
	if err != nil {
		return nil, fmt.Errorf(
			err.Error(), errmsg.Message(err),
		)
	}

	sprint, ok := resp.Result().(*CreateSprintResp)
	if !ok {
		return nil, errmsg.AddMessage(
			fmt.Errorf("failed to convert response to `Create Sprint` Output"),
			fmt.Sprintf("failed to convert %v to `Create Sprint` Output", resp.Result()),
		)
	}

	out := &CreateSprintOutput{
		ID:            sprint.ID,
		Self:          sprint.Self,
		State:         sprint.State,
		Name:          sprint.Name,
		StartDate:     sprint.StartDate,
		EndDate:       sprint.EndDate,
		CompleteDate:  sprint.CompleteDate,
		OriginBoardID: sprint.OriginBoardID,
		Goal:          sprint.Goal,
	}
	return base.ConvertToStructpb(out)
}

type UpdateSprintInput struct {
	SprintID       int    `json:"sprint-id"`
	Name           string `json:"name"`
	Goal           string `json:"goal"`
	StartDate      string `json:"start-date"`
	EndDate        string `json:"end-date"`
	CurrentState   string `json:"current-state"`
	EnterNextState bool   `json:"enter-next-state"`
}

type UpdateSprintRequest struct {
	Name      string `json:"name,omitempty"`
	Goal      string `json:"goal,omitempty"`
	StartDate string `json:"startDate,omitempty"`
	EndDate   string `json:"endDate,omitempty"`
	State     string `json:"state,omitempty"`
}

type UpdateSprintResp struct {
	Sprint
}
type UpdateSprintOutput struct {
	ID            int    `json:"id"`
	Self          string `json:"self"`
	State         string `json:"state"`
	Name          string `json:"name"`
	StartDate     string `json:"start-date"`
	EndDate       string `json:"end-date"`
	CompleteDate  string `json:"complete-date"`
	OriginBoardID int    `json:"origin-board-id"`
	Goal          string `json:"goal"`
}

func (jiraClient *Client) updateSprintTask(ctx context.Context, props *structpb.Struct) (*structpb.Struct, error) {
	var opt UpdateSprintInput
	if err := base.ConvertFromStructpb(props, &opt); err != nil {
		return nil, err
	}

	apiEndpoint := fmt.Sprintf("rest/agile/1.0/sprint/%v", opt.SprintID)

	var body UpdateSprintRequest
	structOpt, err := base.ConvertToStructpb(opt)
	if err != nil {
		return nil, err
	}
	if err := base.ConvertFromStructpb(structOpt, &body); err != nil {
		return nil, err
	}
	body.StartDate = opt.StartDate
	body.EndDate = opt.EndDate
	if _, err := time.Parse(time.RFC3339, body.StartDate); err != nil {
		if body.StartDate == "" {
			body.StartDate = time.Now().Format(time.RFC3339)
		} else if _, err := time.Parse(time.RFC3339, body.StartDate+"T00:00:00Z"); err == nil {
			body.StartDate = body.StartDate + "T00:00:00.000Z"
		} else {
			return nil, errmsg.AddMessage(
				err,
				fmt.Sprintf("invalid start date format: %v", opt.StartDate),
			)
		}
	}
	if _, err := time.Parse(time.RFC3339, body.EndDate); err != nil {
		if body.EndDate == "" {
			return nil, errmsg.AddMessage(
				fmt.Errorf("end date is required"),
				"end date is required",
			)
		} else if _, err := time.Parse(time.RFC3339, body.EndDate+"T00:00:00Z"); err == nil {
			body.EndDate = body.EndDate + "T00:00:00.000Z"
		} else {
			return nil, errmsg.AddMessage(
				err,
				fmt.Sprintf("invalid end date format: %s", opt.EndDate),
			)
		}
	}
	if opt.EnterNextState {
		switch opt.CurrentState {
		case "future":
			body.State = "active"
			startTime, _ := time.Parse(time.RFC3339, body.StartDate)
			if time.Now().Compare(startTime) == -1 {
				body.StartDate = time.Now().Format(time.RFC3339)
			}
		case "active":
			body.State = "closed"
		case "closed":
			body.State = "closed"
		}
	} else {
		body.State = opt.CurrentState
	}
	jsonOpt, err := base.ConvertToStructpb(body)
	if err != nil {
		return nil, err
	}
	req := jiraClient.Client.R().SetResult(&Sprint{}).SetBody(jsonOpt)

	resp, err := req.Put(apiEndpoint)

	if err != nil {
		return nil, err
	}

	updatedSprint, ok := resp.Result().(*Sprint)
	if !ok {
		return nil, errmsg.AddMessage(
			fmt.Errorf("failed to convert response to `Update Sprint` Output"),
			fmt.Sprintf("failed to convert %v to `Update Sprint` Output", resp.Result()),
		)
	}
	out := extractSprintOutput(updatedSprint)
	return base.ConvertToStructpb(out)
}
