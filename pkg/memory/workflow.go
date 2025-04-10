package memory

import (
	"context"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/instill-ai/pipeline-backend/pkg/data"
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
	"github.com/instill-ai/pipeline-backend/pkg/data/path"
	"github.com/instill-ai/pipeline-backend/pkg/pubsub"
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
	// Templates originate from the recipe and are used to render the actual
	// input or setup data.
	ComponentDataInputTemplate ComponentDataType = "input-template"
	ComponentDataSetupTemplate ComponentDataType = "setup-template"

	ComponentDataInput   ComponentDataType = "input"
	ComponentDataOutput  ComponentDataType = "output"
	ComponentDataElement ComponentDataType = "element"
	ComponentDataSetup   ComponentDataType = "setup"
	ComponentDataError   ComponentDataType = "error"
	ComponentDataStatus  ComponentDataType = "status"
)

// WorkflowMemory holds the information of a pipeline trigger during or after
// the workflow execution.
type WorkflowMemory struct {
	mu        sync.Mutex
	id        string
	data      []format.Value
	streaming bool

	publishWFStatus func(_ context.Context, topic string, _ pubsub.Event) error
}

type ComponentEventType string
type PipelineEventType string

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

func (wfm *WorkflowMemory) EnableStreaming() {
	wfm.streaming = true
}

func (wfm *WorkflowMemory) IsStreaming() bool {
	return wfm.streaming
}

func (wfm *WorkflowMemory) InitComponent(ctx context.Context, batchIdx int, componentID string) {
	wfm.mu.Lock()
	defer wfm.mu.Unlock()

	compMemory := data.Map{

		string(ComponentDataInputTemplate): data.Map{},
		string(ComponentDataSetupTemplate): data.Map{},
		string(ComponentDataInput):         data.Map{},
		string(ComponentDataOutput):        data.Map{},
		string(ComponentDataSetup):         data.Map{},
		string(ComponentDataError): data.Map{
			"message": data.NewString(""),
		},
		string(ComponentDataStatus): data.Map{
			"started":   data.NewBoolean(false),
			"skipped":   data.NewBoolean(false),
			"errored":   data.NewBoolean(false),
			"completed": data.NewBoolean(false),
		},
	}
	wfm.data[batchIdx].(data.Map)[componentID] = compMemory
}

