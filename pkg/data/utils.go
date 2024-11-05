package data

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/gabriel-vasile/mimetype"

	"github.com/instill-ai/pipeline-backend/pkg/data/format"
)

func decodeDataURI(s string) (b []byte, contentType string, filename string, err error) {
	slices := strings.Split(s, ",")
	if len(slices) == 1 {
		b, err = base64.StdEncoding.DecodeString(s)
		contentType = strings.Split(mimetype.Detect(b).String(), ";")[0]
	} else {
		mime := strings.Split(slices[0], ":")
		tags := ""
		contentType, tags, _ = strings.Cut(mime[1], ";")
		b, err = base64.StdEncoding.DecodeString(slices[1])
		for _, tag := range strings.Split(tags, ";") {

			key, value, _ := strings.Cut(tag, "=")
			if key == "filename" || key == "fileName" || key == "file-name" {
				filename = value
			}
		}
	}

	return
}

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

func NewBinaryFromURL(url string) (format.Value, error) {
	b, contentType, filename, err := convertURLToBytes(url)
	if err != nil {
		return nil, err
	}

	if contentType == "" {
		contentType = strings.Split(mimetype.Detect(b).String(), ";")[0]
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
		strings.HasPrefix(contentType, "text/")
}
