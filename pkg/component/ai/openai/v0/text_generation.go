package openai

const (
	completionsPath = "/v1/chat/completions"
)

type TextMessage struct {
	Role    string    `json:"role"`
	Content []Content `json:"content"`
}

type TextCompletionInput struct {
	Prompt           string                     `json:"prompt"`
	Images           []string                   `json:"images"`
	ChatHistory      []*TextMessage             `json:"chat-history,omitempty"`
	Model            string                     `json:"model"`
	SystemMessage    *string                    `json:"system-message,omitempty"`
	Temperature      *float32                   `json:"temperature,omitempty"`
	TopP             *float32                   `json:"top-p,omitempty"`
	N                *int                       `json:"n,omitempty"`
	Stop             *string                    `json:"stop,omitempty"`
	MaxTokens        *int                       `json:"max-tokens,omitempty"`
	PresencePenalty  *float32                   `json:"presence-penalty,omitempty"`
	FrequencyPenalty *float32                   `json:"frequency-penalty,omitempty"`
	ResponseFormat   *responseFormatInputStruct `json:"response-format,omitempty"`
}

type responseFormatInputStruct struct {
	Type       string `json:"type,omitempty"`
	JSONSchema string `json:"json-schema,omitempty"`
}

type TextCompletionOutput struct {
	Texts []string `json:"texts"`
	Usage usage    `json:"usage"`
}

type textCompletionReq struct {
	Model            string                   `json:"model"`
	Messages         []interface{}            `json:"messages"`
	Temperature      *float32                 `json:"temperature,omitempty"`
	TopP             *float32                 `json:"top_p,omitempty"`
	N                *int                     `json:"n,omitempty"`
	Stop             *string                  `json:"stop,omitempty"`
	MaxTokens        *int                     `json:"max_tokens,omitempty"`
	PresencePenalty  *float32                 `json:"presence_penalty,omitempty"`
	FrequencyPenalty *float32                 `json:"frequency_penalty,omitempty"`
	ResponseFormat   *responseFormatReqStruct `json:"response_format,omitempty"`
	Stream           bool                     `json:"stream"`
	StreamOptions    *streamOptions           `json:"stream_options,omitempty"`
}

type streamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

type responseFormatReqStruct struct {
	Type       string         `json:"type,omitempty"`
	JSONSchema map[string]any `json:"json_schema,omitempty"`
}

type multiModalMessage struct {
	Role    string    `json:"role"`
	Content []Content `json:"content"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ImageURL struct {
	URL string `json:"url"`
}

type Content struct {
	Type     string    `json:"type"`
	Text     *string   `json:"text,omitempty"`
	ImageURL *ImageURL `json:"image_url,omitempty"`
}

type textCompletionStreamResp struct {
	ID      string          `json:"id"`
	Object  string          `json:"object"`
	Created int             `json:"created"`
	Choices []streamChoices `json:"choices"`
	Usage   usageOpenAI     `json:"usage"`
}
type textCompletionResp struct {
	ID      string      `json:"id"`
	Object  string      `json:"object"`
	Created int         `json:"created"`
	Choices []choices   `json:"choices"`
	Usage   usageOpenAI `json:"usage"`
}

type outputMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type streamChoices struct {
	Index        int           `json:"index"`
	FinishReason string        `json:"finish_reason"`
	Delta        outputMessage `json:"delta"`
}

type choices struct {
	Index        int           `json:"index"`
	FinishReason string        `json:"finish_reason"`
	Message      outputMessage `json:"message"`
}

type usageOpenAI struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type usage struct {
	PromptTokens     int `json:"prompt-tokens"`
	CompletionTokens int `json:"completion-tokens"`
	TotalTokens      int `json:"total-tokens"`
}
