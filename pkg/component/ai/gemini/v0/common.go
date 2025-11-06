package gemini

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"mime"
	"path"
	"slices"
	"strings"
	"time"

	"google.golang.org/genai"

	"github.com/instill-ai/pipeline-backend/pkg/data"
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
	"github.com/instill-ai/pipeline-backend/pkg/external"
)

var (
	// Gemini-supported formats - these can be sent directly without conversion
	geminiImageFormats = []string{
		data.PNG,  // PNG
		data.JPEG, // JPEG
		data.WEBP, // WEBP
		data.HEIC, // HEIC
		data.HEIF, // HEIF
	}

	// Instill-supported formats that can be converted to Gemini-supported formats
	convertibleImageFormats = []string{
		data.GIF,  // GIF
		data.BMP,  // BMP
		data.TIFF, // TIFF
		data.AVIF, // AVIF
	}

	geminiAudioFormats = []string{
		data.WAV,  // WAV
		data.MP3,  // MP3
		data.AIFF, // AIFF
		data.AAC,  // AAC
		data.OGG,  // OGG Vorbis
		data.FLAC, // FLAC
	}

	convertibleAudioFormats = []string{
		data.M4A, // M4A (audio/mp4)
		data.WMA, // WMA (audio/x-ms-wma)
	}

	geminiVideoFormats = []string{
		data.MP4,       // MP4
		data.MPEG,      // MPEG
		data.MOV,       // MOV (video/quicktime)
		data.AVI,       // AVI (video/x-msvideo)
		data.FLV,       // FLV (video/x-flv)
		data.WEBMVIDEO, // WEBM
		data.WMV,       // WMV (video/x-ms-wmv)
	}

	convertibleVideoFormats = []string{
		data.MKV, // MKV (video/x-matroska)
	}

	geminiDocumentFormats = []string{
		data.PDF, // PDF - only visual document format supported by Gemini
	}

	convertibleDocumentFormats = []string{
		data.DOC,  // DOC
		data.DOCX, // DOCX
		data.PPT,  // PPT
		data.PPTX, // PPTX
		data.XLS,  // XLS
		data.XLSX, // XLSX
	}

	textBasedDocumentFormats = []string{
		data.HTML,     // HTML
		data.MARKDOWN, // Markdown
		data.TEXT,     // Plain text
		data.PLAIN,    // Plain text
		data.CSV,      // CSV
	}
)

// MultimediaInput defines the interface for inputs that contain multimedia content
type MultimediaInput interface {
	GetPrompt() *string
	GetImages() []format.Image
	GetAudio() []format.Audio
	GetVideos() []format.Video
	GetDocuments() []format.Document
	GetContents() []*genai.Content
}

// SystemMessageInput defines the interface for inputs that contain system message configuration
type SystemMessageInput interface {
	GetSystemMessage() *string
	GetSystemInstruction() *genai.Content
}

// newURIOrDataPart creates a genai.Part from a URI or base64 data string
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
func processTextParts(in MultimediaInput) []genai.Part {
	var textParts []genai.Part

	// Extract text parts from Contents
	if contents := in.GetContents(); len(contents) > 0 {
		last := contents[len(contents)-1]
		for _, part := range last.Parts {
			if part.Text != "" {
				textParts = append(textParts, *part)
			}
		}
	}

	// Add prompt as text part
	if prompt := in.GetPrompt(); prompt != nil && *prompt != "" {
		textParts = append(textParts, genai.Part{Text: *prompt})
	}

	return textParts
}

