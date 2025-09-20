package gemini

import (
	"encoding/base64"
	"os"
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

func Test_buildReqParts_Images(t *testing.T) {
	c := qt.New(t)
	prompt := "Describe the image."
	imgData := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR4nGNgYAAAAAMAASsJTYQAAAAASUVORK5CYII="
	imageBytes, err := base64.StdEncoding.DecodeString(imgData)
	if err != nil {
		t.Fatal(err)
	}
	img, err := data.NewImageFromBytes(imageBytes, "image/png", "test.png", true)
	if err != nil {
		t.Fatal(err)
	}

	in := TaskChatInput{
		Prompt: &prompt,
		Images: []format.Image{img},
	}
	got, err := buildReqParts(in)
	c.Assert(err, qt.IsNil)
	// Expect 1 image + 1 text prompt = 2 parts
	c.Assert(got, qt.HasLen, 2)
	c.Check(got[0].InlineData, qt.Not(qt.IsNil))
	c.Check(got[0].InlineData.MIMEType, qt.Equals, "image/png")
	c.Check(got[1].Text, qt.Equals, prompt) // Prompt is last
}

func Test_buildReqParts_Documents(t *testing.T) {
	c := qt.New(t)
	prompt := "Summarize this document."
	pdfHeader := "JVBERi0xLjQK" // raw base64 PDF header
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
		Documents: []format.Document{doc},
	}
	got, err := buildReqParts(in)
	c.Assert(err, qt.IsNil)
	// Expect 1 PDF doc + 1 text prompt = 2 parts
	c.Assert(got, qt.HasLen, 2)
	c.Check(got[0].InlineData, qt.Not(qt.IsNil))
	c.Check(got[0].InlineData.MIMEType, qt.Equals, "application/pdf")
	c.Check(got[1].Text, qt.Equals, prompt) // Prompt is last
}

func Test_buildReqParts_Audio(t *testing.T) {
	c := qt.New(t)
	prompt := "Describe the audio content."

	// Read real test audio file
	audioPath := "../../../../data/testdata/small_sample.wav"
	audioBytes, err := os.ReadFile(audioPath)
	if err != nil {
		t.Fatal(err)
	}
	audio, err := data.NewAudioFromBytes(audioBytes, "audio/wav", "small_sample.wav", false)
	if err != nil {
		t.Fatal(err)
	}

	in := TaskChatInput{
		Prompt: &prompt,
		Audio:  []format.Audio{audio},
	}
	got, err := buildReqParts(in)
	c.Assert(err, qt.IsNil)
	// Expect 1 audio + 1 text prompt = 2 parts
	c.Assert(got, qt.HasLen, 2)

	// Check audio part
	c.Check(got[0].InlineData, qt.Not(qt.IsNil))
	c.Check(got[0].InlineData.MIMEType, qt.Equals, "audio/wav")

	// Check text prompt (should be last)
	c.Check(got[1].Text, qt.Equals, prompt)
}

func Test_buildReqParts_Video(t *testing.T) {
	c := qt.New(t)
	prompt := "Describe the video content."

	// Read real test video file
	videoPath := "../../../../data/testdata/small_sample.mp4"
	videoBytes, err := os.ReadFile(videoPath)
	if err != nil {
		t.Fatal(err)
	}
	video, err := data.NewVideoFromBytes(videoBytes, "video/mp4", "small_sample.mp4", true)
	if err != nil {
		t.Fatal(err)
	}

	in := TaskChatInput{
		Prompt: &prompt,
		Videos: []format.Video{video},
	}
	got, err := buildReqParts(in)
	c.Assert(err, qt.IsNil)
	// Expect 1 video + 1 text prompt = 2 parts
	c.Assert(got, qt.HasLen, 2)

	// Check video part
	c.Check(got[0].InlineData, qt.Not(qt.IsNil))
	c.Check(got[0].InlineData.MIMEType, qt.Equals, "video/mp4")

	// Check text prompt (should be last)
	c.Check(got[1].Text, qt.Equals, prompt)
}

