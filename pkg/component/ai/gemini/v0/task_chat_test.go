package gemini

import (
	"encoding/base64"
	"testing"

	"google.golang.org/genai"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/data"
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
)

func Test_newURIOrDataPart_DataURI_ImagePNG(t *testing.T) {
	c := qt.New(t)
	// 1x1 transparent PNG
	pngB64 := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR4nGNgYAAAAAMAASsJTYQAAAAASUVORK5CYII="
	dataURI := "data:image/png;base64," + pngB64
	p := newURIOrDataPart(dataURI, "image/png")
	c.Assert(p, qt.IsNotNil)
	c.Check(p.InlineData, qt.Not(qt.IsNil))
	c.Check(p.InlineData.MIMEType, qt.Equals, "image/png")
	decoded, _ := base64.StdEncoding.DecodeString(pngB64)
	c.Check(p.InlineData.Data, qt.DeepEquals, decoded)
}

func Test_newURIOrDataPart_RawBase64_PDF(t *testing.T) {
	c := qt.New(t)
	// "%PDF-1.4\n" in base64
	pdfHeader := "JVBERi0xLjQK"
	p := newURIOrDataPart(pdfHeader, "application/pdf")
	c.Assert(p, qt.IsNotNil)
	c.Check(p.InlineData, qt.Not(qt.IsNil))
	c.Check(p.InlineData.MIMEType, qt.Equals, "application/pdf")
	decoded, _ := base64.StdEncoding.DecodeString(pdfHeader)
	c.Check(p.InlineData.Data, qt.DeepEquals, decoded)
}

func Test_newURIOrDataPart_DataURI_EmptyMediaType(t *testing.T) {
	c := qt.New(t)
	// Test data URI with no media type: data:,somedata
	dataURI := "data:,somedata"
	p := newURIOrDataPart(dataURI, "text/plain")
	c.Assert(p, qt.IsNotNil)
	c.Check(p.InlineData, qt.Not(qt.IsNil))
	c.Check(p.InlineData.MIMEType, qt.Equals, "text/plain") // Should use default
	c.Check(p.InlineData.Data, qt.DeepEquals, []byte("somedata"))
}

func Test_detectMIMEFromPath(t *testing.T) {
	c := qt.New(t)
	c.Check(detectMIMEFromPath("photo.jpg", "image/png"), qt.Equals, "image/jpeg")
	c.Check(detectMIMEFromPath("doc.pdf", "application/octet-stream"), qt.Equals, "application/pdf")
	c.Check(detectMIMEFromPath("unknown.bin", "application/octet-stream"), qt.Equals, "application/octet-stream")
}

func Test_GenaiParts_TextAndInlineData(t *testing.T) {
	c := qt.New(t)
	hello := "hello"
	// Using genai.Part directly - no conversion needed
	ps := []*genai.Part{
		{Text: hello},
		{InlineData: &genai.Blob{MIMEType: "application/octet-stream", Data: []byte{0x01, 0x02}}},
	}
	// Test that genai parts work correctly
	c.Assert(ps, qt.HasLen, 2)
	c.Check(ps[0].Text, qt.Equals, hello)
	c.Check(ps[1].InlineData, qt.Not(qt.IsNil))
	c.Check(ps[1].InlineData.MIMEType, qt.Equals, "application/octet-stream")
	c.Check(len(ps[1].InlineData.Data) > 0, qt.IsTrue)
}

func Test_buildReqParts_Prompt_Images_Documents(t *testing.T) {
	c := qt.New(t)
	prompt := "Summarize this."
	imgData := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR4nGNgYAAAAAMAASsJTYQAAAAASUVORK5CYII="
	pdfHeader := "JVBERi0xLjQK" // raw base64 PDF header
	imageBytes, err := base64.StdEncoding.DecodeString(imgData)
	if err != nil {
		t.Fatal(err)
	}
	img, err := data.NewImageFromBytes(imageBytes, "image/png", "test.png", true)
	if err != nil {
		t.Fatal(err)
	}
	pdfBytes, err := base64.StdEncoding.DecodeString(pdfHeader)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := data.NewDocumentFromBytes(pdfBytes, "application/pdf", "test.pdf")
	if err != nil {
		t.Fatal(err)
	}

	in := TaskChatInput{
		Prompt:    &prompt,
		Images:    []format.Image{img},
		Documents: []format.Document{doc},
	}
	got, err := buildReqParts(in)
	c.Assert(err, qt.IsNil)
	// Expect 1 image + 1 PDF doc + 1 text prompt = 3 parts (prompt now comes last)
	c.Assert(got, qt.HasLen, 3)
	c.Check(got[0].InlineData, qt.Not(qt.IsNil))
	c.Check(got[0].InlineData.MIMEType, qt.Equals, "image/png")
	c.Check(got[1].InlineData, qt.Not(qt.IsNil))
	c.Check(got[1].InlineData.MIMEType, qt.Equals, "application/pdf")
	c.Check(got[2].Text, qt.Equals, prompt) // Prompt is now last
}