// processNonTextContentParts extracts non-text parts from Contents (images, files, etc.).
// These parts are processed first in the final ordering.
func processNonTextContentParts(in MultimediaInput) []genai.Part {
	var nonTextParts []genai.Part

	if contents := in.GetContents(); len(contents) > 0 {
		last := contents[len(contents)-1]
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
		processedImg := img

		// Check if format is supported by Gemini, if not, try to convert
		if !slices.Contains(geminiImageFormats, contentType) {
			if slices.Contains(convertibleImageFormats, contentType) {
				// Convert unsupported format to PNG (widely supported by Gemini)
				convertedImg, err := img.Convert(data.PNG)
				if err != nil {
					return nil, fmt.Errorf("failed to convert image from %s to PNG: %w", contentType, err)
				}
				processedImg = convertedImg
				contentType = data.PNG
			} else {
				// Unknown format - cannot be processed
				return nil, fmt.Errorf("image format %s is not supported and cannot be processed", contentType)
			}
		}

		imgBase64, err := processedImg.Base64()
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
		processedAudio := audioFile

		// Check if format is supported by Gemini, if not, try to convert
		if !slices.Contains(geminiAudioFormats, contentType) {
			if slices.Contains(convertibleAudioFormats, contentType) {
				// Convert unsupported format to FLAC (widely supported by Gemini)
				convertedAudio, err := audioFile.Convert(data.FLAC)
				if err != nil {
					return nil, fmt.Errorf("failed to convert audio from %s to FLAC: %w", contentType, err)
				}
				processedAudio = convertedAudio
				contentType = data.FLAC
			} else {
				// Unknown format - cannot be processed
				return nil, fmt.Errorf("audio format %s is not supported and cannot be processed", contentType)
			}
		}

		audioBase64, err := processedAudio.Base64()
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
		processedVideo := video

		// Check if format is supported by Gemini, if not, try to convert
		if !slices.Contains(geminiVideoFormats, contentType) {
			if slices.Contains(convertibleVideoFormats, contentType) {
				// Convert unsupported format to MP4 (widely supported by Gemini)
				convertedVideo, err := video.Convert(data.MP4)
				if err != nil {
					return nil, fmt.Errorf("failed to convert video from %s to MP4: %w", contentType, err)
				}
				processedVideo = convertedVideo
				contentType = data.MP4
			} else {
				// Unknown format - cannot be processed
				return nil, fmt.Errorf("video format %s is not supported and cannot be processed", contentType)
			}
		}

		videoBase64, err := processedVideo.Base64()
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
// - Office documents: Automatically converted to PDF for visual understanding
func processDocumentParts(documents []format.Document) ([]genai.Part, error) {
	var parts []genai.Part

	for _, doc := range documents {
		contentType := doc.ContentType().String()
		processedDoc := doc

		// Check if format is supported by Gemini, if not, try to convert
		if !slices.Contains(geminiDocumentFormats, contentType) {
			// Text-based documents: Process as plain text
			if slices.Contains(textBasedDocumentFormats, contentType) || strings.HasPrefix(contentType, "text/") {
				// Extract as plain text content
				textContent := doc.String()
				parts = append(parts, genai.Part{Text: textContent})
				continue
			}
			// Office documents: Convert to PDF for visual elements
			if slices.Contains(convertibleDocumentFormats, contentType) {
				convertedDoc, err := doc.PDF()
				if err != nil {
					return nil, fmt.Errorf("failed to convert document from %s to PDF: %w", contentType, err)
				}
				processedDoc = convertedDoc
				contentType = data.PDF
			} else {
				// Other unknown document formats
				return nil, fmt.Errorf("document format %s is not supported and cannot be processed", contentType)
			}
		}

		// Process the document (either original PDF or converted)
		if contentType == data.PDF {
			// PDFs support full document vision capabilities
			// The model can interpret visual elements like charts, diagrams, and formatting
			docBase64, err := processedDoc.Base64()
			if err != nil {
				return nil, err
			}
			if p := newURIOrDataPart(docBase64.String(), detectMIMEFromPath(docBase64.String(), "application/pdf")); p != nil {
				parts = append(parts, *p)
			}
		}
	}

	return parts, nil
}

// buildReqParts constructs the user request parts from input, including prompt/contents, images, audio, videos, and documents.
// Following best practices: text content (from both Contents and Prompt) is placed after visual/multimedia content (images/audio/videos/documents).
func buildReqParts(in MultimediaInput) ([]genai.Part, error) {
	var parts []genai.Part

	// Process non-text parts from Contents first (images, files, etc.)
	nonTextContentParts := processNonTextContentParts(in)
	parts = append(parts, nonTextContentParts...)

	// Process multimedia content in optimal order: images → audio → videos → documents
	imageParts, err := processImageParts(in.GetImages())
	if err != nil {
		return nil, err
	}
	parts = append(parts, imageParts...)

	audioParts, err := processAudioParts(in.GetAudio())
	if err != nil {
		return nil, err
	}
	parts = append(parts, audioParts...)

	videoParts, err := processVideoParts(in.GetVideos())
	if err != nil {
		return nil, err
	}
	parts = append(parts, videoParts...)

	documentParts, err := processDocumentParts(in.GetDocuments())
	if err != nil {
		return nil, err
	}
	parts = append(parts, documentParts...)

	// Process text content last (as per best practices)
	textParts := processTextParts(in)
	parts = append(parts, textParts...)

	return parts, nil
}

// extractSystemMessage extracts system message from input, prioritizing system-message over system-instruction
func extractSystemMessage(in SystemMessageInput) string {
	if systemMessage := in.GetSystemMessage(); systemMessage != nil && *systemMessage != "" {
		return *systemMessage
	}
	if systemInstruction := in.GetSystemInstruction(); systemInstruction != nil && len(systemInstruction.Parts) > 0 {
		for _, p := range systemInstruction.Parts {
			if p.Text != "" {
				return p.Text
			}
		}
	}
	return ""
}

// MaxInlineSize is the 20MB threshold for determining when to use File API based on total request size
const MaxInlineSize = 20 * 1024 * 1024

// FileAPITimeout is the timeout for File API operations (upload and processing)
const FileAPITimeout = 300 * time.Second

// uploadedFile represents a file that was uploaded and needs to be waited for
type uploadedFile struct {
	name     string
	uri      string
	mimeType string
}

// uploadFileAndWait uploads a file and waits for it to become ACTIVE
func (e *execution) uploadFileAndWait(ctx context.Context, client *genai.Client, data []byte, mimeType string, timeout time.Duration) (*uploadedFile, error) {
	// Upload file
	file, err := client.Files.Upload(ctx, bytes.NewReader(data), &genai.UploadFileConfig{
		MIMEType: mimeType,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to upload file to File API: %w", err)
	}

	// Wait for file to become ACTIVE
	if err := e.waitForFileActive(ctx, client, file.Name, timeout); err != nil {
		// Clean up the uploaded file if it failed to become active
		_, err := client.Files.Delete(ctx, file.Name, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to delete uploaded file: %w", err)
		}
		return nil, fmt.Errorf("uploaded file not ready for use: %w", err)
	}

	return &uploadedFile{
		name:     file.Name,
		uri:      file.URI,
		mimeType: mimeType,
	}, nil
}

// processMediaFile handles file upload for any media type with size checking
func (e *execution) processMediaFile(ctx context.Context, client *genai.Client, fileData []byte, contentType string, maxInlineSize int, timeout time.Duration, forceUpload bool) (genai.Part, string, error) {
	fileSize := len(fileData)

	// Use File API if file is large, or if forceUpload is true (e.g., for videos)
	if fileSize > maxInlineSize || forceUpload {
		uploadedFile, err := e.uploadFileAndWait(ctx, client, fileData, contentType, timeout)
		if err != nil {
			return genai.Part{}, "", err
		}

		part := genai.Part{FileData: &genai.FileData{FileURI: uploadedFile.uri, MIMEType: uploadedFile.mimeType}}
		return part, uploadedFile.name, nil
	}

	// Use inline data for small files
	dataURI := fmt.Sprintf("data:%s;base64,%s", contentType, base64.StdEncoding.EncodeToString(fileData))
	if p := newURIOrDataPart(dataURI, contentType); p != nil {
		return *p, "", nil
	}

	return genai.Part{}, "", fmt.Errorf("failed to create part from file data")
}

// waitForFileActive waits for an uploaded file to become ACTIVE state before using it
func (e *execution) waitForFileActive(ctx context.Context, client *genai.Client, fileName string, timeout time.Duration) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	// Check immediately first (file might already be active)
	if file, err := client.Files.Get(ctx, fileName, nil); err == nil {
		if file.State == genai.FileStateActive {
			return nil
		}
		if file.State == genai.FileStateFailed {
			return fmt.Errorf("file %s processing failed", fileName)
		}
	}

	for {
		select {
		case <-timeoutCtx.Done():
			return fmt.Errorf("timeout waiting for file %s to become ACTIVE: %w", fileName, timeoutCtx.Err())
		case <-ticker.C:
			file, err := client.Files.Get(ctx, fileName, nil)
			if err != nil {
				return fmt.Errorf("failed to get file status for %s: %w", fileName, err)
			}

			switch file.State {
			case genai.FileStateActive:
				return nil
			case genai.FileStateFailed:
				return fmt.Errorf("file %s processing failed", fileName)
			case genai.FileStateProcessing:
				continue
			default:
				return fmt.Errorf("file %s in unexpected state: %s", fileName, file.State)
			}
		}
	}
}

// buildReqPartsWithFileAPI constructs request parts from input, using File API when total request size > 20MB or for cache videos
func (e *execution) buildReqPartsWithFileAPI(ctx context.Context, client *genai.Client, in MultimediaInput, isCache bool) ([]genai.Part, []string, error) {
	// Pre-calculate total capacity for better memory allocation
	totalMediaCount := len(in.GetImages()) + len(in.GetAudio()) + len(in.GetVideos()) + len(in.GetDocuments())
	nonTextContentParts := processNonTextContentParts(in)
	textParts := processTextParts(in)

	// Pre-allocate slices with estimated capacity
	parts := make([]genai.Part, 0, totalMediaCount+len(nonTextContentParts)+len(textParts))
	uploadedFiles := make([]string, 0, totalMediaCount) // Conservative estimate

	// Add non-text parts from Contents first
	parts = append(parts, nonTextContentParts...)

	// Calculate total request size to determine if we should use File API
	totalRequestSize := e.calculateTotalRequestSize(in)
	useFileAPI := totalRequestSize > MaxInlineSize

	// Process all media types with unified File API decision logic
	mediaProcessors := []struct {
		name string
		fn   func() ([]genai.Part, []string, error)
	}{
		{"images", func() ([]genai.Part, []string, error) {
			return e.processImagePartsWithTotalSize(ctx, client, in.GetImages(), useFileAPI)
		}},
		{"audio", func() ([]genai.Part, []string, error) {
			return e.processAudioPartsWithTotalSize(ctx, client, in.GetAudio(), useFileAPI)
		}},
		{"videos", func() ([]genai.Part, []string, error) {
			// Videos for caching always use File API, otherwise follow total size rule
			forceFileAPI := isCache || useFileAPI
			return e.processVideoPartsWithTotalSize(ctx, client, in.GetVideos(), forceFileAPI)
		}},
		{"documents", func() ([]genai.Part, []string, error) {
			return e.processDocumentPartsWithTotalSize(ctx, client, in.GetDocuments(), useFileAPI)
		}},
	}

	// Process each media type sequentially (could be parallelized in future)
	for _, processor := range mediaProcessors {
		mediaParts, mediaFiles, err := processor.fn()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to process %s: %w", processor.name, err)
		}
		parts = append(parts, mediaParts...)
		uploadedFiles = append(uploadedFiles, mediaFiles...)
	}

	// Add text content last (as per best practices)
	parts = append(parts, textParts...)

	return parts, uploadedFiles, nil
}

// calculateTotalRequestSize estimates the total size of the request including all content
func (e *execution) calculateTotalRequestSize(in MultimediaInput) int {
	totalSize := 0

	// Calculate text content size
	if prompt := in.GetPrompt(); prompt != nil {
		totalSize += len(*prompt)
	}

	// Calculate system message size
	if sysInput, ok := in.(SystemMessageInput); ok {
		if sysMsg := sysInput.GetSystemMessage(); sysMsg != nil {
			totalSize += len(*sysMsg)
		}
		if sysInstr := sysInput.GetSystemInstruction(); sysInstr != nil {
			for _, part := range sysInstr.Parts {
				if part.Text != "" {
					totalSize += len(part.Text)
				}
			}
		}
	}

	// Calculate Contents size
	for _, content := range in.GetContents() {
		for _, part := range content.Parts {
			if part.Text != "" {
				totalSize += len(part.Text)
			}
			if part.InlineData != nil {
				totalSize += len(part.InlineData.Data)
			}
		}
	}

	// Calculate media files size
	for _, img := range in.GetImages() {
		if binary, err := img.Binary(); err == nil {
			totalSize += len(binary.ByteArray())
		}
	}

	for _, audio := range in.GetAudio() {
		if binary, err := audio.Binary(); err == nil {
			totalSize += len(binary.ByteArray())
		}
	}

	for _, video := range in.GetVideos() {
		if binary, err := video.Binary(); err == nil {
			totalSize += len(binary.ByteArray())
		}
	}

	for _, doc := range in.GetDocuments() {
		if binary, err := doc.Binary(); err == nil {
			totalSize += len(binary.ByteArray())
		} else {
			// For text-based documents, use string length
			totalSize += len(doc.String())
		}
	}

	return totalSize
}

// processImagePartsWithTotalSize processes images, using File API based on total request size decision
func (e *execution) processImagePartsWithTotalSize(ctx context.Context, client *genai.Client, images []format.Image, useFileAPI bool) ([]genai.Part, []string, error) {
	if len(images) == 0 {
		return nil, nil, nil
	}

	parts := make([]genai.Part, 0, len(images))
	uploadedFiles := make([]string, 0, len(images))

	for _, img := range images {
		contentType := img.ContentType().String()
		processedImg := img

		// Check if format is supported by Gemini, if not, try to convert
		if !slices.Contains(geminiImageFormats, contentType) {
			if slices.Contains(convertibleImageFormats, contentType) {
				// Convert unsupported format to PNG (widely supported by Gemini)
				convertedImg, err := img.Convert(data.PNG)
				if err != nil {
					return nil, nil, fmt.Errorf("failed to convert image from %s to PNG: %w", contentType, err)
				}
				processedImg = convertedImg
				contentType = data.PNG
			} else {
				// Unknown format - cannot be processed
				return nil, nil, fmt.Errorf("image format %s is not supported and cannot be processed", contentType)
			}
		}

		// Get binary data
		binary, err := processedImg.Binary()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get image binary: %w", err)
		}

		// Process using File API decision based on total request size
		part, fileName, err := e.processMediaFile(ctx, client, binary.ByteArray(), contentType, 0, FileAPITimeout, useFileAPI)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to process image: %w", err)
		}

		parts = append(parts, part)
		if fileName != "" {
			uploadedFiles = append(uploadedFiles, fileName)
		}
	}

	return parts, uploadedFiles, nil
}

