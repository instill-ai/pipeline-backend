package data

import (
	"bytes"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
)

type Image struct {
	File
	Width  int
	Height int
}

func (Image) isValue() {}

const JPEG = "image/jpeg"
const PNG = "image/png"
const GIF = "image/gif"

func NewImageFromBytes(b []byte, contentType, fileName string) (image *Image, err error) {
	f, err := NewFileFromBytes(b, contentType, fileName)
	if err != nil {
		return
	}
	return newImage(f)
}

func NewImageFromURL(url string) (image *Image, err error) {
	f, err := NewFileFromURL(url)
	if err != nil {
		return
	}
	return newImage(f)

}
func newImage(f *File) (image *Image, err error) {
	w, h := getImageShape(f.Raw, f.ContentType)
	return &Image{
		File:   *f,
		Width:  w,
		Height: h,
	}, nil
}

func getImageShape(raw []byte, contentType string) (width, height int) {
	var img image.Image
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
	}
	return img.Bounds().Dx(), img.Bounds().Dy()
}

func (i *Image) GetWidth() *Number {
	return NewNumberFromInteger(i.Width)
}

func (i *Image) GetHeight() *Number {
	return NewNumberFromInteger(i.Height)
}

func (i *Image) Get(path string) (v Value, err error) {
	v, err = i.File.Get(path)
	if err == nil {
		return
	}
	switch {

	// TODO: we use data-url as default format for now
	case comparePath(path, ""):
		return i.GetDataURL(i.ContentType)
	case comparePath(path, ".jpeg") || comparePath(path, ".jpg"):
		return i.GetDataURL(JPEG)
	case comparePath(path, ".png"):
		return i.GetDataURL(PNG)
	case comparePath(path, ".gif"):
		return i.GetDataURL(GIF)

	case comparePath(path, ".base64"):
		return i.GetBase64(i.ContentType)
	case comparePath(path, ".base64.jpeg") || comparePath(path, ".base64.jpg"):
		return i.GetBase64(JPEG)
	case comparePath(path, ".base64.png"):
		return i.GetBase64(PNG)
	case comparePath(path, ".base64.gif"):
		return i.GetBase64(GIF)

	case comparePath(path, ".data-url"):
		return i.GetDataURL(i.ContentType)
	case comparePath(path, ".data-url.jpeg") || comparePath(path, ".data-url.jpg"):
		return i.GetDataURL(JPEG)
	case comparePath(path, ".data-url.png"):
		return i.GetDataURL(PNG)
	case comparePath(path, ".data-url.gif"):
		return i.GetDataURL(GIF)

	case comparePath(path, ".byte-array"):
		return i.GetByteArray(i.ContentType)
	case comparePath(path, ".byte-array.jpeg") || comparePath(path, ".byte-array.jpg"):
		return i.GetByteArray(JPEG)
	case comparePath(path, ".byte-array.png"):
		return i.GetByteArray(PNG)
	case comparePath(path, ".byte-array.gif"):
		return i.GetByteArray(GIF)

	case comparePath(path, ".width"):
		return i.GetWidth(), nil
	case comparePath(path, ".height"):
		return i.GetHeight(), nil
	}
	return nil, fmt.Errorf("wrong path")
}
