package base

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/lestrrat-go/jsref/provider"
	"go.uber.org/zap"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
	"gopkg.in/yaml.v3"

	jsoniter "github.com/json-iterator/go"
	temporalclient "go.temporal.io/sdk/client"

	"github.com/instill-ai/pipeline-backend/pkg/component/internal/jsonref"
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
	"github.com/instill-ai/pipeline-backend/pkg/external"

	pb "github.com/instill-ai/protogen-go/pipeline/pipeline/v1beta"
)

const conditionJSON = `
{
	"uiOrder": 1,
	"shortDescription": "config whether the component will be executed or skipped",
	"type": "string",
	"upstreamTypes": ["value", "template"]
}
`

// InstillExtension is the extension for the component.
type InstillExtension struct {
	jsoniter.DummyExtension
}

// UpdateStructDescriptor updates the struct descriptor for the component.
func (e *InstillExtension) UpdateStructDescriptor(structDescriptor *jsoniter.StructDescriptor) {

	// We use kebab-case for JSON keys in component input and output, while
	// vendors sometimes use camelCase or snake_case in requests and responses.
	// This often requires us to write two structs and convert between them.
	// However, most of the time, they can share the same struct with only
	// different tags. `jsoniter` is a good tool that can help us manage using
	// different tags to encode/decode JSON. Here, we implement an extension
	// where the JSON encoder/decoder will first use the `instill` tag as the
	// JSON key; if no `instill` tag is present, it will use the default `json`
	// tag. We'll use jsoniter in `ConvertFromStructpb()` and
	// `ConvertToStructpb()`.

	for _, v := range structDescriptor.Fields {
		t := v.Field.Tag()
		if instill, ok := t.Lookup("instill"); ok {
			v.FromNames = []string{instill}
			v.ToNames = []string{instill}
		}
	}
}

func init() {
	jsoniter.RegisterExtension(&InstillExtension{})
}

// IComponent is the interface that wraps the basic component methods.
// All component need to implement this interface.
type IComponent interface {
	GetDefinitionID() string
	GetDefinitionUID() uuid.UUID
	GetLogger() *zap.Logger

	LoadDefinition(definitionJSON, setupJSON, tasksJSON []byte, eventJSONBytes []byte, additionalJSONBytes map[string][]byte) error

	// Note: Some content in the definition JSON schema needs to be generated
	// by sysVars or component setting.
	GetDefinition(sysVars map[string]any, compConfig *ComponentConfig) (*pb.ComponentDefinition, error)

	// CreateExecution takes a ComponentExecution that can be used to compose
	// the core component behaviour with the particular business logic in the
	// implementation.
	CreateExecution(base ComponentExecution) (IExecution, error)
	Test(sysVars map[string]any, config *structpb.Struct) error

	IsSecretField(target string) bool
	SupportsOAuth() bool

	// Note: These functions are for the pipeline run-on-event feature,
	// which is still experimental and may change at any time.

	// RegisterEvent registers an event handler for the component. It performs
	// two main tasks:
	// 1. Registers a webhook URL with the vendor service if required
	// 2. Generates a identifier for the event registration that will be used to
	//    route incoming events to the correct pipeline
	//
	// The identifier returned by this method will be stored in backend and used
	// later to match incoming webhook events with their corresponding pipeline.
	RegisterEvent(ctx context.Context, settings *RegisterEventSettings) (identifier []Identifier, err error)

	// UnregisterEvent unregisters an event handler for the component.
	UnregisterEvent(ctx context.Context, settings *UnregisterEventSettings, identifier []Identifier) error

	// IdentifyEvent identifies the event and returns the identifiers.
	IdentifyEvent(ctx context.Context, rawEvent *RawEvent) (identifierResult *IdentifierResult, err error)

	// ParseEvent parses the raw event and returns a parsed event.
	// The parsed event contains:
	// - parsed message: the processed event data
	// - webhook response: any response that should be sent back to the webhook caller
	ParseEvent(ctx context.Context, rawEvent *RawEvent) (parsedEvent *ParsedEvent, err error)

	UsageHandlerCreator() UsageHandlerCreator
}

// Identifier is the identifier for the event.
type Identifier map[string]any

