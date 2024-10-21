package data

import (
	"fmt"

	"github.com/instill-ai/pipeline-backend/pkg/data/value"
)

type videoData struct {
	fileData
}

func (videoData) IsValue() {}

func NewVideoFromBytes(b []byte, contentType, fileName string) (video *videoData, err error) {
	f, err := NewFileFromBytes(b, contentType, fileName)
	if err != nil {
		return
	}
	return newVideo(f)
}

func NewVideoFromURL(url string) (video *videoData, err error) {
	f, err := NewFileFromURL(url)
	if err != nil {
		return
	}
	return newVideo(f)
}

func newVideo(f *fileData) (video *videoData, err error) {
	return &videoData{
		fileData: *f,
	}, nil
}

func (vid *videoData) Get(path string) (v value.Value, err error) {
	v, err = vid.fileData.Get(path)
	if err == nil {
		return
	}
	switch {

	// TODO: we use data-uri as default format for now
	case comparePath(path, ""):
		return vid, nil

	}
	return nil, fmt.Errorf("wrong path")
}
