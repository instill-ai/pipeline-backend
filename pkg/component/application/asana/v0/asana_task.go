package asana

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"

	errorsx "github.com/instill-ai/x/errors"
)

type AsanaTask struct {
	Action string `json:"action"`
}

func (c *Client) GoalRelatedTask(ctx context.Context, props *structpb.Struct) (*structpb.Struct, error) {
	var task AsanaTask
	if err := base.ConvertFromStructpb(props, &task); err != nil {
		return nil, err
	}
	switch task.Action {
	case "create":
		return c.CreateGoal(ctx, props)
	case "update":
		return c.UpdateGoal(ctx, props)
	case "delete":
		return c.DeleteGoal(ctx, props)
	case "get":
		return c.GetGoal(ctx, props)
	default:
		return nil, errorsx.AddMessage(
			fmt.Errorf("invalid action"),
			"invalid action for \"Goal\"",
		)
	}
}

func (c *Client) TaskRelatedTask(ctx context.Context, props *structpb.Struct) (*structpb.Struct, error) {
	var task AsanaTask
	if err := base.ConvertFromStructpb(props, &task); err != nil {
		return nil, err
	}
	switch task.Action {
	case "create":
		return c.CreateTask(ctx, props)
	case "update":
		return c.UpdateTask(ctx, props)
	case "delete":
		return c.DeleteTask(ctx, props)
	case "get":
		return c.GetTask(ctx, props)
	case "duplicate":
		return c.DuplicateTask(ctx, props)
	case "set parent":
		return c.TaskSetParent(ctx, props)
	case "edit tag":
		return c.TaskEditTag(ctx, props)
	case "edit follower":
		return c.TaskEditFollower(ctx, props)
	case "edit project":
		return c.TaskEditProject(ctx, props)
	default:
		return nil, errorsx.AddMessage(
			fmt.Errorf("invalid action"),
			"invalid action for \"Task\"",
		)
	}
}

func (c *Client) ProjectRelatedTask(ctx context.Context, props *structpb.Struct) (*structpb.Struct, error) {
	var task AsanaTask
	if err := base.ConvertFromStructpb(props, &task); err != nil {
		return nil, err
	}
	switch task.Action {
	case "create":
		return c.CreateProject(ctx, props)
	case "update":
		return c.UpdateProject(ctx, props)
	case "delete":
		return c.DeleteProject(ctx, props)
	case "get":
		return c.GetProject(ctx, props)
	case "duplicate":
		return c.DuplicateProject(ctx, props)
	default:
		return nil, errorsx.AddMessage(
			fmt.Errorf("invalid action"),
			"invalid action for \"Project\"",
		)
	}
}

func (c *Client) PortfolioRelatedTask(ctx context.Context, props *structpb.Struct) (*structpb.Struct, error) {
	var task AsanaTask
	if err := base.ConvertFromStructpb(props, &task); err != nil {
		return nil, err
	}
	switch task.Action {
	case "create":
		return c.CreatePortfolio(ctx, props)
	case "update":
		return c.UpdatePortfolio(ctx, props)
	case "delete":
		return c.DeletePortfolio(ctx, props)
	case "get":
		return c.GetPortfolio(ctx, props)
	default:
		return nil, errorsx.AddMessage(
			fmt.Errorf("invalid action"),
			"invalid action for \"Portfolio\"",
		)
	}
}
