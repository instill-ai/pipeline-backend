package data

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image/gif"
	"image/jpeg"
	"image/png"

	"golang.org/x/image/bmp"
	"golang.org/x/image/tiff"
	"golang.org/x/image/webp"

	goimage "image"

	"github.com/instill-ai/pipeline-backend/pkg/data/cgo"
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
	"github.com/instill-ai/pipeline-backend/pkg/data/path"
	"github.com/instill-ai/pipeline-backend/pkg/external"
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
	HEIC = "image/heic"
	HEIF = "image/heif"
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
	"heic":   func(i *imageData) (format.Value, error) { return i.Convert(HEIC) },
	"heif":   func(i *imageData) (format.Value, error) { return i.Convert(HEIF) },
}

// NewImageFromBytes creates a new imageData from byte slice
func NewImageFromBytes(b []byte, contentType, filename string, isUnified bool) (*imageData, error) {
	return createImageData(b, contentType, filename, isUnified)
}

// NewImageFromURL creates a new imageData from a URL
func NewImageFromURL(ctx context.Context, binaryFetcher external.BinaryFetcher, url string, isUnified bool) (*imageData, error) {
	b, contentType, filename, err := binaryFetcher.FetchFromURL(ctx, url)
	if err != nil {
		return nil, err
	}
	return createImageData(b, contentType, filename, isUnified)
}

// createImageData is a helper function to create imageData
func createImageData(b []byte, contentType, filename string, isUnified bool) (*imageData, error) {
	// Normalize MIME type first
	normalizedContentType := normalizeMIMEType(contentType)

	var err error
	finalContentType := normalizedContentType

	// If the image should be unified, convert it to PNG (the internal unified image format)
	if isUnified {
		if normalizedContentType != PNG {
			b, err = convertImage(b, normalizedContentType, PNG)
			if err != nil {
				return nil, err
			}
			finalContentType = PNG
		}
	}

	f, err := NewFileFromBytes(b, finalContentType, filename)
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
	case HEIC, HEIF:
		width, height = cgo.GetHEIFImageProperties(raw)
		return
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

func (i *imageData) Resize(width, height int) (format.Image, error) {
	img, _, err := goimage.Decode(bytes.NewReader(i.raw))
	if err != nil {
		return nil, fmt.Errorf("error decoding image for resize: %v", err)
	}

	// Create new image with desired dimensions
	resized := goimage.NewRGBA(goimage.Rect(0, 0, width, height))

	// Simple nearest-neighbor scaling
	scaleX := float64(img.Bounds().Dx()) / float64(width)
	scaleY := float64(img.Bounds().Dy()) / float64(height)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			srcX := int(float64(x) * scaleX)
			srcY := int(float64(y) * scaleY)
			resized.Set(x, y, img.At(srcX, srcY))
		}
	}

	// Encode resized image to PNG format
	buf := new(bytes.Buffer)
	if err := png.Encode(buf, resized); err != nil {
		return nil, fmt.Errorf("error encoding resized image: %v", err)
	}

	// Create new image data from encoded bytes
	return NewImageFromBytes(buf.Bytes(), PNG, "", false)
}

// imageData has unexported fields, which cannot be accessed by the regular
// encoder / decoder. A custom encode/decode method pair is defined to send and
// receive the type with the gob package.

// encImageData is redundant with imageData but allows us not to modify the
// format.Image interface signature.
type encImageData struct {
	encFileData
	Width  int
	Height int
}

func (i *imageData) GobEncode() ([]byte, error) {
	return json.Marshal(encImageData{
		encFileData: i.asEncodedStruct(),
		Width:       i.width,
		Height:      i.height,
	})
}

func (i *imageData) GobDecode(b []byte) error {
	var ei encImageData
	if err := json.Unmarshal(b, &ei); err != nil {
		return err
	}

	i.fileData = ei.asFileData()
	i.width = ei.Width
	i.height = ei.Height

	return nil
}
