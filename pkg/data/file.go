package data

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gabriel-vasile/mimetype"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/operator/document/v0"
	"github.com/instill-ai/pipeline-backend/pkg/data/value"
)

type fileData struct {
	raw         []byte
	contentType string
	fileName    string
	sourceURL   string
	cache       map[string][]byte
}

func (fileData) IsValue() {}

func NewFileFromBytes(b []byte, contentType, fileName string) (bin *fileData, err error) {
	if contentType == "" {
		contentType = strings.Split(mimetype.Detect(b).String(), ";")[0]
	}
	cache := map[string][]byte{}
	cache[contentType] = b
	return &fileData{
		raw:         b,
		contentType: contentType,
		fileName:    fileName,
		cache:       cache,
	}, nil
}

func NewFileFromURL(url string) (bin *fileData, err error) {
	if strings.HasPrefix(url, "data:") {
		return newFileFromDataURI(url)
	}

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	contentType := ""
	if headers, ok := resp.Header["Content-Type"]; ok && len(headers) > 0 {
		contentType = headers[0]
	}

	bin, err = NewFileFromBytes(body, contentType, "")
	if err != nil {
		return nil, err
	}
	bin.sourceURL = url
	return bin, nil
}

func newFileFromDataURI(url string) (bin *fileData, err error) {
	b, contentType, fileName, err := decodeDataURI(url)
	if err != nil {
		return
	}
	cache := map[string][]byte{}
	cache[contentType] = b
	return &fileData{
		raw:         b,
		contentType: contentType,
		fileName:    fileName,
		cache:       cache,
	}, nil
}

func (f *fileData) Binary(contentType string) (ba *byteArrayData, err error) {
	if c, ok := f.cache[contentType]; ok {
		return NewByteArray(c), nil
	}

	b, err := convertFile(f.raw, f.contentType, contentType)
	if err != nil {
		return nil, fmt.Errorf("can not convert data from %s to %s", f.contentType, contentType)
	}
	f.cache[contentType] = b
	return NewByteArray(b), nil
}

func (f *fileData) String() (val string) {

	// TODO: Refactor to share implementation with Document format
	dataURI, err := f.DataURI(f.contentType)
	if err != nil {
		return ""
	}

	res, err := document.ConvertDocumentToMarkdown(
		&document.ConvertDocumentToMarkdownInput{
			Document: dataURI.String(),
			Filename: f.fileName,
		}, document.GetMarkdownTransformer)
	if err != nil {
		return ""
	}
	return res.Body
}

func (f *fileData) DataURI(contentType string) (url *stringData, err error) {
	ba, err := f.Binary(contentType)
	if err != nil {
		return
	}
	s, err := encodeDataURI(ba.ByteArray(), contentType)
	if err != nil {
		return
	}
	return NewString(s), nil
}

func (f *fileData) GetBase64(contentType string) (b64 *stringData, err error) {
	ba, err := f.DataURI(contentType)
	if err != nil {
		return
	}
	_, b64str, _ := strings.Cut(ba.String(), ",")
	return NewString(b64str), nil
}

func (f *fileData) FileSize() (size *numberData) {
	return NewNumberFromInteger(len(f.raw))
}

func (f *fileData) ContentType() (t *stringData) {
	return NewString(f.contentType)
}

func (f *fileData) FileName() (t *stringData) {
	return NewString(f.fileName)
}

func (f *fileData) SourceURL() (t *stringData) {
	return NewString(f.sourceURL)
}

func (f *fileData) Get(path string) (v value.Value, err error) {
	switch {
	case comparePath(path, ".source-url"):
		return f.SourceURL(), nil
	case comparePath(path, ".filename"):
		return f.FileName(), nil
	case comparePath(path, ".file-size"):
		return f.FileSize(), nil
	case comparePath(path, ".content-type"):
		return f.ContentType(), nil
	}
	return nil, fmt.Errorf("wrong path")
}

func (f fileData) ToStructValue() (v *structpb.Value, err error) {
	d, err := f.DataURI(f.contentType)
	if err != nil {
		return nil, err
	}
	return structpb.NewStringValue(d.String()), nil
}
