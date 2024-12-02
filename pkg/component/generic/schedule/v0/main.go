package schedule

//go:generate compogen readme ./config ./README.mdx

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
	"github.com/instill-ai/x/errmsg"
)

var (
	//go:embed config/definition.json
	definitionJSON []byte
	//go:embed config/events.json
	eventsJSON []byte
	//go:embed config/tasks.json
	tasksJSON []byte

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
		err := comp.LoadDefinition(definitionJSON, nil, tasksJSON, eventsJSON, nil)
		if err != nil {
			panic(err)
		}
	})
	return comp
}

// CreateExecution initializes a component executor that can be used in a
// pipeline run.
func (c *component) CreateExecution(x base.ComponentExecution) (base.IExecution, error) {
	return nil, errmsg.AddMessage(
		fmt.Errorf("not supported task: %s", x.Task),
		fmt.Sprintf("%s task is not supported.", x.Task),
	)
}

func (c *component) ParseEvent(ctx context.Context, rawEvent *base.RawEvent) (parsedEvent *base.ParsedEvent, err error) {
	// Currently only cron job triggered is supported
	fmt.Println("ParseEvent", rawEvent)
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
	fmt.Println("IdentifyEvent", r)

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

// func (s *service) setSchedulePipeline(ctx context.Context, ns resource.Namespace, pipelineID, pipelineReleaseID string, pipelineUID, releaseUID uuid.UUID, recipe *datamodel.Recipe) error {
// 	// TODO This check could be removed, as the receiver should be initialized
// 	// at this point. However, some tests depend on it, so we would need to
// 	// either mock this interface or (better) communicate with Temporal through
// 	// our own interface.
// 	if s.temporalClient == nil {
// 		return nil
// 	}

// 	crons := []string{}
// 	if recipe != nil && recipe.On != nil {
// 		for _, v := range recipe.On {
// 			// TODO: Introduce Schedule Component to define structured schema
// 			// for schedule setup configuration
// 			if v.Type == "schedule" {
// 				crons = append(crons, v.Config["cron"].(string))
// 			}
// 		}
// 	}

// 	scheduleID := fmt.Sprintf("%s_%s_schedule", pipelineUID, releaseUID)

// 	handle := s.temporalClient.ScheduleClient().GetHandle(ctx, scheduleID)
// 	_ = handle.Delete(ctx)

// 	if len(crons) > 0 {

// 		param := &worker.SchedulePipelineWorkflowParam{
// 			Namespace:          ns,
// 			PipelineID:         pipelineID,
// 			PipelineUID:        pipelineUID,
// 			PipelineReleaseID:  pipelineReleaseID,
// 			PipelineReleaseUID: releaseUID,
// 		}
// 		_, err := s.temporalClient.ScheduleClient().Create(ctx, client.ScheduleOptions{
// 			ID: scheduleID,
// 			Spec: client.ScheduleSpec{
// 				CronExpressions: crons,
// 			},
// 			Action: &client.ScheduleWorkflowAction{
// 				Args:      []any{param},
// 				ID:        scheduleID,
// 				Workflow:  "SchedulePipelineWorkflow",
// 				TaskQueue: worker.TaskQueue,
// 				RetryPolicy: &temporal.RetryPolicy{
// 					MaximumAttempts: 1,
// 				},
// 			},
// 		})

// 		if err != nil {
// 			return err
// 		}
// 	}

//		return nil
//	}
type SchedulePipelineWorkflowParam struct {
	UID     uuid.UUID
	EventID string
}
