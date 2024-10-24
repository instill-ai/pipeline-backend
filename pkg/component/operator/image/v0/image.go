package image

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"

	"github.com/google/uuid"
	"github.com/instill-ai/pipeline-backend/pkg/data"
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
)

func decodeImage(i format.Image) (image.Image, error) {

	binary, err := i.Binary()
	if err != nil {
		return nil, fmt.Errorf("error getting binary data for image: %v", err)
	}

	img, _, err := image.Decode(bytes.NewReader(binary.ByteArray()))
	if err != nil {
		return nil, fmt.Errorf("error decoding image: %v", err)
	}

	return img, nil
}

func encodeImage(img image.Image) (format.Image, error) {
	buf := new(bytes.Buffer)
	err := png.Encode(buf, img)
	if err != nil {
		return nil, fmt.Errorf("error encoding image: %v", err)
	}
	imgData, err := data.NewImageFromBytes(buf.Bytes(), "image/png", fmt.Sprintf("image-%s.png", uuid.New().String()))
	if err != nil {
		return nil, fmt.Errorf("error creating image data: %v", err)
	}

	return imgData, nil
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
