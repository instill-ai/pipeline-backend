package image

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/png"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

// base64Image is a base64 encoded image
type base64Image string

func decodeBase64Image(base64Img string) (image.Image, error) {
	imgBytes, err := base64.StdEncoding.DecodeString(base.TrimBase64Mime(base64Img))
	if err != nil {
		return nil, fmt.Errorf("error decoding base64 image: %v", err)
	}

	img, _, err := image.Decode(bytes.NewReader(imgBytes))
	if err != nil {
		return nil, fmt.Errorf("error decoding image: %v", err)
	}

	return img, nil
}

func encodeBase64Image(img image.Image) (string, error) {
	buf := new(bytes.Buffer)
	err := png.Encode(buf, img)
	if err != nil {
		return "", fmt.Errorf("error encoding image: %v", err)
	}

	base64ByteImg := base64.StdEncoding.EncodeToString(buf.Bytes())

	return base64ByteImg, nil
}

func convertToRGBA(img image.Image) *image.RGBA {
	bounds := img.Bounds()
	rgba := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			originalColor := img.At(x, y)
			rgba.Set(x, y, color.RGBAModel.Convert(originalColor))
		}
	}
	return rgba
}
