package memory

import (
	"context"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/instill-ai/pipeline-backend/pkg/data"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"google.golang.org/protobuf/encoding/protojson"
)

type PipelineStatusType string
type PipelineDataType string
type ComponentStatusType string
type ComponentDataType string

const (
	PipelineVariable   PipelineDataType = "variable"
	PipelineSecret     PipelineDataType = "secret"
	PipelineConnection PipelineDataType = "connection"
	PipelineOutput     PipelineDataType = "output"

	// We preserve the `PipelineOutputTemplate` in memory to re-render the
	// results.
	PipelineOutputTemplate PipelineDataType = "_output"
)

const (
	ComponentStatusStarted   ComponentStatusType = "started"
	ComponentStatusSkipped   ComponentStatusType = "skipped"
	ComponentStatusErrored   ComponentStatusType = "errored"
	ComponentStatusCompleted ComponentStatusType = "completed"
)

const (
	PipelineStatusStarted   PipelineStatusType = "started"
	PipelineStatusErrored   PipelineStatusType = "errored"
	PipelineStatusCompleted PipelineStatusType = "completed"
)

const (
	ComponentDataInput   ComponentDataType = "input"
	ComponentDataOutput  ComponentDataType = "output"
	ComponentDataElement ComponentDataType = "element"
	ComponentDataSetup   ComponentDataType = "setup"
	ComponentDataError   ComponentDataType = "error"
	ComponentDataStatus  ComponentDataType = "status"
)

type MemoryStore interface {
	NewWorkflowMemory(ctx context.Context, workflowID string, recipe *datamodel.Recipe, batchSize int) (workflow WorkflowMemory, err error)
	GetWorkflowMemory(ctx context.Context, workflowID string) (workflow WorkflowMemory, err error)
	PurgeWorkflowMemory(ctx context.Context, workflowID string) (err error)

	SendWorkflowStatusEvent(ctx context.Context, workflowID string, event Event) (err error)
}

type WorkflowMemory interface {
	Set(ctx context.Context, batchIdx int, key string, value data.Value) (err error)
	Get(ctx context.Context, batchIdx int, path string) (value data.Value, err error)

	InitComponent(ctx context.Context, batchIdx int, componentID string)
	SetComponentData(ctx context.Context, batchIdx int, componentID string, t ComponentDataType, value data.Value) (err error)
	GetComponentData(ctx context.Context, batchIdx int, componentID string, t ComponentDataType) (value data.Value, err error)
	SetComponentStatus(ctx context.Context, batchIdx int, componentID string, t ComponentStatusType, value bool) (err error)
	GetComponentStatus(ctx context.Context, batchIdx int, componentID string, t ComponentStatusType) (value bool, err error)
	SetPipelineData(ctx context.Context, batchIdx int, t PipelineDataType, value data.Value) (err error)
	GetPipelineData(ctx context.Context, batchIdx int, t PipelineDataType) (value data.Value, err error)
	SetComponentErrorMessage(ctx context.Context, batchIdx int, componentID string, msg string) (err error)

	EnableStreaming()
	IsStreaming() bool
	SendEvent(ctx context.Context, event *Event)
	ListenEvent(ctx context.Context) chan *Event

	GetBatchSize() int
	SetRecipe(*datamodel.Recipe)
	GetRecipe() *datamodel.Recipe
}

type ComponentStatus struct {
	Started   bool `json:"started"`
	Completed bool `json:"completed"`
	Skipped   bool `json:"skipped"`
}

type memoryStore struct {
	workflows sync.Map
}

type workflowMemory struct {
	mu        sync.Mutex
	ID        string
	Data      []data.Value
	Recipe    *datamodel.Recipe
	Streaming bool
	channel   chan *Event
}

type ComponentEventType string
type PipelineEventType string

type Event struct {
	Event string `json:"event"`
	Data  any    `json:"data"`
}

type PipelineEventData struct {
	UpdateTime time.Time `json:"updateTime"`
	BatchIndex int       `json:"batchIndex"`

	Status map[PipelineStatusType]bool `json:"status"`
}

type PipelineStatusUpdatedEventData struct {
	PipelineEventData
}

type PipelineOutputUpdatedEventData struct {
	PipelineEventData
	Output any `json:"output"`
}

type PipelineErrorUpdatedEventData struct {
	PipelineEventData
	Error MessageError `json:"error"`
}

type ComponentEventData struct {
	UpdateTime  time.Time `json:"updateTime"`
	ComponentID string    `json:"componentID"`
	BatchIndex  int       `json:"batchIndex"`

	Status map[ComponentStatusType]bool `json:"status"`
}

type ComponentStatusUpdatedEventData struct {
	ComponentEventData
}

type ComponentInputUpdatedEventData struct {
	ComponentEventData
	Input any `json:"input"`
}
type ComponentOutputUpdatedEventData struct {
	ComponentEventData
	Output any `json:"output"`
}

