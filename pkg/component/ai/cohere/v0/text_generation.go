package cohere

import (
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"

	cohereSDK "github.com/cohere-ai/cohere-go/v2"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

type ChatMessage struct {
	Role    string              `json:"role"`
	Content []MultiModalContent `json:"content"`
}
type URL struct {
	URL string `json:"url"`
}

type MultiModalContent struct {
	ImageURL URL    `json:"image-url"`
	Text     string `json:"text"`
	Type     string `json:"type"`
}

type TextGenerationInput struct {
	ChatHistory  []ChatMessage `json:"chat-history"`
	MaxNewTokens int           `json:"max-new-tokens"`
	ModelName    string        `json:"model-name"`
	Prompt       string        `json:"prompt"`
	PromptImages []string      `json:"prompt-images"`
	Seed         int           `json:"seed"`
	SystemMsg    string        `json:"system-message"`
	Temperature  float64       `json:"temperature"`
	TopK         int           `json:"top-k"`
	Documents    []string      `json:"documents"`
}

type citation struct {
	Start int    `json:"start"`
	End   int    `json:"end"`
	Text  string `json:"text"`
}

type commandUsage struct {
	InputTokens  int `json:"input-tokens"`
	OutputTokens int `json:"output-tokens"`
}

type TextGenerationOutput struct {
	Text      string       `json:"text"`
	Citations []citation   `json:"citations"`
	Usage     commandUsage `json:"usage"`
}

func (e *execution) taskTextGeneration(in *structpb.Struct) (*structpb.Struct, error) {

	inputStruct := TextGenerationInput{}
	err := base.ConvertFromStructpb(in, &inputStruct)
	if err != nil {
		return nil, fmt.Errorf("error generating input struct: %v", err)
	}
	messages := []*cohereSDK.Message{}

	if inputStruct.SystemMsg != "" {
		message := cohereSDK.Message{}
		message.Role = "SYSTEM"
		message.Chatbot = &cohereSDK.ChatMessage{Message: inputStruct.SystemMsg}
		messages = append(messages, &message)
	}

	for _, chatMessage := range inputStruct.ChatHistory {
		messageContent := ""
		for _, content := range chatMessage.Content {
			if content.Type == "text" {
				messageContent += content.Text
			}
		}
		message := cohereSDK.Message{}
		message.Role = chatMessage.Role
		switch message.Role {
		case "USER":
			message.User = &cohereSDK.ChatMessage{Message: messageContent}
		case "CHATBOT":
			message.Chatbot = &cohereSDK.ChatMessage{Message: messageContent}
		}
		messages = append(messages, &message)
	}

	documents := []map[string]string{}
	for _, doc := range inputStruct.Documents {
		document := map[string]string{}
		document["text"] = doc
		documents = append(documents, document)
	}

	req := cohereSDK.ChatRequest{
		Message:     inputStruct.Prompt,
		Model:       &inputStruct.ModelName,
		ChatHistory: messages,
		MaxTokens:   &inputStruct.MaxNewTokens,
		Temperature: &inputStruct.Temperature,
		K:           &inputStruct.TopK,
		Seed:        &inputStruct.Seed,
		Documents:   documents,
	}

	resp, err := e.client.generateTextChat(req)

	if err != nil {
		return nil, err
	}

	citations := []citation{}

	for _, c := range resp.Citations {
		citation := citation{
			Start: c.Start,
			End:   c.End,
			Text:  c.Text,
		}
		citations = append(citations, citation)
	}
	bills := resp.Meta.BilledUnits
	inputTokens := *bills.InputTokens
	outputTokens := *bills.OutputTokens

	outputStruct := TextGenerationOutput{
		Text:      resp.Text,
		Citations: citations,
		Usage: commandUsage{
			InputTokens:  int(inputTokens),
			OutputTokens: int(outputTokens),
		},
	}

	output, err := base.ConvertToStructpb(outputStruct)
	if err != nil {
		return nil, err
	}
	return output, nil
}