// EventSettings is the settings for the event.
type EventSettings struct {
	// TODO: The Config field represents the component configuration settings
	// while Setup contains initialization parameters. Consider renaming to
	// a more explicit name for clarity.
	Config format.Value
	Setup  format.Value
}

// RegisterEventSettings is the settings for registering an event.
type RegisterEventSettings struct {
	EventSettings
	RegistrationUID uuid.UUID
}

// UnregisterEventSettings is the settings for unregistering an event.
type UnregisterEventSettings struct {
	EventSettings
}

// RawEvent is the raw event from the webhook.
type RawEvent struct {
	EventSettings
	Header  map[string][]string
	Message format.Value
}

// ParsedEvent is the parsed event from the raw event.
type ParsedEvent struct {
	ParsedMessage format.Value
	Response      format.Value
}

// IdentifierResult is the result of identifying an event.
type IdentifierResult struct {
	SkipTrigger bool
	Identifiers []Identifier
	Response    format.Value
}

// Component implements the common component methods.
type Component struct {
	Logger          *zap.Logger
	NewUsageHandler UsageHandlerCreator

	definition   *pb.ComponentDefinition
	secretFields []string

	BinaryFetcher  external.BinaryFetcher
	TemporalClient temporalclient.Client
}

// IdentifyEvent is not implemented for the base component.
func (c *Component) IdentifyEvent(ctx context.Context, rawEvent *RawEvent) (identifierResult *IdentifierResult, err error) {
	return nil, fmt.Errorf("not implemented")
}

// ParseEvent is not implemented for the base component.
func (c *Component) ParseEvent(ctx context.Context, rawEvent *RawEvent) (*ParsedEvent, error) {
	return nil, fmt.Errorf("not implemented")
}

// RegisterEvent is not implemented for the base component.
func (c *Component) RegisterEvent(context.Context, *RegisterEventSettings) ([]Identifier, error) {
	return nil, fmt.Errorf("not implemented")
}

// UnregisterEvent is not implemented for the base component.
func (c *Component) UnregisterEvent(context.Context, *UnregisterEventSettings, []Identifier) error {
	return fmt.Errorf("not implemented")
}

func convertDataSpecToCompSpec(dataSpec *structpb.Struct) (*structpb.Struct, error) {

	compSpec := proto.Clone(dataSpec).(*structpb.Struct)
	if _, ok := compSpec.Fields["const"]; ok {
		return compSpec, nil
	}

	// Handle composite schemas (anyOf/allOf/oneOf)
	for _, target := range []string{"allOf", "anyOf", "oneOf"} {
		if fields, ok := compSpec.Fields[target]; ok {
			for idx, schema := range fields.GetListValue().GetValues() {
				converted, err := convertDataSpecToCompSpec(schema.GetStructValue())
				if err != nil {
					return nil, err
				}
				fields.GetListValue().Values[idx] = structpb.NewStructValue(converted)
			}
		}
	}

	if compSpec.Fields["type"] != nil && compSpec.Fields["type"].GetStringValue() == "object" {
		// Always add required field for object type if missing
		if _, ok := compSpec.Fields["required"]; !ok {
			compSpec.Fields["required"] = structpb.NewListValue(&structpb.ListValue{Values: []*structpb.Value{}})
		}
	}
	if _, ok := compSpec.Fields["properties"]; ok {
		for k, v := range compSpec.Fields["properties"].GetStructValue().AsMap() {
			switch val := v.(type) {
			case map[string]any:
				s, err := structpb.NewStruct(val)
				if err != nil {
					return nil, err
				}
				converted, err := convertDataSpecToCompSpec(s)
				if err != nil {
					return nil, err
				}
				compSpec.Fields["properties"].GetStructValue().Fields[k] = structpb.NewStructValue(converted)
			case []interface{}:
				listValue := &structpb.ListValue{
					Values: make([]*structpb.Value, len(val)),
				}
				for i, item := range val {
					value, err := structpb.NewValue(item)
					if err != nil {
						return nil, err
					}
					listValue.Values[i] = value
				}
				compSpec.Fields["properties"].GetStructValue().Fields[k] = structpb.NewListValue(listValue)
			case string, bool, float64, int64:
				value, err := structpb.NewValue(val)
				if err != nil {
					return nil, err
				}
				compSpec.Fields["properties"].GetStructValue().Fields[k] = value
			default:
				return nil, fmt.Errorf("unsupported type: %T", v)
			}
		}
	}
	if _, ok := compSpec.Fields["patternProperties"]; ok {
		for k, v := range compSpec.Fields["patternProperties"].GetStructValue().AsMap() {
			switch val := v.(type) {
			case map[string]any:
				s, err := structpb.NewStruct(val)
				if err != nil {
					return nil, err
				}
				converted, err := convertDataSpecToCompSpec(s)
				if err != nil {
					return nil, err
				}
				compSpec.Fields["patternProperties"].GetStructValue().Fields[k] = structpb.NewStructValue(converted)
			case string, bool, float64, int64:
				value, err := structpb.NewValue(val)
				if err != nil {
					return nil, err
				}
				compSpec.Fields["patternProperties"].GetStructValue().Fields[k] = value
			default:
				return nil, fmt.Errorf("unsupported type: %T", v)
			}
		}
	}
	for _, target := range []string{"allOf", "anyOf", "oneOf"} {
		if _, ok := compSpec.Fields[target]; ok {
			for idx, item := range compSpec.Fields[target].GetListValue().AsSlice() {
				s, err := structpb.NewStruct(item.(map[string]any))
				if err != nil {
					return nil, err
				}
				converted, err := convertDataSpecToCompSpec(s)
				if err != nil {
					return nil, err
				}
				compSpec.Fields[target].GetListValue().Values[idx] = structpb.NewStructValue(converted)
			}
		}
	}

	if _, ok := compSpec.Fields["uiOrder"]; !ok {
		compSpec.Fields["uiOrder"] = structpb.NewNumberValue(0)
	}

	return compSpec, nil
}

