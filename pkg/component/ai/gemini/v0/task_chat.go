package gemini

import (
	"context"
	"encoding/base64"
	"mime"
	"path"
	"strings"

	"google.golang.org/genai"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
	"github.com/instill-ai/pipeline-backend/pkg/external"
)

func (e *execution) chat(ctx context.Context, job *base.Job) error {
	// Read input
	in := TaskChatInput{}
	if err := job.Input.ReadData(ctx, &in); err != nil {
		job.Error.Error(ctx, err)
		return nil
	}

	// Create Gemini client
	client, err := e.createGeminiClient(ctx)
	if err != nil {
		job.Error.Error(ctx, err)
		return nil
	}

	// Prepare request components
	systemMessage := extractSystemMessage(in)
	cfg := buildGenerateContentConfig(in, systemMessage)
	contents, err := e.buildRequestContents(in)
	if err != nil {
		job.Error.Error(ctx, err)
		return nil
	}
	if len(contents) == 0 {
		return nil
	}

	// Execute request (streaming or non-streaming)
	streamEnabled := in.Stream != nil && *in.Stream
	if streamEnabled {
		return e.handleStreamingRequest(ctx, job, client, in.Model, contents, cfg)
	}
	return e.handleNonStreamingRequest(ctx, job, client, in.Model, contents, cfg)
}

// createGeminiClient creates a new Gemini API client
func (e *execution) createGeminiClient(ctx context.Context) (*genai.Client, error) {
	apiKey := e.Setup.GetFields()[cfgAPIKey].GetStringValue()
	return genai.NewClient(ctx, &genai.ClientConfig{APIKey: apiKey, Backend: genai.BackendGeminiAPI})
}

// extractSystemMessage extracts system message from input, prioritizing system-message over system-instruction
func extractSystemMessage(in TaskChatInput) string {
	if in.SystemMessage != nil && *in.SystemMessage != "" {
		return *in.SystemMessage
	}
	if in.SystemInstruction != nil && len(in.SystemInstruction.Parts) > 0 {
		for _, p := range in.SystemInstruction.Parts {
			if p.Text != "" {
				return p.Text
			}
		}
	}
	return ""
}

// buildRequestContents builds the complete request contents from input (history + current message)
func (e *execution) buildRequestContents(in TaskChatInput) ([]*genai.Content, error) {
	// Build user parts (prompt/contents + images + documents)
	inParts, err := buildReqParts(in)
	if err != nil {
		return nil, err
	}
	if len(inParts) == 0 {
		return nil, nil
	}

	// Merge chat history and current message into contents
	contents := make([]*genai.Content, 0)
	if len(in.ChatHistory) > 0 {
		for _, h := range in.ChatHistory {
			role := genai.RoleUser
			if h.Role == "model" {
				role = genai.RoleModel
			}
			if len(h.Parts) == 0 {
				continue
			}
			contents = append(contents, &genai.Content{Role: role, Parts: h.Parts})
		}
	}

	// Append current user message as last turn
	partsPtrs := make([]*genai.Part, 0, len(inParts))
	for i := range inParts {
		p := inParts[i]
		partsPtrs = append(partsPtrs, &p)
	}
	contents = append(contents, &genai.Content{Role: genai.RoleUser, Parts: partsPtrs})

	return contents, nil
}

// handleStreamingRequest processes streaming requests
func (e *execution) handleStreamingRequest(ctx context.Context, job *base.Job, client *genai.Client, model string, contents []*genai.Content, cfg *genai.GenerateContentConfig) error {
	texts := make([]string, 0)
	var finalResp *genai.GenerateContentResponse

	for r, err := range client.Models.GenerateContentStream(ctx, model, contents, cfg) {
		if err != nil {
			job.Error.Error(ctx, err)
			return nil
		}

		// Accumulate text from candidates
		e.accumulateTexts(r, &texts)

		// Merge response chunks
		e.mergeResponseChunk(r, &finalResp)

		// Stream incremental output
		streamOutput := e.buildStreamOutput(texts, finalResp)
		if err := job.Output.WriteData(ctx, streamOutput); err != nil {
			job.Error.Error(ctx, err)
			return nil
		}
	}

	// Send final output
	finalOut := renderFinal(finalResp, nil)
	if err := job.Output.WriteData(ctx, finalOut); err != nil {
		job.Error.Error(ctx, err)
		return nil
	}
	return nil
}

