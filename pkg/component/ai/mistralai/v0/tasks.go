package mistralai

import (
	"context"
	"encoding/json"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	mistralSDK "github.com/gage-technologies/mistral-go"

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
	TopP         float64       `json:"top-p"`
	Safe         bool          `json:"safe"`
}

type chatUsage struct {
	InputTokens  int `json:"input-tokens"`
	OutputTokens int `json:"output-tokens"`
}

type TextGenerationOutput struct {
	Text  string    `json:"text"`
	Usage chatUsage `json:"usage"`
}

type TextEmbeddingInput struct {
	Text      string `json:"text"`
	ModelName string `json:"model-name"`
}

type textEmbeddingUsage struct {
	Tokens int `json:"tokens"`
}

type TextEmbeddingOutput struct {
	Embedding []float64          `json:"embedding"`
	Usage     textEmbeddingUsage `json:"usage"`
}

func (e *execution) taskTextGeneration(_ *structpb.Struct, job *base.Job, ctx context.Context) (*structpb.Struct, error) {

	input, err := job.Input.Read(ctx)
	if err != nil {
		return nil, fmt.Errorf("error reading input: %v", err)
	}

	inputStruct := TextGenerationInput{}
	err = base.ConvertFromStructpb(input, &inputStruct)
	if err != nil {
		return nil, fmt.Errorf("error generating input struct: %v", err)
	}

	messages := []mistralSDK.ChatMessage{}

	if inputStruct.SystemMsg != "" {
		messages = append(messages, mistralSDK.ChatMessage{
			Role:    "system",
			Content: inputStruct.SystemMsg,
		})
	}
	for _, chatMessage := range inputStruct.ChatHistory {
		messageContent := ""
		for _, content := range chatMessage.Content {
			if content.Type == "text" {
				messageContent += content.Text
			}
		}
		if messageContent == "" {
			continue
		}
		messages = append(messages, mistralSDK.ChatMessage{
			Role:    chatMessage.Role,
			Content: messageContent,
		})
	}

	promptMessage := mistralSDK.ChatMessage{
		Role:    "user",
		Content: inputStruct.Prompt,
	}

	messages = append(messages, promptMessage)

	params := mistralSDK.ChatRequestParams{
		Temperature: inputStruct.Temperature,
		RandomSeed:  inputStruct.Seed,
		MaxTokens:   inputStruct.MaxNewTokens,
		TopP:        inputStruct.TopP,
		SafePrompt:  inputStruct.Safe,
	}

	respStream, err := e.client.sdkClient.ChatStream(
		inputStruct.ModelName,
		messages,
		&params,
	)

	if err != nil {
		return nil, fmt.Errorf("error calling Chat: %v", err)
	}

	outputStruct := TextGenerationOutput{}
	outputStruct.Text = ""

	output := &structpb.Struct{}
	for resp := range respStream {
		fmt.Println("resp")
		fmt.Println(resp.Choices[0].Delta.Content)
		fmt.Println(resp.Usage)

		outputStruct.Text += resp.Choices[0].Delta.Content
		outputStruct.Usage = chatUsage{
			InputTokens:  resp.Usage.PromptTokens,
			OutputTokens: resp.Usage.CompletionTokens,
		}
		outputJSON, err := json.Marshal(outputStruct)
		if err != nil {
			job.Error.Error(ctx, err)
			return nil, err
		}

		err = protojson.Unmarshal(outputJSON, output)
		if err != nil {
			job.Error.Error(ctx, err)
			return nil, err
		}

		err = job.Output.Write(ctx, output)
		if err != nil {
			job.Error.Error(ctx, err)
			return nil, err
		}
	}

	return output, nil
}

func (e *execution) taskTextEmbedding(in *structpb.Struct, job *base.Job, ctx context.Context) (*structpb.Struct, error) {
	inputStruct := TextEmbeddingInput{}
	err := base.ConvertFromStructpb(in, &inputStruct)
	if err != nil {
		return nil, fmt.Errorf("error generating input struct: %v", err)
	}

	resp, err := e.client.sdkClient.Embeddings(inputStruct.ModelName, []string{inputStruct.Text})
	if err != nil {
		return nil, fmt.Errorf("error calling Embeddings: %v", err)
	}
	outputStruct := TextEmbeddingOutput{
		Embedding: resp.Data[0].Embedding,
		Usage: textEmbeddingUsage{
			Tokens: resp.Usage.TotalTokens,
		},
	}
	output, err := base.ConvertToStructpb(outputStruct)
	if err != nil {
		return nil, err
	}
	return output, nil

}