const taskPrefix = "TASK_"

// TaskIDToTitle builds a Task title from its ID. This is used when the `title`
// key in the task definition isn't present.
func TaskIDToTitle(id string) string {
	title := strings.ReplaceAll(id, taskPrefix, "")
	title = strings.ReplaceAll(title, "_", " ")
	return cases.Title(language.English).String(title)
}

const eventPrefix = "EVENT_"

// EventIDToTitle builds a Event title from its ID. This is used when the `title`
// key in the task definition isn't present.
func EventIDToTitle(id string) string {
	title := strings.ReplaceAll(id, eventPrefix, "")
	title = strings.ReplaceAll(title, "_", " ")
	return cases.Title(language.English).String(title)
}

func generateComponentTaskCards(tasks []string, taskStructs map[string]*structpb.Struct) []*pb.ComponentTask {
	taskCards := make([]*pb.ComponentTask, 0, len(tasks))
	for _, k := range tasks {
		if v, ok := taskStructs[k]; ok {
			title := v.Fields["title"].GetStringValue()
			if title == "" {
				title = TaskIDToTitle(k)
			}

			description := taskStructs[k].Fields["shortDescription"].GetStringValue()
			if description == "" {
				description = v.Fields["description"].GetStringValue()
			}

			taskCards = append(taskCards, &pb.ComponentTask{
				Name:        k,
				Title:       title,
				Description: description,
			})
		}
	}

	return taskCards
}

func generateComponentEventCards(events []string, eventStructs map[string]*structpb.Struct) []*pb.ComponentEvent {
	eventCards := make([]*pb.ComponentEvent, 0, len(events))
	for _, k := range events {
		if v, ok := eventStructs[k]; ok {
			title := v.Fields["title"].GetStringValue()
			if title == "" {
				title = TaskIDToTitle(k)
			}

			description := eventStructs[k].Fields["shortDescription"].GetStringValue()
			if description == "" {
				description = v.Fields["description"].GetStringValue()
			}

			eventCards = append(eventCards, &pb.ComponentEvent{
				Name:        k,
				Title:       title,
				Description: description,
			})
		}
	}
	return eventCards
}

