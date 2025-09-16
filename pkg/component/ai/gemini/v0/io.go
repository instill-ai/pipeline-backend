package gemini

import (
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
)

// INPUT

// TaskChatInput is the input for the TASK_CHAT task.
type TaskChatInput struct {
	// Flattened chat input properties
	Stream          *bool             `instill:"stream"`
	Prompt          *string           `instill:"prompt"`
	Images          []format.Image    `instill:"images"`
	Documents       []format.Document `instill:"documents"`
	SystemMessage   *string           `instill:"system-message"`
	ChatHistory     []content         `instill:"chat-history"`
	MaxOutputTokens *int32            `instill:"max-output-tokens"`
	Temperature     *float32          `instill:"temperature"`
	TopP            *float32          `instill:"top-p"`
	TopK            *int32            `instill:"top-k"`
	Seed            *int32            `instill:"seed"`

	// Other properties
	Model             string            `instill:"model"`
	Contents          []content         `instill:"contents"`
	Tools             []tool            `instill:"tools"`
	ToolConfig        *toolConfig       `instill:"tool-config"`
	SafetySettings    []safetySetting   `instill:"safety-settings"`
	SystemInstruction *content          `instill:"system-instruction"`
	GenerationConfig  *generationConfig `instill:"generation-config"`
	CachedContent     *string           `instill:"cached-content"`
}

type content struct {
	Role  *string `instill:"role"`
	Parts []part  `instill:"parts"`
}

type part struct {
	// Common optional annotations
	Thought          *bool          `instill:"thought"`
	ThoughtSignature *string        `instill:"thought-signature"`
	VideoMetadata    *videoMetadata `instill:"video-metadata"`

	// Union payloads (only one expected)
	Text                *string              `instill:"text"`
	InlineData          *blob                `instill:"inline-data"`
	FileData            *fileData            `instill:"file-data"`
	FunctionCall        *functionCall        `instill:"function-call"`
	FunctionResponse    *functionResponse    `instill:"function-response"`
	ExecutableCode      *executableCode      `instill:"executable-code"`
	CodeExecutionResult *codeExecutionResult `instill:"code-execution-result"`
}

type blob struct {
	MIMEType string `instill:"mimeType"`
	Data     string `instill:"data"`
}

type fileData struct {
	MIMEType string `instill:"mimeType"`
	URI      string `instill:"uri"`
}

type functionCall struct {
	ID   *string                 `instill:"id"`
	Name string                  `instill:"name"`
	Args map[string]format.Value `instill:"args"`
}

type functionResponse struct {
	ID           *string                 `instill:"id"`
	Name         string                  `instill:"name"`
	Response     map[string]format.Value `instill:"response"`
	WillContinue *bool                   `instill:"will-continue"`
	Scheduling   *string                 `instill:"scheduling"`
}

type videoMetadata struct {
	StartOffset *string  `instill:"start-offset"`
	EndOffset   *string  `instill:"end-offset"`
	FPS         *float32 `instill:"fps"`
}

type tool struct {
	FunctionDeclarations  []functionDeclaration  `instill:"function-declarations"`
	GoogleSearchRetrieval *googleSearchRetrieval `instill:"google-search-retrieval"`
	CodeExecution         *codeExecution         `instill:"code-execution"`
	GoogleSearch          *googleSearch          `instill:"google-search"`
	URLContext            *urlContext            `instill:"url-context"`
}

type functionDeclaration struct {
	Name        string      `instill:"name"`
	Description *string     `instill:"description"`
	Parameters  *jsonSchema `instill:"parameters"`
}

type dynamicRetrievalConfig struct {
	Mode             *string  `instill:"mode"`
	DynamicThreshold *float64 `instill:"dynamic-threshold"`
}

type googleSearchRetrieval struct {
	DynamicRetrievalConfig *dynamicRetrievalConfig `instill:"dynamic-retrieval-config"`
}

type interval struct {
	StartTime *string `instill:"start-time"`
	EndTime   *string `instill:"end-time"`
}

type googleSearch struct {
	TimeRangeFilter *interval `instill:"time-range-filter"`
}

type urlContext struct{}

type codeExecution struct{}

type toolConfig struct {
	FunctionCallingConfig *functionCallingConfig `instill:"function-calling-config"`
}

type functionCallingConfig struct {
	Mode                 *string  `instill:"mode"`
	AllowedFunctionNames []string `instill:"allowed-function-names"`
}

type safetySetting struct {
	Category  string `instill:"category"`
	Threshold string `instill:"threshold"`
}

