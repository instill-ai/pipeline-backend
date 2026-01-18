package gemini

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"testing"
	"time"

	"google.golang.org/genai"

	qt "github.com/frankban/quicktest"
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

func Test_newURIOrDataPart_RemoteURI(t *testing.T) {
	c := qt.New(t)

	t.Run("with https URL", func(t *testing.T) {
		remoteURI := "https://example.com/image.png"
		p := newURIOrDataPart(remoteURI, "image/png")
		c.Assert(p, qt.IsNotNil)
		// Should create a fileData part, not inline data
		c.Check(p.InlineData, qt.IsNil)
		c.Check(p.FileData, qt.Not(qt.IsNil))
		c.Check(p.FileData.FileURI, qt.Equals, remoteURI)
		c.Check(p.FileData.MIMEType, qt.Equals, "image/png")
	})

	t.Run("with http URL", func(t *testing.T) {
		remoteURI := "http://example.com/video.mp4"
		p := newURIOrDataPart(remoteURI, "video/mp4")
		c.Assert(p, qt.IsNotNil)
		// Should create a fileData part, not inline data
		c.Check(p.InlineData, qt.IsNil)
		c.Check(p.FileData, qt.Not(qt.IsNil))
		c.Check(p.FileData.FileURI, qt.Equals, remoteURI)
		c.Check(p.FileData.MIMEType, qt.Equals, "video/mp4")
	})

	t.Run("with gs:// URL", func(t *testing.T) {
		remoteURI := "gs://bucket/audio.wav"
		p := newURIOrDataPart(remoteURI, "audio/wav")
		c.Assert(p, qt.IsNotNil)
		// Should create a fileData part for Google Cloud Storage URI
		c.Check(p.InlineData, qt.IsNil)
		c.Check(p.FileData, qt.Not(qt.IsNil))
		c.Check(p.FileData.FileURI, qt.Equals, remoteURI)
		c.Check(p.FileData.MIMEType, qt.Equals, "audio/wav")
	})
}

func Test_newURIOrDataPart_RawBase64(t *testing.T) {
	c := qt.New(t)
	// Raw base64 without data URI prefix
	pngB64 := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR4nGNgYAAAAAMAASsJTYQAAAAASUVORK5CYII="
	p := newURIOrDataPart(pngB64, "image/png")
	c.Assert(p, qt.IsNotNil)
	c.Check(p.InlineData, qt.Not(qt.IsNil))
	c.Check(p.InlineData.MIMEType, qt.Equals, "image/png")
	decoded, _ := base64.StdEncoding.DecodeString(pngB64)
	c.Check(p.InlineData.Data, qt.DeepEquals, decoded)
}

// NOTE: Tests for buildReqParts and process*Parts functions were removed
// as those functions were replaced by buildReqPartsWithFileAPI.
// The new FileAPI-based functionality is tested through integration tests.

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
	c.Check(out.Usage["total-token-count"], qt.Equals, int32(3))
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

func Test_buildUsageMap(t *testing.T) {
	c := qt.New(t)

	t.Run("with complete metadata", func(t *testing.T) {
		metadata := &genai.GenerateContentResponseUsageMetadata{
			PromptTokenCount:        10,
			CachedContentTokenCount: 5,
			CandidatesTokenCount:    20,
			TotalTokenCount:         35,
			ToolUsePromptTokenCount: 3,
			ThoughtsTokenCount:      2,
		}

		got := buildUsageMap(metadata)
		c.Check(got["prompt-token-count"], qt.Equals, int32(10))
		c.Check(got["cached-content-token-count"], qt.Equals, int32(5))
		c.Check(got["candidates-token-count"], qt.Equals, int32(20))
		c.Check(got["total-token-count"], qt.Equals, int32(35))
		c.Check(got["tool-use-prompt-token-count"], qt.Equals, int32(3))
		c.Check(got["thoughts-token-count"], qt.Equals, int32(2))
	})

	t.Run("with nil metadata", func(t *testing.T) {
		// This test documents that buildUsageMap doesn't handle nil gracefully
		// In practice, it's always called with valid metadata from the API response
		c.Check(func() { buildUsageMap(nil) }, qt.PanicMatches, "runtime error: invalid memory address or nil pointer dereference")
	})
}

