package data

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"

	"github.com/gabriel-vasile/mimetype"

	"github.com/instill-ai/pipeline-backend/pkg/data/format"
	"github.com/instill-ai/pipeline-backend/pkg/external"
)

func encodeDataURI(b []byte, contentType string) (s string, err error) {
	s = fmt.Sprintf("data:%s;base64,%s", contentType, base64.StdEncoding.EncodeToString(b))
	return
}

// normalizeMIMEType maps non-standard MIME types to their standard equivalents
func normalizeMIMEType(contentType string) string {
	switch contentType {
	// Audio normalizations
	case "audio/mp3":
		return MP3 // Maps to "audio/mpeg"
	case "audio/x-m4a":
		return M4A // Maps to "audio/mp4"

	// Video normalizations
	case "video/mov":
		return MOV // Maps to "video/quicktime"
	case "video/avi":
		return AVI // Maps to "video/x-msvideo"
	case "video/wmv":
		return WMV // Maps to "video/x-ms-wmv"
	case "video/mpg":
		return MPEG // Maps to "video/mpeg"
	case "video/3gpp":
		return MP4 // 3GPP files are typically MP4 containers

	// Image normalizations
	case "image/x-heic":
		return HEIC // Maps to "image/heic"
	case "image/x-heif":
		return HEIF // Maps to "image/heif"
	case "image/x-avif":
		return AVIF // Maps to "image/avif"

	default:
		return contentType
	}
}

// NewBinaryFromBytes creates a new binary from a byte slice.
func NewBinaryFromBytes(b []byte, contentType, filename string) (format.Value, error) {
	if contentType == "" {
		contentType = strings.Split(mimetype.Detect(b).String(), ";")[0]
	} else {
		// Normalize provided content type: strip parameters and lowercase
		contentType = strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
		// Normalize non-standard MIME types to standard ones
		contentType = normalizeMIMEType(contentType)
	}

	switch {
	case isImageContentType(contentType):
		return NewImageFromBytes(b, contentType, filename, true)
	case isAudioContentType(contentType):
		return NewAudioFromBytes(b, contentType, filename, true)
	case isVideoContentType(contentType):
		return NewVideoFromBytes(b, contentType, filename, true)
	case isDocumentContentType(contentType):
		return NewDocumentFromBytes(b, contentType, filename)
	default:
		return NewFileFromBytes(b, contentType, filename)
	}
}

// NewBinaryFromURL creates a new binary from a URL.
func NewBinaryFromURL(ctx context.Context, binaryFetcher external.BinaryFetcher, urlStr string) (format.Value, error) {
	b, contentType, filename, err := binaryFetcher.FetchFromURL(ctx, urlStr)
	if err != nil {
		return nil, err
	}

	// If no filename was extracted, try to get it from the URL
	if filename == "" {
		if parsedURL, err := url.Parse(urlStr); err == nil {
			if path := parsedURL.Path; path != "" {
				// Extract filename from URL path
				if lastSlash := strings.LastIndex(path, "/"); lastSlash != -1 && lastSlash < len(path)-1 {
					filename = path[lastSlash+1:]
					// Remove query parameters if present
					if questionMark := strings.Index(filename, "?"); questionMark != -1 {
						filename = filename[:questionMark]
					}
				}
			}
		}
	}

	if contentType == "" {
		contentType = strings.Split(mimetype.Detect(b).String(), ";")[0]
	} else {
		// Normalize provided content type: strip parameters and lowercase
		contentType = strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
		// Normalize non-standard MIME types to standard ones
		contentType = normalizeMIMEType(contentType)
	}

	switch {
	case isImageContentType(contentType):
		return NewImageFromBytes(b, contentType, filename, false)
	case isAudioContentType(contentType):
		return NewAudioFromBytes(b, contentType, filename, true)
	case isVideoContentType(contentType):
		return NewVideoFromBytes(b, contentType, filename, true)
	case isDocumentContentType(contentType):
		return NewDocumentFromBytes(b, contentType, filename)
	default:
		return NewFileFromBytes(b, contentType, filename)
	}
}

func isImageContentType(contentType string) bool {
	return contentType == JPEG ||
		contentType == PNG ||
		contentType == GIF ||
		contentType == BMP ||
		contentType == WEBP ||
		contentType == TIFF ||
		contentType == HEIC ||
		contentType == HEIF ||
		contentType == AVIF
}

func isAudioContentType(contentType string) bool {
	return contentType == AIFF ||
		contentType == MP3 ||
		contentType == WAV ||
		contentType == AAC ||
		contentType == OGG ||
		contentType == FLAC ||
		contentType == M4A ||
		contentType == WMA ||
		contentType == WEBMAUDIO
}

func isVideoContentType(contentType string) bool {
	return contentType == MPEG ||
		contentType == AVI ||
		contentType == MOV ||
		contentType == MKV ||
		contentType == FLV ||
		contentType == WMV ||
		contentType == MP4 ||
		contentType == WEBMVIDEO
}

func isDocumentContentType(contentType string) bool {
	return contentType == DOC ||
		contentType == DOCX ||
		contentType == PPT ||
		contentType == PPTX ||
		contentType == XLS ||
		contentType == XLSX ||
		contentType == HTML ||
		contentType == PLAIN ||
		contentType == TEXT ||
		contentType == MARKDOWN ||
		contentType == CSV ||
		contentType == PDF ||
		contentType == OLE ||
		strings.HasPrefix(contentType, "text/")
}
