package data

import (
	"os"
	"testing"

	qt "github.com/frankban/quicktest"
)

func TestNewImageFromBytes(t *testing.T) {
	t.Parallel()
	c := qt.New(t)

	testCases := []struct {
		name        string
		filename    string
		contentType string
		width       int
		height      int
	}{
		{"Valid PNG image", "sample_640_426.png", "image/png", 640, 426},
		{"Valid JPEG image", "sample_640_426.jpeg", "image/jpeg", 640, 426},
		{"Valid TIFF image", "sample_640_426.tiff", "image/tiff", 640, 426},
		{"Invalid file type", "sample1.mp3", "", 0, 0},
		{"Empty image bytes", "", "", 0, 0},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *qt.C) {
			var imageBytes []byte
			var err error

			if tc.filename != "" {
				imageBytes, err = os.ReadFile("testdata/" + tc.filename)
				c.Assert(err, qt.IsNil)
			}

			image, err := NewImageFromBytes(imageBytes, tc.contentType, tc.filename)

			if tc.contentType == "" {
				c.Assert(err, qt.Not(qt.IsNil))
				return
			}

			c.Assert(err, qt.IsNil)
			c.Assert(image.ContentType().String(), qt.Equals, "image/png")
			c.Assert(image.Width().Integer(), qt.Equals, tc.width)
			c.Assert(image.Height().Integer(), qt.Equals, tc.height)
		})
	}
}

func TestNewImageFromURL(t *testing.T) {
	t.Parallel()
	c := qt.New(t)

	testCases := []struct {
		name string
		url  string
	}{
		{"Valid image URL", "https://raw.githubusercontent.com/instill-ai/pipeline-backend/24153e2c57ba4ce508059a0bd1af8528b07b5ed3/pkg/data/testdata/sample_640_426.png"},
		{"Invalid URL", "https://invalid-url.com/image.png"},
		{"Non-existent URL", "https://filesamples.com/samples/image/png/non_existent.png"},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *qt.C) {
			image, err := NewImageFromURL(tc.url)

			if tc.name == "Valid image URL" {
				c.Assert(err, qt.IsNil)
				c.Assert(image.ContentType().String(), qt.Equals, "image/png")
				c.Assert(image.Width().Integer(), qt.Equals, 640)
				c.Assert(image.Height().Integer(), qt.Equals, 426)
			} else {
				c.Assert(err, qt.Not(qt.IsNil))
			}
		})
	}
}

func TestImageProperties(t *testing.T) {
	t.Parallel()
	c := qt.New(t)

	testCases := []struct {
		name        string
		filename    string
		contentType string
		width       int
		height      int
	}{
		{"PNG image", "sample_640_426.png", "image/png", 640, 426},
		{"JPEG image", "sample_640_426.jpeg", "image/jpeg", 640, 426},
		{"TIFF image", "sample_640_426.tiff", "image/tiff", 640, 426},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *qt.C) {
			imageBytes, err := os.ReadFile("testdata/" + tc.filename)
			c.Assert(err, qt.IsNil)

			image, err := NewImageFromBytes(imageBytes, tc.contentType, tc.filename)
			c.Assert(err, qt.IsNil)

			c.Assert(image.ContentType().String(), qt.Equals, "image/png")
			c.Assert(image.Width().Integer(), qt.Equals, tc.width)
			c.Assert(image.Height().Integer(), qt.Equals, tc.height)
		})
	}
}

func TestImageConvert(t *testing.T) {
	t.Parallel()
	c := qt.New(t)

	testCases := []struct {
		name           string
		filename       string
		contentType    string
		expectedFormat string
	}{
		{"PNG to JPEG", "sample_640_426.png", "image/png", "image/jpeg"},
		{"JPEG to TIFF", "sample_640_426.jpeg", "image/jpeg", "image/tiff"},
		{"TIFF to JPEG", "sample_640_426.tiff", "image/tiff", "image/jpeg"},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *qt.C) {
			imageBytes, err := os.ReadFile("testdata/" + tc.filename)
			c.Assert(err, qt.IsNil)

			image, err := NewImageFromBytes(imageBytes, tc.contentType, tc.filename)
			c.Assert(err, qt.IsNil)

			convertedImage, err := image.Convert(tc.expectedFormat)
			c.Assert(err, qt.IsNil)
			c.Assert(convertedImage, qt.Not(qt.IsNil))
			c.Assert(convertedImage.ContentType().String(), qt.Equals, tc.expectedFormat)

			// Check that the converted image has the same dimensions as the original
			c.Assert(convertedImage.Width().Integer(), qt.Equals, image.Width().Integer())
			c.Assert(convertedImage.Height().Integer(), qt.Equals, image.Height().Integer())

			// Check that the converted image is different from the original
			c.Assert(convertedImage.(*imageData).raw, qt.Not(qt.DeepEquals), image.raw)
		})
	}

	c.Run("Invalid target format", func(c *qt.C) {
		imageBytes, err := os.ReadFile("testdata/sample_640_426.png")
		c.Assert(err, qt.IsNil)

		image, err := NewImageFromBytes(imageBytes, "image/png", "sample_640_426.png")
		c.Assert(err, qt.IsNil)

		_, err = image.Convert("invalid_format")
		c.Assert(err, qt.Not(qt.IsNil))
	})
}