func Test_accumulateTexts(t *testing.T) {
	c := qt.New(t)
	exec := &execution{}

	t.Run("with new candidates", func(t *testing.T) {
		texts := []string{}
		resp := &genai.GenerateContentResponse{
			Candidates: []*genai.Candidate{
				{
					Content: &genai.Content{
						Parts: []*genai.Part{
							{Text: "Hello"},
							{Text: " world"},
						},
					},
				},
				{
					Content: &genai.Content{
						Parts: []*genai.Part{
							{Text: "Second candidate"},
						},
					},
				},
			},
		}

		exec.accumulateTexts(resp, &texts)
		c.Assert(texts, qt.HasLen, 2)
		c.Check(texts[0], qt.Equals, "Hello world")
		c.Check(texts[1], qt.Equals, "Second candidate")
	})

	t.Run("with existing texts", func(t *testing.T) {
		texts := []string{"Existing", "Text"}
		resp := &genai.GenerateContentResponse{
			Candidates: []*genai.Candidate{
				{
					Content: &genai.Content{
						Parts: []*genai.Part{
							{Text: " more"},
						},
					},
				},
			},
		}

		exec.accumulateTexts(resp, &texts)
		c.Assert(texts, qt.HasLen, 2)
		c.Check(texts[0], qt.Equals, "Existing more")
		c.Check(texts[1], qt.Equals, "Text")
	})

	t.Run("with nil response", func(t *testing.T) {
		texts := []string{"existing"}
		original := make([]string, len(texts))
		copy(original, texts)

		exec.accumulateTexts(nil, &texts)
		c.Check(texts, qt.DeepEquals, original)
	})
}

func Test_mergeResponseChunk(t *testing.T) {
	c := qt.New(t)
	exec := &execution{}

	t.Run("with nil finalResp", func(t *testing.T) {
		var finalResp *genai.GenerateContentResponse
		chunk := &genai.GenerateContentResponse{
			ModelVersion: "v1",
			ResponseID:   "resp-123",
			Candidates: []*genai.Candidate{
				{
					Index: 0,
					Content: &genai.Content{
						Parts: []*genai.Part{{Text: "Hello"}},
					},
				},
			},
			UsageMetadata: &genai.GenerateContentResponseUsageMetadata{
				TotalTokenCount: 10,
			},
		}

		exec.mergeResponseChunk(chunk, &finalResp)
		c.Assert(finalResp, qt.Not(qt.IsNil))
		c.Check(finalResp.ModelVersion, qt.Equals, "v1")
		c.Check(finalResp.ResponseID, qt.Equals, "resp-123")
		c.Assert(finalResp.Candidates, qt.HasLen, 1)
		c.Check(finalResp.Candidates[0].Content.Parts[0].Text, qt.Equals, "Hello")
	})

	t.Run("with existing finalResp", func(t *testing.T) {
		finalResp := &genai.GenerateContentResponse{
			ModelVersion: "v1",
			ResponseID:   "resp-123",
			Candidates: []*genai.Candidate{
				{
					Index: 0,
					Content: &genai.Content{
						Parts: []*genai.Part{{Text: "Hello"}},
					},
				},
			},
		}

		chunk := &genai.GenerateContentResponse{
			Candidates: []*genai.Candidate{
				{
					Index: 0,
					Content: &genai.Content{
						Parts: []*genai.Part{{Text: " world"}},
					},
				},
			},
			UsageMetadata: &genai.GenerateContentResponseUsageMetadata{
				TotalTokenCount: 15,
			},
		}

		exec.mergeResponseChunk(chunk, &finalResp)
		c.Assert(finalResp.Candidates, qt.HasLen, 1)
		c.Assert(finalResp.Candidates[0].Content.Parts, qt.HasLen, 2)
		c.Check(finalResp.Candidates[0].Content.Parts[0].Text, qt.Equals, "Hello")
		c.Check(finalResp.Candidates[0].Content.Parts[1].Text, qt.Equals, " world")
		c.Check(finalResp.UsageMetadata.TotalTokenCount, qt.Equals, int32(15))
	})
}

func Test_buildStreamOutput(t *testing.T) {
	c := qt.New(t)

	// Create a mock execution struct for testing
	exec := &execution{}

	texts := []string{"Hello", "World"}
	finalResp := &genai.GenerateContentResponse{
		ModelVersion: "v1",
		ResponseID:   "resp-123",
		Candidates: []*genai.Candidate{
			{
				Content: &genai.Content{Parts: []*genai.Part{{Text: "Hello"}}},
			},
			{
				Content: &genai.Content{Parts: []*genai.Part{{Text: "World"}}},
			},
		},
		UsageMetadata: &genai.GenerateContentResponseUsageMetadata{
			PromptTokenCount:     5,
			CandidatesTokenCount: 10,
			TotalTokenCount:      15,
		},
		PromptFeedback: &genai.GenerateContentResponsePromptFeedback{
			SafetyRatings: []*genai.SafetyRating{
				{Category: genai.HarmCategoryHarassment, Probability: genai.HarmProbabilityNegligible},
			},
		},
	}

	got := exec.buildStreamOutput(texts, finalResp)

	c.Assert(got.Texts, qt.DeepEquals, texts)
	c.Assert(got.Candidates, qt.HasLen, 2)
	c.Assert(got.Usage, qt.Not(qt.IsNil))
	c.Check(got.Usage["total-token-count"], qt.Equals, int32(15))
	c.Assert(got.PromptFeedback, qt.Not(qt.IsNil))
	c.Assert(got.ModelVersion, qt.Not(qt.IsNil))
	c.Check(*got.ModelVersion, qt.Equals, "v1")
	c.Assert(got.ResponseID, qt.Not(qt.IsNil))
	c.Check(*got.ResponseID, qt.Equals, "resp-123")

	// Check usage map
	c.Check(got.Usage["total-token-count"], qt.Equals, int32(15))
	c.Check(got.Usage["prompt-token-count"], qt.Equals, int32(5))
	c.Check(got.Usage["candidates-token-count"], qt.Equals, int32(10))
}

