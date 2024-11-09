package json

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	_ "embed"

	"github.com/itchyny/gojq"
	"github.com/xeipuuv/gojsonschema"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/x/errmsg"
)

const (
	taskMarshal      = "TASK_MARSHAL"
	taskUnmarshal    = "TASK_UNMARSHAL"
	taskJQ           = "TASK_JQ"
	taskRenameFields = "TASK_RENAME_FIELDS"
)

var (
	//go:embed config/definition.json
	definitionJSON []byte
	//go:embed config/tasks.json
	tasksJSON []byte
	//go:embed config/tasks.json
	schemaJSON []byte

	once   sync.Once
	comp   *component
	schema *gojsonschema.Schema
)

type component struct {
	base.Component
}

type execution struct {
	base.ComponentExecution

	execute func(*structpb.Struct) (*structpb.Struct, error)
}

func Init(bc base.Component) *component {
	once.Do(func() {
		comp = &component{Component: bc}
		err := comp.LoadDefinition(definitionJSON, nil, tasksJSON, nil)
		if err != nil {
			panic(err)
		}

		schemaLoader := gojsonschema.NewStringLoader(string(schemaJSON))
		schema, err = gojsonschema.NewSchema(schemaLoader)
		if err != nil {
			panic(fmt.Sprintf("Failed to load JSON schema: %v", err))
		}
	})
	return comp
}

func (c *component) CreateExecution(x base.ComponentExecution) (base.IExecution, error) {
	e := &execution{ComponentExecution: x}

	switch x.Task {
	case taskMarshal:
		e.execute = e.marshal
	case taskUnmarshal:
		e.execute = e.unmarshal
	case taskJQ:
		e.execute = e.jq
	case taskRenameFields:
		e.execute = e.renameFields
	default:
		return nil, errmsg.AddMessage(
			fmt.Errorf("not supported task: %s", x.Task),
			fmt.Sprintf("%s task is not supported.", x.Task),
		)
	}
	return e, nil
}

func validateJSON(input any) error {
	documentLoader := gojsonschema.NewGoLoader(input)
	result, err := schema.Validate(documentLoader)
	if err != nil {
		return fmt.Errorf("validation error: %v", err)
	}
	if !result.Valid() {
		errMsg := "JSON does not conform to the schema: "
		for _, desc := range result.Errors() {
			errMsg += fmt.Sprintf("%s; ", desc)
		}
		return errors.New(errMsg)
	}
	return nil
}

func (e *execution) marshal(in *structpb.Struct) (*structpb.Struct, error) {
	out := new(structpb.Struct)

	input := in.AsMap()
	if err := validateJSON(input); err != nil {
		return nil, errmsg.AddMessage(err, "Validation failed for marshal task.")
	}

	b, err := protojson.Marshal(in.Fields["json"])
	if err != nil {
		return nil, errmsg.AddMessage(err, "Couldn't convert the provided object to JSON.")
	}

	out.Fields = map[string]*structpb.Value{
		"string": structpb.NewStringValue(string(b)),
	}

	return out, nil
}

func (e *execution) unmarshal(in *structpb.Struct) (*structpb.Struct, error) {
	out := new(structpb.Struct)

	b := []byte(in.Fields["string"].GetStringValue())
	obj := new(structpb.Value)
	if err := protojson.Unmarshal(b, obj); err != nil {
		return nil, errmsg.AddMessage(err, "Couldn't parse the JSON string. Please check the syntax is correct.")
	}

	if err := validateJSON(obj.AsInterface()); err != nil {
		return nil, errmsg.AddMessage(err, "Validation failed for unmarshal task.")
	}

	out.Fields = map[string]*structpb.Value{"json": obj}

	return out, nil
}

func (e *execution) jq(in *structpb.Struct) (*structpb.Struct, error) {
	out := new(structpb.Struct)

	input := in.Fields["json-value"].AsInterface()
	if input == nil {
		b := []byte(in.Fields["json-string"].GetStringValue())
		if err := json.Unmarshal(b, &input); err != nil {
			return nil, errmsg.AddMessage(err, "Couldn't parse the JSON input. Please check the syntax is correct.")
		}
	}

	if err := validateJSON(input); err != nil {
		return nil, errmsg.AddMessage(err, "Validation failed for jq task.")
	}

	queryStr := in.Fields["jq-filter"].GetStringValue()
	q, err := gojq.Parse(queryStr)
	if err != nil {
		msg := fmt.Sprintf("Couldn't parse the jq filter: %s. Please check the syntax is correct.", err.Error())
		return nil, errmsg.AddMessage(err, msg)
	}

	results := []any{}
	iter := q.Run(input)
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}

		if err, ok := v.(error); ok {
			msg := fmt.Sprintf("Couldn't apply the jq filter: %s.", err.Error())
			return nil, errmsg.AddMessage(err, msg)
		}

		results = append(results, v)
	}

	list, err := structpb.NewList(results)
	if err != nil {
		return nil, err
	}

	out.Fields = map[string]*structpb.Value{
		"results": structpb.NewListValue(list),
	}

	return out, nil
}

func (e *execution) renameFields(in *structpb.Struct) (*structpb.Struct, error) {
	out := new(structpb.Struct)

	// Validate presence of required fields: "json" and "fields"
	jsonField, ok := in.Fields["json"]
	if !ok || jsonField == nil {
		return nil, errmsg.AddMessage(fmt.Errorf("missing required field: json"), "JSON and fields are required.")
	}
	jsonValue := jsonField.AsInterface().(map[string]any)

	fieldsValue, ok := in.Fields["fields"]
	if !ok || fieldsValue == nil || len(fieldsValue.GetListValue().Values) == 0 {
		return nil, errmsg.AddMessage(fmt.Errorf("missing required field: fields"), "JSON and fields are required.")
	}
	fields := fieldsValue.GetListValue().Values

	// Conflict resolution strategy validation
	conflictResolution := in.Fields["conflict-resolution"].GetStringValue()
	if conflictResolution != "overwrite" && conflictResolution != "skip" && conflictResolution != "error" {
		return nil, errmsg.AddMessage(fmt.Errorf("invalid conflict resolution strategy"), "Conflict resolution strategy is invalid.")
	}

	// Process renaming fields with conflict resolution
	for _, field := range fields {
		from := field.GetStructValue().Fields["from"].GetStringValue()
		to := field.GetStructValue().Fields["to"].GetStringValue()

		if val, ok := jsonValue[from]; ok {
			switch conflictResolution {
			case "overwrite":
				delete(jsonValue, from)
				jsonValue[to] = val
			case "skip":
				if _, exists := jsonValue[to]; !exists {
					jsonValue[to] = val
				}
				delete(jsonValue, from)
			case "error":
				if _, exists := jsonValue[to]; exists {
					return nil, errmsg.AddMessage(fmt.Errorf("field conflict: '%s' already exists", to), "Field conflict.")
				}
				delete(jsonValue, from)
				jsonValue[to] = val
			}
		}
	}

	// Validate the final JSON structure
	if err := validateJSON(jsonValue); err != nil {
		return nil, errmsg.AddMessage(err, "Validation failed for renamed JSON object.")
	}

	// Convert to structpb.Struct and assign to output
	structValue, err := structpb.NewStruct(jsonValue)
	if err != nil {
		return nil, errmsg.AddMessage(err, "Failed to create structpb.Struct for output.")
	}

	out.Fields = map[string]*structpb.Value{
		"json": structpb.NewStructValue(structValue),
	}

	return out, nil
}

func (e *execution) Execute(ctx context.Context, jobs []*base.Job) error {
	return base.SequentialExecutor(ctx, jobs, e.execute)
}
