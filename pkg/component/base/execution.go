package base

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/x/errmsg"
)

// IExecution allows components to be executed.
type IExecution interface {
	GetTask() string
	GetLogger() *zap.Logger
	GetTaskInputSchema() string
	GetTaskOutputSchema() string
	GetSystemVariables() map[string]any
	GetComponent() IComponent
	GetComponentID() string
	UsesInstillCredentials() bool

	Execute(context.Context, []*Job) error
}

// Job is the job for the component.
type Job struct {
	Input  InputReader
	Output OutputWriter
	Error  ErrorHandler
}

// InputReader is an interface for reading input data from a job.
type InputReader interface {
	// ReadData reads the input data from the job into the provided struct.
	ReadData(ctx context.Context, input any) (err error)

	// Deprecated: Read() is deprecated and will be removed in a future version.
	// Use ReadData() instead. structpb is not suitable for handling binary data
	// and will be phased out gradually.
	Read(ctx context.Context) (input *structpb.Struct, err error)
}

// OutputWriter is an interface for writing output data to a job.
type OutputWriter interface {
	// WriteData writes the output data to the job from the provided struct.
	WriteData(ctx context.Context, output any) (err error)

	// Deprecated: Write() is deprecated and will be removed in a future
	// version. Use WriteData() instead. structpb is not suitable for handling
	// binary data and will be phased out gradually.
	Write(ctx context.Context, output *structpb.Struct) (err error)
}

// ErrorHandler is an interface for handling errors from a job.
type ErrorHandler interface {
	Error(ctx context.Context, err error)
}

// ComponentExecution implements the common methods for component execution.
type ComponentExecution struct {
	Component IComponent

	// Component ID is the ID of the component *as defined in the recipe*. This
	// identifies an instance of a component, which holds a given configuration
	// (task, setup, input parameters, etc.).
	//
	// NOTE: this is a property of the component not of the execution. However,
	// right now components are being created on startup and only executions
	// are created every time a pipeline is triggered. Therefore, at the moment
	// there's no intermediate entity reflecting "a component within a
	// pipeline". Since we need to access the component ID for e.g. logging /
	// metric collection purposes, for now this information will live in the
	// execution, but note that several executions might have the same
	// component ID.
	ComponentID     string
	SystemVariables map[string]any
	Setup           *structpb.Struct
	Task            string
}

// GetComponent returns the component interface that is triggering the execution.
func (e *ComponentExecution) GetComponent() IComponent { return e.Component }

// GetComponentID returns the ID of the component that's being executed.
func (e *ComponentExecution) GetComponentID() string { return e.ComponentID }

// GetTask returns the task that the component is executing.
func (e *ComponentExecution) GetTask() string { return e.Task }

// GetSetup returns the setup of the component.
func (e *ComponentExecution) GetSetup() *structpb.Struct { return e.Setup }

// GetSystemVariables returns the system variables of the component.
func (e *ComponentExecution) GetSystemVariables() map[string]any { return e.SystemVariables }

// GetLogger returns the logger of the component.
func (e *ComponentExecution) GetLogger() *zap.Logger { return e.Component.GetLogger() }

// GetTaskInputSchema returns the input schema of the task.
func (e *ComponentExecution) GetTaskInputSchema() string {
	return e.Component.GetTaskInputSchemas()[e.Task]
}

// GetTaskOutputSchema returns the output schema of the task.
func (e *ComponentExecution) GetTaskOutputSchema() string {
	return e.Component.GetTaskOutputSchemas()[e.Task]
}

// UsesInstillCredentials indicates whether the component setup includes the use
// of global secrets (as opposed to a bring-your-own-key configuration) to
// connect to external services. Components should override this method when
// they have the ability to read global secrets and be executed without
// explicit credentials.
func (e *ComponentExecution) UsesInstillCredentials() bool { return false }

func (e *ComponentExecution) getInputSchemaJSON(task string) (map[string]interface{}, error) {
	taskSpec, ok := e.Component.GetTaskInputSchemas()[task]
	if !ok {
		return nil, errmsg.AddMessage(
			fmt.Errorf("task %s not found", task),
			fmt.Sprintf("Task %s not found", task),
		)
	}
	var taskSpecMap map[string]interface{}
	err := json.Unmarshal([]byte(taskSpec), &taskSpecMap)
	if err != nil {
		return nil, errmsg.AddMessage(
			err,
			"Failed to unmarshal input",
		)
	}
	inputMap := taskSpecMap["properties"].(map[string]interface{})
	return inputMap, nil
}

