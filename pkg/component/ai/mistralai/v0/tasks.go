package mistralai

import (
	"context"
	"fmt"

	mistralSDK "github.com/gage-technologies/mistral-go"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func (e *execution) taskTextGeneration(ctx context.Context, job *base.Job) error {

	inputStruct := TextGenerationInput{}
	err := job.Input.ReadData(ctx, &inputStruct)
	if err != nil {
		return fmt.Errorf("error generating input struct: %v", err)
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
		return fmt.Errorf("error calling Chat: %v", err)
	}

	outputStruct := TextGenerationOutput{}
	outputStruct.Text = ""

	for resp := range respStream {

		outputStruct.Text += resp.Choices[0].Delta.Content
		outputStruct.Usage = chatUsage{
			InputTokens:  resp.Usage.PromptTokens,
			OutputTokens: resp.Usage.CompletionTokens,
		}
		err = job.Output.WriteData(ctx, outputStruct)
		if err != nil {
			job.Error.Error(ctx, err)
			return err
		}
	}

	return nil
}

func (e *execution) taskTextEmbedding(ctx context.Context, job *base.Job) error {

	inputStruct := TextEmbeddingInput{}
	err := job.Input.ReadData(ctx, &inputStruct)
	if err != nil {
		return fmt.Errorf("error generating input struct: %v", err)
	}

	resp, err := e.client.sdkClient.Embeddings(inputStruct.ModelName, []string{inputStruct.Text})
	if err != nil {
		return fmt.Errorf("error calling Embeddings: %v", err)
	}
	outputStruct := TextEmbeddingOutput{
		Embedding: resp.Data[0].Embedding,
		Usage: textEmbeddingUsage{
			Tokens: resp.Usage.TotalTokens,
		},
	}
	err = job.Output.WriteData(ctx, outputStruct)
	if err != nil {
		job.Error.Error(ctx, err)
		return err
	}
	return nil
}