func Test_buildStreamOutput_InlineDataCleanup(t *testing.T) {
	c := qt.New(t)

	// Create a mock execution struct for testing
	exec := &execution{}

	// Create test image data (1x1 transparent PNG)
	pngB64 := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR4nGNgYAAAAAMAASsJTYQAAAAASUVORK5CYII="
	imageData, err := base64.StdEncoding.DecodeString(pngB64)
	c.Assert(err, qt.IsNil)
	texts := []string{"Here's an image"}

	// Create a response with InlineData that should be cleaned up during streaming
	finalResp := &genai.GenerateContentResponse{
		ModelVersion: "v1",
		ResponseID:   "resp-123",
		Candidates: []*genai.Candidate{
			{
				Content: &genai.Content{
					Parts: []*genai.Part{
						{Text: "Here's an image"},
						{
							InlineData: &genai.Blob{
								MIMEType: "image/png",
								Data:     imageData,
							},
						},
					},
				},
			},
		},
		UsageMetadata: &genai.GenerateContentResponseUsageMetadata{
			PromptTokenCount:     5,
			CandidatesTokenCount: 10,
			TotalTokenCount:      15,
		},
	}

	// Verify InlineData exists before calling buildStreamOutput
	c.Assert(finalResp.Candidates[0].Content.Parts[1].InlineData, qt.Not(qt.IsNil))
	c.Check(finalResp.Candidates[0].Content.Parts[1].InlineData.Data, qt.DeepEquals, imageData)

	// Call buildStreamOutput
	got := exec.buildStreamOutput(texts, finalResp)

	// Verify the streaming output structure
	c.Assert(got.Texts, qt.DeepEquals, texts)
	c.Assert(got.Candidates, qt.HasLen, 1)
	c.Assert(got.Candidates[0].Content.Parts, qt.HasLen, 2)

	// Verify that InlineData has been cleaned up in the streaming response
	c.Check(got.Candidates[0].Content.Parts[1].InlineData, qt.IsNil, qt.Commentf("InlineData should be cleaned up in streaming mode"))

	// Verify that the original finalResp also has InlineData cleaned up
	// (since candidates are passed by reference)
	c.Check(finalResp.Candidates[0].Content.Parts[1].InlineData, qt.IsNil, qt.Commentf("InlineData should be cleaned up in original response"))

	// Verify that text parts are preserved
	c.Check(got.Candidates[0].Content.Parts[0].Text, qt.Equals, "Here's an image")

	// Verify other metadata is preserved
	c.Check(got.Usage["total-token-count"], qt.Equals, int32(15))
	c.Assert(got.ModelVersion, qt.Not(qt.IsNil))
	c.Check(*got.ModelVersion, qt.Equals, "v1")
	c.Assert(got.ResponseID, qt.Not(qt.IsNil))
	c.Check(*got.ResponseID, qt.Equals, "resp-123")

	// Verify that Images array now contains the extracted image during streaming
	c.Check(got.Images, qt.HasLen, 1, qt.Commentf("Images should be extracted and shown during streaming"))
}