func Test_processTextParts(t *testing.T) {
	c := qt.New(t)

	t.Run("with prompt only", func(t *testing.T) {
		prompt := "Test prompt"
		in := TaskChatInput{Prompt: &prompt}

		got := processTextParts(in)
		c.Assert(got, qt.HasLen, 1)
		c.Check(got[0].Text, qt.Equals, prompt)
	})

	t.Run("with contents text only", func(t *testing.T) {
		textContent := "Content text"
		in := TaskChatInput{
			Contents: []*genai.Content{
				{
					Parts: []*genai.Part{
						{Text: textContent},
					},
				},
			},
		}

		got := processTextParts(in)
		c.Assert(got, qt.HasLen, 1)
		c.Check(got[0].Text, qt.Equals, textContent)
	})

	t.Run("with both prompt and contents", func(t *testing.T) {
		prompt := "Test prompt"
		textContent := "Content text"
		in := TaskChatInput{
			Prompt: &prompt,
			Contents: []*genai.Content{
				{
					Parts: []*genai.Part{
						{Text: textContent},
					},
				},
			},
		}

		got := processTextParts(in)
		c.Assert(got, qt.HasLen, 2)
		c.Check(got[0].Text, qt.Equals, textContent) // Content text comes first
		c.Check(got[1].Text, qt.Equals, prompt)      // Prompt comes last
	})

	t.Run("with empty prompt", func(t *testing.T) {
		emptyPrompt := ""
		in := TaskChatInput{Prompt: &emptyPrompt}

		got := processTextParts(in)
		c.Assert(got, qt.HasLen, 0) // Empty prompt should not be added
	})

	t.Run("with nil prompt", func(t *testing.T) {
		in := TaskChatInput{Prompt: nil}

		got := processTextParts(in)
		c.Assert(got, qt.HasLen, 0)
	})
}

func Test_processNonTextContentParts(t *testing.T) {
	c := qt.New(t)

	t.Run("with non-text parts", func(t *testing.T) {
		in := TaskChatInput{
			Contents: []*genai.Content{
				{
					Parts: []*genai.Part{
						{Text: "text part"},
						{InlineData: &genai.Blob{MIMEType: "image/png", Data: []byte("fake")}},
						{Text: "another text"},
						{InlineData: &genai.Blob{MIMEType: "audio/wav", Data: []byte("fake")}},
					},
				},
			},
		}

		got := processNonTextContentParts(in)
		c.Assert(got, qt.HasLen, 2) // Only non-text parts
		c.Check(got[0].InlineData.MIMEType, qt.Equals, "image/png")
		c.Check(got[1].InlineData.MIMEType, qt.Equals, "audio/wav")
	})

	t.Run("with only text parts", func(t *testing.T) {
		in := TaskChatInput{
			Contents: []*genai.Content{
				{
					Parts: []*genai.Part{
						{Text: "text part 1"},
						{Text: "text part 2"},
					},
				},
			},
		}

		got := processNonTextContentParts(in)
		c.Assert(got, qt.HasLen, 0) // No non-text parts
	})

	t.Run("with empty contents", func(t *testing.T) {
		in := TaskChatInput{Contents: nil}

		got := processNonTextContentParts(in)
		c.Assert(got, qt.HasLen, 0)
	})
}

