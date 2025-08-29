package data

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/gabriel-vasile/mimetype"

	"github.com/instill-ai/pipeline-backend/pkg/data/format"
	"github.com/instill-ai/pipeline-backend/pkg/external"
)

func encodeDataURI(b []byte, contentType string) (s string, err error) {
	s = fmt.Sprintf("data:%s;base64,%s", contentType, base64.StdEncoding.EncodeToString(b))
	return
}

func StandardizePath(path string) (newPath string, err error) {
	splits := strings.FieldsFunc(path, func(s rune) bool {
		return s == '.' || s == '['
	})
	for _, split := range splits {
		if strings.HasSuffix(split, "]") {
			// Array Index
			newPath += fmt.Sprintf("[%s", split)
		} else {
			// Map Key
			newPath += fmt.Sprintf("[\"%s\"]", split)
		}
	}
	return newPath, err
}
func NewBinaryFromBytes(b []byte, contentType, filename string) (format.Value, error) {
	if contentType == "" {
		contentType = strings.Split(mimetype.Detect(b).String(), ";")[0]
	} else {
		// Normalize provided content type: strip parameters and lowercase
		contentType = strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	}

	switch {
	case isImageContentType(contentType):
		return NewImageFromBytes(b, contentType, filename)
	case isAudioContentType(contentType):
		return NewAudioFromBytes(b, contentType, filename)
	case isVideoContentType(contentType):
		return NewVideoFromBytes(b, contentType, filename)
	case isDocumentContentType(contentType):
		return NewDocumentFromBytes(b, contentType, filename)
	default:
		return NewFileFromBytes(b, contentType, filename)
	}
}

func NewBinaryFromURL(ctx context.Context, binaryFetcher external.BinaryFetcher, url string) (format.Value, error) {
	b, contentType, filename, err := binaryFetcher.FetchFromURL(ctx, url)
	if err != nil {
		return nil, err
	}

	if contentType == "" {
		contentType = strings.Split(mimetype.Detect(b).String(), ";")[0]
	} else {
		// Normalize provided content type: strip parameters and lowercase
		contentType = strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	}

	switch {
	case isImageContentType(contentType):
		return NewImageFromBytes(b, contentType, filename)
	case isAudioContentType(contentType):
		return NewAudioFromBytes(b, contentType, filename)
	case isVideoContentType(contentType):
		return NewVideoFromBytes(b, contentType, filename)
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
		contentType == TIFF
}

func isAudioContentType(contentType string) bool {
	return contentType == AIFF ||
		contentType == MP3 ||
		contentType == WAV ||
		contentType == AAC ||
		contentType == OGG ||
		contentType == FLAC ||
		contentType == M4A ||
		contentType == WMA
}

func isVideoContentType(contentType string) bool {
	return contentType == MPEG ||
		contentType == AVI ||
		contentType == MOV ||
		contentType == WEBM ||
		contentType == MKV ||
		contentType == FLV ||
		contentType == WMV ||
		contentType == MP4
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
