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

type Job struct {
	Input  InputReader
	Output OutputWriter
	Error  ErrorHandler
}

type InputReader interface {
	Read(ctx context.Context) (input *structpb.Struct, err error)
}
type OutputWriter interface {
	Write(ctx context.Context, output *structpb.Struct) (err error)
}
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

func (e *ComponentExecution) GetTask() string                    { return e.Task }
func (e *ComponentExecution) GetSetup() *structpb.Struct         { return e.Setup }
func (e *ComponentExecution) GetSystemVariables() map[string]any { return e.SystemVariables }
func (e *ComponentExecution) GetLogger() *zap.Logger             { return e.Component.GetLogger() }

func (e *ComponentExecution) GetTaskInputSchema() string {
	return e.Component.GetTaskInputSchemas()[e.Task]
}
func (e *ComponentExecution) GetTaskOutputSchema() string {
	return e.Component.GetTaskOutputSchemas()[e.Task]
}

// UsesInstillCredentials indicates wether the component setup includes the use
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
func (e *ComponentExecution) FillInDefaultValues(input *structpb.Struct) (*structpb.Struct, error) {
	inputMap, err := e.getInputSchemaJSON(e.Task)
	if err != nil {
		return nil, err
	}
	return e.fillInDefaultValuesWithReference(input, inputMap)
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
func (e *ComponentExecution) fillInDefaultValuesWithReference(input *structpb.Struct, reference map[string]interface{}) (*structpb.Struct, error) {
	for key, value := range reference {
		valueMap, ok := value.(map[string]interface{})
		if !ok {
			continue
		}
		if _, ok := valueMap["default"]; !ok {
			if !hasNextLevel(valueMap) {
				continue
			}
			if _, ok := input.GetFields()[key]; !ok {
				input.GetFields()[key] = structpb.NewStructValue(&structpb.Struct{Fields: make(map[string]*structpb.Value)})
			}
			var properties map[string]interface{}
			if _, ok := valueMap["properties"]; !ok {
				var requiredFieldsRaw []interface{}
				if requiredFieldsRaw, ok = valueMap["required"].([]interface{}); !ok {
					continue
				}
				requiredFields := make([]string, len(requiredFieldsRaw))
				for idx, v := range requiredFieldsRaw {
					requiredFields[idx] = fmt.Sprintf("%v", v)
				}
				for _, target := range []string{"allOf", "anyOf", "oneOf"} {
					var items []interface{}
					if items, ok = valueMap[target].([]interface{}); !ok {
						continue
					}
					for _, v := range items {
						if properties, ok = v.(map[string]interface{})["properties"].(map[string]interface{}); !ok {
							continue
						}
						inputSubField := input.GetFields()[key].GetStructValue()
						if target == "oneOf" && !optionMatch(inputSubField, properties, requiredFields) {
							continue
						}
						subField, err := e.fillInDefaultValuesWithReference(inputSubField, properties)
						if err != nil {
							return nil, err
						}
						input.GetFields()[key] = structpb.NewStructValue(subField)
					}
				}
			} else {
				if properties, ok = valueMap["properties"].(map[string]interface{}); !ok {
					continue
				}
				subField, err := e.fillInDefaultValuesWithReference(input.GetFields()[key].GetStructValue(), properties)
				if err != nil {
					return nil, err
				}
				input.GetFields()[key] = structpb.NewStructValue(subField)
			}
			continue
		}
		if _, ok := input.GetFields()[key]; ok {
			continue
		}
		defaultValue := valueMap["default"]
		typeValue := valueMap["type"]
		switch typeValue {
		case "string", "integer", "number", "boolean":
			val, err := structpb.NewValue(defaultValue)
			if err != nil {
				continue
			}
			input.GetFields()[key] = val
		case "array":
			tempArray := &structpb.ListValue{Values: []*structpb.Value{}}
			itemType := valueMap["items"].(map[string]interface{})["type"]
			switch itemType {
			case "string", "integer", "number", "boolean":
				for _, v := range defaultValue.([]interface{}) {
					val, err := structpb.NewValue(v)
					if err != nil {
						continue
					}
					tempArray.Values = append(tempArray.Values, val)
				}
			default:
				continue
			}
			input.GetFields()[key] = structpb.NewListValue(tempArray)
		}
	}
	return input, nil
}

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
