package jira

import (
	"context"
	"fmt"
	"time"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/x/errmsg"
)

func (c *client) updateSprint(ctx context.Context, job *base.Job) error {
	var input updateSprintInput
	if err := job.Input.ReadData(ctx, &input); err != nil {
		return fmt.Errorf("reading input data: %w", err)
	}

	apiEndpoint := fmt.Sprintf("rest/agile/1.0/sprint/%v", input.SprintID)

	var body updateSprintReq
	structOpt, err := base.ConvertToStructpb(input)
	if err != nil {
		return fmt.Errorf("converting to structpb: %w", err)
	}
	if err := base.ConvertFromStructpb(structOpt, &body); err != nil {
		return fmt.Errorf("converting to structpb: %w", err)
	}
	body.StartDate = input.StartDate
	body.EndDate = input.EndDate
	if _, err := time.Parse(time.RFC3339, body.StartDate); err != nil {
		if body.StartDate == "" {
			body.StartDate = time.Now().Format(time.RFC3339)
		} else if _, err := time.Parse(time.RFC3339, body.StartDate+"T00:00:00Z"); err == nil {
			body.StartDate = body.StartDate + "T00:00:00.000Z"
		} else {
			return errmsg.AddMessage(
				err,
				fmt.Sprintf("invalid start date format: %v", input.StartDate),
			)
		}
	}
	if _, err := time.Parse(time.RFC3339, body.EndDate); err != nil {
		if body.EndDate == "" {
			return errmsg.AddMessage(
				fmt.Errorf("end date is required"),
				"end date is required",
			)
		} else if _, err := time.Parse(time.RFC3339, body.EndDate+"T00:00:00Z"); err == nil {
			body.EndDate = body.EndDate + "T00:00:00.000Z"
		} else {
			return errmsg.AddMessage(
				err,
				fmt.Sprintf("invalid end date format: %s", input.EndDate),
			)
		}
	}
	if input.EnterNextState {
		switch input.CurrentState {
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
		body.State = input.CurrentState
	}
	jsonOpt, err := base.ConvertToStructpb(body)
	if err != nil {
		return fmt.Errorf("converting to structpb: %w", err)
	}
	req := c.R().SetResult(&Sprint{}).SetBody(jsonOpt)

	resp, err := req.Put(apiEndpoint)

	if err != nil {
		return fmt.Errorf("updating sprint: %w", err)
	}

	updatedSprint, ok := resp.Result().(*Sprint)
	if !ok {
		return errmsg.AddMessage(
			fmt.Errorf("failed to convert response to `Update Sprint` Output"),
			fmt.Sprintf("failed to convert %v to `Update Sprint` Output", resp.Result()),
		)
	}
	outputSprint := extractSprintOutput(updatedSprint)
	output := updateSprintOutput{
		ID:            outputSprint.ID,
		Self:          outputSprint.Self,
		State:         outputSprint.State,
		Name:          outputSprint.Name,
		StartDate:     outputSprint.StartDate,
		EndDate:       outputSprint.EndDate,
		CompleteDate:  outputSprint.CompleteDate,
		OriginBoardID: outputSprint.OriginBoardID,
		Goal:          outputSprint.Goal,
	}
	return job.Output.WriteData(ctx, output)
}
