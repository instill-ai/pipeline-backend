package data

import (
	"bytes"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"strings"
)

type Image struct {
	File
	Width  int
	Height int
}

func (*Image) isValue() {}

const JPEG = "image/jpeg"
const PNG = "image/png"
const GIF = "image/gif"

func NewImageFromBytes(b []byte, contentType, fileName string) (image *Image, err error) {
	f, err := NewFileFromBytes(b, contentType, fileName, convertImage)
	if err != nil {
		return
	}
	return newImage(f)
}

func NewImageFromURL(url string) (image *Image, err error) {
	f, err := NewFileFromURL(url, convertImage)
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

func convertImage(raw []byte, sourceContentType, targetContentType string) (b []byte, err error) {
	var img image.Image
	switch sourceContentType {
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
	default:
		return nil, fmt.Errorf("conversion error")
	}

	buf := new(bytes.Buffer)
	switch targetContentType {
	case PNG:
		if err = png.Encode(buf, img); err != nil {
			return
		}
	case JPEG:
		if err = jpeg.Encode(buf, img, nil); err != nil {
			return
		}
	case GIF:
		if err = gif.Encode(buf, img, nil); err != nil {
			return
		}
	default:
		return nil, fmt.Errorf("conversion error")
	}
	return buf.Bytes(), nil
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
	case path == "":
		// TODO: we use data-url for now
		return i.GetDataURL(i.ContentType)
	case path == ".width":
		return i.GetWidth(), nil
	case path == ".height":
		return i.GetHeight(), nil
	case strings.HasPrefix(path, ".base64"):
		ss := strings.Split(path, ".")
		if len(ss) > 2 {
			switch ss[2] {
			case "jpeg", "jpg":
				return i.GetBase64(JPEG)
			case "png":
				return i.GetBase64(PNG)
			case "gif":
				return i.GetBase64(GIF)
			}
		}
		return i.GetBase64(i.ContentType)
	case strings.HasPrefix(path, ".data-url"):
		ss := strings.Split(path, ".")
		if len(ss) > 2 {
			switch ss[2] {
			case "jpeg", "jpg":
				return i.GetDataURL(JPEG)
			case "png":
				return i.GetDataURL(PNG)
			case "gif":
				return i.GetDataURL(GIF)
			}
		}
		return i.GetDataURL(i.ContentType)
	case strings.HasPrefix(path, ".byte-array"):
		ss := strings.Split(path, ".")
		if len(ss) > 2 {
			switch ss[2] {
			case "jpeg", "jpg":
				return i.GetByteArray(JPEG)
			case "png":
				return i.GetByteArray(PNG)
			case "gif":
				return i.GetByteArray(GIF)
			}
		}
		return i.GetByteArray(i.ContentType)
	}
	return nil, fmt.Errorf("wrong path")
}