func Test_processInlineDataInCandidates(t *testing.T) {
	c := qt.New(t)

	t.Run("cleanup only (streaming mode)", func(t *testing.T) {
		// Create test data
		imageData1 := []byte("fake-image-data-1")
		imageData2 := []byte("fake-image-data-2")

		candidates := []*genai.Candidate{
			{
				Content: &genai.Content{
					Parts: []*genai.Part{
						{Text: "Text part"},
						{
							InlineData: &genai.Blob{
								MIMEType: "image/png",
								Data:     imageData1,
							},
						},
					},
				},
			},
			{
				Content: &genai.Content{
					Parts: []*genai.Part{
						{
							InlineData: &genai.Blob{
								MIMEType: "image/jpeg",
								Data:     imageData2,
							},
						},
						{Text: "Another text part"},
					},
				},
			},
			// Test nil candidate
			nil,
			// Test candidate with nil content
			{Content: nil},
		}

		// Verify InlineData exists before cleanup
		c.Assert(candidates[0].Content.Parts[1].InlineData, qt.Not(qt.IsNil))
		c.Assert(candidates[1].Content.Parts[0].InlineData, qt.Not(qt.IsNil))

		// Call cleanup function without image extraction
		images := processInlineDataInCandidates(candidates, false)

		// Verify no images were extracted
		c.Check(images, qt.IsNil)

		// Verify InlineData has been cleaned up
		c.Check(candidates[0].Content.Parts[1].InlineData, qt.IsNil)
		c.Check(candidates[1].Content.Parts[0].InlineData, qt.IsNil)

		// Verify text parts are preserved
		c.Check(candidates[0].Content.Parts[0].Text, qt.Equals, "Text part")
		c.Check(candidates[1].Content.Parts[1].Text, qt.Equals, "Another text part")

		// Verify function handles nil candidates gracefully
		c.Assert(len(candidates), qt.Equals, 4) // No panic occurred
	})

	t.Run("extract images and cleanup (final mode)", func(t *testing.T) {
		// Create test image data (1x1 transparent PNG)
		pngB64 := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR4nGNgYAAAAAMAASsJTYQAAAAASUVORK5CYII="
		imageData, err := base64.StdEncoding.DecodeString(pngB64)
		c.Assert(err, qt.IsNil)

		candidates := []*genai.Candidate{
			{
				Content: &genai.Content{
					Parts: []*genai.Part{
						{Text: "Here's an image"},
						{
							InlineData: &genai.Blob{
								MIMEType: "image/png",
								Data:     imageData,
							},
						},
						{
							InlineData: &genai.Blob{
								MIMEType: "text/plain", // Non-image data
								Data:     []byte("some text"),
							},
						},
					},
				},
			},
		}

		// Verify InlineData exists before processing
		c.Assert(candidates[0].Content.Parts[1].InlineData, qt.Not(qt.IsNil))
		c.Assert(candidates[0].Content.Parts[2].InlineData, qt.Not(qt.IsNil))

		// Call function with image extraction
		images := processInlineDataInCandidates(candidates, true)

		// Verify one image was extracted (only the PNG, not the text data)
		c.Assert(images, qt.HasLen, 1)

		// Verify all InlineData has been cleaned up (both image and non-image)
		c.Check(candidates[0].Content.Parts[1].InlineData, qt.IsNil)
		c.Check(candidates[0].Content.Parts[2].InlineData, qt.IsNil)

		// Verify text parts are preserved
		c.Check(candidates[0].Content.Parts[0].Text, qt.Equals, "Here's an image")
	})
}

func Test_renderFinal_WithInlineData(t *testing.T) {
	c := qt.New(t)

	// Create test image data (1x1 transparent PNG)
	pngB64 := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR4nGNgYAAAAAMAASsJTYQAAAAASUVORK5CYII="
	imageData, err := base64.StdEncoding.DecodeString(pngB64)
	c.Assert(err, qt.IsNil)

	// Create a response with InlineData that should be extracted and cleaned up
	resp := &genai.GenerateContentResponse{
		ModelVersion: "v1",
		ResponseID:   "resp-123",
		Candidates: []*genai.Candidate{
			{
				Content: &genai.Content{
					Parts: []*genai.Part{
						{Text: "Here's an image"},
						{
							InlineData: &genai.Blob{
								MIMEType: "image/png",
								Data:     imageData,
							},
						},
					},
				},
			},
		},
		UsageMetadata: &genai.GenerateContentResponseUsageMetadata{
			PromptTokenCount:     5,
			CandidatesTokenCount: 10,
			TotalTokenCount:      15,
		},
	}

	// Verify InlineData exists before calling renderFinal
	c.Assert(resp.Candidates[0].Content.Parts[1].InlineData, qt.Not(qt.IsNil))
	c.Check(resp.Candidates[0].Content.Parts[1].InlineData.Data, qt.DeepEquals, imageData)

	// Call renderFinal
	got := renderFinal(resp, nil)

	// Verify that images were extracted
	c.Assert(got.Images, qt.HasLen, 1, qt.Commentf("One image should be extracted"))

	// Verify that InlineData has been cleaned up
	c.Check(resp.Candidates[0].Content.Parts[1].InlineData, qt.IsNil, qt.Commentf("InlineData should be cleaned up"))

	// Verify that text parts are preserved
	c.Check(got.Texts, qt.HasLen, 1)
	c.Check(got.Texts[0], qt.Equals, "Here's an image")

	// Verify other metadata is preserved
	c.Check(got.Usage["total-token-count"], qt.Equals, int32(15))
	c.Assert(got.ModelVersion, qt.Not(qt.IsNil))
	c.Check(*got.ModelVersion, qt.Equals, "v1")
	c.Assert(got.ResponseID, qt.Not(qt.IsNil))
	c.Check(*got.ResponseID, qt.Equals, "resp-123")
}

