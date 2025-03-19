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

	// Due to the size of the workflow memory data, it needs to be persisted in
	// MinIO (or another datastore) in order to be shared between different
	// processes.
	// TODO [INS-7456,INS-7457]: As we further refactor the workflow code, we
	// might consider using an alternative datastore to remove duplication or
	// not to hold data blobs in the memory, which would remove the need to
	// commit and fetch the worfklow memory data.
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

func wfmFilePath(workflowID string) string {
	return fmt.Sprintf("pipeline-runs/wfm/%s.json", workflowID)
}

// CleanupWorkflowMemory removes the worfklow memory data from the in-memory
// map and from the remote datastore.
func (s *Store) CleanupWorkflowMemory(ctx context.Context, userUID uuid.UUID, workflowID string) error {
	s.PurgeWorkflowMemory(workflowID)
	if err := s.minioClient.DeleteFile(ctx, userUID, wfmFilePath(workflowID)); err != nil {
		return fmt.Errorf("deleting workflow memory data from minIO: %w", err)
	}

	return nil
}

// PurgeWorkflowMemory removes the worfklow memory data from the in-memory map.
func (s *Store) PurgeWorkflowMemory(workflowID string) {
	s.workflows.Delete(workflowID)
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

	err = s.minioClient.UploadPrivateFileBytes(ctx, minio.UploadFileBytesParam{
		UserUID:       userUID,
		FilePath:      wfmFilePath(wfm.id),
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
	b, err := s.minioClient.GetFile(ctx, userUID, wfmFilePath(workflowID))
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