// processAudioPartsWithTotalSize processes audio files, using File API based on total request size decision
func (e *execution) processAudioPartsWithTotalSize(ctx context.Context, client *genai.Client, audio []format.Audio, useFileAPI bool) ([]genai.Part, []string, error) {
	if len(audio) == 0 {
		return nil, nil, nil
	}

	parts := make([]genai.Part, 0, len(audio))
	uploadedFiles := make([]string, 0, len(audio))

	for _, audioFile := range audio {
		contentType := audioFile.ContentType().String()
		processedAudio := audioFile

		// Check if format is supported by Gemini, if not, try to convert
		if !slices.Contains(geminiAudioFormats, contentType) {
			if slices.Contains(convertibleAudioFormats, contentType) {
				// Convert unsupported format to FLAC (widely supported by Gemini)
				convertedAudio, err := audioFile.Convert(data.FLAC)
				if err != nil {
					return nil, nil, fmt.Errorf("failed to convert audio from %s to FLAC: %w", contentType, err)
				}
				processedAudio = convertedAudio
				contentType = data.FLAC
			} else {
				// Unknown format - cannot be processed
				return nil, nil, fmt.Errorf("audio format %s is not supported and cannot be processed", contentType)
			}
		}

		// Get binary data
		binary, err := processedAudio.Binary()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get audio binary: %w", err)
		}

		// Process using File API decision based on total request size
		part, fileName, err := e.processMediaFile(ctx, client, binary.ByteArray(), contentType, 0, FileAPITimeout, useFileAPI)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to process audio: %w", err)
		}

		parts = append(parts, part)
		if fileName != "" {
			uploadedFiles = append(uploadedFiles, fileName)
		}
	}

	return parts, uploadedFiles, nil
}