func Test_processImageParts(t *testing.T) {
	c := qt.New(t)

	t.Run("with supported format (PNG)", func(t *testing.T) {
		imgData := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR4nGNgYAAAAAMAASsJTYQAAAAASUVORK5CYII="
		imageBytes, err := base64.StdEncoding.DecodeString(imgData)
		c.Assert(err, qt.IsNil)

		img, err := data.NewImageFromBytes(imageBytes, "image/png", "test.png", false)
		c.Assert(err, qt.IsNil)

		got, err := processImageParts([]format.Image{img})
		c.Assert(err, qt.IsNil)
		c.Assert(got, qt.HasLen, 1)
		c.Check(got[0].InlineData, qt.Not(qt.IsNil))
		c.Check(got[0].InlineData.MIMEType, qt.Equals, "image/png")
	})

	t.Run("with supported format (JPEG)", func(t *testing.T) {
		// JPEG header bytes for a minimal 1x1 image
		jpegBytes := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46}
		img, err := data.NewImageFromBytes(jpegBytes, "image/jpeg", "test.jpg", false)
		if err != nil {
			// Skip if we can't create a valid JPEG for testing
			t.Skip("Cannot create JPEG for testing")
		}

		got, err := processImageParts([]format.Image{img})
		c.Assert(err, qt.IsNil)
		c.Assert(got, qt.HasLen, 1)
		c.Check(got[0].InlineData.MIMEType, qt.Equals, "image/jpeg")
	})

	t.Run("with unsupported format (GIF)", func(t *testing.T) {
		// GIF header for a minimal 1x1 transparent GIF
		gifBytes := []byte{0x47, 0x49, 0x46, 0x38, 0x39, 0x61, 0x01, 0x00, 0x01, 0x00}
		img, err := data.NewImageFromBytes(gifBytes, "image/gif", "test.gif", false)
		if err != nil {
			// Skip if we can't create a valid GIF for testing
			t.Skip("Cannot create GIF for testing")
		}

		got, err := processImageParts([]format.Image{img})
		c.Assert(err, qt.Not(qt.IsNil))
		c.Check(err.Error(), qt.Contains, "image format image/gif is not supported by Gemini API")
		c.Check(err.Error(), qt.Contains, "such as \":png\"")
		c.Check(err.Error(), qt.Contains, "Use \":\" syntax to convert GIF, BMP, TIFF to PNG, JPEG, WEBP")
		c.Assert(got, qt.IsNil)
	})

	t.Run("with unsupported format (BMP)", func(t *testing.T) {
		// BMP header for a minimal bitmap
		bmpBytes := []byte{0x42, 0x4D, 0x3A, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
		img, err := data.NewImageFromBytes(bmpBytes, "image/bmp", "test.bmp", false)
		if err != nil {
			// Skip if we can't create a valid BMP for testing
			t.Skip("Cannot create BMP for testing")
		}

		got, err := processImageParts([]format.Image{img})
		c.Assert(err, qt.Not(qt.IsNil))
		c.Check(err.Error(), qt.Contains, "image format image/bmp is not supported by Gemini API")
		c.Check(err.Error(), qt.Contains, "such as \":png\"")
		c.Assert(got, qt.IsNil)
	})

	t.Run("with completely unknown format", func(t *testing.T) {
		// Create a fake image with an unknown format
		imgData := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A} // PNG header
		img, err := data.NewImageFromBytes(imgData, "image/unknown", "test.unknown", false)
		if err != nil {
			t.Skip("Cannot create unknown image format for testing")
		}

		got, err := processImageParts([]format.Image{img})
		c.Assert(err, qt.Not(qt.IsNil))
		c.Check(err.Error(), qt.Contains, "image format image/unknown is not supported and cannot be processed")
		c.Assert(got, qt.IsNil)
	})
}

