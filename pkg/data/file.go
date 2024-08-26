package data

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gabriel-vasile/mimetype"
	"google.golang.org/protobuf/types/known/structpb"
)

type File struct {
	Raw         []byte
	ContentType string
	FileName    string
	SourceURL   string
	Cache       map[string][]byte
}

func NewFileFromBytes(b []byte, contentType, fileName string) (bin *File, err error) {
	if contentType == "" {
		contentType = strings.Split(mimetype.Detect(b).String(), ";")[0]
	}
	cache := map[string][]byte{}
	cache[contentType] = b
	return &File{
		Raw:         b,
		ContentType: contentType,
		FileName:    fileName,
		Cache:       cache,
	}, nil
}

func NewFileFromURL(url string) (bin *File, err error) {
	if strings.HasPrefix(url, "data:") {
		return newFileFromDataURL(url)
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
	bin.SourceURL = url
	return bin, nil
}

func newFileFromDataURL(url string) (bin *File, err error) {
	b, contentType, fileName, err := decodeDataURL(url)
	if err != nil {
		return
	}
	cache := map[string][]byte{}
	cache[contentType] = b
	return &File{
		Raw:         b,
		ContentType: contentType,
		FileName:    fileName,
		Cache:       cache,
	}, nil
}

func (f *File) GetByteArray(contentType string) (ba *ByteArray, err error) {
	if c, ok := f.Cache[contentType]; ok {
		return NewByteArray(c), nil
	}

	b, err := convertFile(f.Raw, f.ContentType, contentType)
	if err != nil {
		return nil, fmt.Errorf("can not convert data from %s to %s", f.ContentType, contentType)
	}
	f.Cache[contentType] = b
	return NewByteArray(b), nil
}

func (f *File) GetDataURL(contentType string) (url *String, err error) {
	ba, err := f.GetByteArray(contentType)
	if err != nil {
		return
	}
	s, err := encodeDataURL(ba.GetByteArray(), contentType)
	if err != nil {
		return
	}
	return NewString(s), nil
}

func (f *File) GetBase64(contentType string) (b64 *String, err error) {
	ba, err := f.GetDataURL(contentType)
	if err != nil {
		return
	}
	_, b64str, _ := strings.Cut(ba.GetString(), ",")
	return NewString(b64str), nil
}

func (f *File) GetFileSize() (size *Number) {
	return NewNumberFromInteger(len(f.Raw))
}

func (f *File) GetContentType() (t *String) {
	return NewString(f.ContentType)
}

func (f *File) GetFileName() (t *String) {
	return NewString(f.FileName)
}

func (f *File) GetSourceURL() (t *String) {
	return NewString(f.SourceURL)
}

func (f *File) Get(path string) (v Value, err error) {
	switch {
	case comparePath(path, ".source-url"):
		return f.GetSourceURL(), nil
	case comparePath(path, ".file-name"):
		return f.GetFileName(), nil
	case comparePath(path, ".file-size"):
		return f.GetFileSize(), nil
	case comparePath(path, ".content-type"):
		return f.GetContentType(), nil
	}
	return nil, fmt.Errorf("wrong path")
}

func (f File) ToStructValue() (v *structpb.Value, err error) {
	d, err := f.GetDataURL(f.ContentType)
	if err != nil {
		return nil, err
	}
	return structpb.NewStringValue(d.GetString()), nil
}
