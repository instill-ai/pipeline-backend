package memory

import (
	"context"
	"fmt"
	"sync"

	"github.com/gofrs/uuid"

	"github.com/instill-ai/pipeline-backend/pkg/data"
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
	"github.com/instill-ai/pipeline-backend/pkg/pubsub"
	"github.com/instill-ai/x/minio"
)

// Store is used to communicate the pipeline trigger information (inputs,
// outputs, status) between processes (server, worker) during or after workflow
// execution.
type Store struct {
	workflows sync.Map
	publisher pubsub.EventPublisher

	// This memory store implementation stores the workflow memory as a blob in
	// MinIO. This is a first step in refactoring the pipeline trigger workflow
	// that allows us to decouple the server and worker processes. We will
	// further refactor this code to remove the communication through MinIO
	// blobs.
	minioClient minio.Client
}

// NewStore returns an initialized memory store.
func NewStore(pub pubsub.EventPublisher, minioClient minio.Client) *Store {
	return &Store{
		workflows:   sync.Map{},
		publisher:   pub,
		minioClient: minioClient,
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

// WorkflowMemoryExpiryRuleTag will be used to automatically delete the
// workflow memory blobs.
const WorkflowMemoryExpiryRuleTag = "workflow-memory"

// CommitWorkflowData persists the workflow memory data in a datastore.
func (s *Store) CommitWorkflowData(ctx context.Context, userUID uuid.UUID, wfm *WorkflowMemory) error {
	b, err := data.Encode(data.Array(wfm.data))
	if err != nil {
		return fmt.Errorf("serializing workflow memory data: %w", err)
	}

	_, _, err = s.minioClient.UploadFileBytes(ctx, &minio.UploadFileBytesParam{
		UserUID:       userUID,
		FilePath:      fmt.Sprintf("pipeline-runs/wfm/%s.json", wfm.id),
		FileBytes:     b,
		ExpiryRuleTag: WorkflowMemoryExpiryRuleTag,
	})
	if err != nil {
		return fmt.Errorf("uploading workflow memory to MinIO: %w", err)
	}

	return nil
}

// FetchWorkflowMemory loads the workflow data into memory. This relies on
// the workflow memory being stored via the CommitWorkflowData method,
// and is used when separate processes want to share the workflow data.
func (s *Store) FetchWorkflowMemory(ctx context.Context, userUID uuid.UUID, workflowID string) (*WorkflowMemory, error) {
	objectName := fmt.Sprintf("pipeline-runs/wfm/%s.json", workflowID)
	b, err := s.minioClient.GetFile(ctx, userUID, objectName)
	if err != nil {
		return nil, fmt.Errorf("downloading workflow memory from MinIO: %w", err)
	}

	v, err := data.Decode(b)
	if err != nil {
		return nil, fmt.Errorf("deserializing workflow memory data: %w", err)
	}

	wData, ok := v.(data.Array)
	if !ok {
		return nil, fmt.Errorf("workflow memory data should be an array")
	}

	var wfm *WorkflowMemory
	if v, existsInMem := s.workflows.Load(workflowID); existsInMem {
		wfm = v.(*WorkflowMemory)
	} else {
		wfm, err = s.NewWorkflowMemory(ctx, workflowID, len(wData))
		if err != nil {
			return nil, fmt.Errorf("creating workflow memory: %w", err)
		}
	}

	wfm.mu.Lock()
	defer wfm.mu.Unlock()
	wfm.data = wData

	return wfm, nil
}