// FillInDefaultValues fills in default values for the input based on the task's input schema
func (e *ComponentExecution) FillInDefaultValues(input any) error {
	inputMap, err := e.getInputSchemaJSON(e.Task)
	if err != nil {
		return err
	}
	_, err = e.fillInDefaultValuesWithReference(input, inputMap)
	return err
}

func hasNextLevel(valueMap map[string]interface{}) bool {
	if valType, ok := valueMap["type"]; ok {
		if valType != "object" {
			return false
		}
	}
	if _, ok := valueMap["properties"]; ok {
		return true
	}
	for _, target := range []string{"allOf", "anyOf", "oneOf"} {
		if _, ok := valueMap[target]; ok {
			items := valueMap[target].([]interface{})
			for _, v := range items {
				if _, ok := v.(map[string]interface{})["properties"].(map[string]interface{}); ok {
					return true
				}
			}
		}
	}
	return false
}

func optionMatch(valueMap *structpb.Struct, reference map[string]interface{}, checkFields []string) bool {
	for _, checkField := range checkFields {
		if _, ok := valueMap.GetFields()[checkField]; !ok {
			return false
		}
		if val, ok := reference[checkField].(map[string]interface{})["const"]; ok {
			if valueMap.GetFields()[checkField].GetStringValue() != val {
				return false
			}
		}
	}
	return true
}

func convertToStructPb(input any) (*structpb.Struct, error) {
	if s, ok := input.(*structpb.Struct); ok {
		return s, nil
	}

	jsonBytes, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal input: %w", err)
	}

	result := &structpb.Struct{Fields: make(map[string]*structpb.Value)}
	if err := protojson.Unmarshal(jsonBytes, result); err != nil {
		return nil, fmt.Errorf("failed to convert input to struct: %w", err)
	}

	return result, nil
}

func handleNestedObject(e *ComponentExecution, inputStruct *structpb.Struct, key string, valueMap map[string]interface{}) error {
	if !hasNextLevel(valueMap) {
		return nil
	}

	if _, ok := inputStruct.GetFields()[key]; !ok {
		inputStruct.GetFields()[key] = structpb.NewStructValue(&structpb.Struct{Fields: make(map[string]*structpb.Value)})
	}

	if properties, ok := valueMap["properties"].(map[string]interface{}); ok {
		return handleProperties(e, inputStruct, key, properties)
	}

	return handleCompositeTypes(e, inputStruct, key, valueMap)
}

func handleProperties(e *ComponentExecution, inputStruct *structpb.Struct, key string, properties map[string]interface{}) error {
	subField, err := e.fillInDefaultValuesWithReference(inputStruct.GetFields()[key].GetStructValue(), properties)
	if err != nil {
		return err
	}
	inputStruct.GetFields()[key] = structpb.NewStructValue(subField.(*structpb.Struct))
	return nil
}

func extractRequiredFields(valueMap map[string]interface{}) ([]string, bool) {
	requiredFieldsRaw, ok := valueMap["required"].([]interface{})
	if !ok {
		return nil, false
	}

	requiredFields := make([]string, len(requiredFieldsRaw))
	for idx, v := range requiredFieldsRaw {
		requiredFields[idx] = fmt.Sprintf("%v", v)
	}

	return requiredFields, true
}

func handleCompositeTypes(e *ComponentExecution, inputStruct *structpb.Struct, key string, valueMap map[string]interface{}) error {
	requiredFields, ok := extractRequiredFields(valueMap)
	if !ok {
		return nil
	}

	for _, target := range []string{"allOf", "anyOf", "oneOf"} {
		items, ok := valueMap[target].([]interface{})
		if !ok {
			continue
		}

		for _, v := range items {
			properties, ok := v.(map[string]interface{})["properties"].(map[string]interface{})
			if !ok {
				continue
			}

			inputSubField := inputStruct.GetFields()[key].GetStructValue()
			if target == "oneOf" && !optionMatch(inputSubField, properties, requiredFields) {
				continue
			}

			if err := handleProperties(e, inputStruct, key, properties); err != nil {
				return err
			}
		}
	}
	return nil
}

func handleDefaultValue(inputStruct *structpb.Struct, key string, valueMap map[string]interface{}) error {
	if _, exists := inputStruct.GetFields()[key]; exists {
		return nil
	}

	defaultValue := valueMap["default"]
	typeValue := valueMap["type"].(string)

	switch typeValue {
	case "string", "integer", "number", "boolean":
		val, err := structpb.NewValue(defaultValue)
		if err == nil {
			inputStruct.GetFields()[key] = val
		}
	case "array":
		if err := handleDefaultArray(inputStruct, key, valueMap); err != nil {
			return err
		}
	}
	return nil
}

