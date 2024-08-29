package data

import "fmt"

type Video struct {
	File
}

func (Video) isValue() {}

func NewVideoFromBytes(b []byte, contentType, fileName string) (video *Video, err error) {
	f, err := NewFileFromBytes(b, contentType, fileName)
	if err != nil {
		return
	}
	return newVideo(f)
}

func NewVideoFromURL(url string) (video *Video, err error) {
	f, err := NewFileFromURL(url)
	if err != nil {
		return
	}
	return newVideo(f)
}

func newVideo(f *File) (video *Video, err error) {
	return &Video{
		File: *f,
	}, nil
}

func (vid *Video) Get(path string) (v Value, err error) {
	v, err = vid.File.Get(path)
	if err == nil {
		return
	}
	switch {

	// TODO: we use data-url as default format for now
	case comparePath(path, ""):
		return vid.GetDataURL(vid.ContentType)

	}
	return nil, fmt.Errorf("wrong path")
}