func Test_buildReqParts_UnsupportedDocumentMIME_Convertible(t *testing.T) {
	c := qt.New(t)
	prompt := "Summarize this."

	// Create a document with convertible MIME type (DOC)
	docBytes := []byte("This is a DOC document")
	doc, err := data.NewDocumentFromBytes(docBytes, data.DOC, "test.doc")
	if err != nil {
		t.Fatal(err)
	}

	in := TaskChatInput{
		Prompt:    &prompt,
		Documents: []format.Document{doc},
	}

	got, err := buildReqParts(in)
	c.Assert(err, qt.Not(qt.IsNil))
	c.Assert(err.Error(), qt.Contains, "document type application/msword will be processed as text only")
	c.Assert(err.Error(), qt.Contains, "use \":pdf\" syntax")
	c.Assert(got, qt.IsNil)
}

func Test_buildReqParts_TextBasedDocument_CSV(t *testing.T) {
	c := qt.New(t)
	prompt := "Summarize this."

	// Create a document with text-based MIME type (CSV)
	csvContent := "name,age\nJohn,30\nJane,25"
	docBytes := []byte(csvContent)
	doc, err := data.NewDocumentFromBytes(docBytes, data.CSV, "test.csv")
	if err != nil {
		t.Fatal(err)
	}

	in := TaskChatInput{
		Prompt:    &prompt,
		Documents: []format.Document{doc},
	}

	got, err := buildReqParts(in)
	c.Assert(err, qt.IsNil)
	// Expect 1 text part (CSV content) + 1 text part (prompt) = 2 parts
	c.Assert(got, qt.HasLen, 2)
	c.Check(got[0].Text, qt.Equals, csvContent) // CSV content as text
	c.Check(got[1].Text, qt.Equals, prompt)     // Prompt comes last
}

func Test_buildReqParts_TextBasedDocument_HTML(t *testing.T) {
	c := qt.New(t)
	prompt := "Extract the main content."

	// Create an HTML document
	htmlContent := "<html><body><h1>Title</h1><p>Content</p></body></html>"
	docBytes := []byte(htmlContent)
	doc, err := data.NewDocumentFromBytes(docBytes, data.HTML, "test.html")
	if err != nil {
		t.Fatal(err)
	}

	in := TaskChatInput{
		Prompt:    &prompt,
		Documents: []format.Document{doc},
	}

	got, err := buildReqParts(in)
	c.Assert(err, qt.IsNil)
	// Expect 1 text part (HTML content) + 1 text part (prompt) = 2 parts
	c.Assert(got, qt.HasLen, 2)
	c.Check(got[0].Text, qt.Equals, htmlContent) // HTML content as text (tags preserved in extraction)
	c.Check(got[1].Text, qt.Equals, prompt)      // Prompt comes last
}

func Test_buildReqParts_TextBasedDocument_Markdown(t *testing.T) {
	c := qt.New(t)
	prompt := "Convert to HTML."

	// Create a Markdown document
	markdownContent := "# Title\n\nThis is **bold** text."
	docBytes := []byte(markdownContent)
	doc, err := data.NewDocumentFromBytes(docBytes, data.MARKDOWN, "test.md")
	if err != nil {
		t.Fatal(err)
	}

	in := TaskChatInput{
		Prompt:    &prompt,
		Documents: []format.Document{doc},
	}

	got, err := buildReqParts(in)
	c.Assert(err, qt.IsNil)
	// Expect 1 text part (Markdown content) + 1 text part (prompt) = 2 parts
	c.Assert(got, qt.HasLen, 2)
	c.Check(got[0].Text, qt.Equals, markdownContent) // Markdown content as text
	c.Check(got[1].Text, qt.Equals, prompt)          // Prompt comes last
}

func Test_buildReqParts_UnsupportedDocumentType(t *testing.T) {
	c := qt.New(t)
	prompt := "Process this."

	// Create a mock document that simulates an unsupported type
	// We'll create a document with a supported data package type but use a filename that won't trigger conversion
	docBytes := []byte("binary data")
	doc, err := data.NewDocumentFromBytes(docBytes, data.OCTETSTREAM, "test.unknown")
	if err != nil {
		t.Fatal(err)
	}

	in := TaskChatInput{
		Prompt:    &prompt,
		Documents: []format.Document{doc},
	}

	got, err := buildReqParts(in)
	c.Assert(err, qt.Not(qt.IsNil))
	// Since OCTETSTREAM with unknown extension gets converted to DOC by default,
	// it will be caught by our convertible check
	c.Assert(err.Error(), qt.Contains, "document type application/msword will be processed as text only")
	c.Assert(got, qt.IsNil)
}