func Test_processAudioParts(t *testing.T) {
	c := qt.New(t)

	t.Run("with supported format (WAV)", func(t *testing.T) {
		// Read real test audio file
		audioPath := "../../../../data/testdata/small_sample.wav"
		audioBytes, err := os.ReadFile(audioPath)
		c.Assert(err, qt.IsNil)

		audio, err := data.NewAudioFromBytes(audioBytes, "audio/wav", "small_sample.wav", false)
		c.Assert(err, qt.IsNil)

		got, err := processAudioParts([]format.Audio{audio})
		c.Assert(err, qt.IsNil)
		c.Assert(got, qt.HasLen, 1)
		c.Check(got[0].InlineData, qt.Not(qt.IsNil))
		c.Check(got[0].InlineData.MIMEType, qt.Equals, "audio/wav")
	})

	t.Run("with supported format (OGG)", func(t *testing.T) {
		// Use the real WAV file but create it as OGG format for testing
		audioPath := "../../../../data/testdata/small_sample.wav"
		audioBytes, err := os.ReadFile(audioPath)
		c.Assert(err, qt.IsNil)

		// Create audio with OGG content type (unified format in pipeline-backend)
		audio, err := data.NewAudioFromBytes(audioBytes, "audio/ogg", "test.ogg", false)
		if err != nil {
			t.Skip("Cannot create OGG audio for testing")
		}

		got, err := processAudioParts([]format.Audio{audio})
		c.Assert(err, qt.IsNil)
		c.Assert(got, qt.HasLen, 1)
		c.Check(got[0].InlineData.MIMEType, qt.Equals, "audio/ogg")
	})

	t.Run("with completely unknown format", func(t *testing.T) {
		// Use a real audio file but with an unknown content type
		audioPath := "../../../../data/testdata/small_sample.wav"
		audioBytes, err := os.ReadFile(audioPath)
		c.Assert(err, qt.IsNil)

		audio, err := data.NewAudioFromBytes(audioBytes, "audio/unknown", "test.unknown", false)
		if err != nil {
			t.Skip("Cannot create unknown audio format for testing")
		}

		got, err := processAudioParts([]format.Audio{audio})
		c.Assert(err, qt.Not(qt.IsNil))
		c.Check(err.Error(), qt.Contains, "audio format audio/unknown is not supported and cannot be processed")
		c.Assert(got, qt.IsNil)
	})
}

func Test_processVideoParts(t *testing.T) {
	c := qt.New(t)

	t.Run("with supported format (MP4)", func(t *testing.T) {
		// Read real test video file
		videoPath := "../../../../data/testdata/small_sample.mp4"
		videoBytes, err := os.ReadFile(videoPath)
		c.Assert(err, qt.IsNil)

		video, err := data.NewVideoFromBytes(videoBytes, "video/mp4", "small_sample.mp4", false)
		c.Assert(err, qt.IsNil)

		got, err := processVideoParts([]format.Video{video})
		c.Assert(err, qt.IsNil)
		c.Assert(got, qt.HasLen, 1)
		c.Check(got[0].InlineData, qt.Not(qt.IsNil))
		c.Check(got[0].InlineData.MIMEType, qt.Equals, "video/mp4")
	})

	t.Run("with supported format (WEBM)", func(t *testing.T) {
		// Use the real MP4 file but create it as WEBM format for testing
		videoPath := "../../../../data/testdata/small_sample.mp4"
		videoBytes, err := os.ReadFile(videoPath)
		c.Assert(err, qt.IsNil)

		// Create video with WEBM content type
		video, err := data.NewVideoFromBytes(videoBytes, "video/webm", "test.webm", false)
		if err != nil {
			t.Skip("Cannot create WEBM video for testing")
		}

		got, err := processVideoParts([]format.Video{video})
		c.Assert(err, qt.IsNil)
		c.Assert(got, qt.HasLen, 1)
		c.Check(got[0].InlineData.MIMEType, qt.Equals, "video/webm")
	})

	t.Run("with unsupported format (MKV)", func(t *testing.T) {
		// Use a real video file but with an unsupported content type for testing validation logic
		videoPath := "../../../../data/testdata/small_sample.mp4"
		videoBytes, err := os.ReadFile(videoPath)
		c.Assert(err, qt.IsNil)

		// Create video with unsupported format (MKV) to test validation
		video, err := data.NewVideoFromBytes(videoBytes, "video/x-matroska", "test.mkv", false)
		if err != nil {
			t.Skip("Cannot create MKV video for testing")
		}

		got, err := processVideoParts([]format.Video{video})
		c.Assert(err, qt.Not(qt.IsNil))
		c.Check(err.Error(), qt.Contains, "video format video/x-matroska is not supported by Gemini API")
		c.Check(err.Error(), qt.Contains, "such as \":mp4\"")
		c.Check(err.Error(), qt.Contains, "Use \":\" syntax to convert MKV to MP4, MPEG, MOV, AVI, FLV, WEBM, WMV")
		c.Assert(got, qt.IsNil)
	})

	t.Run("with completely unknown format", func(t *testing.T) {
		// Use a real video file but with an unknown content type
		videoPath := "../../../../data/testdata/small_sample.mp4"
		videoBytes, err := os.ReadFile(videoPath)
		c.Assert(err, qt.IsNil)

		video, err := data.NewVideoFromBytes(videoBytes, "video/unknown", "test.unknown", false)
		if err != nil {
			t.Skip("Cannot create unknown video format for testing")
		}

		got, err := processVideoParts([]format.Video{video})
		c.Assert(err, qt.Not(qt.IsNil))
		c.Check(err.Error(), qt.Contains, "video format video/unknown is not supported and cannot be processed")
		c.Assert(got, qt.IsNil)
	})
}

