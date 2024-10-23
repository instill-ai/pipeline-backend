package image

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"

	"github.com/google/uuid"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/data"
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
)

// Helper function to create a test image with a solid color
func createTestImage(c *qt.C, width, height int, color color.Color) format.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color)
		}
	}
	buf := new(bytes.Buffer)
	err := png.Encode(buf, img)
	if err != nil {
		c.Assert(err, qt.IsNil)
	}
	imgData, err := data.NewImageFromBytes(buf.Bytes(), "image/png", fmt.Sprintf("image-%s.png", uuid.New().String()))
	if err != nil {
		c.Assert(err, qt.IsNil)
	}
	return imgData
}