// handleNonStreamingRequest processes non-streaming requests
func (e *execution) handleNonStreamingRequest(ctx context.Context, job *base.Job, client *genai.Client, model string, contents []*genai.Content, cfg *genai.GenerateContentConfig) error {
	resp, err := client.Models.GenerateContent(ctx, model, contents, cfg)
	if err != nil {
		job.Error.Error(ctx, err)
		return nil
	}

	finalOut := renderFinal(resp, nil)
	if err := job.Output.WriteData(ctx, finalOut); err != nil {
		job.Error.Error(ctx, err)
		return nil
	}
	return nil
}

// accumulateTexts accumulates text content from streaming response chunks
func (e *execution) accumulateTexts(r *genai.GenerateContentResponse, texts *[]string) {
	if r != nil && len(r.Candidates) > 0 {
		// Ensure texts slice has enough capacity
		for len(*texts) < len(r.Candidates) {
			*texts = append(*texts, "")
		}
		// Accumulate text from each candidate
		for i, c := range r.Candidates {
			if c.Content != nil {
				for _, p := range c.Content.Parts {
					if p != nil && p.Text != "" {
						(*texts)[i] += p.Text
					}
				}
			}
		}
	}
}

// mergeResponseChunk merges streaming response chunks into a final response
func (e *execution) mergeResponseChunk(r *genai.GenerateContentResponse, finalResp **genai.GenerateContentResponse) {
	if r == nil {
		return
	}

	if *finalResp == nil {
		// Initialize with first chunk
		*finalResp = &genai.GenerateContentResponse{
			ModelVersion:   r.ModelVersion,
			ResponseID:     r.ResponseID,
			UsageMetadata:  r.UsageMetadata,
			PromptFeedback: r.PromptFeedback,
			Candidates:     make([]*genai.Candidate, len(r.Candidates)),
		}
		// Deep copy candidates
		for i, c := range r.Candidates {
			if c != nil {
				(*finalResp).Candidates[i] = &genai.Candidate{
					Index:              c.Index,
					SafetyRatings:      c.SafetyRatings,
					FinishReason:       c.FinishReason,
					CitationMetadata:   c.CitationMetadata,
					TokenCount:         c.TokenCount,
					LogprobsResult:     c.LogprobsResult,
					AvgLogprobs:        c.AvgLogprobs,
					URLContextMetadata: c.URLContextMetadata,
					GroundingMetadata:  c.GroundingMetadata,
					Content:            &genai.Content{Role: c.Content.Role, Parts: []*genai.Part{}},
				}
				// Copy parts
				if c.Content != nil {
					for _, p := range c.Content.Parts {
						if p != nil {
							(*finalResp).Candidates[i].Content.Parts = append((*finalResp).Candidates[i].Content.Parts, p)
						}
					}
				}
			}
		}
	} else {
		// Merge subsequent chunks - append parts to existing candidates
		for i, c := range r.Candidates {
			if c != nil && i < len((*finalResp).Candidates) && (*finalResp).Candidates[i] != nil {
				// Update metadata from latest chunk
				(*finalResp).Candidates[i].FinishReason = c.FinishReason
				(*finalResp).Candidates[i].TokenCount = c.TokenCount
				(*finalResp).Candidates[i].AvgLogprobs = c.AvgLogprobs
				if c.SafetyRatings != nil {
					(*finalResp).Candidates[i].SafetyRatings = c.SafetyRatings
				}
				if c.CitationMetadata != nil {
					(*finalResp).Candidates[i].CitationMetadata = c.CitationMetadata
				}
				if c.LogprobsResult != nil {
					(*finalResp).Candidates[i].LogprobsResult = c.LogprobsResult
				}
				if c.URLContextMetadata != nil {
					(*finalResp).Candidates[i].URLContextMetadata = c.URLContextMetadata
				}
				if c.GroundingMetadata != nil {
					(*finalResp).Candidates[i].GroundingMetadata = c.GroundingMetadata
				}

				// Append new parts
				if c.Content != nil {
					for _, p := range c.Content.Parts {
						if p != nil {
							(*finalResp).Candidates[i].Content.Parts = append((*finalResp).Candidates[i].Content.Parts, p)
						}
					}
				}
			}
		}
		// Update response-level metadata from latest chunk
		if r.UsageMetadata != nil {
			(*finalResp).UsageMetadata = r.UsageMetadata
		}
		if r.PromptFeedback != nil {
			(*finalResp).PromptFeedback = r.PromptFeedback
		}
	}
}

