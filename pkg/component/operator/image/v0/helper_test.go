package image

import (
	"bytes"
	"embed"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"strings"

	"github.com/google/uuid"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/data"
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
)

var (
	//go:embed testdata/*
	testdata embed.FS
)

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

func compareTestImage(c *qt.C, img format.Image, name string) {

	fileName := fmt.Sprintf("testdata/test_output_%s_%s_%s.jpeg", name, strings.ToLower(strings.Split(c.Name(), "/")[0]), strings.ToLower(strings.Split(c.Name(), "/")[1]))
	expectedImageBytes, err := testdata.ReadFile(fileName)
	c.Assert(err, qt.IsNil)

	expectedImage, err := data.NewImageFromBytes(expectedImageBytes, "image/jpeg", fileName)
	c.Assert(err, qt.IsNil)
	converted, err := img.Convert("image/jpeg")
	c.Assert(err, qt.IsNil)
	compareImage(c, converted, expectedImage)
}

func compareImage(c *qt.C, img format.Image, expectedImage format.Image) {
	// Decode the actual image
	actualImg, err := decodeImage(img)
	c.Assert(err, qt.IsNil)

	// Decode the expected image
	expectedImg, err := decodeImage(expectedImage)
	c.Assert(err, qt.IsNil)

	// Compare dimensions
	c.Assert(actualImg.Bounds(), qt.DeepEquals, expectedImg.Bounds(), qt.Commentf("Image dimensions do not match"))

	// Compare pixel by pixel with tolerance
	bounds := actualImg.Bounds()
	tolerance := uint32(2000) // Allow a small difference in color values
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			actualColor := actualImg.At(x, y)
			expectedColor := expectedImg.At(x, y)

			ar, ag, ab, aa := actualColor.RGBA()
			er, eg, eb, ea := expectedColor.RGBA()

			c.Assert(absDiff(ar, er) <= tolerance, qt.IsTrue, qt.Commentf("Red channel mismatch at pixel (%d, %d): got %d, want %d", x, y, ar, er))
			c.Assert(absDiff(ag, eg) <= tolerance, qt.IsTrue, qt.Commentf("Green channel mismatch at pixel (%d, %d): got %d, want %d", x, y, ag, eg))
			c.Assert(absDiff(ab, eb) <= tolerance, qt.IsTrue, qt.Commentf("Blue channel mismatch at pixel (%d, %d): got %d, want %d", x, y, ab, eb))
			c.Assert(absDiff(aa, ea) <= tolerance, qt.IsTrue, qt.Commentf("Alpha channel mismatch at pixel (%d, %d): got %d, want %d", x, y, aa, ea))
		}
	}

}

func absDiff(a, b uint32) uint32 {
	if a > b {
		return a - b
	}
	return b - a
}