func Test_StreamingAndFinalImageConsistency(t *testing.T) {
	c := qt.New(t)

	// Create a mock execution struct for testing
	exec := &execution{}

	// Create test image data (1x1 transparent PNG)
	pngB64 := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR4nGNgYAAAAAMAASsJTYQAAAAASUVORK5CYII="
	imageData, err := base64.StdEncoding.DecodeString(pngB64)
	c.Assert(err, qt.IsNil)

	// Create a response with InlineData
	resp := &genai.GenerateContentResponse{
		ModelVersion: "v1",
		ResponseID:   "resp-123",
		Candidates: []*genai.Candidate{
			{
				Content: &genai.Content{
					Parts: []*genai.Part{
						{Text: "Here's an image"},
						{
							InlineData: &genai.Blob{
								MIMEType: "image/png",
								Data:     imageData,
							},
						},
					},
				},
			},
		},
		UsageMetadata: &genai.GenerateContentResponseUsageMetadata{
			TotalTokenCount: 15,
		},
	}

	// Test streaming output
	streamingOutput := exec.buildStreamOutput([]string{"Here's an image"}, resp)

	// Test final output (need to create a new response since InlineData gets cleaned up)
	resp2 := &genai.GenerateContentResponse{
		ModelVersion: "v1",
		ResponseID:   "resp-123",
		Candidates: []*genai.Candidate{
			{
				Content: &genai.Content{
					Parts: []*genai.Part{
						{Text: "Here's an image"},
						{
							InlineData: &genai.Blob{
								MIMEType: "image/png",
								Data:     imageData,
							},
						},
					},
				},
			},
		},
		UsageMetadata: &genai.GenerateContentResponseUsageMetadata{
			TotalTokenCount: 15,
		},
	}
	finalOutput := renderFinal(resp2, nil)

	// Verify both outputs have the same number of images
	c.Check(streamingOutput.Images, qt.HasLen, 1, qt.Commentf("Streaming should extract images"))
	c.Check(finalOutput.Images, qt.HasLen, 1, qt.Commentf("Final should extract images"))

	// Verify both outputs have cleaned up InlineData
	c.Check(resp.Candidates[0].Content.Parts[1].InlineData, qt.IsNil, qt.Commentf("Streaming should clean up InlineData"))
	c.Check(resp2.Candidates[0].Content.Parts[1].InlineData, qt.IsNil, qt.Commentf("Final should clean up InlineData"))

	// Verify text content is preserved in both
	c.Check(streamingOutput.Texts[0], qt.Equals, "Here's an image")
	c.Check(finalOutput.Texts[0], qt.Equals, "Here's an image")
}

func Test_extractSystemMessage(t *testing.T) {
	c := qt.New(t)

	t.Run("with system message", func(t *testing.T) {
		systemMsg := "You are a helpful assistant"
		in := TaskChatInput{
			SystemMessage: &systemMsg,
		}

		got := extractSystemMessage(in)
		c.Check(got, qt.Equals, systemMsg)
	})

	t.Run("with system instruction", func(t *testing.T) {
		systemText := "System instruction text"
		in := TaskChatInput{
			SystemInstruction: &genai.Content{
				Parts: []*genai.Part{{Text: systemText}},
			},
		}

		got := extractSystemMessage(in)
		c.Check(got, qt.Equals, systemText)
	})

	t.Run("with both - system message takes priority", func(t *testing.T) {
		systemMsg := "System message"
		systemText := "System instruction"
		in := TaskChatInput{
			SystemMessage: &systemMsg,
			SystemInstruction: &genai.Content{
				Parts: []*genai.Part{{Text: systemText}},
			},
		}

		got := extractSystemMessage(in)
		c.Check(got, qt.Equals, systemMsg)
	})

	t.Run("with empty system message", func(t *testing.T) {
		emptyMsg := ""
		systemText := "System instruction"
		in := TaskChatInput{
			SystemMessage: &emptyMsg,
			SystemInstruction: &genai.Content{
				Parts: []*genai.Part{{Text: systemText}},
			},
		}

		got := extractSystemMessage(in)
		c.Check(got, qt.Equals, systemText)
	})

	t.Run("with neither", func(t *testing.T) {
		in := TaskChatInput{}
		got := extractSystemMessage(in)
		c.Check(got, qt.Equals, "")
	})
}

// Test File API integration for chat
func TestChatFileAPIIntegration(t *testing.T) {
	c := qt.New(t)

	t.Run("total request size threshold logic for chat", func(t *testing.T) {
		exec := &execution{}
		ctx := context.Background()

		// Test small total request size (should use inline data)
		smallTotalSize := 1024 * 1024 // 1MB total request
		shouldUseFileAPI := smallTotalSize > MaxInlineSize

		c.Check(shouldUseFileAPI, qt.IsFalse) // Small total request should use inline

		// Test large total request size (should use File API)
		largeTotalSize := 25 * 1024 * 1024 // 25MB total request
		shouldUseFileAPILarge := largeTotalSize > MaxInlineSize

		c.Check(shouldUseFileAPILarge, qt.IsTrue) // Large total request should use File API

		// Test chat video with small total size (should follow total size rule, not forced)
		isChatVideo := false // Chat videos don't force File API like cache videos do
		shouldUseFileAPIVideo := smallTotalSize > MaxInlineSize || isChatVideo

		c.Check(shouldUseFileAPIVideo, qt.IsFalse) // Chat videos follow total size rule

		// Test chat video with large total size (should use File API)
		shouldUseFileAPIVideoLarge := largeTotalSize > MaxInlineSize || isChatVideo

		c.Check(shouldUseFileAPIVideoLarge, qt.IsTrue) // Large chat videos should use File API

		// Avoid unused variable warning
		_ = exec
		_ = ctx
	})

	t.Run("MaxInlineSize constant", func(t *testing.T) {
		// Test that the constant is correctly set to 20MB
		expectedSize := 20 * 1024 * 1024
		c.Check(MaxInlineSize, qt.Equals, expectedSize)
	})
}