func handleDefaultArray(inputStruct *structpb.Struct, key string, valueMap map[string]interface{}) error {
	tempArray := &structpb.ListValue{Values: []*structpb.Value{}}
	itemType := valueMap["items"].(map[string]interface{})["type"].(string)

	if itemType != "string" && itemType != "integer" && itemType != "number" && itemType != "boolean" {
		return nil
	}

	for _, v := range valueMap["default"].([]interface{}) {
		if val, err := structpb.NewValue(v); err == nil {
			tempArray.Values = append(tempArray.Values, val)
		}
	}

	inputStruct.GetFields()[key] = structpb.NewListValue(tempArray)
	return nil
}

// Main function
func (e *ComponentExecution) fillInDefaultValuesWithReference(input any, reference map[string]interface{}) (any, error) {
	inputStruct, err := convertToStructPb(input)
	if err != nil {
		return nil, err
	}

	for key, value := range reference {
		valueMap, ok := value.(map[string]interface{})
		if !ok {
			continue
		}

		if _, hasDefault := valueMap["default"]; !hasDefault {
			if err := handleNestedObject(e, inputStruct, key, valueMap); err != nil {
				return nil, err
			}
			continue
		}

		if err := handleDefaultValue(inputStruct, key, valueMap); err != nil {
			return nil, err
		}
	}

	resultJSON, err := protojson.Marshal(inputStruct)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	if err := json.Unmarshal(resultJSON, input); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result back to original type: %w", err)
	}

	return input, nil
}

// FormatErrors formats the errors from the jsonschema validation
func FormatErrors(inputPath string, e jsonschema.Detailed, errors *[]string) {
	path := inputPath + e.InstanceLocation

	pathItems := strings.Split(path, "/")
	formatedPath := pathItems[0]
	for _, pathItem := range pathItems[1:] {
		if _, err := strconv.Atoi(pathItem); err == nil {
			formatedPath += fmt.Sprintf("[%s]", pathItem)
		} else {
			formatedPath += fmt.Sprintf(".%s", pathItem)
		}

	}
	*errors = append(*errors, fmt.Sprintf("%s: %s", formatedPath, e.Error))
}

// Validate the input and output format
func Validate(data *structpb.Struct, jsonSchema string, target string) error {

	schStruct := &structpb.Struct{}
	err := protojson.Unmarshal([]byte(jsonSchema), schStruct)
	if err != nil {
		return err
	}

	err = CompileInstillAcceptFormats(schStruct)
	if err != nil {
		return err
	}

	schStr, err := protojson.Marshal(schStruct)
	if err != nil {
		return err
	}

	c := jsonschema.NewCompiler()
	c.RegisterExtension("instillAcceptFormats", InstillAcceptFormatsMeta, InstillAcceptFormatsCompiler{})
	c.RegisterExtension("instillFormat", InstillFormatMeta, InstillFormatCompiler{})
	if err := c.AddResource("schema.json", strings.NewReader(string(schStr))); err != nil {
		return err
	}
	sch, err := c.Compile("schema.json")
	if err != nil {
		return err
	}
	errors := []string{}

	var v interface{}
	jsonData, err := protojson.Marshal(data)
	if err != nil {
		errors = append(errors, fmt.Sprintf("%s: data error", target))
		return fmt.Errorf("%s", strings.Join(errors, "; "))
	}

	if err := json.Unmarshal(jsonData, &v); err != nil {
		errors = append(errors, fmt.Sprintf("%s: data error", target))
		return fmt.Errorf("%s", strings.Join(errors, "; "))
	}

	if err = sch.Validate(v); err != nil {
		e := err.(*jsonschema.ValidationError)
		for _, valErr := range e.DetailedOutput().Errors {
			inputPath := target
			FormatErrors(inputPath, valErr, &errors)
			for _, subValErr := range valErr.Errors {
				FormatErrors(inputPath, subValErr, &errors)
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("%s", strings.Join(errors, "; "))
	}

	return nil
}

// SecretKeyword is a keyword to reference a secret in a component
// configuration. When a component detects this value in a configuration
// parameter, it will used the pre-configured value, injected at
// initialization.
const SecretKeyword = "__INSTILL_SECRET"

// NewUnresolvedCredential returns an end-user error signaling that the
// component setup contains credentials that reference a global secret that
// wasn't injected into the component.
func NewUnresolvedCredential(key string) error {
	return errmsg.AddMessage(
		fmt.Errorf("unresolved global credential"),
		fmt.Sprintf("The configuration field %s references a global secret "+
			"but it doesn't support Instill Credentials.", key),
	)
}
