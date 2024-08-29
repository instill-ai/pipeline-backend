package data

import "fmt"

type Audio struct {
	File
}

func (Audio) isValue() {}

func NewAudioFromBytes(b []byte, contentType, fileName string) (audio *Audio, err error) {
	f, err := NewFileFromBytes(b, contentType, fileName)
	if err != nil {
		return
	}
	return newAudio(f)
}

func NewAudioFromURL(url string) (audio *Audio, err error) {
	f, err := NewFileFromURL(url)
	if err != nil {
		return
	}
	return newAudio(f)
}

func newAudio(f *File) (audio *Audio, err error) {
	return &Audio{
		File: *f,
	}, nil
}

func (a *Audio) Get(path string) (v Value, err error) {
	v, err = a.File.Get(path)
	if err == nil {
		return
	}
	switch {

	// TODO: we use data-url as default format for now
	case comparePath(path, ""):
		return a.GetDataURL(a.ContentType)

	}
	return nil, fmt.Errorf("wrong path")
}