// buildStreamOutput creates streaming output with all available fields
func (e *execution) buildStreamOutput(texts []string, finalResp *genai.GenerateContentResponse) TaskChatOutput {
	streamOutput := TaskChatOutput{
		Texts:          texts,
		Usage:          map[string]any{},
		Candidates:     []*genai.Candidate{},
		UsageMetadata:  nil,
		PromptFeedback: nil,
		ModelVersion:   nil,
		ResponseID:     nil,
	}

	if finalResp != nil {
		streamOutput.Candidates = finalResp.Candidates
		streamOutput.UsageMetadata = finalResp.UsageMetadata
		streamOutput.PromptFeedback = finalResp.PromptFeedback
		if finalResp.ModelVersion != "" {
			mv := finalResp.ModelVersion
			streamOutput.ModelVersion = &mv
		}
		if finalResp.ResponseID != "" {
			ri := finalResp.ResponseID
			streamOutput.ResponseID = &ri
		}

		// Build usage map from UsageMetadata if available
		if finalResp.UsageMetadata != nil {
			streamOutput.Usage = buildUsageMap(finalResp.UsageMetadata)
		}
	}

	return streamOutput
}

// buildUsageMap creates a usage map from UsageMetadata with kebab-case keys
func buildUsageMap(metadata *genai.GenerateContentResponseUsageMetadata) map[string]any {
	usage := make(map[string]any)
	usage["prompt-token-count"] = metadata.PromptTokenCount
	usage["cached-content-token-count"] = metadata.CachedContentTokenCount
	usage["candidates-token-count"] = metadata.CandidatesTokenCount
	usage["total-token-count"] = metadata.TotalTokenCount
	usage["tool-use-prompt-token-count"] = metadata.ToolUsePromptTokenCount
	usage["thoughts-token-count"] = metadata.ThoughtsTokenCount

	if len(metadata.PromptTokensDetails) > 0 {
		arr := make([]map[string]any, 0, len(metadata.PromptTokensDetails))
		for _, d := range metadata.PromptTokensDetails {
			if d == nil {
				continue
			}
			arr = append(arr, map[string]any{"modality": string(d.Modality), "token-count": int(d.TokenCount)})
		}
		usage["prompt-tokens-details"] = arr
	}
	if len(metadata.CacheTokensDetails) > 0 {
		arr := make([]map[string]any, 0, len(metadata.CacheTokensDetails))
		for _, d := range metadata.CacheTokensDetails {
			if d == nil {
				continue
			}
			arr = append(arr, map[string]any{"modality": string(d.Modality), "token-count": int(d.TokenCount)})
		}
		usage["cache-tokens-details"] = arr
	}
	if len(metadata.CandidatesTokensDetails) > 0 {
		arr := make([]map[string]any, 0, len(metadata.CandidatesTokensDetails))
		for _, d := range metadata.CandidatesTokensDetails {
			if d == nil {
				continue
			}
			arr = append(arr, map[string]any{"modality": string(d.Modality), "token-count": int(d.TokenCount)})
		}
		usage["candidates-tokens-details"] = arr
	}
	if len(metadata.ToolUsePromptTokensDetails) > 0 {
		arr := make([]map[string]any, 0, len(metadata.ToolUsePromptTokensDetails))
		for _, d := range metadata.ToolUsePromptTokensDetails {
			if d == nil {
				continue
			}
			arr = append(arr, map[string]any{"modality": string(d.Modality), "token-count": int(d.TokenCount)})
		}
		usage["tool-use-prompt-tokens-details"] = arr
	}

	return usage
}

