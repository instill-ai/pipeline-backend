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
	vendor := ModelVendorMap[inputStruct.Data.Model]

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

var ModelVendorMap = map[string]string{
	"o1-preview":             "openai",
	"o1-mini":                "openai",
	"gpt-4o-mini":            "openai",
	"gpt-4o":                 "openai",
	"gpt-4o-2024-05-13":      "openai",
	"gpt-4o-2024-08-06":      "openai",
	"gpt-4-turbo":            "openai",
	"gpt-4-turbo-2024-04-09": "openai",
	"gpt-4-0125-preview":     "openai",
	"gpt-4-turbo-preview":    "openai",
	"gpt-4-1106-preview":     "openai",
	"gpt-4-vision-preview":   "openai",
	"gpt-4":                  "openai",
	"gpt-4-0314":             "openai",
	"gpt-4-0613":             "openai",
	"gpt-4-32k":              "openai",
	"gpt-4-32k-0314":         "openai",
	"gpt-4-32k-0613":         "openai",
	"gpt-3.5-turbo":          "openai",
	"gpt-3.5-turbo-16k":      "openai",
	"gpt-3.5-turbo-0301":     "openai",
	"gpt-3.5-turbo-0613":     "openai",
	"gpt-3.5-turbo-1106":     "openai",
	"gpt-3.5-turbo-0125":     "openai",
	"gpt-3.5-turbo-16k-0613": "openai",
}
