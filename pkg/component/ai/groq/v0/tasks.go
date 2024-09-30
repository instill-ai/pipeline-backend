package groq

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

	// additional parameters
	TopP float32 `json:"top-p"`
	User string  `json:"user"`
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

	messages := []GroqChatMessageInterface{}

	if input.SystemMsg != "" {
		messages = append(messages, GroqSystemMessage{
			Role:    "system",
			Content: input.SystemMsg,
		})
	}
	for _, msg := range input.ChatHistory {
		messageContents := []GroqChatContent{}
		for _, inputContent := range msg.Content {
			if inputContent.Type == "text" {
				messageContents = append(messageContents, GroqChatContent{Text: inputContent.Text, Type: GroqChatContentTypeText})
			} else {
				messageContents = append(messageContents, GroqChatContent{ImageURL: &GroqURL{URL: inputContent.ImageURL.URL}, Type: GroqChatContentTypeImage})
			}
		}
		messages = append(messages, GroqChatMessage{
			Role:    msg.Role,
			Content: messageContents,
		})
	}

	promptContents := []GroqChatContent{}

	for _, promptImage := range input.PromptImages {
		promptContents = append(promptContents, GroqChatContent{ImageURL: &GroqURL{URL: promptImage}, Type: GroqChatContentTypeImage})
	}

	promptContents = append(promptContents, GroqChatContent{Text: input.Prompt, Type: GroqChatContentTypeText})

	messages = append(messages, GroqChatMessage{
		Role:    "user",
		Content: promptContents,
	})

	request := ChatRequest{
		MaxTokens:   input.MaxNewTokens,
		Model:       input.Model,
		Messages:    messages,
		N:           1,
		Seed:        input.Seed,
		Temperature: input.Temperature,
		TopP:        input.TopP,
		Stop:        []string{},
		User:        input.User,
	}

	response, err := e.client.Chat(request)
	if err != nil {
		return nil, err
	}

	output := TaskTextGenerationChatOuput{
		Text: response.Choices[0].Message.Content,
		Usage: TaskTextGenerationChatUsage{
			InputTokens:  response.Usage.PromptTokens,
			OutputTokens: response.Usage.CompletionTokens,
		},
	}
	return base.ConvertToStructpb(output)
}
