package data

type Video struct {
	File
}

func (Video) isValue() {}

func NewVideoFromBytes(b []byte, contentType, fileName string) (video *Video, err error) {
	f, err := NewFileFromBytes(b, contentType, fileName, nil)
	if err != nil {
		return
	}
	return newVideo(f)
}

func NewVideoFromURL(url string) (video *Video, err error) {
	f, err := NewFileFromURL(url, nil)
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