func generateComponentSpec(title string, tasks []*pb.ComponentTask, taskStructs map[string]*structpb.Struct) (*structpb.Struct, error) {
	var err error
	componentSpec := &structpb.Struct{Fields: map[string]*structpb.Value{}}
	componentSpec.Fields["title"] = structpb.NewStringValue(fmt.Sprintf("%s Component", title))
	componentSpec.Fields["type"] = structpb.NewStringValue("object")

	oneOfList := &structpb.ListValue{
		Values: []*structpb.Value{},
	}
	for _, task := range tasks {
		taskName := task.Name

		oneOf := &structpb.Struct{Fields: map[string]*structpb.Value{}}
		oneOf.Fields["type"] = structpb.NewStringValue("object")
		oneOf.Fields["properties"] = structpb.NewStructValue(&structpb.Struct{Fields: make(map[string]*structpb.Value)})

		oneOf.Fields["properties"].GetStructValue().Fields["task"], err = structpb.NewValue(map[string]any{
			"const": task.Name,
			"title": task.Title,
		})
		if err != nil {
			return nil, err
		}

		if taskStructs[taskName].Fields["description"].GetStringValue() != "" {
			oneOf.Fields["properties"].GetStructValue().Fields["task"].GetStructValue().Fields["description"] = structpb.NewStringValue(taskStructs[taskName].Fields["description"].GetStringValue())
		}

		if task.Description != "" {
			oneOf.Fields["properties"].GetStructValue().Fields["task"].GetStructValue().Fields["shortDescription"] = structpb.NewStringValue(task.Description)
		}
		taskJSONStruct := proto.Clone(taskStructs[taskName]).(*structpb.Struct).Fields["input"].GetStructValue()

		compInputStruct, err := convertDataSpecToCompSpec(taskJSONStruct)
		if err != nil {
			return nil, fmt.Errorf("task %s: %s error: %+v", title, task, err)
		}

		condition := &structpb.Struct{}
		err = protojson.Unmarshal([]byte(conditionJSON), condition)
		if err != nil {
			panic(err)
		}
		oneOf.Fields["properties"].GetStructValue().Fields["condition"] = structpb.NewStructValue(condition)
		oneOf.Fields["properties"].GetStructValue().Fields["input"] = structpb.NewStructValue(compInputStruct)
		if taskStructs[taskName].Fields["metadata"] != nil {
			metadataStruct := proto.Clone(taskStructs[taskName]).(*structpb.Struct).Fields["metadata"].GetStructValue()
			oneOf.Fields["properties"].GetStructValue().Fields["metadata"] = structpb.NewStructValue(metadataStruct)
		}

		// oneOf
		oneOfList.Values = append(oneOfList.Values, structpb.NewStructValue(oneOf))
	}

	componentSpec.Fields["oneOf"] = structpb.NewListValue(oneOfList)

	if err != nil {
		return nil, err
	}

	return componentSpec, nil
}

// EventJSON is the JSON for the event.
type EventJSON map[string]Event

// Event is the event for the component.
type Event struct {
	Title           string `json:"title"`
	Description     string `json:"description"`
	ConfigSchema    any    `json:"configSchema"`
	MessageSchema   any    `json:"messageSchema"`
	MessageExamples []any  `json:"messageExamples"`
}

func generateEventSpecs(eventJSONBytes []byte) (map[string]*pb.EventSpecification, error) {

	specs := map[string]*pb.EventSpecification{}
	var j EventJSON
	err := json.Unmarshal(eventJSONBytes, &j)
	if err != nil {
		return nil, err
	}
	for t, e := range j {
		c, err := json.Marshal(e.ConfigSchema)
		if err != nil {
			return nil, err
		}
		pbConfigSchema := &structpb.Struct{}
		err = protojson.Unmarshal(c, pbConfigSchema)
		if err != nil {
			return nil, err
		}

		m, err := json.Marshal(e.MessageSchema)
		if err != nil {
			return nil, err
		}
		pbMessageSchema := &structpb.Struct{}
		err = protojson.Unmarshal(m, pbMessageSchema)
		if err != nil {
			return nil, err
		}
		pbMessageExamples := make([]*structpb.Struct, 0, len(e.MessageExamples))
		for _, ex := range e.MessageExamples {
			pbMessageExample := &structpb.Struct{}
			exs, err := json.Marshal(ex)
			if err != nil {
				return nil, err
			}
			err = protojson.Unmarshal(exs, pbMessageExample)
			if err != nil {
				return nil, err
			}
			pbMessageExamples = append(pbMessageExamples, pbMessageExample)
		}
		specs[t] = &pb.EventSpecification{
			Title:           e.Title,
			Description:     e.Description,
			ConfigSchema:    pbConfigSchema,
			MessageSchema:   pbMessageSchema,
			MessageExamples: pbMessageExamples,
		}
	}
	return specs, nil
}

