package openai

import "github.com/instill-ai/pipeline-backend/pkg/data"

type taskTextGenerationInput struct {
	Prompt           string                     `key:"prompt"`
	Images           []data.Image               `key:"images"`
	ChatHistory      []*textMessage             `key:"chat-history"`
	Model            string                     `key:"model"`
	SystemMessage    *string                    `key:"system-message"`
	Temperature      *float32                   `key:"temperature"`
	TopP             *float32                   `key:"top-p"`
	N                *int                       `key:"n"`
	Stop             *string                    `key:"stop"`
	MaxTokens        *int                       `key:"max-tokens"`
	PresencePenalty  *float32                   `key:"presence-penalty"`
	FrequencyPenalty *float32                   `key:"frequency-penalty"`
	ResponseFormat   *responseFormatInputStruct `key:"response-format"`
}

type taskTextGenerationOutput struct {
	Texts []string `key:"texts"`
	Usage usage    `key:"usage"`
}

type usage struct {
	PromptTokens     int `key:"prompt-tokens"`
	CompletionTokens int `key:"completion-tokens"`
	TotalTokens      int `key:"total-tokens"`
}

type responseFormatInputStruct struct {
	Type       string `key:"type"`
	JSONSchema string `key:"json-schema"`
}

type textMessage struct {
	Role    string               `key:"role"`
	Content []textMessageContent `key:"content"`
}

type textMessageContent struct {
	Type     string    `key:"type"`
	Text     *string   `key:"text"`
	ImageURL *imageURL `key:"image_url"`
}

type imageURL struct {
	URL string `key:"url"`
}
