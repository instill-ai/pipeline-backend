package gemini

import (
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
	"google.golang.org/genai"
)

// TaskChatInput is the input for the TASK_CHAT task.
type TaskChatInput struct {
	// Flattened chat input properties
	Stream          *bool             `instill:"stream"`
	Prompt          *string           `instill:"prompt"`
	Images          []format.Image    `instill:"images"`
	Audio           []format.Audio    `instill:"audio"`
	Videos          []format.Video    `instill:"videos"`
	Documents       []format.Document `instill:"documents"`
	SystemMessage   *string           `instill:"system-message"`
	ChatHistory     []*genai.Content  `instill:"chat-history"`
	MaxOutputTokens *int32            `instill:"max-output-tokens"`
	Temperature     *float32          `instill:"temperature"`
	TopP            *float32          `instill:"top-p"`
	TopK            *int32            `instill:"top-k"`
	Seed            *int32            `instill:"seed"`

	// Other properties
	Model             string                  `instill:"model"`
	Contents          []*genai.Content        `instill:"contents"`
	Tools             []*genai.Tool           `instill:"tools"`
	ToolConfig        *genai.ToolConfig       `instill:"tool-config"`
	SafetySettings    []*genai.SafetySetting  `instill:"safety-settings"`
	SystemInstruction *genai.Content          `instill:"system-instruction"`
	GenerationConfig  *genai.GenerationConfig `instill:"generation-config"`
	CachedContent     *string                 `instill:"cached-content"`
}

// TaskChatOutput is the output for the TASK_CHAT task.
type TaskChatOutput struct {
	// Flattened chat output properties
	Texts []string       `instill:"texts"`
	Usage map[string]any `instill:"usage"`

	// Use genai types directly with instill tags
	Candidates     []*genai.Candidate                           `instill:"candidates"`
	UsageMetadata  *genai.GenerateContentResponseUsageMetadata  `instill:"usage-metadata"`
	PromptFeedback *genai.GenerateContentResponsePromptFeedback `instill:"prompt-feedback"`
	ModelVersion   *string                                      `instill:"model-version"`
	ResponseID     *string                                      `instill:"response-id"`
}