func (wfm *WorkflowMemory) SetComponentData(ctx context.Context, batchIdx int, componentID string, t ComponentDataType, value format.Value) (err error) {
	wfm.mu.Lock()
	defer wfm.mu.Unlock()

	if _, ok := wfm.data[batchIdx].(data.Map)[componentID]; !ok {
		return fmt.Errorf("component %s not exist", componentID)
	}
	wfm.data[batchIdx].(data.Map)[componentID].(data.Map)[string(t)] = value

	// TODO: For binary data fields, we should return a URL to access the blob instead of the raw data
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
func (wfm *WorkflowMemory) GetComponentData(ctx context.Context, batchIdx int, componentID string, t ComponentDataType) (value format.Value, err error) {
	wfm.mu.Lock()
	defer wfm.mu.Unlock()

	if _, ok := wfm.data[batchIdx].(data.Map)[componentID]; !ok {
		return nil, fmt.Errorf("component %s not exist", componentID)
	}
	return wfm.data[batchIdx].(data.Map)[componentID].(data.Map)[string(t)], nil
}

func (wfm *WorkflowMemory) SetComponentStatus(ctx context.Context, batchIdx int, componentID string, t ComponentStatusType, value bool) (err error) {
	wfm.mu.Lock()
	defer wfm.mu.Unlock()

	if _, ok := wfm.data[batchIdx].(data.Map)[componentID]; !ok {
		return fmt.Errorf("component %s not exist", componentID)
	}
	wfm.data[batchIdx].(data.Map)[componentID].(data.Map)["status"].(data.Map)[string(t)] = data.NewBoolean(value)

	if err := wfm.sendComponentEvent(ctx, batchIdx, componentID, ComponentStatusUpdated); err != nil {
		return err
	}

	return nil
}
func (wfm *WorkflowMemory) SetComponentErrorMessage(ctx context.Context, batchIdx int, componentID string, msg string) (err error) {
	wfm.mu.Lock()
	defer wfm.mu.Unlock()

	if _, ok := wfm.data[batchIdx].(data.Map)[componentID]; !ok {
		return fmt.Errorf("component %s not exist", componentID)
	}
	wfm.data[batchIdx].(data.Map)[componentID].(data.Map)["error"].(data.Map)["message"] = data.NewString(msg)

	if err := wfm.sendComponentEvent(ctx, batchIdx, componentID, ComponentErrorUpdated); err != nil {
		return err
	}

	return nil
}

func (wfm *WorkflowMemory) GetComponentErrorMessage(ctx context.Context, batchIdx int, componentID string) (string, error) {
	v, err := wfm.GetComponentData(ctx, batchIdx, componentID, ComponentDataError)
	if err != nil {
		return "", fmt.Errorf("fetching component data: %w", err)
	}

	asStruct, err := v.ToStructValue()
	if err != nil {
		return "", fmt.Errorf("converting error data to struct: %w", err)
	}

	return asStruct.GetStructValue().AsMap()["message"].(string), nil
}

func (wfm *WorkflowMemory) GetComponentStatus(ctx context.Context, batchIdx int, componentID string, t ComponentStatusType) (v bool, err error) {
	wfm.mu.Lock()
	defer wfm.mu.Unlock()

	if _, ok := wfm.data[batchIdx].(data.Map)[componentID]; !ok {
		return false, fmt.Errorf("component %s not exist", componentID)
	}
	return wfm.data[batchIdx].(data.Map)[componentID].(data.Map)["status"].(data.Map)[string(t)].(format.Boolean).Boolean(), nil
}

func (wfm *WorkflowMemory) SetPipelineData(ctx context.Context, batchIdx int, t PipelineDataType, value format.Value) (err error) {
	wfm.mu.Lock()
	defer wfm.mu.Unlock()

	wfm.data[batchIdx].(data.Map)[string(t)] = value

	if !wfm.streaming {
		return nil
	}

	if t != PipelineOutput {
		return nil
	}

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

	event := pubsub.Event{
		Name: string(PipelineOutputUpdated),
		Data: PipelineOutputUpdatedEventData{
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
		},
	}

	return wfm.publishWFStatus(ctx, wfm.id, event)
}

func (wfm *WorkflowMemory) GetPipelineData(ctx context.Context, batchIdx int, t PipelineDataType) (value format.Value, err error) {
	wfm.mu.Lock()
	defer wfm.mu.Unlock()

	if v, ok := wfm.data[batchIdx].(data.Map)[string(t)]; !ok {
		return nil, fmt.Errorf("%s not exist", string(t))
	} else {
		return v, nil
	}
}

func (wfm *WorkflowMemory) Set(ctx context.Context, batchIdx int, key string, value format.Value) (err error) {
	wfm.mu.Lock()
	defer wfm.mu.Unlock()

	wfm.data[batchIdx].(data.Map)[key] = value
	return nil
}

func (wfm *WorkflowMemory) Get(ctx context.Context, batchIdx int, p string) (memory format.Value, err error) {
	wfm.mu.Lock()
	defer wfm.mu.Unlock()

	pt, err := path.NewPath(p)
	if err != nil {
		return nil, err
	}
	return wfm.data[batchIdx].Get(pt)

}

func (wfm *WorkflowMemory) GetBatchSize() int {
	return len(wfm.data)
}

func (wfm *WorkflowMemory) getComponentEventData(_ context.Context, batchIdx int, componentID string) ComponentEventData {
	// TODO: simplify struct conversion
	st := wfm.data[batchIdx].(data.Map)[componentID].(data.Map)["status"].(data.Map)
	started := st[string(ComponentStatusStarted)].(format.Boolean).Boolean()
	skipped := st[string(ComponentStatusSkipped)].(format.Boolean).Boolean()
	errored := st[string(ComponentStatusErrored)].(format.Boolean).Boolean()
	completed := st[string(ComponentStatusCompleted)].(format.Boolean).Boolean()

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

func (wfm *WorkflowMemory) sendComponentEvent(ctx context.Context, batchIdx int, componentID string, t ComponentEventType) (err error) {
	if !wfm.streaming {
		return nil
	}

	var event pubsub.Event
	switch t {
	case ComponentInputUpdated:
		value := wfm.data[batchIdx].(data.Map)[componentID].(data.Map)[string(ComponentDataInput)]

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

		event = pubsub.Event{
			Name: string(ComponentInputUpdated),
			Data: ComponentInputUpdatedEventData{
				ComponentEventData: wfm.getComponentEventData(ctx, batchIdx, componentID),
				Input:              data,
			},
		}

	case ComponentOutputUpdated:

		value := wfm.data[batchIdx].(data.Map)[componentID].(data.Map)[string(ComponentDataOutput)]

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

		event = pubsub.Event{
			Name: string(ComponentOutputUpdated),
			Data: ComponentOutputUpdatedEventData{
				ComponentEventData: wfm.getComponentEventData(ctx, batchIdx, componentID),
				Output:             data,
			},
		}

	case ComponentErrorUpdated:
		message := wfm.data[batchIdx].(data.Map)[componentID].(data.Map)["error"].(data.Map)["message"].(format.String)
		event = pubsub.Event{
			Name: string(ComponentErrorUpdated),
			Data: ComponentErrorUpdatedEventData{
				ComponentEventData: wfm.getComponentEventData(ctx, batchIdx, componentID),
				Error: MessageError{
					Message: message.String(),
				},
			},
		}

	case ComponentStatusUpdated:
		event = pubsub.Event{
			Name: string(ComponentStatusUpdated),
			Data: ComponentStatusUpdatedEventData{
				ComponentEventData: wfm.getComponentEventData(ctx, batchIdx, componentID),
			},
		}
	default:
		return nil
	}

	return wfm.publishWFStatus(ctx, wfm.id, event)
}