// renderFinal builds a complete output from a final genai response.
func renderFinal(resp *genai.GenerateContentResponse, texts []string) TaskChatOutput {
	out := TaskChatOutput{
		Texts:          []string{},
		Usage:          map[string]any{},
		Candidates:     []*genai.Candidate{},
		UsageMetadata:  nil,
		PromptFeedback: nil,
	}
	if resp == nil {
		return out
	}
	out.Candidates = resp.Candidates
	out.UsageMetadata = resp.UsageMetadata
	out.PromptFeedback = resp.PromptFeedback
	if resp.ModelVersion != "" {
		mv := resp.ModelVersion
		out.ModelVersion = &mv
	}
	if resp.ResponseID != "" {
		ri := resp.ResponseID
		out.ResponseID = &ri
	}
	// Texts and usage map
	if len(texts) > 0 {
		out.Texts = texts
	} else if len(resp.Candidates) > 0 {
		acc := make([]string, 0, len(resp.Candidates))
		for _, c := range resp.Candidates {
			candidateText := ""
			if c.Content != nil {
				for _, p := range c.Content.Parts {
					if p != nil && p.Text != "" {
						candidateText += p.Text
					}
				}
			}
			acc = append(acc, candidateText)
		}
		out.Texts = acc
	}
	if resp.UsageMetadata != nil {
		out.Usage = buildUsageMap(resp.UsageMetadata)
	}
	return out
}

// Helpers for Images/Audio/Videos/Documents strings to genai.Part
func newURIOrDataPart(s string, defaultMIME string) *genai.Part {
	if s == "" {
		return nil
	}

	// Handle data URIs - these need to be decoded and embedded as inline data
	if strings.HasPrefix(s, "data:") {
		fetcher := external.NewBinaryFetcher()
		b, contentType, _, err := fetcher.FetchFromURL(context.Background(), s)
		if err != nil {
			return nil
		}
		mimeType := contentType
		if mimeType == "" {
			mimeType = defaultMIME
		}
		return &genai.Part{InlineData: &genai.Blob{MIMEType: mimeType, Data: b}}
	}

	// Handle remote URIs (http/https) - let Gemini API fetch these directly
	if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
		// Use genai.NewPartFromURI to create a fileData part that references the remote URI
		// This leverages Gemini API's native ability to fetch remote files
		if u := genai.NewPartFromURI(s, defaultMIME); u != nil {
			return u
		}
		return nil
	}

	// Try raw base64 (no prefix) - decode and embed as inline data
	if decoded, err := base64.StdEncoding.DecodeString(s); err == nil {
		return &genai.Part{InlineData: &genai.Blob{MIMEType: defaultMIME, Data: decoded}}
	}

	// Handle other URI schemes (file://, gs://, etc.) - let Gemini API handle these
	if strings.Contains(s, "://") {
		if u := genai.NewPartFromURI(s, defaultMIME); u != nil {
			return u
		}
	}

	return nil
}

// detectMIMEFromPath determines MIME using the standard mime package; falls back to default when unknown.
func detectMIMEFromPath(u string, defaultMIME string) string {
	ext := strings.ToLower(path.Ext(u))
	if ext != "" {
		if t := mime.TypeByExtension(ext); t != "" {
			return t
		}
	}
	return defaultMIME
}

// processTextParts extracts text content from Contents and Prompt, returning them as genai.Part objects.
// Text parts are processed last according to best practices.
func processTextParts(in TaskChatInput) []genai.Part {
	var textParts []genai.Part

	// Extract text parts from Contents
	if len(in.Contents) > 0 {
		last := in.Contents[len(in.Contents)-1]
		for _, part := range last.Parts {
			if part.Text != "" {
				textParts = append(textParts, *part)
			}
		}
	}

	// Add prompt as text part
	if in.Prompt != nil && *in.Prompt != "" {
		textParts = append(textParts, genai.Part{Text: *in.Prompt})
	}

	return textParts
}

// processNonTextContentParts extracts non-text parts from Contents (images, files, etc.).
// These parts are processed first in the final ordering.
func processNonTextContentParts(in TaskChatInput) []genai.Part {
	var nonTextParts []genai.Part

	if len(in.Contents) > 0 {
		last := in.Contents[len(in.Contents)-1]
		for _, part := range last.Parts {
			if part.Text == "" {
				nonTextParts = append(nonTextParts, *part)
			}
		}
	}

	return nonTextParts
}

