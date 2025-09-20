package gemini

import (
	"fmt"
	"slices"
	"strings"

	"github.com/instill-ai/pipeline-backend/pkg/data"
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
