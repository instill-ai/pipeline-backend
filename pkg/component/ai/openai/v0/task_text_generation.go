package openai

const (
	completionsPath = "/v1/chat/completions"
)

type TextMessage struct {
	Role    string    `json:"role"`
	Content []Content `json:"content"`
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
	Prediction       *predictionReqStruct     `json:"prediction,omitempty"`
	Tools            []toolReqStruct          `json:"tools,omitempty"`
	ToolChoice       any                      `json:"tool_choice,omitempty"`
	ReasoningEffort  *string                  `json:"reasoning_effort,omitempty"`
	Verbosity        *string                  `json:"verbosity,omitempty"`
}

type predictionReqStruct struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

type toolReqStruct struct {
	Type     string            `json:"type"`
	Function functionReqStruct `json:"function"`
}

type functionReqStruct struct {
	Description string         `json:"description"`
	Name        string         `json:"name"`
	Parameters  map[string]any `json:"parameters"`
	Strict      *bool          `json:"strict"`
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

type outputMessage struct {
	Role      string         `json:"role"`
	Content   string         `json:"content"`
	ToolCalls []toolCallResp `json:"tool_calls,omitempty"`
}

type toolCallResp struct {
	Index    int              `json:"index"`
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function functionCallResp `json:"function"`
}

type functionCallResp struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type streamChoices struct {
	Index        int           `json:"index"`
	FinishReason string        `json:"finish_reason"`
	Delta        outputMessage `json:"delta"`
}

type usageOpenAI struct {
	PromptTokens           int                          `json:"prompt_tokens"`
	CompletionTokens       int                          `json:"completion_tokens"`
	TotalTokens            int                          `json:"total_tokens"`
	PromptTokenDetails     promptTokenDetailsOpenAI     `json:"prompt_token_details"`
	CompletionTokenDetails completionTokenDetailsOpenAI `json:"completion_tokens_details"`
}

type promptTokenDetailsOpenAI struct {
	AudioTokens  int `json:"audio_tokens"`
	CachedTokens int `json:"cached_tokens"`
}

type completionTokenDetailsOpenAI struct {
	ReasoningTokens          int `json:"reasoning_tokens"`
	AudioTokens              int `json:"audio_tokens"`
	AcceptedPredictionTokens int `json:"accepted_prediction_tokens"`
	RejectedPredictionTokens int `json:"rejected_prediction_tokens"`
}
