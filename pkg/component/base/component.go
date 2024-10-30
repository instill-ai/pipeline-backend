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

	jsoniter "github.com/json-iterator/go"

	"github.com/instill-ai/pipeline-backend/pkg/component/internal/jsonref"

	pb "github.com/instill-ai/protogen-go/vdp/pipeline/v1beta"
)

const conditionJSON = `
{
	"type": "string",
	"instillUIOrder": 1,
	"instillShortDescription": "config whether the component will be executed or skipped",
	"instillAcceptFormats": ["string"],
    "instillUpstreamTypes": ["value", "template"]
}
`

type InstillExtension struct {
	jsoniter.DummyExtension
}

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
	GetTaskInputSchemas() map[string]string
	GetTaskOutputSchemas() map[string]string

	LoadDefinition(definitionJSON, setupJSON, tasksJSON []byte, additionalJSONBytes map[string][]byte) error

	// Note: Some content in the definition JSON schema needs to be generated
	// by sysVars or component setting.
	GetDefinition(sysVars map[string]any, compConfig *ComponentConfig) (*pb.ComponentDefinition, error)

	// CreateExecution takes a ComponentExecution that can be used to compose
	// the core component behaviour with the particular business logic in the
	// implmentation.
	CreateExecution(base ComponentExecution) (IExecution, error)
	Test(sysVars map[string]any, config *structpb.Struct) error

	IsSecretField(target string) bool
	SupportsOAuth() bool

	// Note: These two functions are for the pipeline run-on-event feature,
	// which is still experimental and may change at any time.
	HandleVerificationEvent(header map[string][]string, req *structpb.Struct, setup map[string]any) (isVerification bool, resp *structpb.Struct, err error)
	ParseEvent(ctx context.Context, req *structpb.Struct, setup map[string]any) (parsed *structpb.Struct, err error)

	UsageHandlerCreator() UsageHandlerCreator
}

// Component implements the common component methods.
type Component struct {
	Logger          *zap.Logger
	NewUsageHandler UsageHandlerCreator

	taskInputSchemas  map[string]string
	taskOutputSchemas map[string]string

	definition               *pb.ComponentDefinition
	secretFields             []string
	inputAcceptFormatsFields map[string]map[string][]string
	outputFormatsFields      map[string]map[string]string
}

func (c *Component) HandleVerificationEvent(header map[string][]string, req *structpb.Struct, setup map[string]any) (isVerification bool, resp *structpb.Struct, err error) {
	return false, nil, nil
}

func (c *Component) ParseEvent(ctx context.Context, req *structpb.Struct, setup map[string]any) (parsed *structpb.Struct, err error) {
	return req, nil
}

