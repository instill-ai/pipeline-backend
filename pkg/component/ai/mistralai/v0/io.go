package mistralai

type ChatMessage struct {
	Role    string              `instill:"role"`
	Content []MultiModalContent `instill:"content"`
}
type URL struct {
	URL string `instill:"url"`
}

type MultiModalContent struct {
	ImageURL URL    `instill:"image-url"`
	Text     string `instill:"text"`
	Type     string `instill:"type"`
}

type TextGenerationInput struct {
	ChatHistory  []ChatMessage `instill:"chat-history"`
	MaxNewTokens int           `instill:"max-new-tokens"`
	ModelName    string        `instill:"model-name"`
	Prompt       string        `instill:"prompt"`
	PromptImages []string      `instill:"prompt-images"`
	Seed         int           `instill:"seed"`
	SystemMsg    string        `instill:"system-message"`
	Temperature  float64       `instill:"temperature"`
	TopK         int           `instill:"top-k"`
	TopP         float64       `instill:"top-p"`
	Safe         bool          `instill:"safe"`
}

type chatUsage struct {
	InputTokens  int `instill:"input-tokens"`
	OutputTokens int `instill:"output-tokens"`
}

type TextGenerationOutput struct {
	Text  string    `instill:"text"`
	Usage chatUsage `instill:"usage"`
}

type TextEmbeddingInput struct {
	Text      string `instill:"text"`
	ModelName string `instill:"model-name"`
}

type textEmbeddingUsage struct {
	Tokens int `instill:"tokens"`
}

type TextEmbeddingOutput struct {
	Embedding []float64          `instill:"embedding"`
	Usage     textEmbeddingUsage `instill:"usage"`
}
