package instillmodel

import (
	"encoding/base64"
	"fmt"

	"github.com/gabriel-vasile/mimetype"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/ai/openai/v0"
	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util"

	modelPB "github.com/instill-ai/protogen-go/model/model/v1alpha"
)

type TextGenerationInput struct {
	Prompt        string   `json:"prompt"`
	SystemMessage *string  `json:"system-message,omitempty"`
	PromptImages  []string `json:"prompt-images,omitempty"`

	// Note: We're currently sharing the same struct in the OpenAI component,
	// but this will be moved to the standardized format later.
	ChatHistory []*openai.TextMessage `json:"chat-history,omitempty"`
}

type ChatRequestData struct {
	Messages []Message `json:"messages,omitempty"`
}

type Message struct {
	Content []Content `json:"content,omitempty"`
	Role    string    `json:"role,omitempty"`
}

type Content struct {
	Text        string `json:"text,omitempty"`
	ImageBase64 string `json:"image-base64,omitempty"`
	Type        string `json:"type,omitempty"`
}

type ChatParameter struct {
	MaxTokens   int     `json:"max-tokens,omitempty"`
	Seed        int     `json:"seed,omitempty"`
	N           int     `json:"n,omitempty"`
	Temperature float32 `json:"temperature,omitempty"`
	TopP        int     `json:"top-p,omitempty"`
}

func (e *execution) executeTextGenerationChat(grpcClient modelPB.ModelPublicServiceClient, nsID string, modelID string, version string, inputs []*structpb.Struct) ([]*structpb.Struct, error) {

	if len(inputs) <= 0 {
		return nil, fmt.Errorf("invalid input: %v for model: %s/%s/%s", inputs, nsID, modelID, version)
	}

	if grpcClient == nil {
		return nil, fmt.Errorf("uninitialized client")
	}

	taskInputs := []*structpb.Struct{}
	for _, input := range inputs {
		inputStruct := TextGenerationInput{}
		err := base.ConvertFromStructpb(input, &inputStruct)
		if err != nil {
			return nil, err
		}
		messages := []Message{}

		// If chat history is provided, add it to the messages, and ignore the system message
		if inputStruct.ChatHistory != nil {
			for _, chat := range inputStruct.ChatHistory {
				contents := make([]Content, len(chat.Content))
				for i, c := range chat.Content {
					if c.Type == "text" {
						contents[i] = Content{
							Text: *c.Text,
							Type: "text",
						}
					} else {
						contents[i] = Content{
							ImageBase64: c.ImageURL.URL,
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

		i := &RequestWrapper{
			Data: &ChatRequestData{
				Messages: messages,
			},
			Parameter: &ChatParameter{
				N: 1,
			},
		}
		if _, ok := input.GetFields()["seed"]; ok {
			v := int(input.GetFields()["seed"].GetNumberValue())
			i.Parameter.(*ChatParameter).Seed = v
		}
		if _, ok := input.GetFields()["max-new-tokens"]; ok {
			v := int(input.GetFields()["max-new-tokens"].GetNumberValue())
			i.Parameter.(*ChatParameter).MaxTokens = v
		}
		if _, ok := input.GetFields()["temperature"]; ok {
			v := float32(input.GetFields()["temperature"].GetNumberValue())
			i.Parameter.(*ChatParameter).Temperature = v
		}
		taskInput, err := base.ConvertToStructpb(i)
		if err != nil {
			return nil, err
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

	outputs := []*structpb.Struct{}
	for idx := range inputs {
		choices := taskOutputs[idx].Fields["data"].GetStructValue().Fields["choices"].GetListValue()
		output := structpb.Struct{Fields: make(map[string]*structpb.Value)}
		output.Fields["text"] = structpb.NewStringValue(
			choices.GetValues()[0].GetStructValue().
				Fields["message"].GetStructValue().
				Fields["content"].GetStringValue())
		outputs = append(outputs, &output)
	}
	return outputs, nil
}