type ComponentErrorUpdatedEventData struct {
	ComponentEventData
	Error MessageError `json:"error"`
}
type MessageError struct {
	Message string `json:"message"`
}

const (
	PipelineStatusUpdated PipelineEventType = "PIPELINE_STATUS_UPDATED"
	PipelineOutputUpdated PipelineEventType = "PIPELINE_OUTPUT_UPDATED"
	PipelineErrorUpdated  PipelineEventType = "PIPELINE_ERROR_UPDATED"
	PipelineClosed        PipelineEventType = "PIPELINE_CLOSED"

	ComponentStatusUpdated ComponentEventType = "COMPONENT_STATUS_UPDATED"
	ComponentInputUpdated  ComponentEventType = "COMPONENT_INPUT_UPDATED"
	ComponentOutputUpdated ComponentEventType = "COMPONENT_OUTPUT_UPDATED"
	ComponentErrorUpdated  ComponentEventType = "COMPONENT_ERROR_UPDATED"
)

func init() {
	gob.Register(ComponentStatusUpdatedEventData{})
	gob.Register(ComponentInputUpdatedEventData{})
	gob.Register(ComponentOutputUpdatedEventData{})
	gob.Register(ComponentErrorUpdatedEventData{})
	gob.Register(MessageError{})
	gob.Register(PipelineStatusUpdatedEventData{})
	gob.Register(PipelineOutputUpdatedEventData{})
	gob.Register(PipelineErrorUpdatedEventData{})
}

func NewMemoryStore() MemoryStore {
	return &memoryStore{
		workflows: sync.Map{},
	}
}

func (ms *memoryStore) NewWorkflowMemory(ctx context.Context, workflowID string, r *datamodel.Recipe, batchSize int) (workflow WorkflowMemory, err error) {
	wfmData := make([]data.Value, batchSize)
	for idx := range batchSize {
		m := data.NewMap(map[string]data.Value{
			string(PipelineVariable):   data.NewMap(nil),
			string(PipelineSecret):     data.NewMap(nil),
			string(PipelineConnection): data.NewMap(nil),
			string(PipelineOutput):     data.NewMap(nil),
		})

		wfmData[idx] = m
	}

	ms.workflows.Store(workflowID, &workflowMemory{
		mu:      sync.Mutex{},
		ID:      workflowID,
		Data:    wfmData,
		Recipe:  r,
		channel: make(chan *Event),
	})

	wfm, ok := ms.workflows.Load(workflowID)
	if !ok {
		return nil, fmt.Errorf("workflow memory not found")
	}

	return wfm.(WorkflowMemory), nil
}

func (ms *memoryStore) GetWorkflowMemory(ctx context.Context, workflowID string) (workflow WorkflowMemory, err error) {
	wfm, ok := ms.workflows.Load(workflowID)
	if !ok {
		return nil, fmt.Errorf("workflow memory not found")
	}

	return wfm.(WorkflowMemory), nil
}

func (ms *memoryStore) PurgeWorkflowMemory(ctx context.Context, workflowID string) (err error) {
	ms.workflows.Delete(workflowID)
	return nil
}

func (ms *memoryStore) SendWorkflowStatusEvent(ctx context.Context, workflowID string, event Event) (err error) {
	wfm, err := ms.GetWorkflowMemory(ctx, workflowID)
	if err != nil {
		return err
	}
	wfm.SendEvent(ctx, &event)
	return nil
}

func (wfm *workflowMemory) EnableStreaming() {
	wfm.Streaming = true
}

func (wfm *workflowMemory) IsStreaming() bool {
	return wfm.Streaming
}

func (wfm *workflowMemory) InitComponent(ctx context.Context, batchIdx int, componentID string) {
	wfm.mu.Lock()
	defer wfm.mu.Unlock()

	compMemory := data.NewMap(
		map[string]data.Value{
			string(ComponentDataInput):  data.NewMap(nil),
			string(ComponentDataOutput): data.NewMap(nil),
			string(ComponentDataSetup):  data.NewMap(nil),
			string(ComponentDataError): data.NewMap(
				map[string]data.Value{
					"message": data.NewString(""),
				},
			),
			string(ComponentDataStatus): data.NewMap(
				map[string]data.Value{
					"started":   data.NewBoolean(false),
					"skipped":   data.NewBoolean(false),
					"errored":   data.NewBoolean(false),
					"completed": data.NewBoolean(false),
				},
			),
		},
	)
	wfm.Data[batchIdx].(*data.Map).Fields[componentID] = compMemory
}

