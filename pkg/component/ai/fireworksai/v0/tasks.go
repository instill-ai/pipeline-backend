package fireworksai

import (
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

const (
	chatModelPrefix = "accounts/fireworks/models/"
)

type TaskTextGenerationChatInput struct {
	ChatHistory  []ChatMessage `json:"chat-history"`
	MaxNewTokens int           `json:"max-new-tokens"`
	Model        string        `json:"model"`
	Prompt       string        `json:"prompt"`
	PromptImages []string      `json:"prompt-images"`
	Seed         int           `json:"seed"`
	SystemMsg    string        `json:"system-message"`
	Temperature  float32       `json:"temperature"`
	TopK         int           `json:"top-k"`
	TopP         float32       `json:"top-p"`
}

type ChatMessage struct {
	Role    string              `json:"role"`
	Content []MultiModalContent `json:"content"`
}

type MultiModalContent struct {
	ImageURL URL    `json:"image-url"`
	Text     string `json:"text"`
	Type     string `json:"type"`
}

type URL struct {
	URL string `json:"url"`
}

type TaskTextGenerationChatOuput struct {
	Text  string                      `json:"text"`
	Usage TaskTextGenerationChatUsage `json:"usage"`
}

type TaskTextGenerationChatUsage struct {
	InputTokens  int `json:"input-tokens"`
	OutputTokens int `json:"output-tokens"`
}

func (e *execution) TaskTextGenerationChat(in *structpb.Struct) (*structpb.Struct, error) {
	input := TaskTextGenerationChatInput{}
	if err := base.ConvertFromStructpb(in, &input); err != nil {
		return nil, err
	}

	messages := []FireworksChatRequestMessage{}
	if input.SystemMsg != "" {
		messages = append(messages, FireworksChatRequestMessage{
			Role: FireworksChatMessageRoleSystem,
			Content: []FireworksMultiModalContent{{
				Text: input.SystemMsg,
				Type: FireworksContentTypeText,
			}},
		})
	}

	for _, msg := range input.ChatHistory {
		contents := []FireworksMultiModalContent{}
		for _, c := range msg.Content {
			if c.ImageURL.URL != "" {
				contents = append(contents, FireworksMultiModalContent{
					ImageURL: FireworksURL{URL: c.ImageURL.URL},
					Type:     FireworksContentTypeImageURL,
				})
				continue
			}
			if c.Text != "" {
				contents = append(contents, FireworksMultiModalContent{
					Text: c.Text,
					Type: FireworksContentTypeText,
				})
			}
		}
		messages = append(messages, FireworksChatRequestMessage{
			Role:    FireworksChatMessageRole(msg.Role),
			Content: contents,
		})
	}

	if input.Prompt != "" {
		messages = append(messages, FireworksChatRequestMessage{
			Role: FireworksChatMessageRoleUser,
			Content: []FireworksMultiModalContent{{
				Text: input.Prompt, Type: FireworksContentTypeText,
			}},
		})
	}

	if len(input.PromptImages) > 0 {
		contents := []FireworksMultiModalContent{}
		for _, img := range input.PromptImages {
			contents = append(contents, FireworksMultiModalContent{
				ImageURL: FireworksURL{URL: img},
				Type:     FireworksContentTypeImageURL,
			})
		}
		messages[len(messages)-1].Content = append(messages[len(messages)-1].Content, contents...)
	}

	req := ChatRequest{
		Messages:    messages,
		Model:       chatModelPrefix + input.Model,
		Tools:       nil,
		MaxTokens:   input.MaxNewTokens,
		Temperature: input.Temperature,
		TopK:        input.TopK,
		N:           1,
		TopP:        input.TopP,
	}

	resp, err := e.client.Chat(req)

	if err != nil {
		return nil, err
	}

	output := TaskTextGenerationChatOuput{
		Text: resp.Choices[0].Message.Content,
		Usage: TaskTextGenerationChatUsage{
			InputTokens:  resp.Usage.PromptTokens,
			OutputTokens: resp.Usage.CompletionTokens,
		},
	}
	return base.ConvertToStructpb(output)
}

type TaskTextEmbeddingsUsage struct {
	Tokens int `json:"tokens"`
}

type TaskTextEmbeddingsInput struct {
	Text  string `json:"text"`
	Model string `json:"model"`
}

type TaskTextEmbeddingsOutput struct {
	Embedding []float32               `json:"embedding"`
	Usage     TaskTextEmbeddingsUsage `json:"usage"`
}

func (e *execution) TaskTextEmbeddings(in *structpb.Struct) (*structpb.Struct, error) {
	input := TaskTextEmbeddingsInput{}
	if err := base.ConvertFromStructpb(in, &input); err != nil {
		return nil, err
	}

	req := EmbedRequest{
		Input: input.Text,
		Model: input.Model,
	}

	resp, err := e.client.Embed(req)

	if err != nil {
		return nil, err
	}

	output := TaskTextEmbeddingsOutput{
		Embedding: resp.Data[0].Embedding,
		Usage: TaskTextEmbeddingsUsage{
			Tokens: resp.Usage.TotalTokens,
		},
	}
	return base.ConvertToStructpb(output)
}
