package data

import (
	"bytes"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"

	"golang.org/x/image/webp"
)

func convertFile(raw []byte, sourceContentType, targetContentType string) (b []byte, err error) {
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
	case WEBP:
		if img, err = webp.Decode(bytes.NewReader(raw)); err != nil {
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
