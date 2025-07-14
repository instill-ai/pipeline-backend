package instillmodel

import (
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util"
	modelpb "github.com/instill-ai/protogen-go/model/model/v1alpha"
)

type VisionRequestData struct {
	ImageBase64 string `json:"image-base64"`
	Type        string `json:"type"`
}

func (e *execution) executeVisionTask(grpcClient modelpb.ModelPublicServiceClient, nsID string, modelID string, version string, inputs []*structpb.Struct) ([]*structpb.Struct, error) {
	if len(inputs) <= 0 {
		return nil, fmt.Errorf("invalid input: %v for model: %s/%s/%s", inputs, nsID, modelID, version)
	}

	if grpcClient == nil {
		return nil, fmt.Errorf("uninitialized client")
	}

	taskInputs := []*structpb.Struct{}
	for _, input := range inputs {
		i := &RequestWrapper{
			Data: &VisionRequestData{
				ImageBase64: util.TrimBase64Mime(input.Fields["image-base64"].GetStringValue()),
				Type:        "image-base64",
			},
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
		outputs = append(outputs, taskOutputs[idx].Fields["data"].GetStructValue())
	}
	return outputs, nil
}
