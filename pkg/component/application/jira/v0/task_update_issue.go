package jira

import (
	"context"
	"fmt"
	"strings"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"

	errorsx "github.com/instill-ai/x/errors"
)

func (c *client) updateIssue(ctx context.Context, job *base.Job) error {
	var err error

	var input updateIssueInput
	if err := job.Input.ReadData(ctx, &input); err != nil {
		return fmt.Errorf("reading input data: %w", err)
	}

	var updatedIssue *updateIssueResp
	switch input.Update.UpdateType {
	case "Custom Update":
		updatedIssue, err = updateIssue(c.Client, &input)
		if err != nil {
			return fmt.Errorf("updating issue: %w", err)
		}
		output := updateIssueOutput{
			Issue: Issue{
				ID:          updatedIssue.ID,
				Key:         updatedIssue.Key,
				Description: updatedIssue.Description,
				Summary:     updatedIssue.Summary,
				Fields:      updatedIssue.Fields,
				Self:        updatedIssue.Self,
				IssueType:   updatedIssue.IssueType,
				Status:      updatedIssue.Status,
			},
		}
		return job.Output.WriteData(ctx, output)
	case "Move Issue to Epic":
		err = moveIssueToEpic(c.Client, input.IssueKey, input.Update.EpicKey)
		if err != nil {
			if !strings.Contains(errorsx.Message(err), "The request contains a next-gen issue") {
				return fmt.Errorf("moving issue to epic: %w", err)
			}
			input.Update.UpdateType = "Custom Update"
			input.Update.UpdateFields = append(input.Update.UpdateFields, updateField{
				Action:    "set",
				FieldName: "parent",
				Value: map[string]string{
					"key": input.Update.EpicKey,
				},
			})
			if _, err = updateIssue(c.Client, &input); err != nil {
				return errorsx.AddMessage(
					fmt.Errorf("failed to update issue with parent key"),
					"You can only move issues to epics. The Jira API response with: "+errorsx.Message(err),
				)
			}
		}
		// get issue
		issue, err := getIssue(c.Client, input.IssueKey, false)
		if err != nil {
			return fmt.Errorf("getting issue: %w", err)
		}

		output := updateIssueOutput{Issue: *issue}
		return job.Output.WriteData(ctx, output)
	default:
		return errorsx.AddMessage(
			fmt.Errorf("invalid update type"),
			fmt.Sprintf("%s is an invalid update type", input.Update.UpdateType),
		)
	}
}
