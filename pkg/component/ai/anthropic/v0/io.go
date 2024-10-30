package anthropic

type MessagesInput struct {
	ChatHistory  []ChatMessage `instill:"chat-history"`
	MaxNewTokens int           `instill:"max-new-tokens"`
	ModelName    string        `instill:"model-name"`
	Prompt       string        `instill:"prompt"`
	PromptImages []string      `instill:"prompt-images"`
	Seed         int           `instill:"seed"`
	SystemMsg    string        `instill:"system-message"`
	Temperature  float32       `instill:"temperature"`
	TopK         int           `instill:"top-k"`
}

type ChatMessage struct {
	Role    string              `instill:"role"`
	Content []MultiModalContent `instill:"content"`
}

type MultiModalContent struct {
	ImageURL URL    `instill:"image-url"`
	Text     string `instill:"text"`
	Type     string `instill:"type"`
}

type URL struct {
	URL string `instill:"url"`
}

type MessagesOutput struct {
	Text  string        `instill:"text"`
	Usage messagesUsage `instill:"usage"`
}

type messagesUsage struct {
	InputTokens  int `instill:"input-tokens"`
	OutputTokens int `instill:"output-tokens"`
}
