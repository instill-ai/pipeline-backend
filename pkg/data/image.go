package data

import (
	"bytes"
	"fmt"
	gif "image/gif"
	jpeg "image/jpeg"
	png "image/png"

	goimage "image"

	webp "golang.org/x/image/webp"

	"github.com/instill-ai/pipeline-backend/pkg/data/value"
)

type imageData struct {
	fileData
	width  int
	height int
}

func (imageData) IsValue() {}

const JPEG = "image/jpeg"
const PNG = "image/png"
const GIF = "image/gif"
const WEBP = "image/webp"

func NewImageFromBytes(b []byte, contentType, fileName string) (img *imageData, err error) {
	f, err := NewFileFromBytes(b, contentType, fileName)
	if err != nil {
		return
	}
	return newImage(f)
}

func NewImageFromURL(url string) (img *imageData, err error) {
	f, err := NewFileFromURL(url)
	if err != nil {
		return
	}
	return newImage(f)

}
func newImage(f *fileData) (img *imageData, err error) {
	w, h := getImageShape(f.raw, f.contentType)
	return &imageData{
		fileData: *f,
		width:    w,
		height:   h,
	}, nil
}

func getImageShape(raw []byte, contentType string) (width, height int) {
	var img goimage.Image
	var err error
	switch contentType {
	case PNG:
		if img, err = png.Decode(bytes.NewReader(raw)); err != nil {
			return
		}
	case JPEG:
		if img, err = jpeg.Decode(bytes.NewReader(raw)); err != nil {
			return
		}
	case GIF:
		if img, err = gif.Decode(bytes.NewReader(raw)); err != nil {
			return
		}
	case WEBP:
		if img, err = webp.Decode(bytes.NewReader(raw)); err != nil {
			return
		}

	}
	return img.Bounds().Dx(), img.Bounds().Dy()
}

func (i *imageData) Width() *numberData {
	return NewNumberFromInteger(i.width)
}

func (i *imageData) Height() *numberData {
	return NewNumberFromInteger(i.height)
}

func (i *imageData) Get(path string) (v value.Value, err error) {
	v, err = i.fileData.Get(path)
	if err == nil {
		return
	}
	switch {

	// TODO: we use data-uri as default format for now
	case comparePath(path, ""):
		return i, nil
	case comparePath(path, ".jpeg") || comparePath(path, ".jpg"):
		return i.DataURI(JPEG)
	case comparePath(path, ".png"):
		return i.DataURI(PNG)
	case comparePath(path, ".gif"):
		return i.DataURI(GIF)
	case comparePath(path, ".webp"):
		return i.DataURI(WEBP)

	case comparePath(path, ".base64"):
		return i.GetBase64(i.contentType)
	case comparePath(path, ".base64.jpeg") || comparePath(path, ".base64.jpg"):
		return i.GetBase64(JPEG)
	case comparePath(path, ".base64.png"):
		return i.GetBase64(PNG)
	case comparePath(path, ".base64.gif"):
		return i.GetBase64(GIF)
	case comparePath(path, ".base64.webp"):
		return i.GetBase64(WEBP)

	case comparePath(path, ".data-uri"):
		return i.DataURI(i.contentType)
	case comparePath(path, ".data-uri.jpeg") || comparePath(path, ".data-uri.jpg"):
		return i.DataURI(JPEG)
	case comparePath(path, ".data-uri.png"):
		return i.DataURI(PNG)
	case comparePath(path, ".data-uri.gif"):
		return i.DataURI(GIF)
	case comparePath(path, ".data-uri.webp"):
		return i.DataURI(WEBP)

	case comparePath(path, ".byte-array"):
		return i.Binary(i.contentType)
	case comparePath(path, ".byte-array.jpeg") || comparePath(path, ".byte-array.jpg"):
		return i.Binary(JPEG)
	case comparePath(path, ".byte-array.png"):
		return i.Binary(PNG)
	case comparePath(path, ".byte-array.gif"):
		return i.Binary(GIF)
	case comparePath(path, ".byte-array.webp"):
		return i.Binary(WEBP)

	case comparePath(path, ".width"):
		return i.Width(), nil
	case comparePath(path, ".height"):
		return i.Height(), nil
	}
	return nil, fmt.Errorf("wrong path")
}
