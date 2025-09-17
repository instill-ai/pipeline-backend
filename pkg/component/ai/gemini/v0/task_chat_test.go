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

func Test_buildParts_TextAndInlineData(t *testing.T) {
	c := qt.New(t)
	hello := "hello"
	// simple inline bytes
	inline := base64.StdEncoding.EncodeToString([]byte{0x01, 0x02})
	ps := []part{
		{Text: &hello},
		{InlineData: &blob{MIMEType: "application/octet-stream", Data: inline}},
	}
	got := buildParts(ps)
	c.Assert(got, qt.HasLen, 2)
	c.Check(got[0].Text, qt.Equals, hello)
	c.Check(got[1].InlineData, qt.Not(qt.IsNil))
	c.Check(got[1].InlineData.MIMEType, qt.Equals, "application/octet-stream")
	c.Check(len(got[1].InlineData.Data) > 0, qt.IsTrue)
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
	// Expect 1 text + 1 image + 1 doc = 3 parts
	c.Assert(got, qt.HasLen, 3)
	c.Check(got[0].Text, qt.Equals, prompt)
	c.Check(got[1].InlineData, qt.Not(qt.IsNil))
	c.Check(got[1].InlineData.MIMEType, qt.Equals, "image/png")
	c.Check(got[2].InlineData, qt.Not(qt.IsNil))
	c.Check(got[2].InlineData.MIMEType, qt.Equals, "application/pdf")
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
	c.Assert(err.Error(), qt.Contains, "unsupported document MIME type: application/msword")
	c.Assert(err.Error(), qt.Contains, "Use \":pdf\" syntax")
	c.Assert(got, qt.IsNil)
}