func (wfm *workflowMemory) SetComponentData(ctx context.Context, batchIdx int, componentID string, t ComponentDataType, value data.Value) (err error) {
	wfm.mu.Lock()
	defer wfm.mu.Unlock()

	if _, ok := wfm.Data[batchIdx].(*data.Map).Fields[componentID]; !ok {
		return fmt.Errorf("component %s not exist", componentID)
	}
	wfm.Data[batchIdx].(*data.Map).Fields[componentID].(*data.Map).Fields[string(t)] = value

	if t == ComponentDataInput {
		if err := wfm.sendComponentEvent(ctx, batchIdx, componentID, ComponentInputUpdated); err != nil {
			return err
		}
	} else if t == ComponentDataOutput {
		if err := wfm.sendComponentEvent(ctx, batchIdx, componentID, ComponentOutputUpdated); err != nil {
			return err
		}
	}
	return nil
}
func (wfm *workflowMemory) GetComponentData(ctx context.Context, batchIdx int, componentID string, t ComponentDataType) (value data.Value, err error) {
	wfm.mu.Lock()
	defer wfm.mu.Unlock()

	if _, ok := wfm.Data[batchIdx].(*data.Map).Fields[componentID]; !ok {
		return nil, fmt.Errorf("component %s not exist", componentID)
	}
	return wfm.Data[batchIdx].(*data.Map).Fields[componentID].(*data.Map).Fields[string(t)], nil
}

func (wfm *workflowMemory) SetComponentStatus(ctx context.Context, batchIdx int, componentID string, t ComponentStatusType, value bool) (err error) {
	wfm.mu.Lock()
	defer wfm.mu.Unlock()

	if _, ok := wfm.Data[batchIdx].(*data.Map).Fields[componentID]; !ok {
		return fmt.Errorf("component %s not exist", componentID)
	}
	wfm.Data[batchIdx].(*data.Map).Fields[componentID].(*data.Map).Fields["status"].(*data.Map).Fields[string(t)] = data.NewBoolean(value)

	if err := wfm.sendComponentEvent(ctx, batchIdx, componentID, ComponentStatusUpdated); err != nil {
		return err
	}

	return nil
}
func (wfm *workflowMemory) SetComponentErrorMessage(ctx context.Context, batchIdx int, componentID string, msg string) (err error) {
	wfm.mu.Lock()
	defer wfm.mu.Unlock()

	if _, ok := wfm.Data[batchIdx].(*data.Map).Fields[componentID]; !ok {
		return fmt.Errorf("component %s not exist", componentID)
	}
	wfm.Data[batchIdx].(*data.Map).Fields[componentID].(*data.Map).Fields["error"].(*data.Map).Fields["message"] = data.NewString(msg)

	if err := wfm.sendComponentEvent(ctx, batchIdx, componentID, ComponentErrorUpdated); err != nil {
		return err
	}

	return nil
}
func (wfm *workflowMemory) GetComponentStatus(ctx context.Context, batchIdx int, componentID string, t ComponentStatusType) (value bool, err error) {
	wfm.mu.Lock()
	defer wfm.mu.Unlock()

	if _, ok := wfm.Data[batchIdx].(*data.Map).Fields[componentID]; !ok {
		return false, fmt.Errorf("component %s not exist", componentID)
	}
	return wfm.Data[batchIdx].(*data.Map).Fields[componentID].(*data.Map).Fields["status"].(*data.Map).Fields[string(t)].(*data.Boolean).GetBoolean(), nil
}

func (wfm *workflowMemory) SetPipelineData(ctx context.Context, batchIdx int, t PipelineDataType, value data.Value) (err error) {
	wfm.mu.Lock()
	defer wfm.mu.Unlock()

	wfm.Data[batchIdx].(*data.Map).Fields[string(t)] = value

	if wfm.Streaming {
		// TODO: simplify struct conversion
		s, err := value.ToStructValue()
		if err != nil {
			return err
		}
		b, err := protojson.Marshal(s)
		if err != nil {
			return err
		}
		var data map[string]any
		err = json.Unmarshal(b, &data)
		if err != nil {
			return err
		}
		event := Event{}
		if t == PipelineOutput {
			event.Event = string(PipelineOutputUpdated)
			event.Data = PipelineOutputUpdatedEventData{
				PipelineEventData: PipelineEventData{
					UpdateTime: time.Now(),
					BatchIndex: batchIdx,
					Status: map[PipelineStatusType]bool{
						PipelineStatusStarted:   true,
						PipelineStatusErrored:   false,
						PipelineStatusCompleted: false,
					},
				},
				Output: data,
			}
			wfm.SendEvent(ctx, &event)
		}

	}

	return nil
}

func (wfm *workflowMemory) GetPipelineData(ctx context.Context, batchIdx int, t PipelineDataType) (value data.Value, err error) {
	wfm.mu.Lock()
	defer wfm.mu.Unlock()

	if v, ok := wfm.Data[batchIdx].(*data.Map).Fields[string(t)]; !ok {
		return nil, fmt.Errorf("%s not exist", string(t))
	} else {
		return v, nil
	}
}

