package fireworksai

import (
	"fmt"

	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util/httpclient"
)

const (
	chatEndpoint  = "/inference/v1/chat/completions"
	embedEndpoint = "/inference/v1/embeddings"
)

type FireworksClient struct {
	httpClient *httpclient.Client
	logger     *zap.Logger
}

func newClient(apiKey string, baseURL string, logger *zap.Logger) *FireworksClient {
	client := httpclient.New("FireworksAI", baseURL,
		httpclient.WithLogger(logger),
		httpclient.WithEndUserError(new(errBody)),
	)
	client.SetAuthToken(apiKey)
	return &FireworksClient{httpClient: client, logger: logger}
}

func getAPIKey(setup *structpb.Struct) string {
	return setup.GetFields()[cfgAPIKey].GetStringValue()
}

type errBody struct {
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (e errBody) Message() string {
	return e.Error.Message
}

type FireworksChatRequestMessage struct {
	Role    FireworksChatMessageRole     `json:"role"`
	Content []FireworksMultiModalContent `json:"content"`
	Name    string                       `json:"name,omitempty"`
}

type FireworksMultiModalContent struct {
	ImageURL FireworksURL         `json:"image_url,omitempty"`
	Text     string               `json:"text,omitempty"`
	Type     FireworksContentType `json:"type"`
}

type FireworksContentType string

const (
	FireworksContentTypeText     FireworksContentType = "text"
	FireworksContentTypeImageURL FireworksContentType = "image_url"
)

type FireworksURL struct {
	URL string `json:"url"`
}

type FireworksChatMessageRole string

const (
	FireworksChatMessageRoleUser      FireworksChatMessageRole = "user"
	FireworksChatMessageRoleSystem    FireworksChatMessageRole = "system"
	FireworksChatMessageRoleAssistant FireworksChatMessageRole = "assistant"
)

type FireworksTool struct {
	Type     FireworksToolType     `json:"type"`
	Function FireworksToolFunction `json:"function"`
}

type FireworksToolFunction struct {
	Description string         `json:"description"`
	Name        string         `json:"name"`
	Parameters  map[string]any `json:"parameters"`
}

type FireworksToolType string

const (
	FireworksToolTypeFunction FireworksToolType = "function"
)

type ChatRequest struct {
	// reference: https://docs.fireworks.ai/api-reference/post-chatcompletions on 2024-07-23
	Model             string                        `json:"model"`
	Messages          []FireworksChatRequestMessage `json:"messages"`
	Tools             *[]FireworksTool              `json:"tools,omitempty"`
	MaxTokens         int                           `json:"max_tokens"`
	PromptTruncateLen int                           `json:"prompt_truncate_len,omitempty"`
	Temperature       float32                       `json:"temperature,omitempty"`
	TopP              float32                       `json:"top_p,omitempty"`
	TopK              int                           `json:"top_k,omitempty"`
	FrequencyPenalty  float32                       `json:"frequency_penalty,omitempty"`
	PresencePenalty   float32                       `json:"presence_penalty,omitempty"`
	N                 int                           `json:"n,omitempty"`
	User              string                        `json:"user,omitempty"`
}

type ChatResponse struct {
	ID      string             `json:"id"`
	Object  FireworksObject    `json:"object"`
	Created int64              `json:"created"`
	Model   string             `json:"model"`
	Choices []FireWorksChoice  `json:"choices"`
	Usage   FireworksChatUsage `json:"usage"`
}

type FireworksObject string

const (
	FireworksResponseObjectChatCompletion FireworksObject = "chat.completion"
	FireworksResponseObjectEmbedding      FireworksObject = "embedding"
	FireworksObjectList                   FireworksObject = "list"
)

type FireWorksChoice struct {
	Index        int                          `json:"index"`
	FinishReason FireworksFinishReason        `json:"finish_reason"`
	Message      FireworksChatResponseMessage `json:"message"`
}

type FireworksChatResponseMessage struct {
	Role      FireworksChatMessageRole `json:"role"`
	Content   string                   `json:"content"`
	ToolCalls []FireworksToolCall      `json:"tool_calls"`
}

type FireworksToolCall struct {
	ID       string            `json:"id"`
	Type     FireworksToolType `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type FireworksFinishReason string

const (
	FireworksFinishReasonStop   FireworksFinishReason = "stop"
	FireworksFinishReasonLength FireworksFinishReason = "length"
)

type FireworksChatUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

func (c FireworksClient) Chat(request ChatRequest) (ChatResponse, error) {
	response := ChatResponse{}
	req := c.httpClient.R().SetResult(&response).SetBody(request)
	if res, err := req.Post(chatEndpoint); err != nil {
		errMsg := res.Body()
		return response, fmt.Errorf("sending chat request, %v, %s", err, errMsg)
	}
	return response, nil
}

type EmbedRequest struct {
	// reference: https://docs.fireworks.ai/api-reference/creates-an-embedding-vector-representing-the-input-text on 2024-07-24
	Model      string `json:"model"`
	Input      string `json:"input"`
	Dimensions int    `json:"dimensions,omitempty"`
}

type EmbedResponse struct {
	Model  string               `json:"model"`
	Data   []FireworksEmbedData `json:"data"`
	Usage  FireworksEmbedUsage  `json:"usage"`
	Object FireworksObject      `json:"object"`
}

type FireworksEmbedData struct {
	Index     int             `json:"index"`
	Embedding []float32       `json:"embedding"`
	Object    FireworksObject `json:"object"`
}

type FireworksEmbedUsage struct {
	PromptTokens int `json:"prompt_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

func (c FireworksClient) Embed(request EmbedRequest) (EmbedResponse, error) {
	response := EmbedResponse{}
	req := c.httpClient.R().SetResult(&response).SetBody(request)
	if _, err := req.Post(embedEndpoint); err != nil {
		return response, fmt.Errorf("error when sending embedding request %v", err)
	}
	return response, nil
}
