package jira

import (
	"context"
	"fmt"
	"time"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/x/errmsg"
)

func (c *client) createSprint(ctx context.Context, job *base.Job) error {
	var opt createSprintInput
	if err := job.Input.ReadData(ctx, &opt); err != nil {
		return fmt.Errorf("reading input data: %w", err)
	}
	apiBaseURL := "rest/agile/1.0/sprint"

	// Validate timestamp format RFC3339
	if _, err := time.Parse(time.RFC3339, opt.StartDate); err != nil {
		if opt.StartDate == "" {
			opt.StartDate = time.Now().Format(time.RFC3339)
		} else if _, err := time.Parse(time.RFC3339, opt.StartDate+"T00:00:00Z"); err == nil {
			opt.StartDate = opt.StartDate + "T00:00:00.000Z"
		} else {
			return errmsg.AddMessage(
				err,
				fmt.Sprintf("invalid start date format: %s", opt.StartDate),
			)
		}
	}
	if _, err := time.Parse(time.RFC3339, opt.EndDate); err != nil {
		if opt.EndDate == "" {
			return errmsg.AddMessage(
				fmt.Errorf("end date is required"),
				"end date is required",
			)
		} else if _, err := time.Parse(time.RFC3339, opt.EndDate+"T00:00:00Z"); err == nil {
			opt.EndDate = opt.EndDate + "T00:00:00.000Z"
		} else {
			return errmsg.AddMessage(
				err,
				fmt.Sprintf("invalid end date format: %s", opt.EndDate),
			)
		}
	}
	boardName := opt.BoardName
	boards, err := listBoards(c, &listBoardsInput{Name: boardName})
	if err != nil {
		return fmt.Errorf("listing boards: %w", err)
	}

	if len(boards.Boards) == 0 {
		return errmsg.AddMessage(
			fmt.Errorf("board not found"),
			fmt.Sprintf("board with name %s not found", opt.BoardName),
		)
	} else if len(boards.Boards) > 1 {
		return errmsg.AddMessage(
			fmt.Errorf("multiple boards found"),
			fmt.Sprintf("multiple boards are found with the partial name \"%s\". Please provide a more specific name", opt.BoardName),
		)
	}
	board := boards.Boards[0]
	boardID := board.ID

	req := c.R().SetResult(&createSprintResp{}).SetBody(&createSprintReq{
		Name:          opt.Name,
		Goal:          opt.Goal,
		StartDate:     opt.StartDate,
		EndDate:       opt.EndDate,
		OriginBoardID: boardID,
	})

	resp, err := req.Post(apiBaseURL)
	if err != nil {
		return errmsg.AddMessage(fmt.Errorf("creating sprint: %w", err), errmsg.Message(err))
	}

	sprint, ok := resp.Result().(*createSprintResp)
	if !ok {
		return errmsg.AddMessage(
			fmt.Errorf("failed to convert response to `Create Sprint` Output"),
			fmt.Sprintf("failed to convert %v to `Create Sprint` Output", resp.Result()),
		)
	}

	output := createSprintOutput{
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
	return job.Output.WriteData(ctx, output)
}