func generateDataSpecs(tasks map[string]*structpb.Struct) (map[string]*pb.DataSpecification, error) {

	specs := map[string]*pb.DataSpecification{}
	for k := range tasks {
		spec := &pb.DataSpecification{}
		taskJSONStruct := proto.Clone(tasks[k]).(*structpb.Struct)
		spec.Input = taskJSONStruct.Fields["input"].GetStructValue()
		spec.Output = taskJSONStruct.Fields["output"].GetStructValue()
		specs[k] = spec
	}

	return specs, nil
}

func loadTasks(availableTasks []string, tasksJSONBytes []byte) ([]*pb.ComponentTask, map[string]*structpb.Struct, error) {

	taskStructs := map[string]*structpb.Struct{}
	var err error

	tasksJSONMap := map[string]map[string]any{}
	err = json.Unmarshal(tasksJSONBytes, &tasksJSONMap)
	if err != nil {
		return nil, nil, err
	}

	for _, t := range availableTasks {
		if v, ok := tasksJSONMap[t]; ok {
			taskStructs[t], err = structpb.NewStruct(v)
			if err != nil {
				return nil, nil, err
			}

		}
	}
	tasks := generateComponentTaskCards(availableTasks, taskStructs)
	return tasks, taskStructs, nil
}

func loadEvents(availableEvents []string, eventsJSONBytes []byte) ([]*pb.ComponentEvent, error) {
	eventStructs := map[string]*structpb.Struct{}
	var err error

	eventsJSONMap := map[string]map[string]any{}
	err = json.Unmarshal(eventsJSONBytes, &eventsJSONMap)
	if err != nil {
		return nil, err
	}

	for _, t := range availableEvents {
		if v, ok := eventsJSONMap[t]; ok {
			eventStructs[t], err = structpb.NewStruct(v)
			if err != nil {
				return nil, err
			}

		}
	}
	events := generateComponentEventCards(availableEvents, eventStructs)
	return events, nil
}

// ConvertFromStructpb converts from structpb.Struct to a struct
func ConvertFromStructpb(from *structpb.Struct, to any) error {
	inputJSON, err := protojson.Marshal(from)
	if err != nil {
		return err
	}

	err = jsoniter.Unmarshal(inputJSON, to)
	if err != nil {
		return err
	}
	return nil
}

// ConvertToStructpb converts from a struct to structpb.Struct
func ConvertToStructpb(from any) (*structpb.Struct, error) {
	to := &structpb.Struct{}

	outputJSON, err := jsoniter.Marshal(from)
	if err != nil {
		return nil, err
	}

	err = protojson.Unmarshal(outputJSON, to)
	if err != nil {
		return nil, err
	}
	return to, nil
}

// RenderJSON renders the JSON for the component.
func RenderJSON(tasksJSONBytes []byte, additionalJSONBytes map[string][]byte) ([]byte, error) {
	var err error
	mp := provider.NewMap()
	for k, v := range additionalJSONBytes {
		var i any
		err = json.Unmarshal(v, &i)
		if err != nil {
			return nil, err
		}
		err = mp.Set(k, i)
		if err != nil {
			return nil, err
		}
	}
	res := jsonref.New()
	err = res.AddProvider(mp)
	if err != nil {
		return nil, err
	}

	var tasksJSON any
	err = json.Unmarshal(tasksJSONBytes, &tasksJSON)
	if err != nil {
		return nil, err
	}

	result, err := res.Resolve(tasksJSON, "", jsonref.WithRecursiveResolution(true))
	if err != nil {
		return nil, err
	}
	renderedTasksJSON, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	return renderedTasksJSON, nil

}

