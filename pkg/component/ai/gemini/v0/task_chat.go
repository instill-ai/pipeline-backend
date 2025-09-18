package gemini

import (
	"context"
	"encoding/base64"
	"fmt"
	"mime"
	"path"
	"slices"
	"strings"

	"google.golang.org/genai"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/data"
)

func (e *execution) chat(ctx context.Context, job *base.Job) error {
	// Read input
	in := TaskChatInput{}
	if err := job.Input.ReadData(ctx, &in); err != nil {
		job.Error.Error(ctx, err)
		return nil
	}

	// Create Gemini client
	apiKey := e.Setup.GetFields()[cfgAPIKey].GetStringValue()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{APIKey: apiKey, Backend: genai.BackendGeminiAPI})
	if err != nil {
		job.Error.Error(ctx, err)
		return nil
	}

	// Handle system message/instruction - prioritize system-message over system-instruction
	var systemMessage string
	if in.SystemMessage != nil && *in.SystemMessage != "" {
		systemMessage = *in.SystemMessage
	} else if in.SystemInstruction != nil && len(in.SystemInstruction.Parts) > 0 {
		for _, p := range in.SystemInstruction.Parts {
			if p.Text != nil && *p.Text != "" {
				systemMessage = *p.Text
				break
			}
		}
	}

	// Build generation config
	cfg := buildGenerateContentConfig(in, systemMessage)

	// Build user parts (prompt/contents + images + documents)
	inParts, err := buildReqParts(in)
	if err != nil {
		job.Error.Error(ctx, err)
		return nil
	}
	if len(inParts) == 0 {
		return nil
	}

	// Merge chat history and current message into contents
	contents := make([]*genai.Content, 0)
	if len(in.ChatHistory) > 0 {
		for _, h := range in.ChatHistory {
			role := genai.RoleUser
			if h.Role != nil && *h.Role == "model" {
				role = genai.RoleModel
			}
			ps := buildParts(h.Parts)
			if len(ps) == 0 {
				continue
			}
			partsPtrs := make([]*genai.Part, 0, len(ps))
			for i := range ps {
				p := ps[i]
				partsPtrs = append(partsPtrs, &p)
			}
			contents = append(contents, &genai.Content{Role: role, Parts: partsPtrs})
		}
	}

	// Append current user message as last turn
	partsPtrs := make([]*genai.Part, 0, len(inParts))
	for i := range inParts {
		p := inParts[i]
		partsPtrs = append(partsPtrs, &p)
	}
	contents = append(contents, &genai.Content{Role: genai.RoleUser, Parts: partsPtrs})

	// Send message (merged history + current)
	var resp *genai.GenerateContentResponse
	streamEnabled := in.Stream != nil && *in.Stream
	if streamEnabled {
		texts := make([]string, 0)
		for r, err := range client.Models.GenerateContentStream(ctx, in.Model, contents, cfg) {
			if err != nil {
				job.Error.Error(ctx, err)
				return nil
			}
			if r != nil && len(r.Candidates) > 0 {
				for len(texts) < len(r.Candidates) {
					texts = append(texts, "")
				}
				for i, c := range r.Candidates {
					if c.Content != nil {
						for _, p := range c.Content.Parts {
							if p != nil && p.Text != "" {
								texts[i] += p.Text
							}
						}
					}
				}
			}
			// incremental flush
			if err := job.Output.WriteData(ctx, TaskChatOutput{Texts: texts, Usage: map[string]any{}, Candidates: []candidate{}, UsageMetadata: usageMetadata{}, PromptFeedback: promptFeedback{}}); err != nil {
				job.Error.Error(ctx, err)
				return nil
			}
			resp = r
		}
		finalOut := renderFinal(resp, texts)
		if err := job.Output.WriteData(ctx, finalOut); err != nil {
			job.Error.Error(ctx, err)
			return nil
		}
	} else {
		res, err := client.Models.GenerateContent(ctx, in.Model, contents, cfg)
		if err != nil {
			job.Error.Error(ctx, err)
			return nil
		}
		resp = res
		finalOut := renderFinal(resp, nil)
		if err := job.Output.WriteData(ctx, finalOut); err != nil {
			job.Error.Error(ctx, err)
			return nil
		}
	}
	return nil
}

