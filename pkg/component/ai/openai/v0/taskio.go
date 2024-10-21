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

type taskSpeechRecognitionInput struct {
	Audio       data.Audio `key:"audio"`
	Model       string     `key:"model"`
	Prompt      *string    `key:"prompt"`
	Temperature *float32   `key:"temperature"`
	Language    *string    `key:"language"`
}

type taskSpeechRecognitionOutput struct {
	Text     string  `key:"text"`
	Duration float32 `key:"duration"`
}

type taskTextToSpeechInput struct {
	Text           string   `key:"text"`
	Model          string   `key:"model"`
	Voice          string   `key:"voice"`
	ResponseFormat *string  `key:"response-format"`
	Speed          *float32 `key:"speed"`
}

type taskTextToSpeechOutput struct {
	Audio data.Audio `key:"audio"`
}

type taskTextToImageInput struct {
	Prompt  string  `key:"prompt"`
	Model   string  `key:"model"`
	N       *int    `key:"n"`
	Quality *string `key:"quality"`
	Size    *string `key:"size"`
	Style   *string `key:"style"`
}

type taskTextToImageOutput struct {
	Results []imageGenerationsOutputResult `key:"results"`
}

type imageGenerationsOutputResult struct {
	Image         data.Image `key:"image"`
	RevisedPrompt string     `key:"revised-prompt"`
}
type taskTextEmbeddingsInput struct {
	Text       string `key:"text"`
	Model      string `key:"model"`
	Dimensions int    `key:"dimensions"`
}

type taskTextEmbeddingsOutput struct {
	Embedding []float32 `key:"embeddings"`
}