// processVideoPartsWithTotalSize processes video files, using File API based on total size decision or cache requirement
func (e *execution) processVideoPartsWithTotalSize(ctx context.Context, client *genai.Client, videos []format.Video, useFileAPI bool) ([]genai.Part, []string, error) {
	if len(videos) == 0 {
		return nil, nil, nil
	}

	parts := make([]genai.Part, 0, len(videos))
	uploadedFiles := make([]string, 0, len(videos))

	for _, video := range videos {
		contentType := video.ContentType().String()
		processedVideo := video

		// Check if format is supported by Gemini, if not, try to convert
		if !slices.Contains(geminiVideoFormats, contentType) {
			if slices.Contains(convertibleVideoFormats, contentType) {
				// Convert unsupported format to MP4 (widely supported by Gemini)
				convertedVideo, err := video.Convert(data.MP4)
				if err != nil {
					return nil, nil, fmt.Errorf("failed to convert video from %s to MP4: %w", contentType, err)
				}
				processedVideo = convertedVideo
				contentType = data.MP4
			} else {
				// Unknown format - cannot be processed
				return nil, nil, fmt.Errorf("video format %s is not supported and cannot be processed", contentType)
			}
		}

		// Get binary data
		binary, err := processedVideo.Binary()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get video binary: %w", err)
		}

		// Use File API based on decision (longer timeout for videos)
		part, fileName, err := e.processMediaFile(ctx, client, binary.ByteArray(), contentType, 0, FileAPITimeout, useFileAPI)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to process video: %w", err)
		}

		parts = append(parts, part)
		if fileName != "" {
			uploadedFiles = append(uploadedFiles, fileName)
		}
	}

	return parts, uploadedFiles, nil
}