func Test_buildReqParts_UnsupportedDocumentMIME_ConvertibleText(t *testing.T) {
	c := qt.New(t)
	prompt := "Summarize this."

	// Create a document with convertible text MIME type (CSV)
	docBytes := []byte("name,age\nJohn,30\nJane,25")
	doc, err := data.NewDocumentFromBytes(docBytes, data.CSV, "test.csv")
	if err != nil {
		t.Fatal(err)
	}

	in := TaskChatInput{
		Prompt:    &prompt,
		Documents: []format.Document{doc},
	}

	got, err := buildReqParts(in)
	c.Assert(err, qt.Not(qt.IsNil))
	c.Assert(err.Error(), qt.Contains, "unsupported document MIME type: text/csv")
	c.Assert(err.Error(), qt.Contains, "Use \":pdf\" syntax")
	c.Assert(got, qt.IsNil)
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
	systemInstruction := &content{
		Parts: []part{{Text: &systemInstructionText}},
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
		GenerationConfig: &generationConfig{
			Temperature:    &temp,
			CandidateCount: &candidateCount,
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
		GenerationConfig: &generationConfig{
			Temperature: &configTemp,
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

	tools := []tool{
		{
			FunctionDeclarations: []functionDeclaration{
				{
					Name:        funcName,
					Description: &funcDesc,
					Parameters: &jsonSchema{
						Type: "object",
						Properties: map[string]jsonSchema{
							"param1": {Type: "string"},
						},
					},
				},
			},
		},
	}

	result := buildTools(tools)
	c.Assert(result, qt.HasLen, 1)
	c.Assert(result[0].FunctionDeclarations, qt.HasLen, 1)
	c.Check(result[0].FunctionDeclarations[0].Name, qt.Equals, funcName)
	c.Check(result[0].FunctionDeclarations[0].Description, qt.Equals, funcDesc)
	c.Assert(result[0].FunctionDeclarations[0].Parameters, qt.IsNotNil)
	c.Check(result[0].FunctionDeclarations[0].Parameters.Type, qt.Equals, genai.Type("object"))
}

func Test_buildTools_GoogleSearchRetrieval(t *testing.T) {
	c := qt.New(t)
	tools := []tool{
		{
			GoogleSearchRetrieval: &googleSearchRetrieval{
				DynamicRetrievalConfig: &dynamicRetrievalConfig{},
			},
		},
	}

	result := buildTools(tools)
	c.Assert(result, qt.HasLen, 1)
	c.Check(result[0].GoogleSearchRetrieval, qt.IsNotNil)
}

func Test_buildTools_CodeExecution(t *testing.T) {
	c := qt.New(t)
	tools := []tool{
		{
			CodeExecution: &codeExecution{},
		},
	}

	result := buildTools(tools)
	c.Assert(result, qt.HasLen, 1)
	c.Check(result[0].CodeExecution, qt.IsNotNil)
}

func Test_buildToolConfig(t *testing.T) {
	c := qt.New(t)
	allowedFuncs := []string{"func1", "func2"}

	tc := &toolConfig{
		FunctionCallingConfig: &functionCallingConfig{
			AllowedFunctionNames: allowedFuncs,
		},
	}

	result := buildToolConfig(tc)
	c.Assert(result, qt.IsNotNil)
	c.Assert(result.FunctionCallingConfig, qt.IsNotNil)
	c.Check(result.FunctionCallingConfig.AllowedFunctionNames, qt.DeepEquals, allowedFuncs)
}

func Test_buildToolConfig_Nil(t *testing.T) {
	c := qt.New(t)
	result := buildToolConfig(nil)
	c.Check(result, qt.IsNil)
}

func Test_buildSafetySettings(t *testing.T) {
	c := qt.New(t)
	settings := []safetySetting{
		{Category: "HARM_CATEGORY_HARASSMENT", Threshold: "BLOCK_MEDIUM_AND_ABOVE"},
		{Category: "HARM_CATEGORY_HATE_SPEECH", Threshold: "BLOCK_ONLY_HIGH"},
	}

	result := buildSafetySettings(settings)
	c.Assert(result, qt.HasLen, 2)
	c.Check(result[0].Category, qt.Equals, genai.HarmCategory("HARM_CATEGORY_HARASSMENT"))
	c.Check(result[0].Threshold, qt.Equals, genai.HarmBlockThreshold("BLOCK_MEDIUM_AND_ABOVE"))
	c.Check(result[1].Category, qt.Equals, genai.HarmCategory("HARM_CATEGORY_HATE_SPEECH"))
	c.Check(result[1].Threshold, qt.Equals, genai.HarmBlockThreshold("BLOCK_ONLY_HIGH"))
}

func Test_buildContent(t *testing.T) {
	c := qt.New(t)
	text := "Hello world"
	role := "user"

	content := &content{
		Role:  &role,
		Parts: []part{{Text: &text}},
	}

	result := buildContent(content)
	c.Assert(result, qt.IsNotNil)
	c.Check(result.Role, qt.Equals, genai.RoleUser)
	c.Assert(result.Parts, qt.HasLen, 1)
	c.Check(result.Parts[0].Text, qt.Equals, text)
}

func Test_buildContent_ModelRole(t *testing.T) {
	c := qt.New(t)
	text := "I'm a model response"
	role := "model"

	content := &content{
		Role:  &role,
		Parts: []part{{Text: &text}},
	}

	result := buildContent(content)
	c.Assert(result, qt.IsNotNil)
	c.Check(result.Role, qt.Equals, genai.RoleModel)
}

func Test_buildContent_Nil(t *testing.T) {
	c := qt.New(t)
	result := buildContent(nil)
	c.Check(result, qt.IsNil)
}

func Test_buildContent_EmptyParts(t *testing.T) {
	c := qt.New(t)
	content := &content{Parts: []part{}}
	result := buildContent(content)
	c.Check(result, qt.IsNil)
}

func Test_buildSchema_Basic(t *testing.T) {
	c := qt.New(t)
	schema := &jsonSchema{
		Type:        "object",
		Title:       "Test Schema",
		Description: "A test schema",
		Required:    []string{"field1"},
	}

	result := buildSchema(schema)
	c.Assert(result, qt.IsNotNil)
	c.Check(result.Type, qt.Equals, genai.Type("object"))
	c.Check(result.Title, qt.Equals, "Test Schema")
	c.Check(result.Description, qt.Equals, "A test schema")
	c.Check(result.Required, qt.DeepEquals, []string{"field1"})
}

func Test_buildSchema_WithProperties(t *testing.T) {
	c := qt.New(t)
	schema := &jsonSchema{
		Type: "object",
		Properties: map[string]jsonSchema{
			"name": {Type: "string"},
			"age":  {Type: "integer"},
		},
	}

	result := buildSchema(schema)
	c.Assert(result, qt.IsNotNil)
	c.Assert(result.Properties, qt.HasLen, 2)
	c.Check(result.Properties["name"].Type, qt.Equals, genai.Type("string"))
	c.Check(result.Properties["age"].Type, qt.Equals, genai.Type("integer"))
}

func Test_buildSchema_WithItems(t *testing.T) {
	c := qt.New(t)
	schema := &jsonSchema{
		Type:  "array",
		Items: &jsonSchema{Type: "string"},
	}

	result := buildSchema(schema)
	c.Assert(result, qt.IsNotNil)
	c.Check(result.Type, qt.Equals, genai.Type("array"))
	c.Assert(result.Items, qt.IsNotNil)
	c.Check(result.Items.Type, qt.Equals, genai.Type("string"))
}

func Test_buildSchema_WithConstraints(t *testing.T) {
	c := qt.New(t)
	maxItems := int32(10)
	minLength := int32(5)

	schema := &jsonSchema{
		Type:      "array",
		MaxItems:  &maxItems,
		MinLength: &minLength,
	}

	result := buildSchema(schema)
	c.Assert(result, qt.IsNotNil)
	c.Assert(result.MaxItems, qt.IsNotNil)
	c.Check(*result.MaxItems, qt.Equals, int64(10))
	c.Assert(result.MinLength, qt.IsNotNil)
	c.Check(*result.MinLength, qt.Equals, int64(5))
}

func Test_buildSchema_Nil(t *testing.T) {
	c := qt.New(t)
	result := buildSchema(nil)
	c.Check(result, qt.IsNil)
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
		Tools: []tool{
			{
				FunctionDeclarations: []functionDeclaration{
					{Name: "test_func", Description: nil},
				},
			},
		},
		ToolConfig: &toolConfig{
			FunctionCallingConfig: &functionCallingConfig{
				AllowedFunctionNames: []string{"test_func"},
			},
		},
		SafetySettings: []safetySetting{
			{Category: "HARM_CATEGORY_HARASSMENT", Threshold: "BLOCK_MEDIUM_AND_ABOVE"},
		},
		SystemInstruction: &content{
			Parts: []part{{Text: &systemText}},
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