type generationConfig struct {
	MaxOutputTokens            *int32          `instill:"max-output-tokens"`
	Temperature                *float32        `instill:"temperature"`
	TopP                       *float32        `instill:"top-p"`
	TopK                       *float32        `instill:"top-k"`
	StopSequences              []string        `instill:"stop-sequences"`
	CandidateCount             *int32          `instill:"candidate-count"`
	ResponseMimeType           *string         `instill:"response-mime-type,default=text/plain"`
	ResponseSchema             *jsonSchema     `instill:"response-schema"`
	MediaResolution            *string         `instill:"media-resolution"`
	ResponseModalities         []string        `instill:"response-modalities"`
	Seed                       *int            `instill:"seed"`
	PresencePenalty            *float32        `instill:"presence-penalty"`
	FrequencyPenalty           *float32        `instill:"frequency-penalty"`
	ResponseLogprobs           *bool           `instill:"response-logprobs"`
	Logprobs                   *int            `instill:"logprobs"`
	EnableEnhancedCivicAnswers *bool           `instill:"enable-enhanced-civic-answers"`
	SpeechConfig               *speechConfig   `instill:"speech-config"`
	ThinkingConfig             *thinkingConfig `instill:"thinking-config"`
}

// jsonSchema mirrors $defs/schema in tasks.yaml
type jsonSchema struct {
	Type             string                  `instill:"type"`
	Format           string                  `instill:"format"`
	Title            string                  `instill:"title"`
	Description      string                  `instill:"description"`
	Nullable         *bool                   `instill:"nullable"`
	Enum             []string                `instill:"enum"`
	MaxItems         *int32                  `instill:"max-items"`
	MinItems         *int32                  `instill:"min-items"`
	Properties       map[string]jsonSchema   `instill:"properties"`
	Required         []string                `instill:"required"`
	MinProperties    *int32                  `instill:"min-properties"`
	MaxProperties    *int32                  `instill:"max-properties"`
	MinLength        *int32                  `instill:"min-length"`
	MaxLength        *int32                  `instill:"max-length"`
	Pattern          string                  `instill:"pattern"`
	AnyOf            []jsonSchema            `instill:"anyOf"`
	PropertyOrdering []string                `instill:"property-ordering"`
	Default          map[string]format.Value `instill:"default"`
	Items            *jsonSchema             `instill:"items"`
	Minimum          *float64                `instill:"minimum"`
	Maximum          *float64                `instill:"maximum"`
}

type speechConfig struct {
	VoiceConfig             *voiceConfig             `instill:"voice-config"`
	MultiSpeakerVoiceConfig *multiSpeakerVoiceConfig `instill:"multi-speaker-voice-config"`
	LanguageCode            *string                  `instill:"language-code"`
}

type thinkingConfig struct {
	IncludeThoughts *bool  `instill:"include-thoughts"`
	ThinkingBudget  *int32 `instill:"thinking-budget"`
}

type voiceConfig struct {
	PrebuiltVoiceConfig *prebuiltVoiceConfig `instill:"prebuilt-voice-config"`
}

type prebuiltVoiceConfig struct {
	VoiceName string `instill:"voice-name"`
}

type speakerVoiceConfig struct {
	Speaker     string      `instill:"speaker"`
	VoiceConfig voiceConfig `instill:"voice-config"`
}

type multiSpeakerVoiceConfig struct {
	SpeakerVoiceConfigs []speakerVoiceConfig `instill:"speaker-voice-configs"`
}

// Grounding fine-grained types

type attributionSourceID struct {
	GroundingPassage       *groundingPassageID     `instill:"grounding-passage"`
	SemanticRetrieverChunk *semanticRetrieverChunk `instill:"semantic-retriever-chunk"`
}

type groundingPassageID struct {
	PassageID string `instill:"passage-id"`
	PartIndex int32  `instill:"part-index"`
}

type semanticRetrieverChunk struct {
	Source string `instill:"source"`
	Chunk  string `instill:"chunk"`
}

type segment struct {
	PartIndex  int32  `instill:"part-index"`
	StartIndex int32  `instill:"start-index"`
	EndIndex   int32  `instill:"end-index"`
	Text       string `instill:"text"`
}

type webChunk struct {
	URI   string `instill:"uri"`
	Title string `instill:"title"`
}

type groundingChunk struct {
	Web *webChunk `instill:"web"`
}

type groundingSupport struct {
	GroundingChunkIndices []int32   `instill:"grounding-chunk-indices"`
	ConfidenceScores      []float32 `instill:"confidence-scores"`
	Segment               *segment  `instill:"segment"`
}