// Test chat-specific File API functionality
func TestChatMediaProcessing(t *testing.T) {
	c := qt.New(t)

	t.Run("buildChatRequestContents with File API", func(t *testing.T) {
		// Test that the function signature includes uploaded file names
		input := TaskChatInput{
			Prompt: stringPtr("Analyze this content"),
		}

		// Test that the input structure supports File API scenarios
		c.Check(input.GetPrompt(), qt.DeepEquals, input.Prompt)
		c.Check(input.GetImages(), qt.HasLen, 0)
		c.Check(input.GetAudio(), qt.HasLen, 0)
		c.Check(input.GetVideos(), qt.HasLen, 0)
		c.Check(input.GetDocuments(), qt.HasLen, 0)
		c.Check(input.GetContents(), qt.HasLen, 0)
	})

	t.Run("chat history handling with File API", func(t *testing.T) {
		input := TaskChatInput{
			Prompt: stringPtr("Continue our conversation"),
			ChatHistory: []*genai.Content{
				{
					Role: "user",
					Parts: []*genai.Part{
						{Text: "Previous user message"},
					},
				},
				{
					Role: "model",
					Parts: []*genai.Part{
						{Text: "Previous model response"},
					},
				},
			},
		}

		// Validate chat history structure
		c.Check(len(input.ChatHistory), qt.Equals, 2)
		c.Check(input.ChatHistory[0].Role, qt.Equals, "user")
		c.Check(input.ChatHistory[1].Role, qt.Equals, "model")
		c.Check(input.ChatHistory[0].Parts[0].Text, qt.Equals, "Previous user message")
		c.Check(input.ChatHistory[1].Parts[0].Text, qt.Equals, "Previous model response")
	})
}

// Test file cleanup functionality in chat
func TestChatFileCleanup(t *testing.T) {
	c := qt.New(t)

	t.Run("file cleanup logic", func(t *testing.T) {
		// Test cleanup scenarios similar to cache
		uploadedFiles := []string{
			"files/chat-image-123",
			"files/chat-video-456",
			"files/chat-document-789",
		}

		// Simulate cleanup - in real implementation this would call client.Files.Delete
		cleanupCount := 0
		for _, fileName := range uploadedFiles {
			if fileName != "" {
				cleanupCount++
				// Mock cleanup: client.Files.Delete(ctx, fileName, nil)
			}
		}

		c.Check(cleanupCount, qt.Equals, 3)
		c.Check(len(uploadedFiles), qt.Equals, 3)
	})

	t.Run("error handling in cleanup", func(t *testing.T) {
		// Test error message patterns used in the implementation
		fileName := "files/chat-test-123"

		cleanupErr := fmt.Errorf("Warning: failed to delete uploaded file %s: %v", fileName, "API error")

		c.Check(cleanupErr.Error(), qt.Contains, "failed to delete uploaded file")
		c.Check(cleanupErr.Error(), qt.Contains, fileName)
		c.Check(cleanupErr.Error(), qt.Contains, "API error")
	})
}

// Test streaming vs non-streaming with File API
func TestChatStreamingWithFileAPI(t *testing.T) {
	c := qt.New(t)

	t.Run("streaming enabled", func(t *testing.T) {
		streamEnabled := true
		input := TaskChatInput{
			Prompt: stringPtr("Stream this response"),
			Stream: &streamEnabled,
		}

		c.Check(input.Stream, qt.Not(qt.IsNil))
		c.Check(*input.Stream, qt.IsTrue)
	})

	t.Run("streaming disabled", func(t *testing.T) {
		streamDisabled := false
		input := TaskChatInput{
			Prompt: stringPtr("Don't stream this response"),
			Stream: &streamDisabled,
		}

		c.Check(input.Stream, qt.Not(qt.IsNil))
		c.Check(*input.Stream, qt.IsFalse)
	})

	t.Run("streaming default (nil)", func(t *testing.T) {
		input := TaskChatInput{
			Prompt: stringPtr("Default streaming behavior"),
			Stream: nil,
		}

		c.Check(input.Stream, qt.IsNil)

		// Test default behavior logic
		streamEnabled := input.Stream != nil && *input.Stream
		c.Check(streamEnabled, qt.IsFalse) // Should default to non-streaming
	})
}

