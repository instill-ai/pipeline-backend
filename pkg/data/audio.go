package data

import (
	"fmt"

	"github.com/instill-ai/pipeline-backend/pkg/data/value"
)

type audioData struct {
	fileData
}

func (audioData) IsValue() {}

func NewAudioFromBytes(b []byte, contentType, fileName string) (a *audioData, err error) {
	f, err := NewFileFromBytes(b, contentType, fileName)
	if err != nil {
		return
	}
	return newAudio(f)
}

func NewAudioFromURL(url string) (a *audioData, err error) {
	f, err := NewFileFromURL(url)
	if err != nil {
		return
	}
	return newAudio(f)
}

func newAudio(f *fileData) (a *audioData, err error) {
	return &audioData{
		fileData: *f,
	}, nil
}

func (a *audioData) Get(path string) (v value.Value, err error) {
	v, err = a.fileData.Get(path)
	if err == nil {
		return
	}
	switch {

	// TODO: we use data-uri as default format for now
	case comparePath(path, ""):
		return a, nil

	}
	return nil, fmt.Errorf("wrong path")
}
