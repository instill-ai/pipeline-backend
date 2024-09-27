package groq

import (
	"fmt"

	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util/httpclient"
)

const (
	Endpoint = "https://api.groq.com"
)

// reference: https://console.groq.com/docs/api-reference on 2024-08-05

type errBody struct {
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (e errBody) Message() string {
	return e.Error.Message
}

type GroqClient struct {
	httpClient *httpclient.Client
}

func NewClient(token string, logger *zap.Logger) *GroqClient {
	c := httpclient.New("Groq", Endpoint, httpclient.WithLogger(logger),
		httpclient.WithEndUserError(new(errBody)))
	c.SetAuthToken(token)
	return &GroqClient{httpClient: c}
}

type GroqChatMessageInterface interface {
}

type GroqChatMessage struct {
	Role    string            `json:"role"`
	Content []GroqChatContent `json:"content"`
}

type GroqSystemMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type GroqChatContent struct {
	ImageURL *GroqURL            `json:"image_url,omitempty"`
	Text     string              `json:"text"`
	Type     GroqChatContentType `json:"type,omitempty"`
}

type GroqChatContentType string

const (
	GroqChatContentTypeText  GroqChatContentType = "text"
	GroqChatContentTypeImage GroqChatContentType = "image"
)

type GroqURL struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"`
}

type ChatRequest struct {
	FrequencyPenalty  float32                    `json:"frequency_penalty,omitempty"`
	MaxTokens         int                        `json:"max_tokens"`
	Model             string                     `json:"model"`
	Messages          []GroqChatMessageInterface `json:"messages"`
	N                 int                        `json:"n,omitempty"`
	PresencePenalty   float32                    `json:"presence_penalty,omitempty"`
	ParallelToolCalls bool                       `json:"parallel_tool_calls,omitempty"`
	Seed              int                        `json:"seed,omitempty"`
	Stop              []string                   `json:"stop"`
	Stream            bool                       `json:"stream,omitempty"`
	Temperature       float32                    `json:"temperature,omitempty"`
	TopP              float32                    `json:"top_p,omitempty"`
	User              string                     `json:"user,omitempty"`
}

type ChatResponse struct {
	ID      string       `json:"id"`
	Object  string       `json:"object"`
	Created int          `json:"created"`
	Model   string       `json:"model"`
	Choices []GroqChoice `json:"choices"`
	Usage   GroqUsage    `json:"usage"`
}

type GroqChoice struct {
	Index        int                 `json:"index"`
	Message      GroqResponseMessage `json:"message"`
	FinishReason string              `json:"finish_reason"`
}

type GroqResponseMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type GroqUsage struct {
	PromptTokens     int     `json:"prompt_tokens"`
	CompletionTokens int     `json:"completion_tokens"`
	TotalTokens      int     `json:"total_tokens"`
	PromptTime       float32 `json:"prompt_time"`
	CompletionTime   float32 `json:"completion_time"`
	TotalTime        float32 `json:"total_time"`
}

func (c *GroqClient) Chat(request ChatRequest) (ChatResponse, error) {
	response := ChatResponse{}
	req := c.httpClient.R().SetResult(&response).SetBody(request)
	if resp, err := req.Post("/openai/v1/chat/completions"); err != nil {
		if resp != nil {
			respString := string(resp.Body())
			return response, fmt.Errorf("error when sending chat request %v: %s", err, respString)
		}
		return response, fmt.Errorf("error when sending chat request %v", err)
	}
	return response, nil
}

func getAPIKey(setup *structpb.Struct) string {
	return setup.GetFields()[cfgAPIKey].GetStringValue()
}