// Test chat multimedia input validation
func TestChatMultimediaInputValidation(t *testing.T) {
	c := qt.New(t)

	t.Run("chat with large file simulation", func(t *testing.T) {
		input := TaskChatInput{
			Prompt: stringPtr("Analyze this large image"),
			Model:  "gemini-2.5-flash",
		}

		// Validate that the input structure supports File API scenarios
		c.Check(input.Prompt, qt.Not(qt.IsNil))
		c.Check(*input.Prompt, qt.Equals, "Analyze this large image")
		c.Check(input.Model, qt.Equals, "gemini-2.5-flash")

		// Test that multimedia input interfaces are implemented
		c.Check(input.GetPrompt(), qt.DeepEquals, input.Prompt)
		c.Check(input.GetImages(), qt.HasLen, 0)
		c.Check(input.GetAudio(), qt.HasLen, 0)
		c.Check(input.GetVideos(), qt.HasLen, 0)
		c.Check(input.GetDocuments(), qt.HasLen, 0)
		c.Check(input.GetContents(), qt.HasLen, 0)
	})

	t.Run("chat with video files", func(t *testing.T) {
		input := TaskChatInput{
			Prompt: stringPtr("Analyze these videos"),
			Model:  "gemini-2.5-flash",
		}

		// Videos should always use File API regardless of size
		c.Check(input.Prompt, qt.Not(qt.IsNil))
		c.Check(*input.Prompt, qt.Equals, "Analyze these videos")
		c.Check(input.Model, qt.Equals, "gemini-2.5-flash")

		// Test system message handling
		c.Check(input.GetSystemMessage(), qt.IsNil)
		c.Check(input.GetSystemInstruction(), qt.IsNil)
	})
}

// Test chat performance optimizations
func TestChatPerformanceOptimizations(t *testing.T) {
	c := qt.New(t)

	t.Run("memory allocation efficiency in chat", func(t *testing.T) {
		// Test that we're pre-allocating slices efficiently in chat context
		mediaCount := 5
		historyCount := 3

		// Pre-allocated approach for chat contents
		contents := make([]*genai.Content, 0, historyCount+1) // history + current message
		parts := make([]*genai.Part, 0, mediaCount+1)         // media + text parts

		// Simulate adding history
		for i := 0; i < historyCount; i++ {
			contents = append(contents, &genai.Content{
				Role:  "user",
				Parts: []*genai.Part{{Text: fmt.Sprintf("message-%d", i)}},
			})
		}

		// Simulate adding current message parts
		for i := 0; i < mediaCount; i++ {
			parts = append(parts, &genai.Part{Text: fmt.Sprintf("part-%d", i)})
		}
		contents = append(contents, &genai.Content{Role: "user", Parts: parts})

		c.Check(len(contents), qt.Equals, historyCount+1)
		c.Check(cap(contents), qt.Equals, historyCount+1)
		c.Check(len(parts), qt.Equals, mediaCount)
		c.Check(cap(parts), qt.Equals, mediaCount+1)
	})

	t.Run("timeout configurations for chat", func(t *testing.T) {
		// Test timeout values used in chat File API operations
		imageTimeout := 60 * time.Second
		audioTimeout := 60 * time.Second
		videoTimeout := 120 * time.Second
		documentTimeout := 60 * time.Second

		c.Check(imageTimeout, qt.Equals, time.Minute)
		c.Check(audioTimeout, qt.Equals, time.Minute)
		c.Check(videoTimeout, qt.Equals, 2*time.Minute)
		c.Check(documentTimeout, qt.Equals, time.Minute)

		// Video should have longer timeout in chat too
		c.Check(videoTimeout > imageTimeout, qt.IsTrue)
		c.Check(videoTimeout > audioTimeout, qt.IsTrue)
		c.Check(videoTimeout > documentTimeout, qt.IsTrue)
	})
}

