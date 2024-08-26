package data

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