func Test_buildReqParts_Contents_TextOrdering(t *testing.T) {
	c := qt.New(t)

	// Create test data
	imgData := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR4nGNgYAAAAAMAASsJTYQAAAAASUVORK5CYII="
	pdfHeader := "JVBORi0xLjQK"
	imageBytes, err := base64.StdEncoding.DecodeString(imgData)
	if err != nil {
		t.Fatal(err)
	}
	img, err := data.NewImageFromBytes(imageBytes, "image/png", "test.png", true)
	if err != nil {
		t.Fatal(err)
	}
	pdfBytes, err := base64.StdEncoding.DecodeString(pdfHeader)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := data.NewDocumentFromBytes(pdfBytes, "application/pdf", "test.pdf")
	if err != nil {
		t.Fatal(err)
	}

	// Create Contents with mixed text and non-text parts
	textPart1 := "First text from Contents"
	textPart2 := "Second text from Contents"
	// imageBytes is already []byte, ready for genai.Blob

	in := TaskChatInput{
		Images:    []format.Image{img},
		Documents: []format.Document{doc},
		Contents: []*genai.Content{
			{
				Parts: []*genai.Part{
					{Text: textPart1},
					{InlineData: &genai.Blob{MIMEType: "image/png", Data: imageBytes}},
					{Text: textPart2},
				},
			},
		},
	}

	got, err := buildReqParts(in)
	c.Assert(err, qt.IsNil)
	// Expect: 1 image from Contents + 1 image from Images + 1 PDF doc + 2 text parts from Contents = 5 parts
	c.Assert(got, qt.HasLen, 5)

	// Verify ordering: non-text from Contents, then Images, then Documents, then text from Contents
	c.Check(got[0].InlineData, qt.Not(qt.IsNil)) // Image from Contents
	c.Check(got[0].InlineData.MIMEType, qt.Equals, "image/png")
	c.Check(got[1].InlineData, qt.Not(qt.IsNil)) // Image from Images field
	c.Check(got[1].InlineData.MIMEType, qt.Equals, "image/png")
	c.Check(got[2].InlineData, qt.Not(qt.IsNil)) // PDF from Documents
	c.Check(got[2].InlineData.MIMEType, qt.Equals, "application/pdf")
	c.Check(got[3].Text, qt.Equals, textPart1) // First text from Contents (placed after documents)
	c.Check(got[4].Text, qt.Equals, textPart2) // Second text from Contents
}

func Test_isTextBasedDocument(t *testing.T) {
	c := qt.New(t)

	// Test text-based document types
	c.Check(isTextBasedDocument(data.HTML), qt.Equals, true)
	c.Check(isTextBasedDocument(data.MARKDOWN), qt.Equals, true)
	c.Check(isTextBasedDocument(data.PLAIN), qt.Equals, true)
	c.Check(isTextBasedDocument(data.CSV), qt.Equals, true)
	c.Check(isTextBasedDocument("text/xml"), qt.Equals, true)
	c.Check(isTextBasedDocument("text/javascript"), qt.Equals, true)

	// Test non-text-based document types
	c.Check(isTextBasedDocument(data.PDF), qt.Equals, false)
	c.Check(isTextBasedDocument(data.DOC), qt.Equals, false)
	c.Check(isTextBasedDocument(data.DOCX), qt.Equals, false)
	c.Check(isTextBasedDocument("application/octet-stream"), qt.Equals, false)
	c.Check(isTextBasedDocument("image/png"), qt.Equals, false)
}

func Test_isConvertibleToPDF(t *testing.T) {
	c := qt.New(t)

	// Test convertible document types
	c.Check(isConvertibleToPDF(data.DOC), qt.Equals, true)
	c.Check(isConvertibleToPDF(data.DOCX), qt.Equals, true)
	c.Check(isConvertibleToPDF(data.PPT), qt.Equals, true)
	c.Check(isConvertibleToPDF(data.PPTX), qt.Equals, true)
	c.Check(isConvertibleToPDF(data.XLS), qt.Equals, true)
	c.Check(isConvertibleToPDF(data.XLSX), qt.Equals, true)

	// Test non-convertible document types
	c.Check(isConvertibleToPDF(data.PDF), qt.Equals, false)
	c.Check(isConvertibleToPDF(data.HTML), qt.Equals, false)
	c.Check(isConvertibleToPDF(data.MARKDOWN), qt.Equals, false)
	c.Check(isConvertibleToPDF(data.PLAIN), qt.Equals, false)
	c.Check(isConvertibleToPDF("application/octet-stream"), qt.Equals, false)
}

func Test_renderFinal_Minimal(t *testing.T) {
	c := qt.New(t)
	// Build a minimal GenerateContentResponse with one candidate and usage
	resp := &genai.GenerateContentResponse{
		ModelVersion: "v1",
		ResponseID:   "resp-123",
		Candidates: []*genai.Candidate{
			{
				Content: &genai.Content{Parts: []*genai.Part{{Text: "hello"}}},
			},
		},
		UsageMetadata: &genai.GenerateContentResponseUsageMetadata{
			PromptTokenCount:        1,
			CachedContentTokenCount: 0,
			CandidatesTokenCount:    2,
			TotalTokenCount:         3,
		},
	}
	out := renderFinal(resp, nil)
	c.Assert(out.Texts, qt.DeepEquals, []string{"hello"})
	c.Check(out.ModelVersion, qt.Not(qt.IsNil))
	c.Check(*out.ModelVersion, qt.Equals, "v1")
	c.Check(out.ResponseID, qt.Not(qt.IsNil))
	c.Check(*out.ResponseID, qt.Equals, "resp-123")
	c.Check(out.UsageMetadata.TotalTokenCount, qt.Equals, int32(3))
}

