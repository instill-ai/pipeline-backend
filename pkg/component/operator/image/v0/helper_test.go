package image

import (
	"bytes"
	"embed"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
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

	filename := fmt.Sprintf("testdata/test_output_%s_%s_%s.jpeg", name, strings.ToLower(strings.Split(c.Name(), "/")[0]), strings.ToLower(strings.Split(c.Name(), "/")[1]))
	expectedImageBytes, err := testdata.ReadFile(filename)
	c.Assert(err, qt.IsNil)

	expectedImage, err := data.NewImageFromBytes(expectedImageBytes, "image/jpeg", filename)
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

	// TODO: Compare pixel by pixel with tolerance
	bounds := actualImg.Bounds()
	var mse float64
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			actualColor := actualImg.At(x, y)
			expectedColor := expectedImg.At(x, y)

			ar, ag, ab, aa := actualColor.RGBA()
			er, eg, eb, ea := expectedColor.RGBA()

			mse += float64((ar-er)*(ar-er) + (ag-eg)*(ag-eg) + (ab-eb)*(ab-eb) + (aa-ea)*(aa-ea))
		}
	}
	mse /= float64(bounds.Dx() * bounds.Dy() * 4) // 4 channels: R, G, B, A

	if mse == 0 {
		c.Assert(true, qt.IsTrue, qt.Commentf("Images are identical"))
	} else {
		psnr := 10 * math.Log10((65535*65535)/mse)
		c.Assert(psnr >= 30, qt.IsTrue, qt.Commentf("PSNR is too low: %f", psnr))
	}

}