// renderFinal builds a complete output from a final genai response.
func renderFinal(resp *genai.GenerateContentResponse, texts []string) TaskChatOutput {
	out := TaskChatOutput{
		Texts:          []string{},
		Usage:          map[string]any{},
		Candidates:     []candidate{},
		UsageMetadata:  usageMetadata{},
		PromptFeedback: promptFeedback{},
	}
	if resp == nil {
		return out
	}
	// Candidates
	if len(resp.Candidates) > 0 {
		out.Candidates = make([]candidate, 0, len(resp.Candidates))
		for i, c := range resp.Candidates {
			var contentPtr *content
			if c.Content != nil {
				contentPtr = convertGenaiContent(c.Content)
			}
			srSlice := make([]safetyRating, 0, len(c.SafetyRatings))
			for _, sr := range c.SafetyRatings {
				if sr == nil {
					continue
				}
				srSlice = append(srSlice, safetyRating{Category: string(sr.Category), Probability: string(sr.Probability), Blocked: &sr.Blocked})
			}
			finishReason := string(c.FinishReason)
			var citationPtr *citationMetadata
			if c.CitationMetadata != nil {
				citations := make([]citationSource, 0, len(c.CitationMetadata.Citations))
				for _, cs := range c.CitationMetadata.Citations {
					if cs == nil {
						continue
					}
					citations = append(citations, citationSource{URI: cs.URI, Title: cs.Title, StartIndex: cs.StartIndex, EndIndex: cs.EndIndex})
				}
				citationPtr = &citationMetadata{Citations: citations}
			}
			tokenCount := int32(0)
			if c.TokenCount > 0 {
				tokenCount = c.TokenCount
			}
			var logprobPtr *logprobsResult
			if c.LogprobsResult != nil {
				logprobTops := make([]logprobsTopCandidate, 0)
				for _, step := range c.LogprobsResult.TopCandidates {
					if step == nil {
						continue
					}
					for _, tc := range step.Candidates {
						if tc == nil {
							continue
						}
						logprobTops = append(logprobTops, logprobsTopCandidate{Token: tc.Token, Logprob: tc.LogProbability})
					}
				}
				logprobPtr = &logprobsResult{TopCandidates: logprobTops}
			}
			var groundingMetaPtr *groundingMetadata
			if c.GroundingMetadata != nil {
				gm := groundingMetadata{}
				if len(c.GroundingMetadata.GroundingChunks) > 0 {
					gm.GroundingChunks = make([]groundingChunk, 0, len(c.GroundingMetadata.GroundingChunks))
					for _, gch := range c.GroundingMetadata.GroundingChunks {
						if gch == nil {
							continue
						}
						var wc *webChunk
						if gch.Web != nil {
							wc = &webChunk{URI: gch.Web.URI, Title: gch.Web.Title}
						}
						gm.GroundingChunks = append(gm.GroundingChunks, groundingChunk{Web: wc})
					}
				}
				if len(c.GroundingMetadata.GroundingSupports) > 0 {
					gm.GroundingSupports = make([]groundingSupport, 0, len(c.GroundingMetadata.GroundingSupports))
					for _, gs := range c.GroundingMetadata.GroundingSupports {
						if gs == nil {
							continue
						}
						var segPtr *segment
						if gs.Segment != nil {
							segPtr = &segment{PartIndex: gs.Segment.PartIndex, StartIndex: gs.Segment.StartIndex, EndIndex: gs.Segment.EndIndex, Text: gs.Segment.Text}
						}
						gm.GroundingSupports = append(gm.GroundingSupports, groundingSupport{GroundingChunkIndices: gs.GroundingChunkIndices, ConfidenceScores: gs.ConfidenceScores, Segment: segPtr})
					}
				}
				if c.GroundingMetadata.RetrievalMetadata != nil {
					gm.RetrievalMetadata = &retrievalMetadata{GoogleSearchDynamicRetrievalScore: c.GroundingMetadata.RetrievalMetadata.GoogleSearchDynamicRetrievalScore}
				}
				if len(c.GroundingMetadata.WebSearchQueries) > 0 {
					gm.WebSearchQueries = append([]string{}, c.GroundingMetadata.WebSearchQueries...)
				}
				if c.GroundingMetadata.SearchEntryPoint != nil {
					gm.SearchEntryPoint = &searchEntryPoint{RenderedContent: c.GroundingMetadata.SearchEntryPoint.RenderedContent, SDKBlob: string(c.GroundingMetadata.SearchEntryPoint.SDKBlob)}
				}
				groundingMetaPtr = &gm
			}
			out.Candidates = append(out.Candidates, candidate{Index: int32(i), Content: contentPtr, SafetyRatings: srSlice, FinishReason: finishReason, CitationMetadata: citationPtr, TokenCount: tokenCount, LogprobsResult: logprobPtr, AvgLogprobs: c.AvgLogprobs, GroundingMetadata: groundingMetaPtr})
		}
	}
	// Usage metadata
	if resp.UsageMetadata != nil {
		out.UsageMetadata = usageMetadata{PromptTokenCount: resp.UsageMetadata.PromptTokenCount, CachedContentTokenCount: resp.UsageMetadata.CachedContentTokenCount, CandidatesTokenCount: resp.UsageMetadata.CandidatesTokenCount, ToolUsePromptTokenCount: resp.UsageMetadata.ToolUsePromptTokenCount, ThoughtsTokenCount: resp.UsageMetadata.ThoughtsTokenCount, TotalTokenCount: resp.UsageMetadata.TotalTokenCount, PromptTokensDetails: cloneModalityCounts(resp.UsageMetadata.PromptTokensDetails), CacheTokensDetails: cloneModalityCounts(resp.UsageMetadata.CacheTokensDetails), CandidatesTokensDetails: cloneModalityCounts(resp.UsageMetadata.CandidatesTokensDetails), ToolUsePromptTokensDetails: cloneModalityCounts(resp.UsageMetadata.ToolUsePromptTokensDetails)}
	}
	// Prompt feedback
	if resp.PromptFeedback != nil {
		pf := promptFeedback{}
		br := string(resp.PromptFeedback.BlockReason)
		pf.BlockReason = &br
		for _, sr := range resp.PromptFeedback.SafetyRatings {
			if sr == nil {
				continue
			}
			pf.SafetyRatings = append(pf.SafetyRatings, safetyRating{Category: string(sr.Category), Probability: string(sr.Probability), Blocked: &sr.Blocked})
		}
		out.PromptFeedback = pf
	}
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
		usage := make(map[string]any)
		// Kebab-case counts expected by schema
		usage["prompt-token-count"] = resp.UsageMetadata.PromptTokenCount
		usage["cached-content-token-count"] = resp.UsageMetadata.CachedContentTokenCount
		usage["candidates-token-count"] = resp.UsageMetadata.CandidatesTokenCount
		usage["total-token-count"] = resp.UsageMetadata.TotalTokenCount
		usage["tool-use-prompt-token-count"] = resp.UsageMetadata.ToolUsePromptTokenCount
		usage["thoughts-token-count"] = resp.UsageMetadata.ThoughtsTokenCount

		// Details arrays in kebab-case
		if len(resp.UsageMetadata.PromptTokensDetails) > 0 {
			arr := make([]map[string]any, 0, len(resp.UsageMetadata.PromptTokensDetails))
			for _, d := range resp.UsageMetadata.PromptTokensDetails {
				if d == nil {
					continue
				}
				arr = append(arr, map[string]any{"modality": string(d.Modality), "token-count": int(d.TokenCount)})
			}
			usage["prompt-tokens-details"] = arr
		}
		if len(resp.UsageMetadata.CacheTokensDetails) > 0 {
			arr := make([]map[string]any, 0, len(resp.UsageMetadata.CacheTokensDetails))
			for _, d := range resp.UsageMetadata.CacheTokensDetails {
				if d == nil {
					continue
				}
				arr = append(arr, map[string]any{"modality": string(d.Modality), "token-count": int(d.TokenCount)})
			}
			usage["cache-tokens-details"] = arr
		}
		if len(resp.UsageMetadata.CandidatesTokensDetails) > 0 {
			arr := make([]map[string]any, 0, len(resp.UsageMetadata.CandidatesTokensDetails))
			for _, d := range resp.UsageMetadata.CandidatesTokensDetails {
				if d == nil {
					continue
				}
				arr = append(arr, map[string]any{"modality": string(d.Modality), "token-count": int(d.TokenCount)})
			}
			usage["candidates-tokens-details"] = arr
		}
		if len(resp.UsageMetadata.ToolUsePromptTokensDetails) > 0 {
			arr := make([]map[string]any, 0, len(resp.UsageMetadata.ToolUsePromptTokensDetails))
			for _, d := range resp.UsageMetadata.ToolUsePromptTokensDetails {
				if d == nil {
					continue
				}
				arr = append(arr, map[string]any{"modality": string(d.Modality), "token-count": int(d.TokenCount)})
			}
			usage["tool-use-prompt-tokens-details"] = arr
		}

		out.Usage = usage
	}
	return out
}

