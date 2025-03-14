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
	Temperature      *float32                   `instill:"temperature,default=1"`
	TopP             *float32                   `instill:"top-p,default=1"`
	N                *int                       `instill:"n,default=1"`
	Stop             *string                    `instill:"stop"`
	MaxTokens        *int                       `instill:"max-tokens"`
	PresencePenalty  *float32                   `instill:"presence-penalty,default=0"`
	FrequencyPenalty *float32                   `instill:"frequency-penalty,default=0"`
	ResponseFormat   *responseFormatInputStruct `instill:"response-format"`
	Prediction       *predictionStruct          `instill:"prediction"`
	Tools            []toolStruct               `instill:"tools"`
	ToolChoice       format.Value               `instill:"tool-choice"`
}

type taskTextGenerationOutput struct {
	Texts     []string   `instill:"texts"`
	ToolCalls []toolCall `instill:"tool-calls"`
	Usage     usage      `instill:"usage"`
}

type toolCall struct {
	Type     string       `instill:"type"`
	Function functionCall `instill:"function"`
}

type functionCall struct {
	Name      string `instill:"name"`
	Arguments string `instill:"arguments"`
}

type usage struct {
	PromptTokens           int                     `instill:"prompt-tokens"`
	CompletionTokens       int                     `instill:"completion-tokens"`
	TotalTokens            int                     `instill:"total-tokens"`
	CompletionTokenDetails *completionTokenDetails `instill:"completion-token-details"`
	PromptTokenDetails     *promptTokenDetails     `instill:"prompt-token-details"`
}

type promptTokenDetails struct {
	AudioTokens  int `instill:"audio-tokens"`
	CachedTokens int `instill:"cached-tokens"`
}

type completionTokenDetails struct {
	ReasoningTokens          int `instill:"reasoning-tokens"`
	AudioTokens              int `instill:"audio-tokens"`
	AcceptedPredictionTokens int `instill:"accepted-prediction-tokens"`
	RejectedPredictionTokens int `instill:"rejected-prediction-tokens"`
}

type responseFormatInputStruct struct {
	Type       string `instill:"type"`
	JSONSchema string `instill:"json-schema"`
}

type predictionStruct struct {
	Content string `instill:"content"`
}

type toolStruct struct {
	Function functionStruct `instill:"function"`
}

type functionStruct struct {
	Description string                  `instill:"description"`
	Name        string                  `instill:"name"`
	Parameters  map[string]format.Value `instill:"parameters"`
	Strict      *bool                   `instill:"strict,default=false"`
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
	Temperature *float32     `instill:"temperature,default=0"`
	Language    *string      `instill:"language"`
}

type taskSpeechRecognitionOutput struct {
	Text     string  `instill:"text"`
	Duration float32 `instill:"duration"`
}

type taskTextToSpeechInput struct {
	Text           string   `instill:"text"`
	Model          string   `instill:"model,default=tts-1"`
	Voice          string   `instill:"voice,default=alloy"`
	ResponseFormat *string  `instill:"response-format,default=mp3"`
	Speed          *float32 `instill:"speed,default=1"`
}

type taskTextToSpeechOutput struct {
	Audio format.Audio `instill:"audio"`
}

type taskTextToImageInput struct {
	Prompt  string  `instill:"prompt"`
	Model   string  `instill:"model"`
	N       *int    `instill:"n,default=1"`
	Quality *string `instill:"quality,default=standard"`
	Size    *string `instill:"size,default=1024x1024"`
	Style   *string `instill:"style,default=vivid"`
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