type retrievalMetadata struct {
	GoogleSearchDynamicRetrievalScore float32 `instill:"google-search-dynamic-retrieval-score"`
}

type searchEntryPoint struct {
	RenderedContent string `instill:"rendered-content"`
	SDKBlob         string `instill:"sdk-blob"`
}

type groundingMetadata struct {
	GroundingChunks   []groundingChunk   `instill:"grounding-chunks"`
	GroundingSupports []groundingSupport `instill:"grounding-supports"`
	WebSearchQueries  []string           `instill:"web-search-queries"`
	SearchEntryPoint  *searchEntryPoint  `instill:"search-entry-point"`
	RetrievalMetadata *retrievalMetadata `instill:"retrieval-metadata"`
}

type groundingAttribution struct {
	SourceID *attributionSourceID `instill:"source-id"`
	Content  *content             `instill:"content"`
}

// Executable code and result types
type executableCode struct {
	Language string `instill:"language"`
	Code     string `instill:"code"`
}

type codeExecutionResult struct {
	Outcome string  `instill:"outcome"`
	Output  *string `instill:"output"`
}

// URL context types

type urlMetadata struct {
	RetrievedURL       string `instill:"retrieved-url"`
	URLRetrievalStatus string `instill:"url-retrieval-status"`
}

type urlContextMetadata struct {
	URLMetadata []urlMetadata `instill:"url-metadata"`
}

// OUTPUT

// TaskChatOutput is the output for the TASK_CHAT task.
type TaskChatOutput struct {
	// Flattened chat output properties
	Texts []string       `instill:"texts"`
	Usage map[string]any `instill:"usage"`

	// Other properties
	Candidates     []candidate    `instill:"candidates"`
	UsageMetadata  usageMetadata  `instill:"usage-metadata"`
	PromptFeedback promptFeedback `instill:"prompt-feedback"`
	ModelVersion   *string        `instill:"model-version"`
	ResponseID     *string        `instill:"response-id"`
}

type candidate struct {
	Index                 int32                  `instill:"index"`
	Content               *content               `instill:"content"`
	SafetyRatings         []safetyRating         `instill:"safety-ratings"`
	FinishReason          string                 `instill:"finish-reason"`
	CitationMetadata      *citationMetadata      `instill:"citation-metadata"`
	TokenCount            int32                  `instill:"token-count"`
	GroundingAttributions []groundingAttribution `instill:"grounding-attributions"`
	LogprobsResult        *logprobsResult        `instill:"logprobs-result"`
	AvgLogprobs           float64                `instill:"avg-logprobs"`
	URLContextMetadata    *urlContextMetadata    `instill:"url-context-metadata"`
	GroundingMetadata     *groundingMetadata     `instill:"grounding-metadata"`
}

type safetyRating struct {
	Category    string `instill:"category"`
	Probability string `instill:"probability"`
	Blocked     *bool  `instill:"blocked"`
}

type citationMetadata struct {
	Citations []citationSource `instill:"citations"`
}

type citationSource struct {
	URI        string `instill:"uri"`
	Title      string `instill:"title"`
	StartIndex int32  `instill:"start-index"`
	EndIndex   int32  `instill:"end-index"`
}

type logprobsResult struct {
	TopCandidates []logprobsTopCandidate `instill:"top-candidates"`
}

type logprobsTopCandidate struct {
	Token   string  `instill:"token"`
	Logprob float32 `instill:"logprob"`
}

type promptFeedback struct {
	BlockReason   *string        `instill:"block-reason"`
	SafetyRatings []safetyRating `instill:"safety-ratings"`
}

type usageMetadata struct {
	PromptTokenCount           int32                `instill:"prompt-token-count"`
	CachedContentTokenCount    int32                `instill:"cached-content-token-count"`
	CandidatesTokenCount       int32                `instill:"candidates-token-count"`
	ToolUsePromptTokenCount    int32                `instill:"tool-use-prompt-token-count"`
	ThoughtsTokenCount         int32                `instill:"thoughts-token-count"`
	TotalTokenCount            int32                `instill:"total-token-count"`
	PromptTokensDetails        []modalityTokenCount `instill:"prompt-tokens-details"`
	CacheTokensDetails         []modalityTokenCount `instill:"cache-tokens-details"`
	CandidatesTokensDetails    []modalityTokenCount `instill:"candidates-tokens-details"`
	ToolUsePromptTokensDetails []modalityTokenCount `instill:"tool-use-prompt-tokens-details"`
}

type modalityTokenCount struct {
	Modality   string `instill:"modality"`
	TokenCount int    `instill:"token-count"`
}