// convertGenaiContent maps genai.Content to our internal content type.
func convertGenaiContent(c *genai.Content) *content {
	out := content{}
	if c.Role != "" {
		r := string(c.Role)
		out.Role = &r
	}
	for _, p := range c.Parts {
		if p == nil {
			continue
		}
		pp := part{}
		if p.Text != "" {
			pp.Text = &p.Text
			out.Parts = append(out.Parts, pp)
			continue
		}
		if p.InlineData != nil {
			pp.InlineData = &blob{MIMEType: p.InlineData.MIMEType, Data: base64.StdEncoding.EncodeToString(p.InlineData.Data)}
			out.Parts = append(out.Parts, pp)
			continue
		}
		if p.FileData != nil {
			pp.FileData = &fileData{MIMEType: p.FileData.MIMEType, URI: p.FileData.FileURI}
			out.Parts = append(out.Parts, pp)
			continue
		}
		if p.ExecutableCode != nil {
			pp.ExecutableCode = &executableCode{Language: string(p.ExecutableCode.Language), Code: p.ExecutableCode.Code}
			out.Parts = append(out.Parts, pp)
			continue
		}
		if p.CodeExecutionResult != nil {
			var outStr *string
			if p.CodeExecutionResult.Output != "" {
				v := p.CodeExecutionResult.Output
				outStr = &v
			}
			pp.CodeExecutionResult = &codeExecutionResult{Outcome: string(p.CodeExecutionResult.Outcome), Output: outStr}
			out.Parts = append(out.Parts, pp)
			continue
		}
	}
	return &out
}