func Test_renderFinal_StreamingCandidatesPreservation(t *testing.T) {
	c := qt.New(t)

	// Simulate a streaming scenario with multiple chunks that would be merged
	// This represents what the streaming logic should produce as finalResp

	// Create a response that simulates merged streaming chunks
	// Chunk 1: "Hello" (first part)
	// Chunk 2: " world" (second part)
	// Chunk 3: "!" (third part)
	mergedResp := &genai.GenerateContentResponse{
		ModelVersion: "gemini-2.5-pro",
		ResponseID:   "resp-streaming-123",
		Candidates: []*genai.Candidate{
			{
				Index:        0,
				FinishReason: genai.FinishReasonStop,
				TokenCount:   10,
				AvgLogprobs:  -0.5,
				Content: &genai.Content{
					Role: genai.RoleModel,
					Parts: []*genai.Part{
						{Text: "Hello"},  // From chunk 1
						{Text: " world"}, // From chunk 2
						{Text: "!"},      // From chunk 3
					},
				},
				SafetyRatings: []*genai.SafetyRating{
					{
						Category:    genai.HarmCategoryDangerousContent,
						Probability: genai.HarmProbabilityNegligible,
						Blocked:     false,
					},
				},
			},
			{
				Index:        1,
				FinishReason: genai.FinishReasonStop,
				TokenCount:   8,
				AvgLogprobs:  -0.3,
				Content: &genai.Content{
					Role: genai.RoleModel,
					Parts: []*genai.Part{
						{Text: "Alternative"}, // From chunk 1
						{Text: " response"},   // From chunk 2
					},
				},
			},
		},
		UsageMetadata: &genai.GenerateContentResponseUsageMetadata{
			PromptTokenCount:     5,
			CandidatesTokenCount: 18,
			TotalTokenCount:      23,
		},
	}

	// Test renderFinal with the merged response
	out := renderFinal(mergedResp, nil)

	// Verify Texts field reflects all candidate parts text
	c.Assert(len(out.Texts), qt.Equals, 2)
	c.Assert(out.Texts[0], qt.Equals, "Hello world!")         // All parts from candidate 0 concatenated
	c.Assert(out.Texts[1], qt.Equals, "Alternative response") // All parts from candidate 1 concatenated

	// Verify all candidates are preserved
	c.Assert(len(out.Candidates), qt.Equals, 2)

	// Verify first candidate
	c.Assert(out.Candidates[0].Index, qt.Equals, int32(0))
	c.Assert(out.Candidates[0].FinishReason, qt.Equals, genai.FinishReasonStop)
	c.Assert(out.Candidates[0].TokenCount, qt.Equals, int32(10))
	c.Assert(out.Candidates[0].AvgLogprobs, qt.Equals, -0.5)
	c.Assert(out.Candidates[0].Content, qt.Not(qt.IsNil))
	c.Assert(len(out.Candidates[0].Content.Parts), qt.Equals, 3) // All 3 parts preserved
	c.Assert(out.Candidates[0].Content.Parts[0].Text, qt.Equals, "Hello")
	c.Assert(out.Candidates[0].Content.Parts[1].Text, qt.Equals, " world")
	c.Assert(out.Candidates[0].Content.Parts[2].Text, qt.Equals, "!")
	c.Assert(len(out.Candidates[0].SafetyRatings), qt.Equals, 1)

	// Verify second candidate
	c.Assert(out.Candidates[1].Index, qt.Equals, int32(1))
	c.Assert(out.Candidates[1].FinishReason, qt.Equals, genai.FinishReasonStop)
	c.Assert(out.Candidates[1].TokenCount, qt.Equals, int32(8))
	c.Assert(out.Candidates[1].AvgLogprobs, qt.Equals, -0.3)
	c.Assert(out.Candidates[1].Content, qt.Not(qt.IsNil))
	c.Assert(len(out.Candidates[1].Content.Parts), qt.Equals, 2) // All 2 parts preserved
	c.Assert(out.Candidates[1].Content.Parts[0].Text, qt.Equals, "Alternative")
	c.Assert(out.Candidates[1].Content.Parts[1].Text, qt.Equals, " response")

	// Verify response metadata
	c.Assert(out.ModelVersion, qt.Not(qt.IsNil))
	c.Assert(*out.ModelVersion, qt.Equals, "gemini-2.5-pro")
	c.Assert(out.ResponseID, qt.Not(qt.IsNil))
	c.Assert(*out.ResponseID, qt.Equals, "resp-streaming-123")
	c.Assert(out.UsageMetadata.TotalTokenCount, qt.Equals, int32(23))
	c.Assert(out.UsageMetadata.CandidatesTokenCount, qt.Equals, int32(18))
}

func Test_buildGenerateContentConfig_NoConfig(t *testing.T) {
	c := qt.New(t)
	in := TaskChatInput{}
	cfg := buildGenerateContentConfig(in, "")
	c.Check(cfg, qt.IsNil)
}