func (wfm *workflowMemory) Set(ctx context.Context, batchIdx int, key string, value data.Value) (err error) {
	wfm.mu.Lock()
	defer wfm.mu.Unlock()

	wfm.Data[batchIdx].(*data.Map).Fields[key] = value
	return nil
}

func (wfm *workflowMemory) Get(ctx context.Context, batchIdx int, path string) (memory data.Value, err error) {
	wfm.mu.Lock()
	defer wfm.mu.Unlock()

	return wfm.Data[batchIdx].Get(path)

}

func (wfm *workflowMemory) SendEvent(ctx context.Context, event *Event) {
	wfm.channel <- event
}
func (wfm *workflowMemory) ListenEvent(ctx context.Context) chan *Event {
	return wfm.channel
}

func (wfm *workflowMemory) GetBatchSize() int {
	return len(wfm.Data)
}

func (wfm *workflowMemory) SetRecipe(r *datamodel.Recipe) {
	wfm.Recipe = r
}

func (wfm *workflowMemory) GetRecipe() *datamodel.Recipe {
	return wfm.Recipe
}

func (wfm *workflowMemory) getComponentEventData(_ context.Context, batchIdx int, componentID string) ComponentEventData {
	// TODO: simplify struct conversion
	st := wfm.Data[batchIdx].(*data.Map).Fields[componentID].(*data.Map).Fields["status"].(*data.Map)
	started := st.Fields[string(ComponentStatusStarted)].(*data.Boolean).GetBoolean()
	skipped := st.Fields[string(ComponentStatusSkipped)].(*data.Boolean).GetBoolean()
	errored := st.Fields[string(ComponentStatusErrored)].(*data.Boolean).GetBoolean()
	completed := st.Fields[string(ComponentStatusCompleted)].(*data.Boolean).GetBoolean()

	return ComponentEventData{
		UpdateTime:  time.Now(),
		ComponentID: componentID,
		BatchIndex:  batchIdx,
		Status: map[ComponentStatusType]bool{
			ComponentStatusStarted:   started,
			ComponentStatusSkipped:   skipped,
			ComponentStatusErrored:   errored,
			ComponentStatusCompleted: completed,
		},
	}
}

func (wfm *workflowMemory) sendComponentEvent(ctx context.Context, batchIdx int, componentID string, t ComponentEventType) (err error) {

	if wfm.Streaming {
		var event *Event
		switch t {
		case ComponentInputUpdated:
			value := wfm.Data[batchIdx].(*data.Map).Fields[componentID].(*data.Map).Fields[string(ComponentDataInput)]

			// TODO: simplify struct conversion
			s, err := value.ToStructValue()
			if err != nil {
				return err
			}
			b, err := protojson.Marshal(s)
			if err != nil {
				return err
			}
			var data any
			err = json.Unmarshal(b, &data)
			if err != nil {
				return err
			}

			event = &Event{
				Event: string(ComponentInputUpdated),
				Data: ComponentInputUpdatedEventData{
					ComponentEventData: wfm.getComponentEventData(ctx, batchIdx, componentID),
					Input:              data,
				},
			}

		case ComponentOutputUpdated:

			value := wfm.Data[batchIdx].(*data.Map).Fields[componentID].(*data.Map).Fields[string(ComponentDataOutput)]

			// TODO: simplify struct conversion
			s, err := value.ToStructValue()
			if err != nil {
				return err
			}
			b, err := protojson.Marshal(s)
			if err != nil {
				return err
			}
			var data map[string]any
			err = json.Unmarshal(b, &data)
			if err != nil {
				return err
			}

			event = &Event{
				Event: string(ComponentOutputUpdated),
				Data: ComponentOutputUpdatedEventData{
					ComponentEventData: wfm.getComponentEventData(ctx, batchIdx, componentID),
					Output:             data,
				},
			}

		case ComponentErrorUpdated:
			message := wfm.Data[batchIdx].(*data.Map).Fields[componentID].(*data.Map).Fields["error"].(*data.Map).Fields["message"].(*data.String)
			event = &Event{
				Event: string(ComponentErrorUpdated),
				Data: ComponentErrorUpdatedEventData{
					ComponentEventData: wfm.getComponentEventData(ctx, batchIdx, componentID),
					Error: MessageError{
						Message: message.GetString(),
					},
				},
			}

		case ComponentStatusUpdated:
			event = &Event{
				Event: string(ComponentStatusUpdated),
				Data: ComponentStatusUpdatedEventData{
					ComponentEventData: wfm.getComponentEventData(ctx, batchIdx, componentID),
				},
			}
		}

		if event != nil {
			wfm.SendEvent(ctx, event)
		}

	}
	return nil
}
