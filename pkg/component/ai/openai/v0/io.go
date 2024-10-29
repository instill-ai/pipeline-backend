package openai

import (
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
)

type taskTextGenerationInput struct {
	Prompt           string                     `instill:"prompt"`
	Images           []format.Image             `instill:"images"`
	ChatHistory      []*textMessage             `instill:"chat-history"`
	Model            string                     `instill:"model"`
	SystemMessage    *string                    `instill:"system-message"`
	Temperature      *float32                   `instill:"temperature"`
	TopP             *float32                   `instill:"top-p"`
	N                *int                       `instill:"n"`
	Stop             *string                    `instill:"stop"`
	MaxTokens        *int                       `instill:"max-tokens"`
	PresencePenalty  *float32                   `instill:"presence-penalty"`
	FrequencyPenalty *float32                   `instill:"frequency-penalty"`
	ResponseFormat   *responseFormatInputStruct `instill:"response-format"`
}

type taskTextGenerationOutput struct {
	Texts []string `instill:"texts"`
	Usage usage    `instill:"usage"`
}

type usage struct {
	PromptTokens     int `instill:"prompt-tokens"`
	CompletionTokens int `instill:"completion-tokens"`
	TotalTokens      int `instill:"total-tokens"`
}

type responseFormatInputStruct struct {
	Type       string `instill:"type"`
	JSONSchema string `instill:"json-schema"`
}

type textMessage struct {
	Role    string               `instill:"role"`
	Content []textMessageContent `instill:"content"`
}

type textMessageContent struct {
	Type     string    `instill:"type"`
	Text     *string   `instill:"text"`
	ImageURL *imageURL `instill:"image_url"`
}

type imageURL struct {
	URL string `instill:"url"`
}

type taskSpeechRecognitionInput struct {
	Audio       format.Audio `instill:"audio"`
	Model       string       `instill:"model"`
	Prompt      *string      `instill:"prompt"`
	Temperature *float32     `instill:"temperature"`
	Language    *string      `instill:"language"`
}

type taskSpeechRecognitionOutput struct {
	Text     string  `instill:"text"`
	Duration float32 `instill:"duration"`
}

type taskTextToSpeechInput struct {
	Text           string   `instill:"text"`
	Model          string   `instill:"model"`
	Voice          string   `instill:"voice"`
	ResponseFormat *string  `instill:"response-format"`
	Speed          *float32 `instill:"speed"`
}

type taskTextToSpeechOutput struct {
	Audio format.Audio `instill:"audio"`
}

type taskTextToImageInput struct {
	Prompt  string  `instill:"prompt"`
	Model   string  `instill:"model"`
	N       *int    `instill:"n"`
	Quality *string `instill:"quality"`
	Size    *string `instill:"size"`
	Style   *string `instill:"style"`
}

type taskTextToImageOutput struct {
	Results []imageGenerationsOutputResult `instill:"results"`
}

type imageGenerationsOutputResult struct {
	Image         format.Image `instill:"image"`
	RevisedPrompt string       `instill:"revised-prompt"`
}
type taskTextEmbeddingsInput struct {
	Text       string `instill:"text"`
	Model      string `instill:"model"`
	Dimensions int    `instill:"dimensions"`
}

type taskTextEmbeddingsOutput struct {
	Embedding []float32 `instill:"embedding"`
}