// GetDefinitionID returns the component definition ID.
func (c *Component) GetDefinitionID() string {
	return c.definition.Id
}

// GetDefinitionUID returns the component definition UID.
func (c *Component) GetDefinitionUID() uuid.UUID {
	return uuid.FromStringOrNil(c.definition.Uid)
}

// GetLogger returns the component's logger. If it hasn't been initialized, a
// no-op logger is returned.
func (c *Component) GetLogger() *zap.Logger {
	if c.Logger == nil {
		return zap.NewNop()
	}

	return c.Logger
}

// GetDefinition returns the component definition.
func (c *Component) GetDefinition(sysVars map[string]any, compConfig *ComponentConfig) (*pb.ComponentDefinition, error) {

	var err error
	definition := proto.Clone(c.definition).(*pb.ComponentDefinition)
	definition.Spec.ComponentSpecification, err = convertFormatFields(definition.Spec.ComponentSpecification, true)
	if err != nil {
		return nil, err
	}
	for k := range definition.Spec.DataSpecifications {
		definition.Spec.DataSpecifications[k].Input, err = convertFormatFields(definition.Spec.DataSpecifications[k].Input, false)
		if err != nil {
			return nil, err
		}
		definition.Spec.DataSpecifications[k].Output, err = convertFormatFields(definition.Spec.DataSpecifications[k].Output, false)
		if err != nil {
			return nil, err
		}
	}
	for k := range definition.Spec.EventSpecifications {
		definition.Spec.EventSpecifications[k].ConfigSchema, err = convertFormatFields(definition.Spec.EventSpecifications[k].ConfigSchema, false)
		if err != nil {
			return nil, err
		}
		definition.Spec.EventSpecifications[k].MessageSchema, err = convertFormatFields(definition.Spec.EventSpecifications[k].MessageSchema, false)
		if err != nil {
			return nil, err
		}
	}

	return definition, nil
}

func convertYAMLToJSON(yamlBytes []byte) ([]byte, error) {
	if yamlBytes == nil {
		return nil, nil
	}
	var d any
	err := yaml.Unmarshal(yamlBytes, &d)
	if err != nil {
		return nil, err
	}
	return json.Marshal(d)
}