// processImageParts converts image inputs to genai.Part objects with appropriate MIME types.
func processImageParts(images []format.Image) ([]genai.Part, error) {
	var parts []genai.Part

	for _, img := range images {
		contentType := img.ContentType().String()

		// Validate image format
		if _, err := validateFormat(contentType, "image", imageFormats, "GIF, BMP, TIFF", "PNG, JPEG, WEBP", ":png\", \":jpeg\", \":webp"); err != nil {
			return nil, err
		}

		imgBase64, err := img.Base64()
		if err != nil {
			return nil, err
		}
		if p := newURIOrDataPart(imgBase64.String(), detectMIMEFromPath(imgBase64.String(), contentType)); p != nil {
			parts = append(parts, *p)
		}
	}

	return parts, nil
}

// processAudioParts converts audio inputs to genai.Part objects with appropriate MIME types.
func processAudioParts(audio []format.Audio) ([]genai.Part, error) {
	var parts []genai.Part

	for _, audioFile := range audio {
		contentType := audioFile.ContentType().String()

		// Validate audio format
		if _, err := validateFormat(contentType, "audio", audioFormats, "M4A, WMA", "WAV, MP3, AIFF, AAC, OGG, FLAC", ":wav\", \":mp3\", \":ogg"); err != nil {
			return nil, err
		}

		audioBase64, err := audioFile.Base64()
		if err != nil {
			return nil, err
		}
		if p := newURIOrDataPart(audioBase64.String(), contentType); p != nil {
			parts = append(parts, *p)
		}
	}

	return parts, nil
}

// processVideoParts converts video inputs to genai.Part objects with appropriate MIME types.
func processVideoParts(videos []format.Video) ([]genai.Part, error) {
	var parts []genai.Part

	for _, video := range videos {
		contentType := video.ContentType().String()

		// Validate video format
		if _, err := validateFormat(contentType, "video", videoFormats, "MKV", "MP4, MPEG, MOV, AVI, FLV, WEBM, WMV", ":mp4\", \":mov\", \":webm"); err != nil {
			return nil, err
		}

		videoBase64, err := video.Base64()
		if err != nil {
			return nil, err
		}
		if p := newURIOrDataPart(videoBase64.String(), contentType); p != nil {
			parts = append(parts, *p)
		}
	}

	return parts, nil
}

// processDocumentParts converts document inputs to genai.Part objects based on their type and capabilities.
// - PDFs: Full document vision support (charts, diagrams, formatting preserved)
// - Text-based: Extract as plain text (HTML tags, Markdown formatting, etc. lost)
// - Office documents: Recommend PDF conversion for visual understanding
func processDocumentParts(documents []format.Document) ([]genai.Part, error) {
	var parts []genai.Part

	for _, doc := range documents {
		contentType := doc.ContentType().String()

		// Validate document format and get processing mode
		mode, err := validateFormat(contentType, "document", documentFormats, "", "", "")
		if err != nil {
			return nil, err
		}

		switch mode {
		case "visual":
			// PDFs support full document vision capabilities
			// The model can interpret visual elements like charts, diagrams, and formatting
			docBase64, err := doc.Base64()
			if err != nil {
				return nil, err
			}
			if p := newURIOrDataPart(docBase64.String(), detectMIMEFromPath(docBase64.String(), "application/pdf")); p != nil {
				parts = append(parts, *p)
			}
		case "text":
			// Text-based documents (TXT, Markdown, HTML, XML, etc.)
			// Extract as plain text content
			textContent := doc.String()
			parts = append(parts, genai.Part{Text: textContent})
		}
	}

	return parts, nil
}

// buildReqParts constructs the user request parts from input, including prompt/contents, images, audio, videos, and documents.
// Following best practices: text content (from both Contents and Prompt) is placed after visual/multimedia content (images/audio/videos/documents).
func buildReqParts(in TaskChatInput) ([]genai.Part, error) {
	var parts []genai.Part

	// Process non-text parts from Contents first (images, files, etc.)
	nonTextContentParts := processNonTextContentParts(in)
	parts = append(parts, nonTextContentParts...)

	// Process multimedia content in optimal order: images → audio → videos → documents
	imageParts, err := processImageParts(in.Images)
	if err != nil {
		return nil, err
	}
	parts = append(parts, imageParts...)

	audioParts, err := processAudioParts(in.Audio)
	if err != nil {
		return nil, err
	}
	parts = append(parts, audioParts...)

	videoParts, err := processVideoParts(in.Videos)
	if err != nil {
		return nil, err
	}
	parts = append(parts, videoParts...)

	documentParts, err := processDocumentParts(in.Documents)
	if err != nil {
		return nil, err
	}
	parts = append(parts, documentParts...)

	// Process text content last (as per best practices)
	textParts := processTextParts(in)
	parts = append(parts, textParts...)

	return parts, nil
}

