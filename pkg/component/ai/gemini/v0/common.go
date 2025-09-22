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

// formatSupport defines format support levels for different media types
type formatSupport struct {
	gemini  []string
	instill []string
}

var (
	imageFormats = formatSupport{
		gemini: []string{
			data.PNG,     // PNG
			data.JPEG,    // JPEG
			data.WEBP,    // WEBP
			"image/heic", // HEIC
			"image/heif", // HEIF
		},
		instill: []string{
			data.PNG,  // PNG
			data.JPEG, // JPEG
			data.WEBP, // WEBP
			data.GIF,  // GIF
			data.BMP,  // BMP
			data.TIFF, // TIFF
		},
	}

	audioFormats = formatSupport{
		gemini: []string{
			data.WAV,    // WAV
			"audio/mp3", // MP3
			data.MP3,    // MP3 (audio/mpeg - alternative MIME type)
			data.AIFF,   // AIFF
			data.AAC,    // AAC
			data.OGG,    // OGG Vorbis
			data.FLAC,   // FLAC
		},
		instill: []string{
			data.MP3,  // MP3 (audio/mpeg)
			data.WAV,  // WAV
			data.AAC,  // AAC
			data.OGG,  // OGG
			data.FLAC, // FLAC
			data.M4A,  // M4A (audio/mp4)
			data.WMA,  // WMA (audio/x-ms-wma)
			data.AIFF, // AIFF
		},
	}

	videoFormats = formatSupport{
		gemini: []string{
			data.MP4,     // MP4
			data.MPEG,    // MPEG
			data.MOV,     // MOV (video/quicktime)
			"video/mov",  // MOV (standard MIME type)
			data.AVI,     // AVI (video/x-msvideo)
			"video/avi",  // AVI (standard MIME type)
			data.FLV,     // FLV (video/x-flv)
			"video/mpg",  // MPG - supported by Gemini but not defined in video.go
			data.WEBM,    // WEBM
			data.WMV,     // WMV (video/x-ms-wmv)
			"video/wmv",  // WMV (standard MIME type)
			"video/3gpp", // 3GPP - supported by Gemini but not defined in video.go
		},
		instill: []string{
			data.MP4,  // MP4
			data.AVI,  // AVI (video/x-msvideo)
			data.MOV,  // MOV (video/quicktime)
			data.WEBM, // WEBM
			data.MKV,  // MKV (video/x-matroska)
			data.FLV,  // FLV (video/x-flv)
			data.WMV,  // WMV (video/x-ms-wmv)
			data.MPEG, // MPEG
		},
	}

	documentFormats = formatSupport{
		gemini: []string{
			data.PDF, // PDF - only visual document format supported by Gemini
		},
		instill: []string{
			data.PDF,      // PDF
			data.DOC,      // DOC
			data.DOCX,     // DOCX
			data.PPT,      // PPT
			data.PPTX,     // PPTX
			data.XLS,      // XLS
			data.XLSX,     // XLSX
			data.HTML,     // HTML
			data.MARKDOWN, // Markdown
			data.TEXT,     // Plain text
			data.PLAIN,    // Plain text
			data.CSV,      // CSV
		},
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

// validateFormat checks if a format is supported and returns appropriate error messages
// For documents, it also returns the processing mode ("visual", "text", or "" for error)
func validateFormat(contentType, mediaType string, formats formatSupport, convertibleFormats, supportedTargets, examples string) (string, error) {
	// Check if the format is supported by Gemini API
	if slices.Contains(formats.gemini, contentType) {
		// Special handling for documents to determine processing mode
		if mediaType == "document" {
			if contentType == data.PDF {
				return "visual", nil // PDF supports visual processing
			}
			// Text-based documents supported by Gemini (currently none, but future-proof)
			return "text", nil
		}
		return "", nil // Other media types don't need mode
	}

	// Check if it's a known Instill Core format that can be converted
	if slices.Contains(formats.instill, contentType) {
		// Special handling for documents
		if mediaType == "document" {
			// Text-based documents: Process as plain text
			textBasedTypes := []string{data.HTML, data.MARKDOWN, data.TEXT, data.PLAIN, data.CSV}
			if slices.Contains(textBasedTypes, contentType) || strings.HasPrefix(contentType, "text/") {
				return "text", nil
			}

			// Office documents: Need PDF conversion for visual elements
			officeTypes := []string{data.DOC, data.DOCX, data.PPT, data.PPTX, data.XLS, data.XLSX}
			if slices.Contains(officeTypes, contentType) {
				return "", fmt.Errorf("document format %s will be processed as text only, losing visual elements like charts and formatting. Use \":pdf\" syntax to convert to PDF for document vision capabilities", contentType)
			}

			// Other known document formats
			return "", fmt.Errorf("document format %s is not supported by Gemini API. Use \":pdf\" syntax to convert DOC, DOCX, PPT, PPTX, XLS, XLSX to PDF (supported by both Gemini and Instill Core), such as \":pdf\"", contentType)
		}

		// Non-document media types
		return "", fmt.Errorf("%s format %s is not supported by Gemini API. Use \":\" syntax to convert %s to %s (supported by both Gemini and Instill Core), such as \"%s\"", mediaType, contentType, convertibleFormats, supportedTargets, examples)
	}

	// Unknown format - can't be processed at all
	return "", fmt.Errorf("%s format %s is not supported and cannot be processed", mediaType, contentType)
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

		// Validate image format
		if _, err := validateFormat(contentType, "image", imageFormats, "GIF, BMP, TIFF", "PNG, JPEG, WEBP", ":png\", \":jpeg\", \":webp"); err != nil {
			return nil, nil, err
		}

		// Get binary data
		binary, err := img.Binary()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get image binary: %w", err)
		}

		// Process using File API decision based on total request size
		part, fileName, err := e.processMediaFile(ctx, client, binary.ByteArray(), contentType, 0, 60*time.Second, useFileAPI)
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

		// Validate audio format
		if _, err := validateFormat(contentType, "audio", audioFormats, "M4A, WMA", "WAV, MP3, AIFF, AAC, OGG, FLAC", ":wav\", \":mp3\", \":ogg"); err != nil {
			return nil, nil, err
		}

		// Get binary data
		binary, err := audioFile.Binary()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get audio binary: %w", err)
		}

		// Process using File API decision based on total request size
		part, fileName, err := e.processMediaFile(ctx, client, binary.ByteArray(), contentType, 0, 60*time.Second, useFileAPI)
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

		// Validate video format
		if _, err := validateFormat(contentType, "video", videoFormats, "MKV", "MP4, MPEG, MOV, AVI, FLV, WEBM, WMV", ":mp4\", \":mov\", \":webm"); err != nil {
			return nil, nil, err
		}

		// Get binary data
		binary, err := video.Binary()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get video binary: %w", err)
		}

		// Use File API based on decision (longer timeout for videos)
		part, fileName, err := e.processMediaFile(ctx, client, binary.ByteArray(), contentType, 0, 120*time.Second, useFileAPI)
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

		// Validate document format and get processing mode
		mode, err := validateFormat(contentType, "document", documentFormats, "", "", "")
		if err != nil {
			return nil, nil, err
		}

		switch mode {
		case "visual":
			// PDFs support full document vision capabilities
			binary, err := doc.Binary()
			if err != nil {
				return nil, nil, fmt.Errorf("failed to get document binary: %w", err)
			}

			// Process using File API decision based on total request size
			part, fileName, err := e.processMediaFile(ctx, client, binary.ByteArray(), "application/pdf", 0, 60*time.Second, useFileAPI)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to process PDF: %w", err)
			}

			parts = append(parts, part)
			if fileName != "" {
				uploadedFiles = append(uploadedFiles, fileName)
			}
		case "text":
			// Text-based documents: Extract as plain text content (no File API needed)
			textContent := doc.String()
			parts = append(parts, genai.Part{Text: textContent})
		}
	}

	return parts, uploadedFiles, nil
}
