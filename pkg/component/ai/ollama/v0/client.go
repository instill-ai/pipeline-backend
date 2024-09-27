package ollama

import (
	"fmt"
	"slices"

	"go.uber.org/zap"

	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util/httpclient"
)

// reference: https://github.com/ollama/ollama/blob/main/docs/api.md
// Ollama v0.2.5 on 2024-07-17

type errBody struct {
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (e errBody) Message() string {
	return e.Error.Message
}

type OllamaClient struct {
	httpClient *httpclient.Client
	autoPull   bool
}

func NewClient(endpoint string, autoPull bool, logger *zap.Logger) *OllamaClient {
	c := httpclient.New("Ollama", endpoint, httpclient.WithLogger(logger),
		httpclient.WithEndUserError(new(errBody)))
	return &OllamaClient{httpClient: c, autoPull: autoPull}
}

type OllamaModelInfo struct {
	Name       string `json:"name"`
	ModifiedAt string `json:"modified_at"`
	Size       int    `json:"size"`
	Dijest     string `json:"digest"`
	Details    struct {
		Format            string `json:"format"`
		Family            string `json:"family"`
		Families          string `json:"families"`
		ParameterSize     string `json:"parameter_size"`
		QuantizationLevel string `json:"quantization_level"`
	} `json:"details"`
}

type ListLocalModelsRequest struct {
}

type ListLocalModelsResponse struct {
	Models []OllamaModelInfo `json:"models"`
}

func (c *OllamaClient) CheckModelAvailability(modelName string) bool {
	request := &ListLocalModelsRequest{}
	response := &ListLocalModelsResponse{}
	req := c.httpClient.R().SetResult(&response).SetBody(request)
	if _, err := req.Get("/api/tags"); err != nil {
		return false
	}
	localModels := []string{}
	for _, m := range response.Models {
		localModels = append(localModels, m.Name)
	}
	return slices.Contains(localModels, modelName)
}

type PullModelRequest struct {
	Name   string `json:"name"`
	Stream bool   `json:"stream"`
}

type PullModelResponse struct {
}

func (c *OllamaClient) Pull(modelName string) error {
	request := &PullModelRequest{
		Name:   modelName,
		Stream: false,
	}
	response := &PullModelResponse{}
	req := c.httpClient.R().SetResult(&response).SetBody(request)
	if _, err := req.Post("/api/pull"); err != nil {
		return err
	}
	return nil

}

type OllamaChatMessage struct {
	Role    string   `json:"role"`
	Content string   `json:"content"`
	Images  []string `json:"images,omitempty"`
}

type OllamaOptions struct {
	Temperature float32 `json:"temperature,omitempty"`
	TopK        int     `json:"top_k,omitempty"`
	Seed        int     `json:"seed,omitempty"`
}

type ChatRequest struct {
	Model    string              `json:"model"`
	Messages []OllamaChatMessage `json:"messages"`
	Stream   bool                `json:"stream"`
	Options  OllamaOptions       `json:"options"`
}

type ChatResponse struct {
	Model              string            `json:"model"`
	CreatedAt          string            `json:"created_at"`
	Message            OllamaChatMessage `json:"message"`
	Done               bool              `json:"done"`
	DoneReason         string            `json:"done_reason"`
	TotalDuration      int               `json:"total_duration"`
	LoadDuration       int               `json:"load_duration"`
	PromptEvalCount    int               `json:"prompt_eval_count"`
	PromptEvalDuration int               `json:"prompt_eval_duration"`
	EvalCount          int               `json:"eval_count"`
	EvalDuration       int               `json:"eval_duration"`
}

func (c *OllamaClient) Chat(request ChatRequest) (ChatResponse, error) {
	response := ChatResponse{}
	isAvailable := c.CheckModelAvailability(request.Model)

	if !isAvailable && !c.autoPull {
		return response, fmt.Errorf("model %s is not available", request.Model)
	}
	if !isAvailable {
		err := c.Pull(request.Model)
		if err != nil {
			return response, fmt.Errorf("error when auto pulling model %v", err)
		}
	}
	req := c.httpClient.R().SetResult(&response).SetBody(request)
	if _, err := req.Post("/api/chat"); err != nil {
		return response, fmt.Errorf("error when sending chat request %v", err)
	}
	return response, nil
}

type EmbedRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

type EmbedResponse struct {
	Embedding []float32 `json:"embedding"`
}

func (c *OllamaClient) Embed(request EmbedRequest) (EmbedResponse, error) {
	response := EmbedResponse{}
	isAvailable := c.CheckModelAvailability(request.Model)

	if !isAvailable && !c.autoPull {
		return response, fmt.Errorf("model %s is not available", request.Model)
	}
	if !isAvailable {
		err := c.Pull(request.Model)
		if err != nil {
			return response, fmt.Errorf("error when auto pulling model %v", err)
		}
	}
	req := c.httpClient.R().SetResult(&response).SetBody(request)
	if _, err := req.Post("/api/embeddings"); err != nil {
		return response, fmt.Errorf("error when sending embeddings request %v", err)
	}
	return response, nil
}

func (c *OllamaClient) IsAutoPull() bool {
	return c.autoPull
}