// buildGenerateContentConfig creates a genai.GenerateContentConfig from the input parameters
func buildGenerateContentConfig(in TaskChatInput, systemMessage string) *genai.GenerateContentConfig {
	// Check if any config is needed
	needsConfig := in.MaxOutputTokens != nil || in.Temperature != nil || in.TopP != nil || in.TopK != nil ||
		in.Seed != nil || len(in.Tools) > 0 || in.ToolConfig != nil || len(in.SafetySettings) > 0 ||
		systemMessage != "" || in.SystemInstruction != nil || in.GenerationConfig != nil || in.CachedContent != nil

	if !needsConfig {
		return nil
	}

	cfg := &genai.GenerateContentConfig{}

	// Handle flattened fields first (they take precedence over GenerationConfig)
	if in.Temperature != nil {
		cfg.Temperature = genai.Ptr(*in.Temperature)
	}
	if in.TopP != nil {
		cfg.TopP = genai.Ptr(*in.TopP)
	}
	if in.TopK != nil {
		cfg.TopK = genai.Ptr(float32(*in.TopK))
	}
	if in.MaxOutputTokens != nil {
		cfg.MaxOutputTokens = *in.MaxOutputTokens
	}
	if in.Seed != nil {
		cfg.Seed = in.Seed
	}

	// Apply GenerationConfig if present and flattened fields don't override
	if in.GenerationConfig != nil {
		if cfg.Temperature == nil && in.GenerationConfig.Temperature != nil {
			cfg.Temperature = genai.Ptr(*in.GenerationConfig.Temperature)
		}
		if cfg.TopP == nil && in.GenerationConfig.TopP != nil {
			cfg.TopP = genai.Ptr(*in.GenerationConfig.TopP)
		}
		if cfg.TopK == nil && in.GenerationConfig.TopK != nil {
			cfg.TopK = genai.Ptr(*in.GenerationConfig.TopK)
		}
		if cfg.MaxOutputTokens == 0 && in.GenerationConfig.MaxOutputTokens != 0 {
			cfg.MaxOutputTokens = in.GenerationConfig.MaxOutputTokens
		}
		if len(in.GenerationConfig.StopSequences) > 0 {
			cfg.StopSequences = append([]string{}, in.GenerationConfig.StopSequences...)
		}
		if in.GenerationConfig.CandidateCount != 0 {
			cfg.CandidateCount = in.GenerationConfig.CandidateCount
		}
		if in.GenerationConfig.ResponseMIMEType != "" {
			cfg.ResponseMIMEType = in.GenerationConfig.ResponseMIMEType
		}
		if in.GenerationConfig.ResponseSchema != nil {
			cfg.ResponseSchema = in.GenerationConfig.ResponseSchema
		}
		if in.GenerationConfig.ThinkingConfig != nil {
			cfg.ThinkingConfig = &genai.ThinkingConfig{
				IncludeThoughts: in.GenerationConfig.ThinkingConfig.IncludeThoughts,
				ThinkingBudget:  in.GenerationConfig.ThinkingConfig.ThinkingBudget,
			}
		}
	}

	// Convert Tools
	if len(in.Tools) > 0 {
		cfg.Tools = in.Tools
	}

	// Convert ToolConfig
	if in.ToolConfig != nil {
		cfg.ToolConfig = in.ToolConfig
	}

	// Convert SafetySettings
	if len(in.SafetySettings) > 0 {
		cfg.SafetySettings = in.SafetySettings
	}

	// Handle SystemInstruction - prioritize systemMessage over SystemInstruction
	if systemMessage != "" {
		cfg.SystemInstruction = &genai.Content{Parts: []*genai.Part{{Text: systemMessage}}}
	} else if in.SystemInstruction != nil && len(in.SystemInstruction.Parts) > 0 {
		cfg.SystemInstruction = in.SystemInstruction
	}

	// Handle CachedContent
	if in.CachedContent != nil {
		cfg.CachedContent = *in.CachedContent
	}

	return cfg
}
