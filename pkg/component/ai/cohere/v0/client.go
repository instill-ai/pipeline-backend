package cohere

import (
	"context"
	"sync"

	"github.com/cohere-ai/cohere-go/v2/core"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"

	cohereSDK "github.com/cohere-ai/cohere-go/v2"
	cohereClientSDK "github.com/cohere-ai/cohere-go/v2/client"
)

type cohereClient struct {
	sdkClient cohereClientInterface
	logger    *zap.Logger
	lock      sync.Mutex
}

type cohereClientInterface interface {
	Chat(ctx context.Context, request *cohereSDK.ChatRequest, opts ...core.RequestOption) (*cohereSDK.NonStreamedChatResponse, error)
	Embed(ctx context.Context, request *cohereSDK.EmbedRequest, opts ...core.RequestOption) (*cohereSDK.EmbedResponse, error)
	Rerank(ctx context.Context, request *cohereSDK.RerankRequest, opts ...core.RequestOption) (*cohereSDK.RerankResponse, error)
}

func newClient(apiKey string, logger *zap.Logger) *cohereClient {
	client := cohereClientSDK.NewClient(cohereClientSDK.WithToken(apiKey))
	return &cohereClient{sdkClient: client, logger: logger, lock: sync.Mutex{}}
}

func (cl *cohereClient) generateEmbedding(request cohereSDK.EmbedRequest) (cohereSDK.EmbedResponse, error) {
	respPtr, err := cl.sdkClient.Embed(
		context.TODO(),
		&request,
	)
	if err != nil {
		panic(err)
	}
	resp := cohereSDK.EmbedResponse{
		EmbeddingsFloats: respPtr.EmbeddingsFloats,
		EmbeddingsByType: respPtr.EmbeddingsByType,
	}
	return resp, nil
}

func (cl *cohereClient) generateTextChat(request cohereSDK.ChatRequest) (cohereSDK.NonStreamedChatResponse, error) {
	respPtr, err := cl.sdkClient.Chat(
		context.TODO(),
		&request,
	)
	if err != nil {
		panic(err)
	}
	resp := cohereSDK.NonStreamedChatResponse{
		Text:         respPtr.Text,
		GenerationId: respPtr.GenerationId,
		Citations:    respPtr.Citations,
		Meta:         respPtr.Meta,
	}
	return resp, nil
}
func (cl *cohereClient) generateRerank(request cohereSDK.RerankRequest) (cohereSDK.RerankResponse, error) {
	respPtr, err := cl.sdkClient.Rerank(
		context.TODO(),
		&request,
	)
	if err != nil {
		panic(err)
	}
	resp := cohereSDK.RerankResponse{
		Results: respPtr.Results,
		Meta:    respPtr.Meta,
	}
	return resp, nil
}

func getAPIKey(setup *structpb.Struct) string {
	return setup.GetFields()[cfgAPIKey].GetStringValue()
}
