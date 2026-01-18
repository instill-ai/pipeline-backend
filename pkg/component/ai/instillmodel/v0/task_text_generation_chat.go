package instillmodel

import (
	"encoding/base64"
	"fmt"

	"github.com/gabriel-vasile/mimetype"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util"

	modelpb "github.com/instill-ai/protogen-go/model/v1alpha"
)

func (e *execution) executeTextGenerationChat(grpcClient modelpb.ModelPublicServiceClient, nsID string, modelID string, version string, inputs []*structpb.Struct) ([]*structpb.Struct, error) {

	if len(inputs) <= 0 {
		return nil, fmt.Errorf("invalid input: %v for model: %s/%s/%s", inputs, nsID, modelID, version)
	}

	if grpcClient == nil {
		return nil, fmt.Errorf("uninitialized client")
	}

	// Transform inputs to standardized format
	taskInputs := []*structpb.Struct{}
	for _, input := range inputs {
		// Parse the input using TextGenerationChatInput struct
		var inputStruct TextGenerationChatInput
		err := base.ConvertFromStructpb(input, &inputStruct)
		if err != nil {
			return nil, fmt.Errorf("failed to convert input to TextGenerationChatInput struct: %w", err)
		}

		messages := []Message{}

		// If chat history is provided, add it to the messages, and ignore the system message
		if len(inputStruct.ChatHistory) > 0 {
			for _, chat := range inputStruct.ChatHistory {
				contents := make([]Content, len(chat.Content))
				for i, c := range chat.Content {
					if c.Type == "text" {
						contents[i] = Content{
							Text: c.Text,
							Type: "text",
						}
					} else {
						contents[i] = Content{
							ImageBase64: c.ImageURL,
							Type:        "image-base64",
						}
					}
				}
				messages = append(messages, Message{Role: chat.Role, Content: contents})
			}
		} else if inputStruct.SystemMessage != nil {
			contents := make([]Content, 1)
			contents[0] = Content{Text: *inputStruct.SystemMessage, Type: "text"}
			// If chat history is not provided, add the system message to the messages
			messages = append(messages, Message{Role: "system", Content: contents})
		}
		userContents := []Content{}
		userContents = append(userContents, Content{Type: "text", Text: inputStruct.Prompt})
		for _, image := range inputStruct.PromptImages {
			b, err := base64.StdEncoding.DecodeString(util.TrimBase64Mime(image))
			if err != nil {
				return nil, err
			}
			url := fmt.Sprintf("data:%s;base64,%s", mimetype.Detect(b).String(), util.TrimBase64Mime(image))
			userContents = append(userContents, Content{Type: "image-base64", ImageBase64: url})
		}
		messages = append(messages, Message{Role: "user", Content: userContents})

		// Create request parameters
		requestParam := &ChatParameter{
			N: 1,
		}
		if inputStruct.Seed != nil {
			requestParam.Seed = *inputStruct.Seed
		}
		if inputStruct.MaxNewTokens != nil {
			requestParam.MaxTokens = *inputStruct.MaxNewTokens
		}
		if inputStruct.Temperature != nil {
			requestParam.Temperature = *inputStruct.Temperature
		}

		// Create standardized request wrapper
		requestWrapper := &RequestWrapper{
			Data: &ChatRequestData{
				Messages: messages,
			},
			Parameter: requestParam,
		}

		// Convert to structpb
		taskInput, err := base.ConvertToStructpb(requestWrapper)
		if err != nil {
			return nil, fmt.Errorf("failed to convert request wrapper to structpb: %w", err)
		}
		taskInputs = append(taskInputs, taskInput)
	}

	taskOutputs, err := trigger(grpcClient, e.SystemVariables, nsID, modelID, version, taskInputs)
	if err != nil {
		return nil, err
	}
	if len(taskOutputs) <= 0 {
		return nil, fmt.Errorf("invalid output: %v for model: %s/%s/%s", taskOutputs, nsID, modelID, version)
	}

	// Transform raw outputs to standardized TextGenerationChatOutput format
	outputs := []*structpb.Struct{}
	for idx := range inputs {
		choices := taskOutputs[idx].Fields["data"].GetStructValue().Fields["choices"].GetListValue()

		// Create standardized output structure
		textGenChatOutput := TextGenerationChatOutput{
			Text: choices.GetValues()[0].GetStructValue().
				Fields["message"].GetStructValue().
				Fields["content"].GetStringValue(),
		}

		// Convert to structpb
		outputStruct, err := base.ConvertToStructpb(textGenChatOutput)
		if err != nil {
			return nil, fmt.Errorf("failed to convert text generation chat output to structpb: %w", err)
		}

		outputs = append(outputs, outputStruct)
	}
	return outputs, nil
}