func Test_processDocumentParts(t *testing.T) {
	c := qt.New(t)

	t.Run("with PDF document", func(t *testing.T) {
		pdfHeader := "JVBERi0xLjQK" // raw base64 PDF header
		pdfBytes, err := base64.StdEncoding.DecodeString(pdfHeader)
		c.Assert(err, qt.IsNil)

		doc, err := data.NewDocumentFromBytes(pdfBytes, "application/pdf", "test.pdf")
		c.Assert(err, qt.IsNil)

		got, err := processDocumentParts([]format.Document{doc})
		c.Assert(err, qt.IsNil)
		c.Assert(got, qt.HasLen, 1)
		c.Check(got[0].InlineData, qt.Not(qt.IsNil))
		c.Check(got[0].InlineData.MIMEType, qt.Equals, "application/pdf")
	})

	t.Run("with text-based document", func(t *testing.T) {
		textContent := "This is a plain text document"
		textBytes := []byte(textContent)

		doc, err := data.NewDocumentFromBytes(textBytes, "text/plain", "test.txt")
		c.Assert(err, qt.IsNil)

		got, err := processDocumentParts([]format.Document{doc})
		c.Assert(err, qt.IsNil)
		c.Assert(got, qt.HasLen, 1)
		c.Check(got[0].Text, qt.Equals, textContent)
		c.Check(got[0].InlineData, qt.IsNil) // Text-based docs don't use InlineData
	})

	t.Run("with HTML document", func(t *testing.T) {
		htmlContent := "<html><body><h1>Test</h1></body></html>"
		htmlBytes := []byte(htmlContent)

		doc, err := data.NewDocumentFromBytes(htmlBytes, "text/html", "test.html")
		c.Assert(err, qt.IsNil)

		got, err := processDocumentParts([]format.Document{doc})
		c.Assert(err, qt.IsNil)
		c.Assert(got, qt.HasLen, 1)
		c.Check(got[0].Text, qt.Equals, htmlContent)
		c.Check(got[0].InlineData, qt.IsNil)
	})

	t.Run("with unsupported convertible document", func(t *testing.T) {
		docBytes := []byte("This is a DOC document")
		doc, err := data.NewDocumentFromBytes(docBytes, data.DOC, "test.doc")
		c.Assert(err, qt.IsNil)

		got, err := processDocumentParts([]format.Document{doc})
		c.Assert(err, qt.Not(qt.IsNil))
		c.Check(err.Error(), qt.Contains, "document format application/msword will be processed as text only")
		c.Assert(got, qt.IsNil)
	})

	t.Run("with unsupported document type", func(t *testing.T) {
		docBytes := []byte("fake binary data")
		doc, err := data.NewDocumentFromBytes(docBytes, "application/unknown", "test.unknown")
		if err != nil {
			t.Skip("Cannot create document with unknown type for testing")
		}

		got, err := processDocumentParts([]format.Document{doc})
		c.Assert(err, qt.Not(qt.IsNil))
		c.Check(err.Error(), qt.Contains, "unsupported document type: application/unknown")
		c.Assert(got, qt.IsNil)
	})

	t.Run("with empty documents slice", func(t *testing.T) {
		got, err := processDocumentParts([]format.Document{})
		c.Assert(err, qt.IsNil)
		c.Assert(got, qt.HasLen, 0)
	})
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
	c.Assert(err.Error(), qt.Contains, "document format application/msword will be processed as text only")
	c.Assert(err.Error(), qt.Contains, "Use \":pdf\" syntax")
	c.Assert(got, qt.IsNil)
}