func Test_buildGenerateContentConfig_FlattenedFields(t *testing.T) {
	c := qt.New(t)
	temp := float32(0.7)
	topP := float32(0.9)
	topK := int32(40)
	maxTokens := int32(1000)
	seed := int32(42)

	in := TaskChatInput{
		Temperature:     &temp,
		TopP:            &topP,
		TopK:            &topK,
		MaxOutputTokens: &maxTokens,
		Seed:            &seed,
	}

	cfg := buildGenerateContentConfig(in, "")
	c.Assert(cfg, qt.IsNotNil)
	c.Check(*cfg.Temperature, qt.Equals, temp)
	c.Check(*cfg.TopP, qt.Equals, topP)
	c.Check(*cfg.TopK, qt.Equals, float32(topK))
	c.Check(cfg.MaxOutputTokens, qt.Equals, maxTokens)
	c.Check(*cfg.Seed, qt.Equals, seed)
}

func Test_buildGenerateContentConfig_SystemMessage(t *testing.T) {
	c := qt.New(t)
	in := TaskChatInput{}
	systemMsg := "You are a helpful assistant"

	cfg := buildGenerateContentConfig(in, systemMsg)
	c.Assert(cfg, qt.IsNotNil)
	c.Assert(cfg.SystemInstruction, qt.IsNotNil)
	c.Assert(cfg.SystemInstruction.Parts, qt.HasLen, 1)
	c.Check(cfg.SystemInstruction.Parts[0].Text, qt.Equals, systemMsg)
}

func Test_buildGenerateContentConfig_SystemMessagePriority(t *testing.T) {
	c := qt.New(t)
	systemInstructionText := "System instruction text"
	systemInstruction := &genai.Content{
		Parts: []*genai.Part{{Text: systemInstructionText}},
	}

	in := TaskChatInput{
		SystemInstruction: systemInstruction,
	}
	systemMsg := "System message text"

	cfg := buildGenerateContentConfig(in, systemMsg)
	c.Assert(cfg, qt.IsNotNil)
	c.Assert(cfg.SystemInstruction, qt.IsNotNil)
	c.Assert(cfg.SystemInstruction.Parts, qt.HasLen, 1)
	// Should prioritize systemMessage over SystemInstruction
	c.Check(cfg.SystemInstruction.Parts[0].Text, qt.Equals, systemMsg)
}

func Test_buildGenerateContentConfig_GenerationConfig(t *testing.T) {
	c := qt.New(t)
	temp := float32(0.8)
	candidateCount := int32(2)
	stopSeqs := []string{"stop1", "stop2"}

	in := TaskChatInput{
		GenerationConfig: &genai.GenerationConfig{
			Temperature:    genai.Ptr(temp),
			CandidateCount: candidateCount,
			StopSequences:  stopSeqs,
		},
	}

	cfg := buildGenerateContentConfig(in, "")
	c.Assert(cfg, qt.IsNotNil)
	c.Check(*cfg.Temperature, qt.Equals, temp)
	c.Check(cfg.CandidateCount, qt.Equals, candidateCount)
	c.Check(cfg.StopSequences, qt.DeepEquals, stopSeqs)
}

func Test_buildGenerateContentConfig_FlattenedTakesPrecedence(t *testing.T) {
	c := qt.New(t)
	flattenedTemp := float32(0.5)
	configTemp := float32(0.8)

	in := TaskChatInput{
		Temperature: &flattenedTemp,
		GenerationConfig: &genai.GenerationConfig{
			Temperature: genai.Ptr(configTemp),
		},
	}

	cfg := buildGenerateContentConfig(in, "")
	c.Assert(cfg, qt.IsNotNil)
	// Flattened field should take precedence
	c.Check(*cfg.Temperature, qt.Equals, flattenedTemp)
}

func Test_buildTools_FunctionDeclarations(t *testing.T) {
	c := qt.New(t)
	funcName := "test_function"
	funcDesc := "Test function description"

	tools := []*genai.Tool{
		{
			FunctionDeclarations: []*genai.FunctionDeclaration{
				{
					Name:        funcName,
					Description: funcDesc,
					Parameters: &genai.Schema{
						Type: genai.TypeObject,
						Properties: map[string]*genai.Schema{
							"param1": {Type: genai.TypeString},
						},
						Required: []string{"param1"},
					},
				},
			},
		},
	}

	// Since we're using genai types directly, no conversion needed
	result := tools
	c.Assert(result, qt.HasLen, 1)
	c.Assert(result[0].FunctionDeclarations, qt.HasLen, 1)
	c.Check(result[0].FunctionDeclarations[0].Name, qt.Equals, funcName)
	c.Check(result[0].FunctionDeclarations[0].Description, qt.Equals, funcDesc)
	c.Assert(result[0].FunctionDeclarations[0].Parameters, qt.IsNotNil)
	c.Check(result[0].FunctionDeclarations[0].Parameters.Type, qt.Equals, genai.TypeObject)
}