// LoadDefinition loads the component definition, setup, tasks, events and additional JSON files.
// The definition files are currently loaded together but could be refactored to load separately.
func (c *Component) LoadDefinition(definitionYAMLBytes, setupYAMLBytes, tasksYAMLBytes, eventsYAMLBytes []byte, additionalYAMLBytes map[string][]byte) error {

	var err error
	definitionJSONBytes, err := convertYAMLToJSON(definitionYAMLBytes)
	if err != nil {
		return err
	}
	setupJSONBytes, err := convertYAMLToJSON(setupYAMLBytes)
	if err != nil {
		return err
	}
	eventsJSONBytes, err := convertYAMLToJSON(eventsYAMLBytes)
	if err != nil {
		return err
	}
	tasksJSONBytes, err := convertYAMLToJSON(tasksYAMLBytes)
	if err != nil {
		return err
	}
	additionalJSONBytes := map[string][]byte{}
	for k, v := range additionalYAMLBytes {
		v, err = convertYAMLToJSON(v)
		if err != nil {
			return err
		}
		additionalJSONBytes[k] = v
	}

	var definitionJSON any

	c.secretFields = []string{}

	err = json.Unmarshal(definitionJSONBytes, &definitionJSON)
	if err != nil {
		return err
	}
	renderedTasksJSON, err := RenderJSON(tasksJSONBytes, additionalJSONBytes)
	if err != nil {
		return err
	}

	availableTasks := []string{}
	for _, availableTask := range definitionJSON.(map[string]any)["availableTasks"].([]any) {
		availableTasks = append(availableTasks, availableTask.(string))
	}

	tasks, taskStructs, err := loadTasks(availableTasks, renderedTasksJSON)
	if err != nil {
		return err
	}

	c.definition = &pb.ComponentDefinition{}
	err = protojson.UnmarshalOptions{DiscardUnknown: true}.Unmarshal(definitionJSONBytes, c.definition)
	if err != nil {
		return err
	}

	c.definition.Name = fmt.Sprintf("component-definitions/%s", c.definition.Id)
	c.definition.Tasks = tasks
	if c.definition.Spec == nil {
		c.definition.Spec = &pb.ComponentDefinition_Spec{}
	}
	c.definition.Spec.ComponentSpecification, err = generateComponentSpec(c.definition.Title, tasks, taskStructs)
	if err != nil {
		return err
	}

	raw := &structpb.Struct{}
	err = protojson.Unmarshal(definitionJSONBytes, raw)
	if err != nil {
		return err
	}

	// TODO: Avoid using structpb traversal here.
	if setupJSONBytes != nil {
		setup := &structpb.Struct{}
		err = protojson.Unmarshal(setupJSONBytes, setup)
		if err != nil {
			return err
		}
		setup, err := c.refineResourceSpec(setup)
		if err != nil {
			return err
		}
		configPropStruct := &structpb.Struct{Fields: map[string]*structpb.Value{}}
		configPropStruct.Fields["setup"] = structpb.NewStructValue(setup)
		c.definition.Spec.ComponentSpecification.Fields["properties"] = structpb.NewStructValue(configPropStruct)

	}

	if eventsJSONBytes != nil {
		availableEvents := []string{}
		for _, availableEvent := range definitionJSON.(map[string]any)["availableEvents"].([]any) {
			availableEvents = append(availableEvents, availableEvent.(string))
		}
		events, err := loadEvents(availableEvents, eventsJSONBytes)
		if err != nil {
			return err
		}
		c.definition.Events = events
		c.definition.Spec.EventSpecifications, err = generateEventSpecs(eventsJSONBytes)
		if err != nil {
			return err
		}
	}

	c.definition.Spec.DataSpecifications, err = generateDataSpecs(taskStructs)
	if err != nil {
		return err
	}

	c.initSecretField(c.definition)

	return nil

}

func (c *Component) refineResourceSpec(resourceSpec *structpb.Struct) (*structpb.Struct, error) {

	spec := proto.Clone(resourceSpec).(*structpb.Struct)
	if _, ok := spec.Fields["shortDescription"]; !ok {
		spec.Fields["shortDescription"] = structpb.NewStringValue(spec.Fields["description"].GetStringValue())
	}

	if _, ok := spec.Fields["properties"]; ok {
		for k, v := range spec.Fields["properties"].GetStructValue().AsMap() {
			switch val := v.(type) {
			case map[string]any:
				s, err := structpb.NewStruct(val)
				if err != nil {
					return nil, err
				}
				converted, err := c.refineResourceSpec(s)
				if err != nil {
					return nil, err
				}
				spec.Fields["properties"].GetStructValue().Fields[k] = structpb.NewStructValue(converted)
			case string, bool, float64, int64:
				// Handle primitive types directly
				value, err := structpb.NewValue(val)
				if err != nil {
					return nil, err
				}
				spec.Fields["properties"].GetStructValue().Fields[k] = value
			default:
				return nil, fmt.Errorf("unsupported type: %T", v)
			}
		}
	}
	if _, ok := spec.Fields["patternProperties"]; ok {
		for k, v := range spec.Fields["patternProperties"].GetStructValue().AsMap() {
			switch val := v.(type) {
			case map[string]any:
				s, err := structpb.NewStruct(val)
				if err != nil {
					return nil, err
				}
				converted, err := c.refineResourceSpec(s)
				if err != nil {
					return nil, err
				}
				spec.Fields["patternProperties"].GetStructValue().Fields[k] = structpb.NewStructValue(converted)
			case string, bool, float64, int64:
				// Handle primitive types directly
				value, err := structpb.NewValue(val)
				if err != nil {
					return nil, err
				}
				spec.Fields["patternProperties"].GetStructValue().Fields[k] = value
			default:
				return nil, fmt.Errorf("unsupported type: %T", v)
			}
		}
	}
	for _, target := range []string{"allOf", "anyOf", "oneOf"} {
		if _, ok := spec.Fields[target]; ok {
			for idx, item := range spec.Fields[target].GetListValue().AsSlice() {
				switch val := item.(type) {
				case map[string]any:
					s, err := structpb.NewStruct(val)
					if err != nil {
						return nil, err
					}
					converted, err := c.refineResourceSpec(s)
					if err != nil {
						return nil, err
					}
					spec.Fields[target].GetListValue().AsSlice()[idx] = structpb.NewStructValue(converted)
				case string, bool, float64, int64:
					// Handle primitive types directly
					value, err := structpb.NewValue(val)
					if err != nil {
						return nil, err
					}
					spec.Fields[target].GetListValue().AsSlice()[idx] = value
				default:
					return nil, fmt.Errorf("unsupported type: %T", item)
				}
			}
		}
	}

	return spec, nil
}

