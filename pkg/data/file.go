package data

import (
	"bytes"
	"fmt"
	"mime"
	"strings"
	"unicode/utf8"

	"github.com/gabriel-vasile/mimetype"
	"github.com/go-resty/resty/v2"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/data/format"
	"github.com/instill-ai/pipeline-backend/pkg/data/path"
)

const (
	OCTETSTREAM = "application/octet-stream"
)

var fileGetters = map[string]func(*fileData) (format.Value, error){
	"source-url":   func(f *fileData) (format.Value, error) { return f.SourceURL(), nil },
	"filename":     func(f *fileData) (format.Value, error) { return f.Filename(), nil },
	"file-size":    func(f *fileData) (format.Value, error) { return f.FileSize(), nil },
	"content-type": func(f *fileData) (format.Value, error) { return f.ContentType(), nil },
	"binary":       func(f *fileData) (format.Value, error) { return f.Binary() },
	"data-uri":     func(f *fileData) (format.Value, error) { return f.DataURI() },
	"base64":       func(f *fileData) (format.Value, error) { return f.Base64() },
}

type fileData struct {
	raw         []byte
	contentType string
	filename    string
	sourceURL   string
}

func (fileData) IsValue() {}

func NewFileFromBytes(b []byte, contentType, filename string) (bin *fileData, err error) {
	if contentType == "" {
		contentType = strings.Split(mimetype.Detect(b).String(), ";")[0]
	}

	f := &fileData{
		raw:         b,
		contentType: contentType,
		filename:    filename,
	}

	return f, nil
}

func convertURLToBytes(url string) (b []byte, contentType string, filename string, err error) {
	if strings.HasPrefix(url, "data:") {
		return convertDataURIToBytes(url)
	}

	client := resty.New().SetRetryCount(3)
	resp, err := client.R().Get(url)
	if err != nil {
		return nil, "", "", err
	}
	body := resp.Body()
	contentType = ""
	if headers := resp.Header().Get("Content-Type"); headers != "" {
		contentType = headers
	}
	filename = ""
	if disposition := resp.Header().Get("Content-Disposition"); disposition != "" {
		if strings.HasPrefix(disposition, "attachment") {
			if _, params, err := mime.ParseMediaType(disposition); err == nil {
				if fn, ok := params["filename"]; ok {
					filename = fn
				}
			}
		}
	}
	return body, contentType, filename, nil
}

func NewFileFromURL(url string) (bin *fileData, err error) {
	b, contentType, filename, err := convertURLToBytes(url)
	if err != nil {
		return nil, err
	}
	bin, err = NewFileFromBytes(b, contentType, filename)
	if err != nil {
		return nil, err
	}
	bin.sourceURL = url
	return bin, nil
}

func convertDataURIToBytes(url string) (b []byte, contentType string, filename string, err error) {
	b, contentType, filename, err = decodeDataURI(url)
	if err != nil {
		return
	}
	return b, contentType, filename, nil
}

func (f *fileData) String() string {
	if strings.HasPrefix(f.contentType, "text/") || utf8.Valid(f.raw) {
		return string(f.raw)
	}

	// If the file is not a text file, convert it to a data URI
	dataURI, err := f.DataURI()
	if err != nil {
		return ""
	}
	return dataURI.String()
}

func (f *fileData) Binary() (ba format.ByteArray, err error) {
	return NewByteArray(f.raw), nil
}

func (f *fileData) DataURI() (url format.String, err error) {
	ba, err := f.Binary()
	if err != nil {
		return
	}
	s, err := encodeDataURI(ba.ByteArray(), f.contentType)
	if err != nil {
		return
	}
	return NewString(s), nil
}

func (f *fileData) Base64() (b64 format.String, err error) {
	ba, err := f.DataURI()
	if err != nil {
		return
	}
	_, b64str, _ := strings.Cut(ba.String(), ",")
	return NewString(b64str), nil
}

func (f *fileData) FileSize() (size format.Number) {
	return NewNumberFromInteger(len(f.raw))
}

func (f *fileData) ContentType() (t format.String) {
	return NewString(f.contentType)
}

func (f *fileData) Filename() (t format.String) {
	return NewString(f.filename)
}

func (f *fileData) SourceURL() (t format.String) {
	return NewString(f.sourceURL)
}

func (f *fileData) Get(p *path.Path) (v format.Value, err error) {
	if p == nil || p.IsEmpty() {
		return f, nil
	}

	firstSeg, remainingPath, err := p.TrimFirst()
	if err != nil {
		return nil, err
	}

	if firstSeg.SegmentType != path.AttributeSegment {
		return nil, fmt.Errorf("path not found: %s", p)
	}

	getter, exists := fileGetters[firstSeg.Attribute]
	if !exists {
		return nil, fmt.Errorf("path not found: %s", p)
	}

	result, err := getter(f)
	if err != nil {
		return nil, err
	}

	if remainingPath.IsEmpty() {
		return result, nil
	}

	return result.Get(remainingPath)
}

// Deprecated: ToStructValue() is deprecated and will be removed in a future
// version. structpb is not suitable for handling binary data and will be phased
// out gradually.
func (f fileData) ToStructValue() (v *structpb.Value, err error) {
	d, err := f.DataURI()
	if err != nil {
		return nil, err
	}
	return structpb.NewStringValue(d.String()), nil
}

func (f *fileData) Equal(other format.Value) bool {
	if other, ok := other.(format.File); ok {
		ba, err := other.Binary()
		if err != nil {
			return false
		}
		return bytes.Equal(f.raw, ba.ByteArray()) &&
			f.contentType == other.ContentType().String() &&
			f.filename == other.Filename().String() &&
			f.sourceURL == other.SourceURL().String()
	}
	return false
}
