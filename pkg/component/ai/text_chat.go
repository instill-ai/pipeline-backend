package ai

type TextChatInput struct {
	Data      InputData `json:"data"`
	Parameter Parameter `json:"parameter,omitempty"`
}

type InputData struct {
	Model    string         `json:"model"`
	Messages []InputMessage `json:"messages"`
}

type Parameter struct {
	MaxTokens   *int     `json:"max-tokens,omitempty"`
	Seed        *int     `json:"seed,omitempty"`
	N           *int     `json:"n,omitempty"`
	Temperature *float32 `json:"temperature,omitempty"`
	TopP        *float32 `json:"top-p,omitempty"`
	Stream      bool     `json:"stream,omitempty"`
}

type InputMessage struct {
	Contents []Content `json:"content"`
	Role     string    `json:"role"`
	Name     string    `json:"name,omitempty"`
}

type Content struct {
	Type        string `json:"type"`
	Text        string `json:"text,omitempty"`
	ImageURL    string `json:"image-url,omitempty"`
	ImageBase64 string `json:"image-base64,omitempty"`
}

type TextChatOutput struct {
	Data     OutputData `json:"data"`
	Metadata Metadata   `json:"metadata"`
}

type OutputData struct {
	Choices []Choice `json:"choices"`
}

type Choice struct {
	FinishReason string        `json:"finish-reason"`
	Index        int           `json:"index"`
	Message      OutputMessage `json:"message"`
	// The Unix timestamp (in seconds) of when the chat completion was created.
	Created int `json:"created"`
}

type OutputMessage struct {
	Content string `json:"content"`
	Role    string `json:"role"`
}

type Metadata struct {
	Usage Usage `json:"usage"`
}

type Usage struct {
	CompletionTokens int `json:"completion-tokens"`
	PromptTokens     int `json:"prompt-tokens"`
	TotalTokens      int `json:"total-tokens"`
}