// IsSecretField checks if the target field is secret field
func (c *Component) IsSecretField(target string) bool {
	for _, field := range c.secretFields {
		if target == field {
			return true
		}
	}
	return false
}

// ListSecretFields lists the secret fields by definition id
func (c *Component) ListSecretFields() ([]string, error) {
	return c.secretFields, nil
}

func (c *Component) initSecretField(def *pb.ComponentDefinition) {
	if c.secretFields == nil {
		c.secretFields = []string{}
	}
	secretFields := []string{}
	setup := def.Spec.GetComponentSpecification().GetFields()["properties"].GetStructValue().GetFields()["setup"].GetStructValue()
	secretFields = c.traverseSecretField(setup.GetFields()["properties"], "", secretFields)
	if l, ok := setup.GetFields()["oneOf"]; ok {
		for _, v := range l.GetListValue().Values {
			secretFields = c.traverseSecretField(v.GetStructValue().GetFields()["properties"], "", secretFields)
		}
	}
	c.secretFields = secretFields
}

func (c *Component) traverseSecretField(input *structpb.Value, prefix string, secretFields []string) []string {
	for key, v := range input.GetStructValue().GetFields() {
		if isSecret, ok := v.GetStructValue().GetFields()["instillSecret"]; ok {
			if isSecret.GetBoolValue() || isSecret.GetStringValue() == "true" {
				secretFields = append(secretFields, fmt.Sprintf("%s%s", prefix, key))
			}
		}
		if tp, ok := v.GetStructValue().GetFields()["type"]; ok {
			if tp.GetStringValue() == "object" {
				if l, ok := v.GetStructValue().GetFields()["oneOf"]; ok {
					for _, v := range l.GetListValue().Values {
						secretFields = c.traverseSecretField(v.GetStructValue().GetFields()["properties"], fmt.Sprintf("%s%s.", prefix, key), secretFields)
					}
				}
				secretFields = c.traverseSecretField(v.GetStructValue().GetFields()["properties"], fmt.Sprintf("%s%s.", prefix, key), secretFields)
			}

		}
	}

	return secretFields
}

// SupportsOAuth is false by default. To support OAuth, component
// implementations must be composed with `OAuthComponent`.
func (c *Component) SupportsOAuth() bool {
	return false
}

// UsageHandlerCreator returns a function to initialize a UsageHandler. If the
// component doesn't have such function initialized, a no-op usage handler
// creator is returned.
func (c *Component) UsageHandlerCreator() UsageHandlerCreator {
	if c.NewUsageHandler == nil {
		return NewNoopUsageHandler
	}

	return c.NewUsageHandler
}

// Test is not implemented for the base component.
func (c *Component) Test(sysVars map[string]any, setup *structpb.Struct) error {
	return nil
}

// ReadFromGlobalConfig looks up a component credential field from a secret map
// that comes from the environment variable configuration.
//
// Config parameters are defined with snake_case, but the
// environment variable configuration loader replaces underscores by dots,
// so we can't use the parameter key directly.
// TODO using camelCase in configuration fields would fix this issue.
func ReadFromGlobalConfig(key string, secrets map[string]any) string {
	sanitized := strings.ReplaceAll(key, "-", "")
	if v, ok := secrets[sanitized].(string); ok {
		return v
	}

	return ""
}

// ComponentConfig is the config for the component.
type ComponentConfig struct {
	Task  string
	Input map[string]any
	Setup map[string]any
}