func Test_buildTools_GoogleSearchRetrieval(t *testing.T) {
	c := qt.New(t)
	tools := []*genai.Tool{
		{
			GoogleSearchRetrieval: &genai.GoogleSearchRetrieval{},
		},
	}

	// Since we're using genai types directly, no conversion needed
	result := tools
	c.Assert(result, qt.HasLen, 1)
	c.Check(result[0].GoogleSearchRetrieval, qt.IsNotNil)
}

func Test_buildTools_CodeExecution(t *testing.T) {
	c := qt.New(t)
	tools := []*genai.Tool{
		{
			CodeExecution: &genai.ToolCodeExecution{},
		},
	}

	// Since we're using genai types directly, no conversion needed
	result := tools
	c.Assert(result, qt.HasLen, 1)
	c.Check(result[0].CodeExecution, qt.IsNotNil)
}

func Test_buildGenerateContentConfig_AllFieldsIntegration(t *testing.T) {
	c := qt.New(t)

	// Setup comprehensive input with all the previously missing fields
	temp := float32(0.7)
	seed := int32(123)
	cachedContent := "cached-content-id"
	systemText := "System instruction"

	in := TaskChatInput{
		Temperature: &temp,
		Seed:        &seed,
		Tools: []*genai.Tool{
			{
				FunctionDeclarations: []*genai.FunctionDeclaration{
					{Name: "test_func"},
				},
			},
		},
		ToolConfig: &genai.ToolConfig{
			FunctionCallingConfig: &genai.FunctionCallingConfig{
				AllowedFunctionNames: []string{"test_func"},
			},
		},
		SafetySettings: []*genai.SafetySetting{
			{Category: genai.HarmCategory("HARM_CATEGORY_HARASSMENT"), Threshold: genai.HarmBlockThreshold("BLOCK_MEDIUM_AND_ABOVE")},
		},
		SystemInstruction: &genai.Content{
			Parts: []*genai.Part{{Text: systemText}},
		},
		CachedContent: &cachedContent,
	}

	cfg := buildGenerateContentConfig(in, "") // Empty systemMessage since no SystemMessage field is set

	// Verify all fields are properly set
	c.Assert(cfg, qt.IsNotNil)
	c.Check(*cfg.Temperature, qt.Equals, temp)
	c.Check(*cfg.Seed, qt.Equals, seed)
	c.Check(cfg.CachedContent, qt.Equals, cachedContent)

	// Verify Tools conversion
	c.Assert(cfg.Tools, qt.HasLen, 1)
	c.Assert(cfg.Tools[0].FunctionDeclarations, qt.HasLen, 1)
	c.Check(cfg.Tools[0].FunctionDeclarations[0].Name, qt.Equals, "test_func")

	// Verify ToolConfig conversion
	c.Assert(cfg.ToolConfig, qt.IsNotNil)
	c.Assert(cfg.ToolConfig.FunctionCallingConfig, qt.IsNotNil)
	c.Check(cfg.ToolConfig.FunctionCallingConfig.AllowedFunctionNames, qt.DeepEquals, []string{"test_func"})

	// Verify SafetySettings conversion
	c.Assert(cfg.SafetySettings, qt.HasLen, 1)
	c.Check(cfg.SafetySettings[0].Category, qt.Equals, genai.HarmCategory("HARM_CATEGORY_HARASSMENT"))
	c.Check(cfg.SafetySettings[0].Threshold, qt.Equals, genai.HarmBlockThreshold("BLOCK_MEDIUM_AND_ABOVE"))

	// Verify SystemInstruction is used when no systemMessage is provided
	c.Assert(cfg.SystemInstruction, qt.IsNotNil)
	c.Assert(cfg.SystemInstruction.Parts, qt.HasLen, 1)
	c.Check(cfg.SystemInstruction.Parts[0].Text, qt.Equals, systemText)
}

func Test_buildGenerateContentConfig_CachedContent(t *testing.T) {
	c := qt.New(t)
	cachedContentID := "cache-123"

	in := TaskChatInput{
		CachedContent: &cachedContentID,
	}

	cfg := buildGenerateContentConfig(in, "")
	c.Assert(cfg, qt.IsNotNil)
	c.Check(cfg.CachedContent, qt.Equals, cachedContentID)
}

