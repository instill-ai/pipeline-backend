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
type Store struct {
	workflows sync.Map
	publisher pubsub.EventPublisher
}

// NewStore returns an initialized memory store.
func NewStore(pub pubsub.EventPublisher) *Store {
	return &Store{
		workflows: sync.Map{},
		publisher: pub,
	}
}

func (s *Store) NewWorkflowMemory(_ context.Context, workflowID string, batchSize int) (*WorkflowMemory, error) {
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

	s.workflows.Store(workflowID, &WorkflowMemory{
		mu:              sync.Mutex{},
		id:              workflowID,
		data:            wfmData,
		publishWFStatus: s.SendWorkflowStatusEvent,
	})

	wfm, ok := s.workflows.Load(workflowID)
	if !ok {
		return nil, fmt.Errorf("workflow memory not found")
	}

	return wfm.(*WorkflowMemory), nil
}

func (s *Store) GetWorkflowMemory(_ context.Context, workflowID string) (*WorkflowMemory, error) {
	wfm, ok := s.workflows.Load(workflowID)
	if !ok {
		return nil, fmt.Errorf("workflow memory not found")
	}

	return wfm.(*WorkflowMemory), nil
}

func (s *Store) PurgeWorkflowMemory(_ context.Context, workflowID string) error {
	s.workflows.Delete(workflowID)
	return nil
}

func (s *Store) SendWorkflowStatusEvent(ctx context.Context, workflowID string, event pubsub.Event) error {
	channel := pubsub.WorkflowStatusTopic(workflowID)
	if err := s.publisher.PublishEvent(ctx, channel, event); err != nil {
		return fmt.Errorf("publishing event: %w", err)
	}

	return nil
}
