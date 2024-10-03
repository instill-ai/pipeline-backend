package universalai

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/ai"
	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util/httpclient"

	openaiv1 "github.com/instill-ai/pipeline-backend/pkg/component/ai/openai/v1"
)

func (e *execution) ExecuteTextChat(input *structpb.Struct, job *base.Job, ctx context.Context) (*structpb.Struct, error) {

	x := e.ComponentExecution
	model := getModel(x.GetSetup())

	err := insertModel(input, model)

	if err != nil {
		return nil, err
	}

	inputStruct := ai.TextChatInput{}

	if err := base.ConvertFromStructpb(input, &inputStruct); err != nil {
		return nil, err
	}

	vendor, ok := ModelVendorMap[model]

	if !ok {
		return nil, fmt.Errorf("unsupported vendor for model: %s", model)
	}

	client, err := newClient(x.GetSetup(), x.GetLogger(), vendor)

	if err != nil {
		return nil, err
	}

	switch vendor {
	case "openai":
		return openaiv1.ExecuteTextChat(inputStruct, client.(*httpclient.Client), job, ctx)
	default:
		return nil, fmt.Errorf("unsupported vendor: %s", vendor)
	}
}

// In the implementation, the model is more like the input of execution than the setup of the component.
// However, we should set the model in setup to be able to resolve the setup for the key in the vendor map.
// To avoid users inputting the model in the setup and params, we insert the model into input data.
func insertModel(input *structpb.Struct, model string) error {

	inputData, ok := input.Fields["data"]
	if !ok {
		return fmt.Errorf("failed to get data from input: no 'data' field found")
	}

	dataStruct, ok := inputData.GetKind().(*structpb.Value_StructValue)
	if !ok {
		return fmt.Errorf("data field is not a struct")
	}

	dataStruct.StructValue.Fields["model"] = structpb.NewStringValue(model)

	return nil
}