func convertDataSpecToCompSpec(dataSpec *structpb.Struct) (*structpb.Struct, error) {
	// var err error
	compSpec := proto.Clone(dataSpec).(*structpb.Struct)
	if _, ok := compSpec.Fields["const"]; ok {
		return compSpec, nil
	}

	isFreeform := checkFreeForm(compSpec)

	if _, ok := compSpec.Fields["type"]; !ok && !isFreeform {
		return nil, fmt.Errorf("type missing: %+v", compSpec)
	} else if _, ok := compSpec.Fields["instillUpstreamTypes"]; !ok && compSpec.Fields["type"].GetStringValue() == "object" {

		if _, ok := compSpec.Fields["instillUIOrder"]; !ok {
			compSpec.Fields["instillUIOrder"] = structpb.NewNumberValue(0)
		}
		if _, ok := compSpec.Fields["required"]; !ok {
			return nil, fmt.Errorf("required missing: %+v", compSpec)
		}
		if _, ok := compSpec.Fields["instillEditOnNodeFields"]; !ok {
			compSpec.Fields["instillEditOnNodeFields"] = compSpec.Fields["required"]
		}

		if _, ok := compSpec.Fields["properties"]; ok {
			for k, v := range compSpec.Fields["properties"].GetStructValue().AsMap() {
				s, err := structpb.NewStruct(v.(map[string]any))
				if err != nil {
					return nil, err
				}
				converted, err := convertDataSpecToCompSpec(s)
				if err != nil {
					return nil, err
				}
				compSpec.Fields["properties"].GetStructValue().Fields[k] = structpb.NewStructValue(converted)

			}
		}
		if _, ok := compSpec.Fields["patternProperties"]; ok {
			for k, v := range compSpec.Fields["patternProperties"].GetStructValue().AsMap() {
				s, err := structpb.NewStruct(v.(map[string]any))
				if err != nil {
					return nil, err
				}
				converted, err := convertDataSpecToCompSpec(s)
				if err != nil {
					return nil, err
				}
				compSpec.Fields["patternProperties"].GetStructValue().Fields[k] = structpb.NewStructValue(converted)

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

	} else {
		if _, ok := compSpec.Fields["instillUIOrder"]; !ok {
			compSpec.Fields["instillUIOrder"] = structpb.NewNumberValue(0)
		}
		original := proto.Clone(compSpec).(*structpb.Struct)
		delete(original.Fields, "title")
		delete(original.Fields, "description")
		delete(original.Fields, "instillShortDescription")
		delete(original.Fields, "instillAcceptFormats")
		delete(original.Fields, "instillUIOrder")
		delete(original.Fields, "instillUpstreamTypes")

		newCompSpec := &structpb.Struct{Fields: make(map[string]*structpb.Value)}

		newCompSpec.Fields["title"] = structpb.NewStringValue(compSpec.Fields["title"].GetStringValue())
		newCompSpec.Fields["description"] = structpb.NewStringValue(compSpec.Fields["description"].GetStringValue())
		if _, ok := compSpec.Fields["instillShortDescription"]; ok {
			newCompSpec.Fields["instillShortDescription"] = compSpec.Fields["instillShortDescription"]
		} else {
			newCompSpec.Fields["instillShortDescription"] = newCompSpec.Fields["description"]
		}
		newCompSpec.Fields["instillUIOrder"] = structpb.NewNumberValue(compSpec.Fields["instillUIOrder"].GetNumberValue())
		if compSpec.Fields["instillAcceptFormats"] != nil {
			newCompSpec.Fields["instillAcceptFormats"] = structpb.NewListValue(compSpec.Fields["instillAcceptFormats"].GetListValue())
		}
		newCompSpec.Fields["instillUpstreamTypes"] = structpb.NewListValue(compSpec.Fields["instillUpstreamTypes"].GetListValue())
		newCompSpec.Fields["anyOf"] = structpb.NewListValue(&structpb.ListValue{Values: []*structpb.Value{}})

		for _, v := range compSpec.Fields["instillUpstreamTypes"].GetListValue().GetValues() {
			if v.GetStringValue() == "value" {
				original.Fields["instillUpstreamType"] = v
				newCompSpec.Fields["anyOf"].GetListValue().Values = append(newCompSpec.Fields["anyOf"].GetListValue().Values, structpb.NewStructValue(original))
			}
			if v.GetStringValue() == "reference" {
				item, err := structpb.NewValue(
					map[string]any{
						"type":                "string",
						"pattern":             "^\\{.*\\}$",
						"instillUpstreamType": "reference",
					},
				)
				if err != nil {
					return nil, err
				}
				newCompSpec.Fields["anyOf"].GetListValue().Values = append(newCompSpec.Fields["anyOf"].GetListValue().Values, item)
			}
			if v.GetStringValue() == "template" {
				item, err := structpb.NewValue(
					map[string]any{
						"type":                "string",
						"instillUpstreamType": "template",
					},
				)
				if err != nil {
					return nil, err
				}
				newCompSpec.Fields["anyOf"].GetListValue().Values = append(newCompSpec.Fields["anyOf"].GetListValue().Values, item)
			}

		}

		compSpec = newCompSpec

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

func generateComponentTaskCards(tasks []string, taskStructs map[string]*structpb.Struct) []*pb.ComponentTask {
	taskCards := make([]*pb.ComponentTask, 0, len(tasks))
	for _, k := range tasks {
		if v, ok := taskStructs[k]; ok {
			title := v.Fields["title"].GetStringValue()
			if title == "" {
				title = TaskIDToTitle(k)
			}

			description := taskStructs[k].Fields["instillShortDescription"].GetStringValue()

			taskCards = append(taskCards, &pb.ComponentTask{
				Name:        k,
				Title:       title,
				Description: description,
			})
		}
	}

	return taskCards
}

func generateComponentSpec(title string, tasks []*pb.ComponentTask, taskStructs map[string]*structpb.Struct) (*structpb.Struct, error) {
	var err error
	componentSpec := &structpb.Struct{Fields: map[string]*structpb.Value{}}
	componentSpec.Fields["$schema"] = structpb.NewStringValue("http://json-schema.org/draft-07/schema#")
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
			oneOf.Fields["properties"].GetStructValue().Fields["task"].GetStructValue().Fields["instillShortDescription"] = structpb.NewStringValue(task.Description)
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

func formatDataSpec(dataSpec *structpb.Struct) (*structpb.Struct, error) {
	// var err error
	compSpec := proto.Clone(dataSpec).(*structpb.Struct)
	if compSpec == nil {
		return compSpec, nil
	}
	if compSpec.Fields == nil {
		compSpec.Fields = make(map[string]*structpb.Value)
		return compSpec, nil
	}
	if _, ok := compSpec.Fields["const"]; ok {
		return compSpec, nil
	}

	isFreeform := checkFreeForm(compSpec)

	if _, ok := compSpec.Fields["type"]; !ok && !isFreeform {
		return nil, fmt.Errorf("type missing: %+v", compSpec)
	} else if compSpec.Fields["type"].GetStringValue() == "array" {

		if _, ok := compSpec.Fields["instillUIOrder"]; !ok {
			compSpec.Fields["instillUIOrder"] = structpb.NewNumberValue(0)
		}

		converted, err := formatDataSpec(compSpec.Fields["items"].GetStructValue())
		if err != nil {
			return nil, err
		}
		compSpec.Fields["items"] = structpb.NewStructValue(converted)
	} else if compSpec.Fields["type"].GetStringValue() == "object" {

		if _, ok := compSpec.Fields["instillUIOrder"]; !ok {
			compSpec.Fields["instillUIOrder"] = structpb.NewNumberValue(0)
		}
		if _, ok := compSpec.Fields["required"]; !ok {
			return nil, fmt.Errorf("required missing: %+v", compSpec)
		}
		if _, ok := compSpec.Fields["instillEditOnNodeFields"]; !ok {
			compSpec.Fields["instillEditOnNodeFields"] = compSpec.Fields["required"]
		}

		if _, ok := compSpec.Fields["properties"]; ok {
			for k, v := range compSpec.Fields["properties"].GetStructValue().AsMap() {
				s, err := structpb.NewStruct(v.(map[string]any))
				if err != nil {
					return nil, err
				}
				converted, err := formatDataSpec(s)
				if err != nil {
					return nil, err
				}
				compSpec.Fields["properties"].GetStructValue().Fields[k] = structpb.NewStructValue(converted)

			}
		}
		if _, ok := compSpec.Fields["patternProperties"]; ok {
			for k, v := range compSpec.Fields["patternProperties"].GetStructValue().AsMap() {
				s, err := structpb.NewStruct(v.(map[string]any))
				if err != nil {
					return nil, err
				}
				converted, err := formatDataSpec(s)
				if err != nil {
					return nil, err
				}
				compSpec.Fields["patternProperties"].GetStructValue().Fields[k] = structpb.NewStructValue(converted)

			}
		}
		for _, target := range []string{"allOf", "anyOf", "oneOf"} {
			if _, ok := compSpec.Fields[target]; ok {
				for idx, item := range compSpec.Fields[target].GetListValue().AsSlice() {
					s, err := structpb.NewStruct(item.(map[string]any))
					if err != nil {
						return nil, err
					}
					converted, err := formatDataSpec(s)
					if err != nil {
						return nil, err
					}
					compSpec.Fields[target].GetListValue().AsSlice()[idx] = structpb.NewStructValue(converted)
				}
			}
		}

	} else {
		if _, ok := compSpec.Fields["instillUIOrder"]; !ok {
			compSpec.Fields["instillUIOrder"] = structpb.NewNumberValue(0)
		}

		newCompSpec := &structpb.Struct{Fields: make(map[string]*structpb.Value)}

		newCompSpec.Fields["type"] = structpb.NewStringValue(compSpec.Fields["type"].GetStringValue())
		newCompSpec.Fields["title"] = structpb.NewStringValue(compSpec.Fields["title"].GetStringValue())
		newCompSpec.Fields["description"] = structpb.NewStringValue(compSpec.Fields["description"].GetStringValue())
		if _, ok := newCompSpec.Fields["instillShortDescription"]; ok {
			newCompSpec.Fields["instillShortDescription"] = compSpec.Fields["instillShortDescription"]
		} else {
			newCompSpec.Fields["instillShortDescription"] = newCompSpec.Fields["description"]
		}
		newCompSpec.Fields["instillUIOrder"] = structpb.NewNumberValue(compSpec.Fields["instillUIOrder"].GetNumberValue())
		if compSpec.Fields["instillFormat"] != nil {
			newCompSpec.Fields["instillFormat"] = structpb.NewStringValue(compSpec.Fields["instillFormat"].GetStringValue())
		}

		compSpec = newCompSpec

	}
	return compSpec, nil
}

func generateDataSpecs(tasks map[string]*structpb.Struct) (map[string]*pb.DataSpecification, error) {

	specs := map[string]*pb.DataSpecification{}
	for k := range tasks {
		spec := &pb.DataSpecification{}
		var err error
		taskJSONStruct := proto.Clone(tasks[k]).(*structpb.Struct)
		spec.Input, err = formatDataSpec(taskJSONStruct.Fields["input"].GetStructValue())
		if err != nil {
			return nil, err
		}
		spec.Output, err = formatDataSpec(taskJSONStruct.Fields["output"].GetStructValue())
		if err != nil {
			return nil, err
		}
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
	err = res.AddProvider(provider.NewHTTP())
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

// For formats such as `*`, `semi-structured/*`, and `semi-structured/json` we
// treat them as freeform data. Thus, there is no need to set the `type` in the
// JSON schema.
func checkFreeForm(compSpec *structpb.Struct) bool {
	acceptFormats := compSpec.Fields["instillAcceptFormats"].GetListValue().AsSlice()

	formats := make([]any, 0, len(acceptFormats)+1) // This avoids reallocations when appending values to the slice.
	formats = append(formats, acceptFormats...)

	if instillFormat := compSpec.Fields["instillFormat"].GetStringValue(); instillFormat != "" {
		formats = append(formats, instillFormat)
	}
	if len(formats) == 0 {
		return true
	}

	for _, v := range formats {
		if v.(string) == "*" || v.(string) == "semi-structured/*" || v.(string) == "semi-structured/json" {
			return true
		}
	}

	return false
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
func (c *Component) GetDefinition(sysVars map[string]any, compConfig *ComponentConfig) (*pb.ComponentDefinition, error) {
	return c.definition, nil
}

func (c *Component) GetTaskInputSchemas() map[string]string {
	return c.taskInputSchemas
}
func (c *Component) GetTaskOutputSchemas() map[string]string {
	return c.taskOutputSchemas
}

// LoadDefinition loads the component definitions from json files
func (c *Component) LoadDefinition(definitionJSONBytes, setupJSONBytes, tasksJSONBytes []byte, additionalJSONBytes map[string][]byte) error {
	var err error
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

	c.taskInputSchemas = map[string]string{}
	c.taskOutputSchemas = map[string]string{}
	for k := range taskStructs {
		var s []byte
		s, err = protojson.Marshal(taskStructs[k].Fields["input"].GetStructValue())
		if err != nil {
			return err
		}
		c.taskInputSchemas[k] = string(s)

		s, err = protojson.Marshal(taskStructs[k].Fields["output"].GetStructValue())
		if err != nil {
			return err
		}
		c.taskOutputSchemas[k] = string(s)
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

	c.definition.Spec.DataSpecifications, err = generateDataSpecs(taskStructs)
	if err != nil {
		return err
	}

	c.initSecretField(c.definition)
	c.initInputAcceptFormatsFields()
	c.initOutputFormatsFields()

	return nil

}

func (c *Component) refineResourceSpec(resourceSpec *structpb.Struct) (*structpb.Struct, error) {

	spec := proto.Clone(resourceSpec).(*structpb.Struct)
	if _, ok := spec.Fields["instillShortDescription"]; !ok {
		spec.Fields["instillShortDescription"] = structpb.NewStringValue(spec.Fields["description"].GetStringValue())
	}

	if _, ok := spec.Fields["properties"]; ok {
		for k, v := range spec.Fields["properties"].GetStructValue().AsMap() {
			s, err := structpb.NewStruct(v.(map[string]any))
			if err != nil {
				return nil, err
			}
			converted, err := c.refineResourceSpec(s)
			if err != nil {
				return nil, err
			}
			spec.Fields["properties"].GetStructValue().Fields[k] = structpb.NewStructValue(converted)

		}
	}
	if _, ok := spec.Fields["patternProperties"]; ok {
		for k, v := range spec.Fields["patternProperties"].GetStructValue().AsMap() {
			s, err := structpb.NewStruct(v.(map[string]any))
			if err != nil {
				return nil, err
			}
			converted, err := c.refineResourceSpec(s)
			if err != nil {
				return nil, err
			}
			spec.Fields["patternProperties"].GetStructValue().Fields[k] = structpb.NewStructValue(converted)

		}
	}
	for _, target := range []string{"allOf", "anyOf", "oneOf"} {
		if _, ok := spec.Fields[target]; ok {
			for idx, item := range spec.Fields[target].GetListValue().AsSlice() {
				s, err := structpb.NewStruct(item.(map[string]any))
				if err != nil {
					return nil, err
				}
				converted, err := c.refineResourceSpec(s)
				if err != nil {
					return nil, err
				}
				spec.Fields[target].GetListValue().AsSlice()[idx] = structpb.NewStructValue(converted)
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

func (c *Component) ListInputAcceptFormatsFields() (map[string]map[string][]string, error) {
	return c.inputAcceptFormatsFields, nil
}

func (c *Component) initInputAcceptFormatsFields() {
	inputAcceptFormatsFields := map[string]map[string][]string{}

	for task, sch := range c.GetTaskInputSchemas() {
		inputAcceptFormatsFields[task] = map[string][]string{}
		input := &structpb.Struct{}
		_ = protojson.Unmarshal([]byte(sch), input)
		inputAcceptFormatsFields[task] = c.traverseInputAcceptFormatsFields(input.GetFields()["properties"], "", inputAcceptFormatsFields[task])
		if l, ok := input.GetFields()["oneOf"]; ok {
			for _, v := range l.GetListValue().Values {
				inputAcceptFormatsFields[task] = c.traverseInputAcceptFormatsFields(v.GetStructValue().GetFields()["properties"], "", inputAcceptFormatsFields[task])
			}
		}
		c.inputAcceptFormatsFields = inputAcceptFormatsFields
	}

}

func (c *Component) traverseInputAcceptFormatsFields(input *structpb.Value, prefix string, inputAcceptFormatsFields map[string][]string) map[string][]string {
	// fmt.Println("input", input)
	for key, v := range input.GetStructValue().GetFields() {

		if v, ok := v.GetStructValue().GetFields()["instillAcceptFormats"]; ok {
			for _, f := range v.GetListValue().Values {
				k := fmt.Sprintf("%s%s", prefix, key)
				inputAcceptFormatsFields[k] = append(inputAcceptFormatsFields[k], f.GetStringValue())
			}
		}
		if tp, ok := v.GetStructValue().GetFields()["type"]; ok {
			if tp.GetStringValue() == "object" {
				if l, ok := v.GetStructValue().GetFields()["oneOf"]; ok {
					for _, v := range l.GetListValue().Values {
						inputAcceptFormatsFields = c.traverseInputAcceptFormatsFields(v.GetStructValue().GetFields()["properties"], fmt.Sprintf("%s%s.", prefix, key), inputAcceptFormatsFields)
					}
				}
				inputAcceptFormatsFields = c.traverseInputAcceptFormatsFields(v.GetStructValue().GetFields()["properties"], fmt.Sprintf("%s%s.", prefix, key), inputAcceptFormatsFields)
			}

		}
	}

	return inputAcceptFormatsFields
}

func (c *Component) ListOutputFormatsFields() (map[string]map[string]string, error) {
	return c.outputFormatsFields, nil
}

func (c *Component) initOutputFormatsFields() {
	outputFormatsFields := map[string]map[string]string{}

	for task, sch := range c.GetTaskOutputSchemas() {
		outputFormatsFields[task] = map[string]string{}
		output := &structpb.Struct{}
		_ = protojson.Unmarshal([]byte(sch), output)
		outputFormatsFields[task] = c.traverseOutputFormatsFields(output.GetFields()["properties"], "", outputFormatsFields[task])
		if l, ok := output.GetFields()["oneOf"]; ok {
			for _, v := range l.GetListValue().Values {
				outputFormatsFields[task] = c.traverseOutputFormatsFields(v.GetStructValue().GetFields()["properties"], "", outputFormatsFields[task])
			}
		}
		c.outputFormatsFields = outputFormatsFields

	}

}

func (c *Component) traverseOutputFormatsFields(input *structpb.Value, prefix string, outputFormatsFields map[string]string) map[string]string {
	// fmt.Println("input", input)
	for key, v := range input.GetStructValue().GetFields() {

		if v, ok := v.GetStructValue().GetFields()["instillFormat"]; ok {
			k := fmt.Sprintf("%s%s", prefix, key)
			outputFormatsFields[k] = v.GetStringValue()
		}
		if tp, ok := v.GetStructValue().GetFields()["type"]; ok {
			if tp.GetStringValue() == "object" {
				if l, ok := v.GetStructValue().GetFields()["oneOf"]; ok {
					for _, v := range l.GetListValue().Values {
						outputFormatsFields = c.traverseOutputFormatsFields(v.GetStructValue().GetFields()["properties"], fmt.Sprintf("%s%s.", prefix, key), outputFormatsFields)
					}
				}
				outputFormatsFields = c.traverseOutputFormatsFields(v.GetStructValue().GetFields()["properties"], fmt.Sprintf("%s%s.", prefix, key), outputFormatsFields)
			}

		}
	}

	return outputFormatsFields
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

type ComponentConfig struct {
	Task  string
	Input map[string]any
	Setup map[string]any
}
