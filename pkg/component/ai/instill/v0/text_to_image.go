package instill

import (
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"

	modelPB "github.com/instill-ai/protogen-go/model/model/v1alpha"
)

type TextToImageRequestData struct {
	Prompt string `json:"prompt"`
}

type TextToImageRequestParameter struct {
	AspectRatio    string `json:"aspect-ratio,omitempty"`
	NegativePrompt string `json:"negative-prompt,omitempty"`
	N              int    `json:"n,omitempty"`
	Seed           int    `json:"seed,omitempty"`
}

func (e *execution) executeTextToImage(grpcClient modelPB.ModelPublicServiceClient, nsID string, modelID string, version string, inputs []*structpb.Struct) ([]*structpb.Struct, error) {
	if len(inputs) <= 0 {
		return nil, fmt.Errorf("invalid input: %v for model: %s/%s/%s", inputs, nsID, modelID, version)
	}

	if grpcClient == nil {
		return nil, fmt.Errorf("uninitialized client")
	}

	taskInputs := []*structpb.Struct{}

	for _, input := range inputs {
		i := &RequestWrapper{
			Data: &TextToImageRequestData{
				Prompt: input.GetFields()["prompt"].GetStringValue(),
			},
			Parameter: &TextToImageRequestParameter{},
		}
		if _, ok := input.GetFields()["samples"]; ok {
			v := int(input.GetFields()["samples"].GetNumberValue())
			i.Parameter.(*TextToImageRequestParameter).N = v
		}
		if _, ok := input.GetFields()["seed"]; ok {
			v := int(input.GetFields()["seed"].GetNumberValue())
			i.Parameter.(*TextToImageRequestParameter).Seed = v
		}
		if _, ok := input.GetFields()["aspect-ratio"]; ok {
			v := input.GetFields()["aspect-ratio"].GetStringValue()
			i.Parameter.(*TextToImageRequestParameter).AspectRatio = v
		}
		if _, ok := input.GetFields()["negative-prompt"]; ok {
			v := input.GetFields()["negative-prompt"].GetStringValue()
			i.Parameter.(*TextToImageRequestParameter).NegativePrompt = v
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

		images := make([]*structpb.Value, len(choices.Values))
		for i, c := range choices.Values {
			images[i] = c.GetStructValue().Fields["image"]
		}

		output := structpb.Struct{Fields: make(map[string]*structpb.Value)}

		output.Fields["images"] = structpb.NewListValue(&structpb.ListValue{Values: images})
		outputs = append(outputs, &output)
	}

	return outputs, nil
}
