package scheduler

//go:generate compogen readme ./config ./README.mdx --extraContents bottom=.compogen/bottom.mdx

import (
	"context"
	"fmt"
	"sync"

	_ "embed"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"

	"github.com/gofrs/uuid"
	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/data"

	errorsx "github.com/instill-ai/x/errors"
)

var (
	//go:embed config/definition.yaml
	definitionYAML []byte
	//go:embed config/events.yaml
	eventsYAML []byte
	//go:embed config/tasks.yaml
	tasksYAML []byte

	once sync.Once
	comp *component
)

type component struct {
	base.Component
}

// Init initializes a Component that handles scheduled events.
func Init(bc base.Component) *component {
	once.Do(func() {
		comp = &component{Component: bc}
		err := comp.LoadDefinition(definitionYAML, nil, tasksYAML, eventsYAML, nil)
		if err != nil {
			panic(err)
		}
	})
	return comp
}

// CreateExecution initializes a component executor that can be used in a
// pipeline run.
func (c *component) CreateExecution(x base.ComponentExecution) (base.IExecution, error) {
	return nil, errorsx.AddMessage(
		fmt.Errorf("not supported task: %s", x.Task),
		fmt.Sprintf("%s task is not supported.", x.Task),
	)
}

func (c *component) ParseEvent(ctx context.Context, rawEvent *base.RawEvent) (parsedEvent *base.ParsedEvent, err error) {
	// Currently only cron job triggered is supported
	return c.handleEventCronJobTriggered(ctx, rawEvent)
}

func (c *component) RegisterEvent(ctx context.Context, settings *base.RegisterEventSettings) ([]base.Identifier, error) {
	unmarshaler := data.NewUnmarshaler(c.BinaryFetcher)
	cfg := EventCronJobTriggered{}
	err := unmarshaler.Unmarshal(ctx, settings.Config, &cfg)
	if err != nil {
		return nil, err
	}

	uid, _ := uuid.NewV4()
	scheduleID := fmt.Sprintf("schedule_%s", uid)

	param := &SchedulePipelineWorkflowParam{
		UID: uid,
	}
	_, err = c.TemporalClient.ScheduleClient().Create(ctx, client.ScheduleOptions{
		ID: scheduleID,
		Spec: client.ScheduleSpec{
			CronExpressions: []string{cfg.Cron},
		},
		Action: &client.ScheduleWorkflowAction{
			Args:      []any{param},
			ID:        scheduleID,
			Workflow:  "SchedulePipelineWorkflow",
			TaskQueue: "pipeline-backend", // TODO: avoid hardcoding
			RetryPolicy: &temporal.RetryPolicy{
				MaximumAttempts: 1,
			},
		},
	})
	if err != nil {
		return nil, err
	}

	return []base.Identifier{{"uid": uid.String()}}, nil
}

func (c *component) IdentifyEvent(ctx context.Context, rawEvent *base.RawEvent) (*base.IdentifierResult, error) {

	r := &EventCronJobTriggeredMessage{}
	unmarshaler := data.NewUnmarshaler(c.BinaryFetcher)
	err := unmarshaler.Unmarshal(ctx, rawEvent.Message, r)
	if err != nil {
		return nil, err
	}

	return &base.IdentifierResult{
		Identifiers: []base.Identifier{{"uid": r.UID}},
	}, nil
}
func (c *component) UnregisterEvent(ctx context.Context, settings *base.UnregisterEventSettings, identifiers []base.Identifier) error {

	for _, identifier := range identifiers {
		uid, ok := identifier["uid"]
		if ok {
			scheduleID := fmt.Sprintf("schedule_%s", uid)
			handle := c.TemporalClient.ScheduleClient().GetHandle(ctx, scheduleID)
			_ = handle.Delete(ctx)
		}
	}
	return nil
}

type SchedulePipelineWorkflowParam struct {
	UID     uuid.UUID
	EventID string
}