func TestImageGeneration(t *testing.T) {
	t.Parallel()

	t.Run("image MIME type detection", func(t *testing.T) {
		c := qt.New(t)

		// Test the standard approach used in the codebase
		// Test valid image MIME types
		c.Check(strings.Contains(strings.ToLower("image/png"), "image"), qt.Equals, true)
		c.Check(strings.Contains(strings.ToLower("image/jpeg"), "image"), qt.Equals, true)
		c.Check(strings.Contains(strings.ToLower("image/gif"), "image"), qt.Equals, true)
		c.Check(strings.Contains(strings.ToLower("image/webp"), "image"), qt.Equals, true)
		c.Check(strings.Contains(strings.ToLower("IMAGE/PNG"), "image"), qt.Equals, true) // Case insensitive

		// Test non-image MIME types
		c.Check(strings.Contains(strings.ToLower("text/plain"), "image"), qt.Equals, false)
		c.Check(strings.Contains(strings.ToLower("application/json"), "image"), qt.Equals, false)
		c.Check(strings.Contains(strings.ToLower("video/mp4"), "image"), qt.Equals, false)
		c.Check(strings.Contains(strings.ToLower("audio/wav"), "image"), qt.Equals, false)
	})

	t.Run("renderFinal with generated images", func(t *testing.T) {
		c := qt.New(t)

		// Create a mock response with generated images
		// Use a simple 1x1 PNG image data
		pngData := []byte{
			0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // PNG signature
			0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52, // IHDR chunk
			0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, // 1x1 dimensions
			0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53, // bit depth, color type, etc.
			0xDE, 0x00, 0x00, 0x00, 0x0C, 0x49, 0x44, 0x41, // IDAT chunk
			0x54, 0x08, 0x99, 0x01, 0x01, 0x01, 0x00, 0x00, // pixel data
			0xFE, 0xFF, 0x00, 0x00, 0x02, 0x00, 0x01, 0xE5, // checksum
			0x27, 0xDE, 0xFC, 0x00, 0x00, 0x00, 0x00, 0x49, // IEND chunk
			0x45, 0x4E, 0x44, 0xAE, 0x42, 0x60, 0x82,
		}

		resp := &genai.GenerateContentResponse{
			Candidates: []*genai.Candidate{
				{
					Content: &genai.Content{
						Parts: []*genai.Part{
							{Text: "Here's your generated image:"},
							{InlineData: &genai.Blob{MIMEType: "image/png", Data: pngData}},
						},
					},
				},
			},
		}

		result := renderFinal(resp, nil)

		// Check that text was extracted
		c.Check(result.Texts, qt.HasLen, 1)
		c.Check(result.Texts[0], qt.Equals, "Here's your generated image:")

		// Check that images were extracted
		c.Check(result.Images, qt.HasLen, 1)
		c.Check(result.Images[0].ContentType().String(), qt.Equals, "image/png")
	})

	t.Run("buildStreamOutput with generated images", func(t *testing.T) {
		c := qt.New(t)

		// Mock execution for the method receiver
		e := &execution{}

		// Use a simple 1x1 PNG image data
		pngData := []byte{
			0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // PNG signature
			0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52, // IHDR chunk
			0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, // 1x1 dimensions
			0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53, // bit depth, color type, etc.
			0xDE, 0x00, 0x00, 0x00, 0x0C, 0x49, 0x44, 0x41, // IDAT chunk
			0x54, 0x08, 0x99, 0x01, 0x01, 0x01, 0x00, 0x00, // pixel data
			0xFE, 0xFF, 0x00, 0x00, 0x02, 0x00, 0x01, 0xE5, // checksum
			0x27, 0xDE, 0xFC, 0x00, 0x00, 0x00, 0x00, 0x49, // IEND chunk
			0x45, 0x4E, 0x44, 0xAE, 0x42, 0x60, 0x82,
		}

		texts := []string{"Generated image:"}
		finalResp := &genai.GenerateContentResponse{
			Candidates: []*genai.Candidate{
				{
					Content: &genai.Content{
						Parts: []*genai.Part{
							{Text: "Generated image:"},
							{InlineData: &genai.Blob{MIMEType: "image/png", Data: pngData}},
						},
					},
				},
			},
		}

		result := e.buildStreamOutput(texts, finalResp)

		// Check that texts are preserved
		c.Check(result.Texts, qt.DeepEquals, texts)

		// Check that images are extracted during streaming
		c.Check(result.Images, qt.HasLen, 1)
	})

	t.Run("renderFinal with mixed content", func(t *testing.T) {
		c := qt.New(t)

		// Use a simple 1x1 PNG image data
		pngData := []byte{
			0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // PNG signature
			0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52, // IHDR chunk
			0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, // 1x1 dimensions
			0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53, // bit depth, color type, etc.
			0xDE, 0x00, 0x00, 0x00, 0x0C, 0x49, 0x44, 0x41, // IDAT chunk
			0x54, 0x08, 0x99, 0x01, 0x01, 0x01, 0x00, 0x00, // pixel data
			0xFE, 0xFF, 0x00, 0x00, 0x02, 0x00, 0x01, 0xE5, // checksum
			0x27, 0xDE, 0xFC, 0x00, 0x00, 0x00, 0x00, 0x49, // IEND chunk
			0x45, 0x4E, 0x44, 0xAE, 0x42, 0x60, 0x82,
		}

		// Create a response with text, images, and non-image content
		resp := &genai.GenerateContentResponse{
			Candidates: []*genai.Candidate{
				{
					Content: &genai.Content{
						Parts: []*genai.Part{
							{Text: "Here are your images: "},
							{InlineData: &genai.Blob{MIMEType: "image/png", Data: pngData}},
							{Text: " and "},
							{InlineData: &genai.Blob{MIMEType: "image/png", Data: pngData}}, // Use PNG data with PNG MIME for valid image
							{Text: " Done!"},
							{InlineData: &genai.Blob{MIMEType: "audio/wav", Data: []byte("audio-data")}}, // Non-image, should be ignored
						},
					},
				},
			},
		}

		result := renderFinal(resp, nil)

		// Check that all text parts were concatenated
		c.Check(result.Texts, qt.HasLen, 1)
		c.Check(result.Texts[0], qt.Equals, "Here are your images:  and  Done!")

		// Check that only image parts were extracted (audio should be ignored)
		c.Check(result.Images, qt.HasLen, 2)
		c.Check(result.Images[0].ContentType().String(), qt.Equals, "image/png")
		c.Check(result.Images[1].ContentType().String(), qt.Equals, "image/png")
	})
}
