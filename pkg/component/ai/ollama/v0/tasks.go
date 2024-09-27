package ollama

import (
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
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
	Text string `json:"text"`
}

func (e *execution) TaskTextGenerationChat(in *structpb.Struct) (*structpb.Struct, error) {
	input := TaskTextGenerationChatInput{}
	if err := base.ConvertFromStructpb(in, &input); err != nil {
		return nil, err
	}

	messages := []OllamaChatMessage{}

	if input.SystemMsg != "" {
		messages = append(messages, OllamaChatMessage{
			Role:    "system",
			Content: input.SystemMsg,
		})
	}
	for _, msg := range input.ChatHistory {
		textContent := ""
		imageContent := []string{}
		for _, content := range msg.Content {
			if content.Type == "text" {
				textContent = textContent + content.Text
			} else {
				imageContent = append(imageContent, base.TrimBase64Mime(content.ImageURL.URL))
			}
		}
		messages = append(messages, OllamaChatMessage{
			Role:    msg.Role,
			Content: textContent,
			Images:  imageContent,
		})
	}

	images := []string{}

	for _, image := range input.PromptImages {
		input.PromptImages = append(images, base.TrimBase64Mime(image))
	}

	messages = append(messages, OllamaChatMessage{
		Role:    "user",
		Content: input.Prompt,
		Images:  images,
	})

	request := ChatRequest{
		Model:    input.Model,
		Messages: messages,
		Stream:   false,
		Options: OllamaOptions{
			Temperature: input.Temperature,
			TopK:        input.TopK,
			Seed:        input.Seed,
		},
	}

	response, err := e.client.Chat(request)
	if err != nil {
		return nil, err
	}

	output := TaskTextGenerationChatOuput{
		Text: response.Message.Content,
	}
	return base.ConvertToStructpb(output)
}

type TaskTextEmbeddingsInput struct {
	Text  string `json:"text"`
	Model string `json:"model"`
}

type TaskTextEmbeddingsOutput struct {
	Embedding []float32 `json:"embedding"`
}

func (e *execution) TaskTextEmbeddings(in *structpb.Struct) (*structpb.Struct, error) {
	input := TaskTextEmbeddingsInput{}
	if err := base.ConvertFromStructpb(in, &input); err != nil {
		return nil, err
	}

	request := EmbedRequest{
		Model:  input.Model,
		Prompt: input.Text,
	}

	response, err := e.client.Embed(request)
	if err != nil {
		return nil, err
	}

	output := TaskTextEmbeddingsOutput(response)
	return base.ConvertToStructpb(output)
}
