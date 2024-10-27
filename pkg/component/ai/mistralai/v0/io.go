package mistralai

type ChatMessage struct {
	Role    string              `key:"role"`
	Content []MultiModalContent `key:"content"`
}
type URL struct {
	URL string `key:"url"`
}

type MultiModalContent struct {
	ImageURL URL    `key:"image-url"`
	Text     string `key:"text"`
	Type     string `key:"type"`
}

type TextGenerationInput struct {
	ChatHistory  []ChatMessage `key:"chat-history"`
	MaxNewTokens int           `key:"max-new-tokens"`
	ModelName    string        `key:"model-name"`
	Prompt       string        `key:"prompt"`
	PromptImages []string      `key:"prompt-images"`
	Seed         int           `key:"seed"`
	SystemMsg    string        `key:"system-message"`
	Temperature  float64       `key:"temperature"`
	TopK         int           `key:"top-k"`
	TopP         float64       `key:"top-p"`
	Safe         bool          `key:"safe"`
}

type chatUsage struct {
	InputTokens  int `key:"input-tokens"`
	OutputTokens int `key:"output-tokens"`
}

type TextGenerationOutput struct {
	Text  string    `key:"text"`
	Usage chatUsage `key:"usage"`
}

type TextEmbeddingInput struct {
	Text      string `key:"text"`
	ModelName string `key:"model-name"`
}

type textEmbeddingUsage struct {
	Tokens int `key:"tokens"`
}

type TextEmbeddingOutput struct {
	Embedding []float64          `key:"embedding"`
	Usage     textEmbeddingUsage `key:"usage"`
}