// cloneModalityCounts converts []*genai.ModalityTokenCount to []modalityTokenCount.
func cloneModalityCounts(src []*genai.ModalityTokenCount) []modalityTokenCount {
	if len(src) == 0 {
		return nil
	}
	out := make([]modalityTokenCount, 0, len(src))
	for _, m := range src {
		if m == nil {
			continue
		}
		out = append(out, modalityTokenCount{Modality: string(m.Modality), TokenCount: int(m.TokenCount)})
	}
	return out
}

// Helpers for Images/Documents strings to genai.Part
func newURIOrDataPart(s string, defaultMIME string) *genai.Part {
	if s == "" {
		return nil
	}
	if strings.HasPrefix(s, "data:") {
		// data:[<mediatype>][;base64],<data>
		h := s[5:]
		comma := strings.IndexByte(h, ',')
		if comma < 0 {
			return nil
		}
		head := h[:comma]
		data := h[comma+1:]
		mimeType := defaultMIME
		isBase64 := false

		// Parse the media type and check for base64 encoding
		if semi := strings.IndexByte(head, ';'); semi >= 0 {
			mimeType = head[:semi]
			params := head[semi+1:]
			if params == "base64" {
				isBase64 = true
			}
		} else if head != "" {
			mimeType = head
		}

		var b []byte
		var err error
		if isBase64 {
			b, err = base64.StdEncoding.DecodeString(data)
			if err != nil {
				return nil
			}
		} else {
			// URL decode the data for non-base64 data URIs
			b = []byte(data)
		}
		return &genai.Part{InlineData: &genai.Blob{MIMEType: mimeType, Data: b}}
	}
	// Try raw base64 (no prefix)
	if decoded, err := base64.StdEncoding.DecodeString(s); err == nil {
		return &genai.Part{InlineData: &genai.Blob{MIMEType: defaultMIME, Data: decoded}}
	}
	// Otherwise, treat as URI
	mimeType := defaultMIME
	if u := genai.NewPartFromURI(s, mimeType); u != nil {
		return u
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

// buildParts converts our []part to []genai.Part.
func buildParts(ps []part) []genai.Part {
	out := make([]genai.Part, 0, len(ps))
	for _, p := range ps {
		if p.Text != nil {
			out = append(out, genai.Part{Text: *p.Text})
			continue
		}
		if p.FileData != nil {
			if u := genai.NewPartFromURI(p.FileData.URI, p.FileData.MIMEType); u != nil {
				out = append(out, *u)
			}
			continue
		}
		if p.InlineData != nil {
			if b, err := base64.StdEncoding.DecodeString(p.InlineData.Data); err == nil {
				out = append(out, genai.Part{InlineData: &genai.Blob{MIMEType: p.InlineData.MIMEType, Data: b}})
			}
			continue
		}
	}
	return out
}

// buildReqParts constructs the user request parts from input, including prompt/contents, images, and documents.
// Following best practices: text content (from both Contents and Prompt) is placed after visual content (images/documents).
func buildReqParts(in TaskChatInput) ([]genai.Part, error) {
	parts := []genai.Part{}

	// Separate non-text and text parts from Contents for proper ordering
	var nonTextParts []genai.Part
	var textParts []genai.Part
	if len(in.Contents) > 0 {
		last := in.Contents[len(in.Contents)-1]
		contentParts := buildParts(last.Parts)
		for _, part := range contentParts {
			if part.Text != "" {
				textParts = append(textParts, part)
			} else {
				nonTextParts = append(nonTextParts, part)
			}
		}
	}

	// Add non-text parts from Contents first (images, files, etc.)
	parts = append(parts, nonTextParts...)

	// Add images before documents for optimal processing
	for _, img := range in.Images {
		imgBase64, err := img.Base64()
		if err != nil {
			return nil, err
		}
		if p := newURIOrDataPart(imgBase64.String(), detectMIMEFromPath(imgBase64.String(), "image/png")); p != nil {
			parts = append(parts, *p)
		}
	}
	// Process documents according to their capabilities:
	// - PDFs: Full document vision support (charts, diagrams, formatting preserved)
	// - Text-based: Extract as plain text (HTML tags, Markdown formatting, etc. lost)
	// - Office documents: Recommend PDF conversion for visual understanding
	for _, doc := range in.Documents {
		contentType := doc.ContentType().String()

		if contentType == data.PDF {
			// PDFs support full document vision capabilities
			// The model can interpret visual elements like charts, diagrams, and formatting
			docBase64, err := doc.Base64()
			if err != nil {
				return nil, err
			}
			if p := newURIOrDataPart(docBase64.String(), detectMIMEFromPath(docBase64.String(), "application/pdf")); p != nil {
				parts = append(parts, *p)
			}
		} else if isTextBasedDocument(contentType) {
			// Text-based documents (TXT, Markdown, HTML, XML, etc.)
			// Pass as base64 like PDFs for consistent handling
			docBase64, err := doc.Base64()
			if err != nil {
				return nil, err
			}
			if p := newURIOrDataPart(docBase64.String(), detectMIMEFromPath(docBase64.String(), contentType)); p != nil {
				parts = append(parts, *p)
			}
		} else if isConvertibleToPDF(contentType) {
			// Office documents (DOC, DOCX, PPT, PPTX, XLS, XLSX)
			// These can contain visual elements that would be lost in text extraction
			return nil, fmt.Errorf("document type %s will be processed as text only, losing visual elements like charts and formatting; use \":pdf\" syntax in your input variable to convert to PDF for document vision capabilities", contentType)
		} else {
			return nil, fmt.Errorf("unsupported document type: %s", contentType)
		}
	}

	// Add text parts after documents for best results (as per best practices)
	// This includes both text parts from Contents and the Prompt field
	parts = append(parts, textParts...)
	if in.Prompt != nil && *in.Prompt != "" {
		parts = append(parts, genai.Part{Text: *in.Prompt})
	}

	return parts, nil
}

// isTextBasedDocument checks if a document type should be processed as text content.
// Text-based documents are extracted as plain text, losing visual formatting but preserving content.
// This includes HTML (tags removed), Markdown (formatting lost), plain text, CSV, XML, etc.
func isTextBasedDocument(contentType string) bool {
	textBasedTypes := []string{
		data.HTML,     // text/html - HTML tags will be lost, only text content preserved
		data.MARKDOWN, // text/markdown - Markdown formatting will be lost
		data.TEXT,     // text - already plain text
		data.PLAIN,    // text/plain - already plain text
		data.CSV,      // text/csv - processed as structured text
	}

	return slices.Contains(textBasedTypes, contentType) || strings.HasPrefix(contentType, "text/")
}

// isConvertibleToPDF checks if a MIME type can be converted to PDF using the :pdf syntax.
// These document types often contain visual elements (charts, diagrams, formatting)
// that would be lost if processed as text only. PDF conversion preserves visual understanding.
func isConvertibleToPDF(contentType string) bool {
	convertibleTypes := []string{
		data.DOC,  // application/msword - may contain charts, images, formatting
		data.DOCX, // application/vnd.openxmlformats-officedocument.wordprocessingml.document
		data.PPT,  // application/vnd.ms-powerpoint - presentations with slides, charts
		data.PPTX, // application/vnd.openxmlformats-officedocument.presentationml.presentation
		data.XLS,  // application/vnd.ms-excel - spreadsheets with charts, formatting
		data.XLSX, // application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
	}

	return slices.Contains(convertibleTypes, contentType)
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
		if cfg.MaxOutputTokens == 0 && in.GenerationConfig.MaxOutputTokens != nil {
			cfg.MaxOutputTokens = *in.GenerationConfig.MaxOutputTokens
		}
		if len(in.GenerationConfig.StopSequences) > 0 {
			cfg.StopSequences = append([]string{}, in.GenerationConfig.StopSequences...)
		}
		if in.GenerationConfig.CandidateCount != nil {
			cfg.CandidateCount = *in.GenerationConfig.CandidateCount
		}
		if in.GenerationConfig.ResponseMimeType != nil {
			cfg.ResponseMIMEType = *in.GenerationConfig.ResponseMimeType
		}
		if in.GenerationConfig.ResponseSchema != nil {
			cfg.ResponseSchema = buildSchema(in.GenerationConfig.ResponseSchema)
		}
		if in.GenerationConfig.ThinkingConfig != nil {
			thinkingConfig := &genai.ThinkingConfig{
				ThinkingBudget: in.GenerationConfig.ThinkingConfig.ThinkingBudget,
			}
			if in.GenerationConfig.ThinkingConfig.IncludeThoughts != nil {
				thinkingConfig.IncludeThoughts = *in.GenerationConfig.ThinkingConfig.IncludeThoughts
			}
			cfg.ThinkingConfig = thinkingConfig
		}
	}

	// Convert Tools
	if len(in.Tools) > 0 {
		cfg.Tools = buildTools(in.Tools)
	}

	// Convert ToolConfig
	if in.ToolConfig != nil {
		cfg.ToolConfig = buildToolConfig(in.ToolConfig)
	}

	// Convert SafetySettings
	if len(in.SafetySettings) > 0 {
		cfg.SafetySettings = buildSafetySettings(in.SafetySettings)
	}

	// Handle SystemInstruction - prioritize systemMessage over SystemInstruction
	if systemMessage != "" {
		cfg.SystemInstruction = &genai.Content{Parts: []*genai.Part{{Text: systemMessage}}}
	} else if in.SystemInstruction != nil && len(in.SystemInstruction.Parts) > 0 {
		cfg.SystemInstruction = buildContent(in.SystemInstruction)
	}

	// Handle CachedContent
	if in.CachedContent != nil {
		cfg.CachedContent = *in.CachedContent
	}

	return cfg
}

// buildTools converts []tool to []*genai.Tool
func buildTools(tools []tool) []*genai.Tool {
	result := make([]*genai.Tool, 0, len(tools))
	for _, t := range tools {
		genaiTool := &genai.Tool{}

		// Convert function declarations
		if len(t.FunctionDeclarations) > 0 {
			genaiTool.FunctionDeclarations = make([]*genai.FunctionDeclaration, 0, len(t.FunctionDeclarations))
			for _, fd := range t.FunctionDeclarations {
				genaiDecl := &genai.FunctionDeclaration{
					Name: fd.Name,
				}
				if fd.Description != nil {
					genaiDecl.Description = *fd.Description
				}
				if fd.Parameters != nil {
					genaiDecl.Parameters = buildSchema(fd.Parameters)
				}
				genaiTool.FunctionDeclarations = append(genaiTool.FunctionDeclarations, genaiDecl)
			}
		}

		// Convert Google Search Retrieval
		if t.GoogleSearchRetrieval != nil {
			genaiTool.GoogleSearchRetrieval = &genai.GoogleSearchRetrieval{}
			// Note: DynamicRetrievalConfig conversion may need adjustment based on genai package version
		}

		// Convert Code Execution
		if t.CodeExecution != nil {
			genaiTool.CodeExecution = &genai.ToolCodeExecution{}
		}

		result = append(result, genaiTool)
	}
	return result
}

// buildToolConfig converts *toolConfig to *genai.ToolConfig
func buildToolConfig(tc *toolConfig) *genai.ToolConfig {
	if tc == nil {
		return nil
	}

	result := &genai.ToolConfig{}
	if tc.FunctionCallingConfig != nil {
		result.FunctionCallingConfig = &genai.FunctionCallingConfig{}
		// Note: Mode conversion may need adjustment based on genai package version
		if len(tc.FunctionCallingConfig.AllowedFunctionNames) > 0 {
			result.FunctionCallingConfig.AllowedFunctionNames = append([]string{}, tc.FunctionCallingConfig.AllowedFunctionNames...)
		}
	}
	return result
}

// buildSafetySettings converts []safetySetting to []*genai.SafetySetting
func buildSafetySettings(settings []safetySetting) []*genai.SafetySetting {
	result := make([]*genai.SafetySetting, 0, len(settings))
	for _, s := range settings {
		result = append(result, &genai.SafetySetting{
			Category:  genai.HarmCategory(s.Category),
			Threshold: genai.HarmBlockThreshold(s.Threshold),
		})
	}
	return result
}

// buildContent converts *content to *genai.Content
func buildContent(c *content) *genai.Content {
	if c == nil || len(c.Parts) == 0 {
		return nil
	}

	parts := buildParts(c.Parts)
	if len(parts) == 0 {
		return nil
	}

	partsPtrs := make([]*genai.Part, 0, len(parts))
	for i := range parts {
		p := parts[i]
		partsPtrs = append(partsPtrs, &p)
	}

	result := &genai.Content{Parts: partsPtrs}
	if c.Role != nil {
		if *c.Role == "model" {
			result.Role = genai.RoleModel
		} else {
			result.Role = genai.RoleUser
		}
	}
	return result
}

// buildSchema converts *jsonSchema to *genai.Schema
func buildSchema(js *jsonSchema) *genai.Schema {
	if js == nil {
		return nil
	}

	schema := &genai.Schema{
		Type:        genai.Type(js.Type),
		Format:      js.Format,
		Title:       js.Title,
		Description: js.Description,
		Enum:        js.Enum,
		Required:    js.Required,
		Pattern:     js.Pattern,
		Default:     js.Default,
		Minimum:     js.Minimum,
		Maximum:     js.Maximum,
	}

	// Convert int32 to int64 for items counts and lengths
	if js.MaxItems != nil {
		schema.MaxItems = genai.Ptr(int64(*js.MaxItems))
	}
	if js.MinItems != nil {
		schema.MinItems = genai.Ptr(int64(*js.MinItems))
	}
	if js.MinProperties != nil {
		schema.MinProperties = genai.Ptr(int64(*js.MinProperties))
	}
	if js.MaxProperties != nil {
		schema.MaxProperties = genai.Ptr(int64(*js.MaxProperties))
	}
	if js.MinLength != nil {
		schema.MinLength = genai.Ptr(int64(*js.MinLength))
	}
	if js.MaxLength != nil {
		schema.MaxLength = genai.Ptr(int64(*js.MaxLength))
	}

	// Convert Properties map
	if js.Properties != nil {
		schema.Properties = make(map[string]*genai.Schema)
		for k, v := range js.Properties {
			schema.Properties[k] = buildSchema(&v)
		}
	}

	// Convert AnyOf slice
	if js.AnyOf != nil {
		schema.AnyOf = make([]*genai.Schema, 0, len(js.AnyOf))
		for _, v := range js.AnyOf {
			schema.AnyOf = append(schema.AnyOf, buildSchema(&v))
		}
	}

	// Convert Items
	if js.Items != nil {
		schema.Items = buildSchema(js.Items)
	}

	return schema
}
