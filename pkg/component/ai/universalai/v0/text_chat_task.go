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
	inputStruct := ai.TextChatInput{}

	if err := base.ConvertFromStructpb(input, &inputStruct); err != nil {
		return nil, fmt.Errorf("failed to convert input to TextChatInput: %w", err)
	}

	x := e.ComponentExecution
	model := getModel(x.GetSetup())
	vendor := ModelVendorMap[model]

	client, err := newClient(x.GetSetup(), x.GetLogger(), vendor)

	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	switch vendor {
	case "openai":
		return openaiv1.ExecuteTextChat(inputStruct, client.(*httpclient.Client), job, ctx)
	default:
		return nil, fmt.Errorf("unsupported vendor: %s", vendor)
	}
}
