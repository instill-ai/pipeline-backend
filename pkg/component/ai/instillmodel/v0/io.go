package instillmodel

import (
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
)

// RequestWrapper is the standardized request wrapper for the Instill Model.
type RequestWrapper struct {
	Data      any `instill:"data"`
	Parameter any `instill:"parameter"`
}

// EMBEDDING TASK STRUCTS

// EmbeddingInput is the standardized input for the embedding model.
type EmbeddingInput struct {
	// Data is the the standardized input data for the embedding model.
	Data EmbeddingInputData `instill:"data"`
	// Parameter is the standardized parameter for the embedding model.
	Parameter EmbeddingParameter `instill:"parameter"`
}

// EmbeddingInputData is the standardized input data for the embedding model.
type EmbeddingInputData struct {
	// Model is the model name.
	Model string `instill:"model"`
	// Embeddings is the list of data to be embedded.
	Embeddings []InputEmbedding `instill:"embeddings"`
}

// InputEmbedding is the standardized input data to be embedded.
type InputEmbedding struct {
	// Type is the type of the input data. It can be either "text", "image-url", or "image-base64".
	Type string `instill:"type"`
	// Text is the text to be embedded.
	Text string `instill:"text"`
	// ImageURL is the URL of the image to be embedded.
	ImageURL string `instill:"image-url"`
	// ImageBase64 is the base64 encoded image to be embedded.
	ImageBase64 string `instill:"image-base64"`
}

// EmbeddingParameter is the standardized parameter for the embedding model.
type EmbeddingParameter struct {
	// Format is the format of the output embeddings. Default is "float", can be "float" or "base64".
	Format string `instill:"format"`
	// Dimensions is the number of dimensions of the output embeddings.
	Dimensions int `instill:"dimensions"`
	// InputType is the type of the input data. It can be "query" or "data".
	InputType string `instill:"input-type"`
	// Truncate is how to handle inputs longer than the max token length. Defaults to 'End'. Can be 'End', 'Start', or 'None'.
	Truncate string `instill:"truncate"`
}

// EmbeddingOutput is the standardized output for the embedding model.
type EmbeddingOutput struct {
	// Data is the standardized output data for the embedding model.
	Data EmbeddingOutputData `instill:"data"`
}

// EmbeddingOutputData is the standardized output data for the embedding model.
type EmbeddingOutputData struct {
	// Embeddings is the list of output embeddings.
	Embeddings []OutputEmbedding `instill:"embeddings"`
}

// OutputEmbedding is the standardized output embedding.
type OutputEmbedding struct {
	// Index is the index of the output embedding.
	Index int `instill:"index"`
	// Vector is the output embedding.
	Vector []any `instill:"vector"`
	// Created is the Unix timestamp (in seconds) of when the embedding was created.
	Created int `instill:"created"`
}

// TEXT GENERATION TASK STRUCTS

// TextGenerationInput is the standardized input for text generation tasks.
type TextGenerationInput struct {
	ModelName     string   `instill:"model-name"`
	Prompt        string   `instill:"prompt"`
	SystemMessage *string  `instill:"system-message"`
	MaxNewTokens  *int     `instill:"max-new-tokens"`
	Temperature   *float32 `instill:"temperature"`
	Seed          *int     `instill:"seed"`
}

// TextGenerationRequestData is the request data for text generation.
type TextGenerationRequestData struct {
	Prompt        string `instill:"prompt"`
	SystemMessage string `instill:"system-message"`
}

// TextGenerationRequestParameter is the request parameter for text generation.
type TextGenerationRequestParameter struct {
	MaxTokens   int     `instill:"max-tokens"`
	Seed        int     `instill:"seed"`
	N           int     `instill:"n"`
	Temperature float32 `instill:"temperature"`
	TopP        int     `instill:"top-p"`
}

// TextGenerationOutput is the standardized output for text generation tasks.
type TextGenerationOutput struct {
	Text string `instill:"text"`
}

// TEXT GENERATION CHAT TASK STRUCTS

// MultiModalContent represents multimodal content for chat messages.
type MultiModalContent struct {
	ImageURL string `instill:"image-url"`
	Text     string `instill:"text"`
	Type     string `instill:"type"`
}

// ChatMessage represents a chat message.
type ChatMessage struct {
	Role    string              `instill:"role"`
	Content []MultiModalContent `instill:"content"`
}

// TextGenerationChatInput is the standardized input for text generation chat tasks.
type TextGenerationChatInput struct {
	ModelName     string        `instill:"model-name"`
	Prompt        string        `instill:"prompt"`
	PromptImages  []string      `instill:"prompt-images"`
	ChatHistory   []ChatMessage `instill:"chat-history"`
	SystemMessage *string       `instill:"system-message"`
	MaxNewTokens  *int          `instill:"max-new-tokens"`
	Temperature   *float32      `instill:"temperature"`
	Seed          *int          `instill:"seed"`
}

// Content represents message content for chat.
type Content struct {
	Type        string `instill:"type"`
	Text        string `instill:"text"`
	ImageBase64 string `instill:"image-base64"`
}

// Message represents a chat message for the API.
type Message struct {
	Role    string    `instill:"role"`
	Content []Content `instill:"content"`
}

// ChatRequestData is the request data for chat.
type ChatRequestData struct {
	Messages []Message `instill:"messages"`
}

// ChatParameter is the request parameter for chat.
type ChatParameter struct {
	MaxTokens   int     `instill:"max-tokens"`
	Seed        int     `instill:"seed"`
	N           int     `instill:"n"`
	Temperature float32 `instill:"temperature"`
	TopP        int     `instill:"top-p"`
}

// TextGenerationChatOutput is the standardized output for text generation chat tasks.
type TextGenerationChatOutput struct {
	Text string `instill:"text"`
}

// TEXT TO IMAGE TASK STRUCTS

// TextToImageInput is the standardized input for text-to-image tasks.
type TextToImageInput struct {
	ModelName      string  `instill:"model-name"`
	Prompt         string  `instill:"prompt"`
	NegativePrompt *string `instill:"negative-prompt"`
	AspectRatio    *string `instill:"aspect-ratio"`
	Samples        *int    `instill:"samples"`
	Seed           *int    `instill:"seed"`
}

// TextToImageRequestData is the request data for text-to-image.
type TextToImageRequestData struct {
	Prompt string `instill:"prompt"`
}

// TextToImageRequestParameter is the request parameter for text-to-image.
type TextToImageRequestParameter struct {
	AspectRatio    string `instill:"aspect-ratio"`
	NegativePrompt string `instill:"negative-prompt"`
	N              int    `instill:"n"`
	Seed           int    `instill:"seed"`
}

// TextToImageOutput is the standardized output for text-to-image tasks.
type TextToImageOutput struct {
	Images []format.Image `instill:"images"`
}

// VISION TASK STRUCTS

// VisionInput is the standardized input for vision tasks.
type VisionInput struct {
	ModelName   string `instill:"model-name"`
	ImageBase64 string `instill:"image-base64"`
}

// VisionRequestData is the request data for vision tasks.
type VisionRequestData struct {
	ImageBase64 string `instill:"image-base64"`
	Type        string `instill:"type"`
}
