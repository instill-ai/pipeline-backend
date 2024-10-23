package data

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"

	"github.com/gabriel-vasile/mimetype"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/data/format"
	"github.com/instill-ai/pipeline-backend/pkg/data/path"
)

const (
	OCTETSTREAM = "application/octet-stream"
)

var fileGetters = map[string]func(*fileData) (format.Value, error){
	"source-url":   func(f *fileData) (format.Value, error) { return f.SourceURL(), nil },
	"filename":     func(f *fileData) (format.Value, error) { return f.FileName(), nil },
	"file-size":    func(f *fileData) (format.Value, error) { return f.FileSize(), nil },
	"content-type": func(f *fileData) (format.Value, error) { return f.ContentType(), nil },
	"binary":       func(f *fileData) (format.Value, error) { return f.Binary() },
	"data-uri":     func(f *fileData) (format.Value, error) { return f.DataURI() },
	"base64":       func(f *fileData) (format.Value, error) { return f.Base64() },
}

type fileData struct {
	raw         []byte
	contentType string
	fileName    string
	sourceURL   string
}

func (fileData) IsValue() {}

func NewFileFromBytes(b []byte, contentType, fileName string) (bin *fileData, err error) {
	if contentType == "" {
		contentType = strings.Split(mimetype.Detect(b).String(), ";")[0]
	}

	f := &fileData{
		raw:         b,
		contentType: contentType,
		fileName:    fileName,
	}

	return f, nil
}

func convertURLToBytes(url string) (b []byte, contentType string, fileName string, err error) {
	if strings.HasPrefix(url, "data:") {
		return convertDataURIToBytes(url)
	}

	resp, err := http.Get(url)
	if err != nil {
		return nil, "", "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", "", err
	}
	contentType = ""
	if headers, ok := resp.Header["Content-Type"]; ok && len(headers) > 0 {
		contentType = headers[0]
	}
	fileName = ""
	if headers, ok := resp.Header["Content-Disposition"]; ok && len(headers) > 0 {
		if disposition := headers[0]; strings.HasPrefix(disposition, "attachment") {
			if _, params, err := mime.ParseMediaType(disposition); err == nil {
				if fn, ok := params["filename"]; ok {
					fileName = fn
				}
			}
		}
	}
	return body, contentType, fileName, nil
}

func NewFileFromURL(url string) (bin *fileData, err error) {
	b, contentType, fileName, err := convertURLToBytes(url)
	if err != nil {
		return nil, err
	}
	bin, err = NewFileFromBytes(b, contentType, fileName)
	if err != nil {
		return nil, err
	}
	bin.sourceURL = url
	return bin, nil
}

func convertDataURIToBytes(url string) (b []byte, contentType string, fileName string, err error) {
	b, contentType, fileName, err = decodeDataURI(url)
	if err != nil {
		return
	}
	return b, contentType, fileName, nil
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

func (f *fileData) FileName() (t format.String) {
	return NewString(f.fileName)
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

func (f fileData) ToStructValue() (v *structpb.Value, err error) {
	d, err := f.DataURI()
	if err != nil {
		return nil, err
	}
	return structpb.NewStringValue(d.String()), nil
}
