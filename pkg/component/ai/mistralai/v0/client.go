package mistralai

import (
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"

	mistralSDK "github.com/gage-technologies/mistral-go"
)

type MistralClient struct {
	sdkClient mistralClientInterface
	logger    *zap.Logger
}

type mistralClientInterface interface {
	Embeddings(model string, input []string) (*mistralSDK.EmbeddingResponse, error)
	Chat(model string, messages []mistralSDK.ChatMessage, params *mistralSDK.ChatRequestParams) (*mistralSDK.ChatCompletionResponse, error)
}

func newClient(apiKey string, logger *zap.Logger) MistralClient {
	client := mistralSDK.NewMistralClientDefault(apiKey)
	return MistralClient{sdkClient: client, logger: logger}
}

func getAPIKey(setup *structpb.Struct) string {
	return setup.GetFields()[cfgAPIKey].GetStringValue()
}