func Test_buildReqParts_UnsupportedImageFormat(t *testing.T) {
	c := qt.New(t)
	prompt := "Describe this image."

	// Create an image with unsupported format (GIF)
	gifBytes := []byte{0x47, 0x49, 0x46, 0x38, 0x39, 0x61, 0x01, 0x00, 0x01, 0x00}
	img, err := data.NewImageFromBytes(gifBytes, "image/gif", "test.gif", false)
	if err != nil {
		t.Skip("Cannot create GIF for testing")
	}

	in := TaskChatInput{
		Prompt: &prompt,
		Images: []format.Image{img},
	}

	got, err := buildReqParts(in)
	c.Assert(err, qt.Not(qt.IsNil))
	c.Assert(err.Error(), qt.Contains, "image format image/gif is not supported by Gemini API")
	c.Assert(err.Error(), qt.Contains, "Use \":\" syntax to convert GIF, BMP, TIFF to PNG, JPEG, WEBP")
	c.Assert(err.Error(), qt.Contains, "such as \":png\"")
	c.Assert(got, qt.IsNil)
}

func Test_buildReqParts_UnsupportedAudioFormat(t *testing.T) {
	c := qt.New(t)
	prompt := "Describe this audio."

	// Use a real audio file but with an unsupported content type for testing validation logic
	audioPath := "../../../../data/testdata/small_sample.wav"
	audioBytes, err := os.ReadFile(audioPath)
	c.Assert(err, qt.IsNil)

	// Create audio with unsupported format (M4A) to test validation
	audio, err := data.NewAudioFromBytes(audioBytes, "audio/mp4", "test.m4a", false)
	if err != nil {
		t.Skip("Cannot create M4A audio for testing")
	}

	in := TaskChatInput{
		Prompt: &prompt,
		Audio:  []format.Audio{audio},
	}

	got, err := buildReqParts(in)
	c.Assert(err, qt.Not(qt.IsNil))
	c.Assert(err.Error(), qt.Contains, "audio format audio/mp4 is not supported by Gemini API")
	c.Assert(err.Error(), qt.Contains, "Use \":\" syntax to convert M4A, WMA to WAV, MP3, AIFF, AAC, OGG, FLAC")
	c.Assert(err.Error(), qt.Contains, "such as \":wav\"")
	c.Assert(got, qt.IsNil)
}

func Test_buildReqParts_UnsupportedVideoFormat(t *testing.T) {
	c := qt.New(t)
	prompt := "Describe this video."

	// Use a real video file but with an unsupported content type for testing validation logic
	videoPath := "../../../../data/testdata/small_sample.mp4"
	videoBytes, err := os.ReadFile(videoPath)
	c.Assert(err, qt.IsNil)

	// Create video with unsupported format (MKV) to test validation
	video, err := data.NewVideoFromBytes(videoBytes, "video/x-matroska", "test.mkv", false)
	if err != nil {
		t.Skip("Cannot create MKV video for testing")
	}

	in := TaskChatInput{
		Prompt: &prompt,
		Videos: []format.Video{video},
	}

	got, err := buildReqParts(in)
	c.Assert(err, qt.Not(qt.IsNil))
	c.Assert(err.Error(), qt.Contains, "video format video/x-matroska is not supported by Gemini API")
	c.Assert(err.Error(), qt.Contains, "Use \":\" syntax to convert MKV to MP4, MPEG, MOV, AVI, FLV, WEBM, WMV")
	c.Assert(err.Error(), qt.Contains, "such as \":mp4\"")
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
	c.Assert(got.UsageMetadata, qt.Not(qt.IsNil))
	c.Check(got.UsageMetadata.TotalTokenCount, qt.Equals, int32(15))
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
