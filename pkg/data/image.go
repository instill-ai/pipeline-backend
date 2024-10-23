package data

import (
	"bytes"
	"fmt"
	"image/gif"
	"image/jpeg"
	"image/png"

	"golang.org/x/image/bmp"
	"golang.org/x/image/tiff"
	"golang.org/x/image/webp"

	goimage "image"

	"github.com/instill-ai/pipeline-backend/pkg/data/format"
	"github.com/instill-ai/pipeline-backend/pkg/data/path"
)

type imageData struct {
	fileData
	width  int
	height int
}

func (imageData) IsValue() {}

const (
	JPEG = "image/jpeg"
	PNG  = "image/png"
	GIF  = "image/gif"
	WEBP = "image/webp"
	TIFF = "image/tiff"
	BMP  = "image/bmp"
)

var imageGetters = map[string]func(*imageData) (format.Value, error){
	"width":  func(i *imageData) (format.Value, error) { return i.Width(), nil },
	"height": func(i *imageData) (format.Value, error) { return i.Height(), nil },
	"jpeg":   func(i *imageData) (format.Value, error) { return i.Convert(JPEG) },
	"png":    func(i *imageData) (format.Value, error) { return i.Convert(PNG) },
	"gif":    func(i *imageData) (format.Value, error) { return i.Convert(GIF) },
	"webp":   func(i *imageData) (format.Value, error) { return i.Convert(WEBP) },
	"tiff":   func(i *imageData) (format.Value, error) { return i.Convert(TIFF) },
	"bmp":    func(i *imageData) (format.Value, error) { return i.Convert(BMP) },
}

// NewImageFromBytes creates a new imageData from byte slice
func NewImageFromBytes(b []byte, contentType, fileName string) (*imageData, error) {
	return createImageData(b, contentType, fileName)
}

// NewImageFromURL creates a new imageData from a URL
func NewImageFromURL(url string) (*imageData, error) {
	b, contentType, fileName, err := convertURLToBytes(url)
	if err != nil {
		return nil, err
	}
	return createImageData(b, contentType, fileName)
}

// createImageData is a helper function to create imageData
func createImageData(b []byte, contentType, fileName string) (*imageData, error) {
	b, err := convertImage(b, contentType, PNG)
	if err != nil {
		return nil, err
	}

	f, err := NewFileFromBytes(b, PNG, fileName)
	if err != nil {
		return nil, err
	}

	return newImage(f)
}

// newImage creates a new imageData from file data
func newImage(f *fileData) (*imageData, error) {
	w, h := getImageProperties(f.raw, f.contentType)
	i := &imageData{
		fileData: *f,
		width:    w,
		height:   h,
	}

	return i, nil
}

func getImageProperties(raw []byte, contentType string) (width, height int) {
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
	case TIFF:
		if img, err = tiff.Decode(bytes.NewReader(raw)); err != nil {
			return
		}
	case BMP:
		if img, err = bmp.Decode(bytes.NewReader(raw)); err != nil {
			return
		}
	}
	if img == nil {
		return
	}
	return img.Bounds().Dx(), img.Bounds().Dy()
}

func (i *imageData) Width() format.Number {
	return NewNumberFromInteger(i.width)
}

func (i *imageData) Height() format.Number {
	return NewNumberFromInteger(i.height)
}

func (i *imageData) Convert(contentType string) (format.Image, error) {
	b, err := convertImage(i.raw, i.contentType, contentType)
	if err != nil {
		return nil, fmt.Errorf("can not convert data from %s to %s", i.contentType, contentType)
	}
	f, err := NewFileFromBytes(b, contentType, "")
	if err != nil {
		return nil, fmt.Errorf("can not convert data from %s to %s", i.contentType, contentType)
	}
	return newImage(f)
}

func (i *imageData) Get(p *path.Path) (v format.Value, err error) {
	if p == nil || p.IsEmpty() {
		return i, nil
	}

	firstSeg, remainingPath, err := p.TrimFirst()
	if err != nil {
		return nil, err
	}

	if firstSeg.SegmentType != path.AttributeSegment {
		return nil, fmt.Errorf("path not found: %s", p)
	}

	getter, exists := imageGetters[firstSeg.Attribute]
	if !exists {
		return i.fileData.Get(p)
	}

	result, err := getter(i)
	if err != nil {
		return nil, err
	}

	if remainingPath.IsEmpty() {
		return result, nil
	}

	return result.Get(remainingPath)
}
