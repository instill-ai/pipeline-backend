package gemini

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/genai"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/data"
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
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
	contents, uploadedFileNames, err := e.buildChatRequestContents(ctx, client, in)
	if err != nil {
		job.Error.Error(ctx, err)
		return nil
	}
	if len(contents) == 0 {
		return nil
	}

	// Ensure uploaded files are cleaned up after chat completion
	defer func() {
		for _, fileName := range uploadedFileNames {
			if _, deleteErr := client.Files.Delete(ctx, fileName, nil); deleteErr != nil {
				// Log the error but don't fail the operation
				// The files will be automatically deleted after 48 hours anyway
				fmt.Printf("Warning: failed to delete uploaded file %s: %v\n", fileName, deleteErr)
			}
		}
	}()

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

// buildChatRequestContents builds the complete request contents from input (history + current message) using File API for large files and videos
func (e *execution) buildChatRequestContents(ctx context.Context, client *genai.Client, in TaskChatInput) ([]*genai.Content, []string, error) {
	// Build user parts (prompt/contents + images + documents) using File API based on total request size
	inParts, uploadedFileNames, err := e.buildReqPartsWithFileAPI(ctx, client, in, false) // isCache = false
	if err != nil {
		return nil, nil, err
	}
	if len(inParts) == 0 {
		return nil, nil, nil
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

	return contents, uploadedFileNames, nil
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
				(*finalResp).Candidates[i].AvgLogprobs = c.AvgLogprobs

				// Due to a bug in the API, the token count is always 0
				// ref: https://discuss.ai.google.dev/t/why-is-token-count-in-generatecontentresponse-always-0/1917
				(*finalResp).Candidates[i].TokenCount += c.TokenCount

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
		PromptFeedback: nil,
		ModelVersion:   nil,
		ResponseID:     nil,
	}

	if finalResp != nil {
		streamOutput.Candidates = finalResp.Candidates
		streamOutput.PromptFeedback = finalResp.PromptFeedback
		if finalResp.ModelVersion != "" {
			mv := finalResp.ModelVersion
			streamOutput.ModelVersion = &mv
		}
		if finalResp.ResponseID != "" {
			ri := finalResp.ResponseID
			streamOutput.ResponseID = &ri
		}

		// Extract images and clean up InlineData in streaming responses
		// This ensures users see images in streaming responses while preventing binary data exposure
		streamOutput.Images = processInlineDataInCandidates(finalResp.Candidates, true)

		// Build usage map from UsageMetadata if available
		if finalResp.UsageMetadata != nil {
			streamOutput.Usage = buildUsageMap(finalResp.UsageMetadata)
		}
	}

	return streamOutput
}

// processInlineDataInCandidates handles InlineData processing in candidates with unified logic.
// If extractImages is true, it extracts image data and converts to format.Image.
// Always cleans up InlineData to prevent binary data exposure in JSON output.
func processInlineDataInCandidates(candidates []*genai.Candidate, extractImages bool) []format.Image {
	var images []format.Image
	if extractImages {
		images = make([]format.Image, 0)
	}

	for _, c := range candidates {
		if c != nil && c.Content != nil {
			for _, p := range c.Content.Parts {
				if p != nil && p.InlineData != nil {
					// Extract image if requested and the data is an image
					if extractImages && strings.Contains(strings.ToLower(p.InlineData.MIMEType), "image") {
						// Convert blob data to format.Image using the standard data package approach
						// Normalize MIME type and use the existing NewImageFromBytes function
						normalizedMimeType := strings.ToLower(strings.TrimSpace(strings.Split(p.InlineData.MIMEType, ";")[0]))
						img, err := data.NewImageFromBytes(p.InlineData.Data, normalizedMimeType, "", true)
						if err == nil {
							images = append(images, img)
						}
					}
					// Always clean up InlineData to prevent raw binary data from being exposed in JSON output
					// The binary data is already extracted and converted to format.Image above (if requested)
					p.InlineData = nil
				}
			}
		}
	}

	return images
}

// buildUsageMap creates a usage map from UsageMetadata with kebab-case keys
func buildUsageMap(metadata *genai.GenerateContentResponseUsageMetadata) map[string]any {
	usage := make(map[string]any)
	usage["prompt-token-count"] = metadata.PromptTokenCount
	usage["cached-content-token-count"] = metadata.CachedContentTokenCount
	usage["candidates-token-count"] = metadata.CandidatesTokenCount
	usage["tool-use-prompt-token-count"] = metadata.ToolUsePromptTokenCount
	usage["thoughts-token-count"] = metadata.ThoughtsTokenCount
	usage["total-token-count"] = metadata.TotalTokenCount

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
		Images:         []format.Image{},
		Usage:          map[string]any{},
		Candidates:     []*genai.Candidate{},
		PromptFeedback: nil,
	}
	if resp == nil {
		return out
	}
	out.Candidates = resp.Candidates
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

	// Extract generated images from candidates and clean up InlineData to prevent raw binary exposure
	out.Images = processInlineDataInCandidates(resp.Candidates, true)

	if resp.UsageMetadata != nil {
		out.Usage = buildUsageMap(resp.UsageMetadata)
	}
	return out
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
		genConfig := in.GenerationConfig
		if cfg.Temperature == nil && genConfig.Temperature != nil {
			cfg.Temperature = genai.Ptr(*genConfig.Temperature)
		}
		if cfg.TopP == nil && genConfig.TopP != nil {
			cfg.TopP = genai.Ptr(*genConfig.TopP)
		}
		if cfg.TopK == nil && genConfig.TopK != nil {
			cfg.TopK = genai.Ptr(*genConfig.TopK)
		}
		if cfg.MaxOutputTokens == 0 && genConfig.MaxOutputTokens != 0 {
			cfg.MaxOutputTokens = genConfig.MaxOutputTokens
		}
		if len(genConfig.StopSequences) > 0 {
			cfg.StopSequences = append([]string{}, genConfig.StopSequences...)
		}
		if genConfig.CandidateCount != 0 {
			cfg.CandidateCount = genConfig.CandidateCount
		}
		if genConfig.ResponseMIMEType != "" {
			cfg.ResponseMIMEType = genConfig.ResponseMIMEType
		}
		if genConfig.ResponseSchema != nil {
			cfg.ResponseSchema = genConfig.ResponseSchema
		}
		if genConfig.ThinkingConfig != nil {
			cfg.ThinkingConfig = &genai.ThinkingConfig{
				IncludeThoughts: genConfig.ThinkingConfig.IncludeThoughts,
				ThinkingBudget:  genConfig.ThinkingConfig.ThinkingBudget,
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
