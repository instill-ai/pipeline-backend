package instill

import (
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"

	modelPB "github.com/instill-ai/protogen-go/model/model/v1alpha"
)

type TextCompletionRequestData struct {
	Prompt        string `json:"prompt"`
	SystemMessage string `json:"system-message,omitempty"`
}

type TextCompletionRequestParameter struct {
	MaxTokens   int     `json:"max-tokens,omitempty"`
	Seed        int     `json:"seed,omitempty"`
	N           int     `json:"n,omitempty"`
	Temperature float32 `json:"temperature,omitempty"`
	TopP        int     `json:"top-p,omitempty"`
}

func (e *execution) executeTextGeneration(grpcClient modelPB.ModelPublicServiceClient, nsID string, modelID string, version string, inputs []*structpb.Struct) ([]*structpb.Struct, error) {
	if len(inputs) <= 0 {
		return nil, fmt.Errorf("invalid input: %v for model: %s/%s/%s", inputs, nsID, modelID, version)
	}

	if grpcClient == nil {
		return nil, fmt.Errorf("uninitialized client")
	}

	taskInputs := []*structpb.Struct{}
	for _, input := range inputs {
		i := &RequestWrapper{
			Data: &TextCompletionRequestData{
				Prompt: input.GetFields()["prompt"].GetStringValue(),
			},
			Parameter: &TextCompletionRequestParameter{
				N: 1,
			},
		}
		if _, ok := input.GetFields()["system-message"]; ok {
			v := input.GetFields()["system-message"].GetStringValue()
			i.Data.(*TextCompletionRequestData).SystemMessage = v
		}
		if _, ok := input.GetFields()["seed"]; ok {
			v := int(input.GetFields()["seed"].GetNumberValue())
			i.Parameter.(*TextCompletionRequestParameter).Seed = v
		}
		if _, ok := input.GetFields()["max-new-tokens"]; ok {
			v := int(input.GetFields()["max-new-tokens"].GetNumberValue())
			i.Parameter.(*TextCompletionRequestParameter).MaxTokens = v
		}
		if _, ok := input.GetFields()["temperature"]; ok {
			v := float32(input.GetFields()["temperature"].GetNumberValue())
			i.Parameter.(*TextCompletionRequestParameter).Temperature = v
		}
		taskInput, err := base.ConvertToStructpb(i)
		if err != nil {
			return nil, err
		}
		taskInputs = append(taskInputs, taskInput)
	}

	taskOutputs, err := trigger(grpcClient, e.SystemVariables, nsID, modelID, version, taskInputs)
	if err != nil {
		return nil, err
	}
	if len(taskOutputs) <= 0 {
		return nil, fmt.Errorf("invalid output: %v for model: %s/%s/%s", taskOutputs, nsID, modelID, version)
	}

	outputs := []*structpb.Struct{}
	for idx := range inputs {
		choices := taskOutputs[idx].Fields["data"].GetStructValue().Fields["choices"].GetListValue()
		output := structpb.Struct{Fields: make(map[string]*structpb.Value)}
		output.Fields["text"] = structpb.NewStringValue(choices.GetValues()[0].GetStructValue().Fields["content"].GetStringValue())
		outputs = append(outputs, &output)
	}

	return outputs, nil
}