// processDocumentPartsWithTotalSize processes documents, using File API based on total request size decision (for PDFs)
func (e *execution) processDocumentPartsWithTotalSize(ctx context.Context, client *genai.Client, documents []format.Document, useFileAPI bool) ([]genai.Part, []string, error) {
	if len(documents) == 0 {
		return nil, nil, nil
	}

	parts := make([]genai.Part, 0, len(documents))
	uploadedFiles := make([]string, 0, len(documents))

	for _, doc := range documents {
		contentType := doc.ContentType().String()
		processedDoc := doc

		// Check if format is supported by Gemini, if not, try to convert
		if !slices.Contains(geminiDocumentFormats, contentType) {
			// Text-based documents: Process as plain text
			if slices.Contains(textBasedDocumentFormats, contentType) || strings.HasPrefix(contentType, "text/") {
				// Extract as plain text content (no File API needed)
				textContent := doc.String()
				parts = append(parts, genai.Part{Text: textContent})
				continue
			}
			// Office documents: Convert to PDF for visual elements
			if slices.Contains(convertibleDocumentFormats, contentType) {
				convertedDoc, err := doc.PDF()
				if err != nil {
					return nil, nil, fmt.Errorf("failed to convert document from %s to PDF: %w", contentType, err)
				}
				processedDoc = convertedDoc
				contentType = data.PDF
			} else {
				// Other unknown document formats
				return nil, nil, fmt.Errorf("document format %s is not supported and cannot be processed", contentType)
			}
		}

		// Process PDFs (either original or converted)
		if contentType == data.PDF {
			// PDFs support full document vision capabilities
			binary, err := processedDoc.Binary()
			if err != nil {
				return nil, nil, fmt.Errorf("failed to get document binary: %w", err)
			}

			// Process using File API decision based on total request size
			part, fileName, err := e.processMediaFile(ctx, client, binary.ByteArray(), "application/pdf", 0, FileAPITimeout, useFileAPI)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to process PDF: %w", err)
			}

			parts = append(parts, part)
			if fileName != "" {
				uploadedFiles = append(uploadedFiles, fileName)
			}
		}
	}

	return parts, uploadedFiles, nil
}
