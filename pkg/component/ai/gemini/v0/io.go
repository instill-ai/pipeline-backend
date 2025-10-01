package gemini

import (
	"time"

	"github.com/instill-ai/pipeline-backend/pkg/data/format"
	"google.golang.org/genai"
)

// TaskChatInput is the input for the TASK_CHAT task.
type TaskChatInput struct {
	Model             string                  `instill:"model"`
	Stream            *bool                   `instill:"stream"`
	Prompt            *string                 `instill:"prompt"`
	Images            []format.Image          `instill:"images"`
	Audio             []format.Audio          `instill:"audio"`
	Videos            []format.Video          `instill:"videos"`
	Documents         []format.Document       `instill:"documents"`
	SystemMessage     *string                 `instill:"system-message"`
	ChatHistory       []*genai.Content        `instill:"chat-history"`
	MaxOutputTokens   *int32                  `instill:"max-output-tokens"`
	Temperature       *float32                `instill:"temperature"`
	TopP              *float32                `instill:"top-p"`
	TopK              *int32                  `instill:"top-k"`
	Seed              *int32                  `instill:"seed"`
	Contents          []*genai.Content        `instill:"contents"`
	Tools             []*genai.Tool           `instill:"tools"`
	ToolConfig        *genai.ToolConfig       `instill:"tool-config"`
	SafetySettings    []*genai.SafetySetting  `instill:"safety-settings"`
	SystemInstruction *genai.Content          `instill:"system-instruction"`
	GenerationConfig  *genai.GenerationConfig `instill:"generation-config"`
	CachedContent     *string                 `instill:"cached-content"`
}

// GetPrompt implements MultimediaInput interface
func (t TaskChatInput) GetPrompt() *string { return t.Prompt }

// GetImages implements MultimediaInput interface
func (t TaskChatInput) GetImages() []format.Image { return t.Images }

// GetAudio implements MultimediaInput interface
func (t TaskChatInput) GetAudio() []format.Audio { return t.Audio }

// GetVideos implements MultimediaInput interface
func (t TaskChatInput) GetVideos() []format.Video { return t.Videos }

// GetDocuments implements MultimediaInput interface
func (t TaskChatInput) GetDocuments() []format.Document { return t.Documents }

// GetContents implements MultimediaInput interface
func (t TaskChatInput) GetContents() []*genai.Content { return t.Contents }

// GetSystemMessage implements SystemMessageInput interface
func (t TaskChatInput) GetSystemMessage() *string { return t.SystemMessage }

// GetSystemInstruction implements SystemMessageInput interface
func (t TaskChatInput) GetSystemInstruction() *genai.Content { return t.SystemInstruction }

// TaskChatOutput is the output for the TASK_CHAT task.
type TaskChatOutput struct {
	// Flattened chat output properties
	Texts  []string       `instill:"texts"`
	Images []format.Image `instill:"images"`
	Usage  map[string]any `instill:"usage"`

	// Use genai types directly with instill tags
	Candidates     []*genai.Candidate                           `instill:"candidates"`
	PromptFeedback *genai.GenerateContentResponsePromptFeedback `instill:"prompt-feedback"`
	ModelVersion   *string                                      `instill:"model-version"`
	ResponseID     *string                                      `instill:"response-id"`
}

// TaskCacheInput is the input for the TASK_CACHE task.
type TaskCacheInput struct {
	Operation   string  `instill:"operation"`
	Model       string  `instill:"model"`
	CacheName   *string `instill:"cache-name"`
	DisplayName *string `instill:"display-name"`

	// Multimedia inputs (flattened for ease of use)
	Prompt        *string           `instill:"prompt"`
	Images        []format.Image    `instill:"images"`
	Audio         []format.Audio    `instill:"audio"`
	Videos        []format.Video    `instill:"videos"`
	Documents     []format.Document `instill:"documents"`
	SystemMessage *string           `instill:"system-message"`

	// Advanced inputs
	SystemInstruction *genai.Content    `instill:"system-instruction"`
	Contents          []*genai.Content  `instill:"contents"`
	Tools             []*genai.Tool     `instill:"tools"`
	ToolConfig        *genai.ToolConfig `instill:"tool-config"`
	TTL               *time.Duration    `instill:"ttl,pattern=^[0-9]+(\\.([0-9]{1,9}))?s$"`
	ExpireTime        *time.Time        `instill:"expire-time,pattern=^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\\.[0-9]{1,9})?(Z|[+-][0-9]{2}:[0-9]{2})$"`
	PageSize          *int32            `instill:"page-size"`
	PageToken         *string           `instill:"page-token"`
}

// GetPrompt implements MultimediaInput interface
func (t TaskCacheInput) GetPrompt() *string { return t.Prompt }

// GetImages implements MultimediaInput interface
func (t TaskCacheInput) GetImages() []format.Image { return t.Images }

// GetAudio implements MultimediaInput interface
func (t TaskCacheInput) GetAudio() []format.Audio { return t.Audio }

// GetVideos implements MultimediaInput interface
func (t TaskCacheInput) GetVideos() []format.Video { return t.Videos }

// GetDocuments implements MultimediaInput interface
func (t TaskCacheInput) GetDocuments() []format.Document { return t.Documents }

// GetContents implements MultimediaInput interface
func (t TaskCacheInput) GetContents() []*genai.Content { return t.Contents }

// GetSystemMessage implements SystemMessageInput interface
func (t TaskCacheInput) GetSystemMessage() *string { return t.SystemMessage }

// GetSystemInstruction implements SystemMessageInput interface
func (t TaskCacheInput) GetSystemInstruction() *genai.Content { return t.SystemInstruction }

// TaskCacheOutput is the output for the TASK_CACHE task.
type TaskCacheOutput struct {
	Operation      string                 `instill:"operation"`
	CachedContent  *genai.CachedContent   `instill:"cached-content"`
	CachedContents []*genai.CachedContent `instill:"cached-contents"`
	NextPageToken  *string                `instill:"next-page-token"`
}

// TaskTextEmbeddingsInput is the input for the TASK_TEXT_EMBEDDINGS task.
type TaskTextEmbeddingsInput struct {
	Model                string `instill:"model"`
	Text                 string `instill:"text"`
	TaskType             string `instill:"task-type"`
	Title                string `instill:"title"`
	OutputDimensionality *int32 `instill:"output-dimensionality"`
}

// TaskTextEmbeddingsOutput is the output for the TASK_TEXT_EMBEDDINGS task.
type TaskTextEmbeddingsOutput struct {
	Embedding *genai.ContentEmbedding `instill:"embedding"`
}
