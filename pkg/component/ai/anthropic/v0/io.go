package anthropic

type MessagesInput struct {
	ChatHistory  []ChatMessage `key:"chat-history"`
	MaxNewTokens int           `key:"max-new-tokens"`
	ModelName    string        `key:"model-name"`
	Prompt       string        `key:"prompt"`
	PromptImages []string      `key:"prompt-images"`
	Seed         int           `key:"seed"`
	SystemMsg    string        `key:"system-message"`
	Temperature  float32       `key:"temperature"`
	TopK         int           `key:"top-k"`
}

type ChatMessage struct {
	Role    string              `key:"role"`
	Content []MultiModalContent `key:"content"`
}

type MultiModalContent struct {
	ImageURL URL    `key:"image-url"`
	Text     string `key:"text"`
	Type     string `key:"type"`
}

type URL struct {
	URL string `key:"url"`
}

type MessagesOutput struct {
	Text  string        `key:"text"`
	Usage messagesUsage `key:"usage"`
}

type messagesUsage struct {
	InputTokens  int `key:"input-tokens"`
	OutputTokens int `key:"output-tokens"`
}
