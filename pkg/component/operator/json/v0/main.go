//go:generate compogen readme ./config ./README.mdx --extraContents TASK_JQ=.compogen/extra-jq.mdx --extraContents bottom=.compogen/bottom.mdx
package json

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	_ "embed"

	"github.com/itchyny/gojq"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/x/errmsg"

	"strconv"
	"strings"
	"reflect"
)

const (
	taskMarshal   = "TASK_MARSHAL"
	taskUnmarshal = "TASK_UNMARSHAL"
	taskJQ        = "TASK_JQ"
	taskEditValues= "TASK_EDIT_VALUES"
)

var (
	//go:embed config/definition.json
	definitionJSON []byte
	//go:embed config/tasks.json
	tasksJSON []byte

	once sync.Once
	comp *component
)

type component struct {
	base.Component
}

type execution struct {
	base.ComponentExecution

	execute func(*structpb.Struct) (*structpb.Struct, error)
}

// Init returns an implementation of IOperator that processes JSON objects.
func Init(bc base.Component) *component {
	once.Do(func() {
		comp = &component{Component: bc}
		err := comp.LoadDefinition(definitionJSON, nil, tasksJSON, nil)
		if err != nil {
			panic(err)
		}
	})
	return comp
}

// CreateExecution initializes a component executor that can be used in a
// pipeline trigger.
func (c *component) CreateExecution(x base.ComponentExecution) (base.IExecution, error) {
	e := &execution{ComponentExecution: x}

	switch x.Task {
	case taskMarshal:
		e.execute = e.marshal
	case taskUnmarshal:
		e.execute = e.unmarshal
	case taskJQ:
		e.execute = e.jq
	case taskEditValues:
		e.execute = e.updateJson
	default:
		return nil, errmsg.AddMessage(
			fmt.Errorf("not supported task: %s", x.Task),
			fmt.Sprintf("%s task is not supported.", x.Task),
		)
	}
	return e, nil
}

func (e *execution) marshal(in *structpb.Struct) (*structpb.Struct, error) {
	out := new(structpb.Struct)

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

	queryStr := in.Fields["jq-filter"].GetStringValue()
	q, err := gojq.Parse(queryStr)
	if err != nil {
		// Error messages from gojq are human-friendly enough.
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

func (e *execution) Execute(ctx context.Context, jobs []*base.Job) error {
	return base.SequentialExecutor(ctx, jobs, e.execute)
}

func (e *execution) updateJson(in *structpb.Struct) (*structpb.Struct, error) {
	data := in.Fields["data"].AsInterface()
	updates := in.Fields["updates"].GetListValue().AsSlice()
	conflictResolution := in.Fields["conflictResolution"].GetStringValue()
	supportDotNotation := in.Fields["supportDotNotation"].GetBoolValue()

	// Perform deep copy of the data before updates
	updatedData := deepCopy(data)

	switch data := updatedData.(type) {
	case []interface{}:
		// Process each object in the array
		for i, obj := range data {
			if err := applyUpdatesToObject(obj, updates, supportDotNotation, conflictResolution); err != nil {
				msg := fmt.Sprintf("Error in object %d: %v\n", i, err)
				return nil, errmsg.AddMessage(err, msg)
			}
		}
	case map[string]interface{}:
		// Process the single object
		if err := applyUpdatesToObject(data, updates, supportDotNotation, conflictResolution); err != nil {
			msg := fmt.Sprintf("Error in single object: %v\n", err)
			return nil, errmsg.AddMessage(err, msg)
		}
	default:
		msg := fmt.Sprintf("Invalid data format")
		return nil, errmsg.AddMessage(fmt.Errorf("Error "),msg)
	}
	output := map[string]interface{}{
		"data": updatedData,
	}
	outputStruct, err := structpb.NewStruct(output)
	if err != nil {
		msg := fmt.Sprintf("Failed to convert output to structpb.Struct:", err)
		return nil, errmsg.AddMessage(err, msg)
	}
	return outputStruct,nil
}

func applyUpdatesToObject(data interface{}, updates []interface{}, supportDotNotation bool, conflictResolution string) error {
	for _, update := range updates {
		updateMap := update.(map[string]interface{})
		field := updateMap["field"].(string)
		newValue := updateMap["newValue"]

		err := applyUpdate(data, field, newValue, supportDotNotation, conflictResolution)
		if err != nil {
			// Handle the "error" conflictResolution case by stopping and returning the error
			if conflictResolution == "error" {
				return err
			}
			// Continue for other conflictResolution cases
		}
	}
	return nil
}

func applyUpdate(data interface{}, field string, newValue interface{}, supportDotNotation bool, conflictResolution string) error {
	var fieldParts []string
	if supportDotNotation {
		fieldParts = strings.Split(field, ".")
	} else {
		fieldParts = []string{field}
	}

	current := data
	for i, part := range fieldParts {
		if i == len(fieldParts)-1 {
			// We're at the final part of the path, apply the update

			existingValue, fieldExisting := getFieldValue(current, part)

			// Check if the field exists and compare types
			if fieldExisting {
				if !(reflect.TypeOf(existingValue)==reflect.TypeOf(newValue)){
					return fmt.Errorf("type mismatch: existing field '%s' has type '%T' but got value of type '%T'", part, existingValue, newValue)
				}
			}
			switch conflictResolution {
			case "create":
				// Create the field if it doesn't exist
				setFieldValue(current, part, newValue)
			case "skip":
				// Skip if the field doesn't exist
				if !fieldExists(current, part) {
					return nil
				}
				setFieldValue(current, part, newValue)
			case "error":
				// Return an error if the field doesn't exist
				if !fieldExists(current, part) {
					return fmt.Errorf("Field '%s' does not exist", part)
				}
				setFieldValue(current, part, newValue)
			}
		} else {
			// Traverse to the next part of the path
			if next, ok := getFieldValue(current, part); ok {
				current = next
			} else {
				// Field doesn't exist and we're not at the final part
				if conflictResolution == "create" {
					newMap := make(map[string]interface{})
					setFieldValue(current, part, newMap)
					current = newMap
				} else {
					return fmt.Errorf("Field '%s' does not exist", part)
				}
			}
		}
	}
	return nil
}

func setFieldValue(data interface{}, field string, value interface{}) {
	// Update the field in the map or array (handle different data structures)
	switch data := data.(type) {
	case map[string]interface{}:
		data[field] = value
	case []interface{}:
		idx, _ := strconv.Atoi(field)
		if idx >= 0 && idx < len(data) {
			data[idx] = value
		}
	}
}

func getFieldValue(data interface{}, field string) (interface{}, bool) {
	// Retrieve the field value from the map or array
	switch data := data.(type) {
	case map[string]interface{}:
		val, ok := data[field]
		return val, ok
	case []interface{}:
		idx, err := strconv.Atoi(field)
		if err != nil || idx < 0 || idx >= len(data) {
			return nil, false
		}
		return data[idx], true
	}
	return nil, false
}

func fieldExists(data interface{}, field string) bool {
	// Check if the field exists in the map or array
	switch data := data.(type) {
	case map[string]interface{}:
		_, ok := data[field]
		return ok
	case []interface{}:
		idx, err := strconv.Atoi(field)
		return err == nil && idx >= 0 && idx < len(data)
	}
	return false
}

func deepCopy(data interface{}) interface{} {
	// Deep copy the data structure to avoid modifying the original input
	b, _ := json.Marshal(data)
	var copiedData interface{}
	json.Unmarshal(b, &copiedData)
	return copiedData
}