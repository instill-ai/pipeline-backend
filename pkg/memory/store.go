package memory

import (
	"context"
	"fmt"
	"sync"

	"github.com/instill-ai/pipeline-backend/pkg/data"
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
	"github.com/instill-ai/pipeline-backend/pkg/pubsub"
)

// Store is used to communicate the pipeline trigger information (inputs,
// outputs, status) between processes (server, worker) during or after workflow
// execution.
type Store interface {
	NewWorkflowMemory(_ context.Context, workflowID string, batchSize int) (WorkflowMemory, error)
	GetWorkflowMemory(_ context.Context, workflowID string) (WorkflowMemory, error)
	PurgeWorkflowMemory(_ context.Context, workflowID string) error

	SendWorkflowStatusEvent(_ context.Context, workflowID string, event pubsub.Event) error
}

type store struct {
	workflows sync.Map
	publisher pubsub.EventPublisher
}

// NewStore returns an initialized memory stored.
func NewStore(pub pubsub.EventPublisher) Store {
	return &store{
		workflows: sync.Map{},
		publisher: pub,
	}
}

func (s *store) NewWorkflowMemory(_ context.Context, workflowID string, batchSize int) (WorkflowMemory, error) {
	wfmData := make([]format.Value, batchSize)
	for idx := range batchSize {
		m := data.Map{
			string(PipelineVariable):   data.Map{},
			string(PipelineSecret):     data.Map{},
			string(PipelineConnection): data.Map{},
			string(PipelineOutput):     data.Map{},
		}

		wfmData[idx] = m
	}

	s.workflows.Store(workflowID, &workflowMemory{
		mu:              sync.Mutex{},
		id:              workflowID,
		data:            wfmData,
		publishWFStatus: s.SendWorkflowStatusEvent,
	})

	wfm, ok := s.workflows.Load(workflowID)
	if !ok {
		return nil, fmt.Errorf("workflow memory not found")
	}

	return wfm.(WorkflowMemory), nil
}

func (s *store) GetWorkflowMemory(_ context.Context, workflowID string) (WorkflowMemory, error) {
	wfm, ok := s.workflows.Load(workflowID)
	if !ok {
		return nil, fmt.Errorf("workflow memory not found")
	}

	return wfm.(WorkflowMemory), nil
}

func (s *store) PurgeWorkflowMemory(_ context.Context, workflowID string) error {
	s.workflows.Delete(workflowID)
	return nil
}

func (s *store) SendWorkflowStatusEvent(ctx context.Context, workflowID string, event pubsub.Event) error {
	channel := pubsub.WorkflowStatusTopic(workflowID)
	if err := s.publisher.PublishEvent(ctx, channel, event); err != nil {
		return fmt.Errorf("publishing event: %w", err)
	}

	return nil
}