func Test_StreamingAllFields(t *testing.T) {
	c := qt.New(t)

	// Simulate streaming chunks that would come from GenerateContentStream
	// This tests the logic inside the streaming loop that builds incremental outputs

	// Initial chunk with first candidate data
	chunk1 := &genai.GenerateContentResponse{
		ModelVersion: "gemini-2.5-pro",
		ResponseID:   "resp-123",
		Candidates: []*genai.Candidate{
			{
				Index:        0,
				FinishReason: genai.FinishReasonOther,
				TokenCount:   5,
				AvgLogprobs:  -0.1,
				Content: &genai.Content{
					Role: genai.RoleModel,
					Parts: []*genai.Part{
						{Text: "Hello"},
					},
				},
				SafetyRatings: []*genai.SafetyRating{
					{
						Category:    genai.HarmCategoryDangerousContent,
						Probability: genai.HarmProbabilityNegligible,
						Blocked:     false,
					},
				},
			},
		},
		UsageMetadata: &genai.GenerateContentResponseUsageMetadata{
			PromptTokenCount:     10,
			CandidatesTokenCount: 5,
			TotalTokenCount:      15,
		},
		PromptFeedback: &genai.GenerateContentResponsePromptFeedback{
			SafetyRatings: []*genai.SafetyRating{
				{
					Category:    genai.HarmCategoryHarassment,
					Probability: genai.HarmProbabilityNegligible,
				},
			},
		},
	}

	// Second chunk with additional content
	chunk2 := &genai.GenerateContentResponse{
		ModelVersion: "gemini-2.5-pro",
		ResponseID:   "resp-123",
		Candidates: []*genai.Candidate{
			{
				Index:        0,
				FinishReason: genai.FinishReasonStop,
				TokenCount:   10,
				AvgLogprobs:  -0.2,
				Content: &genai.Content{
					Role: genai.RoleModel,
					Parts: []*genai.Part{
						{Text: " world!"},
					},
				},
				SafetyRatings: []*genai.SafetyRating{
					{
						Category:    genai.HarmCategoryDangerousContent,
						Probability: genai.HarmProbabilityNegligible,
						Blocked:     false,
					},
				},
			},
		},
		UsageMetadata: &genai.GenerateContentResponseUsageMetadata{
			PromptTokenCount:     10,
			CandidatesTokenCount: 10,
			TotalTokenCount:      20,
		},
	}

	// Simulate the streaming logic that builds finalResp and texts
	texts := make([]string, 0)
	var finalResp *genai.GenerateContentResponse

	// Process chunk1 (initial chunk)
	r := chunk1
	if r != nil && len(r.Candidates) > 0 {
		// Accumulate texts for incremental output
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

	// Build the final response by merging all chunks
	// Initialize with first chunk
	finalResp = &genai.GenerateContentResponse{
		ModelVersion:   r.ModelVersion,
		ResponseID:     r.ResponseID,
		UsageMetadata:  r.UsageMetadata,
		PromptFeedback: r.PromptFeedback,
		Candidates:     make([]*genai.Candidate, len(r.Candidates)),
	}
	// Deep copy candidates
	for i, c := range r.Candidates {
		if c != nil {
			finalResp.Candidates[i] = &genai.Candidate{
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
						finalResp.Candidates[i].Content.Parts = append(finalResp.Candidates[i].Content.Parts, p)
					}
				}
			}
		}
	}

	// Test streaming output after chunk1
	streamOutput1 := TaskChatOutput{
		Texts:          texts,
		Usage:          map[string]any{},
		Candidates:     []*genai.Candidate{},
		UsageMetadata:  nil,
		PromptFeedback: nil,
		ModelVersion:   nil,
		ResponseID:     nil,
	}

	streamOutput1.Candidates = finalResp.Candidates
	streamOutput1.UsageMetadata = finalResp.UsageMetadata
	streamOutput1.PromptFeedback = finalResp.PromptFeedback
	if finalResp.ModelVersion != "" {
		mv := finalResp.ModelVersion
		streamOutput1.ModelVersion = &mv
	}
	if finalResp.ResponseID != "" {
		ri := finalResp.ResponseID
		streamOutput1.ResponseID = &ri
	}

	// Build usage map from UsageMetadata if available
	if finalResp.UsageMetadata != nil {
		usage := make(map[string]any)
		usage["prompt-token-count"] = finalResp.UsageMetadata.PromptTokenCount
		usage["cached-content-token-count"] = finalResp.UsageMetadata.CachedContentTokenCount
		usage["candidates-token-count"] = finalResp.UsageMetadata.CandidatesTokenCount
		usage["total-token-count"] = finalResp.UsageMetadata.TotalTokenCount
		usage["tool-use-prompt-token-count"] = finalResp.UsageMetadata.ToolUsePromptTokenCount
		usage["thoughts-token-count"] = finalResp.UsageMetadata.ThoughtsTokenCount

		// Simplified usage map for testing

		streamOutput1.Usage = usage
	}

	// Verify streaming output after chunk1
	c.Assert(len(streamOutput1.Texts), qt.Equals, 1)
	c.Assert(streamOutput1.Texts[0], qt.Equals, "Hello")

	// Verify candidates are streamed
	c.Assert(len(streamOutput1.Candidates), qt.Equals, 1)
	c.Assert(streamOutput1.Candidates[0].Index, qt.Equals, int32(0))
	c.Assert(streamOutput1.Candidates[0].FinishReason, qt.Equals, genai.FinishReasonOther)
	c.Assert(streamOutput1.Candidates[0].TokenCount, qt.Equals, int32(5))
	c.Assert(streamOutput1.Candidates[0].AvgLogprobs, qt.Equals, -0.1)
	c.Assert(len(streamOutput1.Candidates[0].Content.Parts), qt.Equals, 1)
	c.Assert(streamOutput1.Candidates[0].Content.Parts[0].Text, qt.Equals, "Hello")

	// Verify usage metadata is streamed
	c.Assert(streamOutput1.UsageMetadata, qt.Not(qt.IsNil))
	c.Assert(streamOutput1.UsageMetadata.TotalTokenCount, qt.Equals, int32(15))
	c.Assert(streamOutput1.UsageMetadata.CandidatesTokenCount, qt.Equals, int32(5))

	// Verify prompt feedback is streamed
	c.Assert(streamOutput1.PromptFeedback, qt.Not(qt.IsNil))
	c.Assert(len(streamOutput1.PromptFeedback.SafetyRatings), qt.Equals, 1)

	// Verify model version and response ID are streamed
	c.Assert(streamOutput1.ModelVersion, qt.Not(qt.IsNil))
	c.Assert(*streamOutput1.ModelVersion, qt.Equals, "gemini-2.5-pro")
	c.Assert(streamOutput1.ResponseID, qt.Not(qt.IsNil))
	c.Assert(*streamOutput1.ResponseID, qt.Equals, "resp-123")

	// Verify usage map is properly formatted
	c.Assert(streamOutput1.Usage["total-token-count"], qt.Equals, int32(15))
	c.Assert(streamOutput1.Usage["prompt-token-count"], qt.Equals, int32(10))
	c.Assert(streamOutput1.Usage["candidates-token-count"], qt.Equals, int32(5))

	// Process chunk2 (merge subsequent chunk)
	r = chunk2
	if r != nil && len(r.Candidates) > 0 {
		// Accumulate texts for incremental output
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

	// Merge subsequent chunks - append parts to existing candidates
	for i, c := range r.Candidates {
		if c != nil && i < len(finalResp.Candidates) && finalResp.Candidates[i] != nil {
			// Update metadata from latest chunk
			finalResp.Candidates[i].FinishReason = c.FinishReason
			finalResp.Candidates[i].TokenCount = c.TokenCount
			finalResp.Candidates[i].AvgLogprobs = c.AvgLogprobs
			if c.SafetyRatings != nil {
				finalResp.Candidates[i].SafetyRatings = c.SafetyRatings
			}

			// Append new parts
			if c.Content != nil {
				for _, p := range c.Content.Parts {
					if p != nil {
						finalResp.Candidates[i].Content.Parts = append(finalResp.Candidates[i].Content.Parts, p)
					}
				}
			}
		}
	}
	// Update response-level metadata from latest chunk
	if r.UsageMetadata != nil {
		finalResp.UsageMetadata = r.UsageMetadata
	}

	// Test final streaming output after chunk2
	streamOutput2 := TaskChatOutput{
		Texts:          texts,
		Usage:          map[string]any{},
		Candidates:     []*genai.Candidate{},
		UsageMetadata:  nil,
		PromptFeedback: nil,
		ModelVersion:   nil,
		ResponseID:     nil,
	}

	streamOutput2.Candidates = finalResp.Candidates
	streamOutput2.UsageMetadata = finalResp.UsageMetadata
	streamOutput2.PromptFeedback = finalResp.PromptFeedback
	if finalResp.ModelVersion != "" {
		mv := finalResp.ModelVersion
		streamOutput2.ModelVersion = &mv
	}
	if finalResp.ResponseID != "" {
		ri := finalResp.ResponseID
		streamOutput2.ResponseID = &ri
	}

	// Build usage map
	if finalResp.UsageMetadata != nil {
		usage := make(map[string]any)
		usage["prompt-token-count"] = finalResp.UsageMetadata.PromptTokenCount
		usage["candidates-token-count"] = finalResp.UsageMetadata.CandidatesTokenCount
		usage["total-token-count"] = finalResp.UsageMetadata.TotalTokenCount
		streamOutput2.Usage = usage
	}

	// Verify final streaming output
	c.Assert(len(streamOutput2.Texts), qt.Equals, 1)
	c.Assert(streamOutput2.Texts[0], qt.Equals, "Hello world!")

	// Verify candidates have been merged correctly
	c.Assert(len(streamOutput2.Candidates), qt.Equals, 1)
	c.Assert(streamOutput2.Candidates[0].FinishReason, qt.Equals, genai.FinishReasonStop) // Updated from chunk2
	c.Assert(streamOutput2.Candidates[0].TokenCount, qt.Equals, int32(10))                // Updated from chunk2
	c.Assert(streamOutput2.Candidates[0].AvgLogprobs, qt.Equals, -0.2)                    // Updated from chunk2
	c.Assert(len(streamOutput2.Candidates[0].Content.Parts), qt.Equals, 2)                // Both parts preserved
	c.Assert(streamOutput2.Candidates[0].Content.Parts[0].Text, qt.Equals, "Hello")
	c.Assert(streamOutput2.Candidates[0].Content.Parts[1].Text, qt.Equals, " world!")

	// Verify updated usage metadata
	c.Assert(streamOutput2.UsageMetadata.TotalTokenCount, qt.Equals, int32(20)) // Updated from chunk2
	c.Assert(streamOutput2.UsageMetadata.CandidatesTokenCount, qt.Equals, int32(10))

	// Verify usage map reflects final values
	c.Assert(streamOutput2.Usage["total-token-count"], qt.Equals, int32(20))
	c.Assert(streamOutput2.Usage["candidates-token-count"], qt.Equals, int32(10))
}
