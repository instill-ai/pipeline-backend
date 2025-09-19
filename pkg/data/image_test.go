package data

import (
	"context"
	"os"
	"testing"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/external"
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
		{"Valid PNG image", "small_sample.png", "image/png", 320, 240},
		{"Valid JPEG image", "small_sample.jpeg", "image/jpeg", 320, 240},
		{"Valid TIFF image", "small_sample.tiff", "image/tiff", 320, 240},
		{"Valid GIF image", "small_sample.gif", "image/gif", 320, 240},
		{"Valid BMP image", "small_sample.bmp", "image/bmp", 320, 240},
		{"Valid WEBP image", "small_sample.webp", "image/webp", 320, 240},
		{"Invalid file type", "small_sample.mp3", "", 0, 0},
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

			image, err := NewImageFromBytes(imageBytes, tc.contentType, tc.filename, true)

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

func TestNewImageFromBytesUnified(t *testing.T) {
	t.Parallel()
	c := qt.New(t)

	testCases := []struct {
		name        string
		filename    string
		contentType string
		width       int
		height      int
	}{
		{"PNG as unified", "small_sample.png", "image/png", 320, 240},
		{"JPEG as unified", "small_sample.jpeg", "image/jpeg", 320, 240},
		{"TIFF as unified", "small_sample.tiff", "image/tiff", 320, 240},
		{"GIF as unified", "small_sample.gif", "image/gif", 320, 240},
		{"BMP as unified", "small_sample.bmp", "image/bmp", 320, 240},
		{"WEBP as unified", "small_sample.webp", "image/webp", 320, 240},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *qt.C) {
			imageBytes, err := os.ReadFile("testdata/" + tc.filename)
			c.Assert(err, qt.IsNil)

			// Test as unified (should convert to PNG)
			image, err := NewImageFromBytes(imageBytes, tc.contentType, tc.filename, true)
			c.Assert(err, qt.IsNil)
			c.Assert(image.ContentType().String(), qt.Equals, "image/png")
			c.Assert(image.Width().Integer(), qt.Equals, tc.width)
			c.Assert(image.Height().Integer(), qt.Equals, tc.height)

			// Test as non-unified (should preserve original format)
			imageOriginal, err := NewImageFromBytes(imageBytes, tc.contentType, tc.filename, false)
			c.Assert(err, qt.IsNil)
			c.Assert(imageOriginal.ContentType().String(), qt.Equals, tc.contentType)
			c.Assert(imageOriginal.Width().Integer(), qt.Equals, tc.width)
			c.Assert(imageOriginal.Height().Integer(), qt.Equals, tc.height)
		})
	}
}

func TestNewImageFromURL(t *testing.T) {
	t.Parallel()
	c := qt.New(t)

	ctx := context.Background()
	binaryFetcher := external.NewBinaryFetcher()
	testCases := []struct {
		name string
		url  string
	}{
		{"Valid image URL", "https://raw.githubusercontent.com/instill-ai/pipeline-backend/24153e2c57ba4ce508059a0bd1af8528b07b5ed3/pkg/data/testdata/sample_640_426.png"},
		{"Invalid URL", "https://invalid-url.com/image.png"},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *qt.C) {
			image, err := NewImageFromURL(ctx, binaryFetcher, tc.url, true)

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

func TestNewImageFromURLUnified(t *testing.T) {
	t.Parallel()
	c := qt.New(t)

	ctx := context.Background()
	binaryFetcher := external.NewBinaryFetcher()
	validURL := "https://raw.githubusercontent.com/instill-ai/pipeline-backend/24153e2c57ba4ce508059a0bd1af8528b07b5ed3/pkg/data/testdata/sample_640_426.png"

	c.Run("Unified converts to PNG", func(c *qt.C) {
		image, err := NewImageFromURL(ctx, binaryFetcher, validURL, true)
		c.Assert(err, qt.IsNil)
		// Should convert to PNG (internal unified format)
		c.Assert(image.ContentType().String(), qt.Equals, "image/png")
		c.Assert(image.Width().Integer(), qt.Equals, 640)
		c.Assert(image.Height().Integer(), qt.Equals, 426)
	})

	c.Run("Non-unified preserves original format", func(c *qt.C) {
		image, err := NewImageFromURL(ctx, binaryFetcher, validURL, false)
		c.Assert(err, qt.IsNil)
		// Should preserve original format (PNG in this case)
		c.Assert(image.ContentType().String(), qt.Equals, "image/png")
		c.Assert(image.Width().Integer(), qt.Equals, 640)
		c.Assert(image.Height().Integer(), qt.Equals, 426)
	})
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
		{"PNG image", "small_sample.png", "image/png", 320, 240},
		{"JPEG image", "small_sample.jpeg", "image/jpeg", 320, 240},
		{"TIFF image", "small_sample.tiff", "image/tiff", 320, 240},
		{"GIF image", "small_sample.gif", "image/gif", 320, 240},
		{"BMP image", "small_sample.bmp", "image/bmp", 320, 240},
		{"WEBP image", "small_sample.webp", "image/webp", 320, 240},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *qt.C) {
			imageBytes, err := os.ReadFile("testdata/" + tc.filename)
			c.Assert(err, qt.IsNil)

			image, err := NewImageFromBytes(imageBytes, tc.contentType, tc.filename, true)
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
		{"PNG to JPEG", "small_sample.png", "image/png", "image/jpeg"},
		{"JPEG to TIFF", "small_sample.jpeg", "image/jpeg", "image/tiff"},
		{"TIFF to JPEG", "small_sample.tiff", "image/tiff", "image/jpeg"},
		{"GIF to PNG", "small_sample.gif", "image/gif", "image/png"},
		{"BMP to PNG", "small_sample.bmp", "image/bmp", "image/png"},
		{"WEBP to JPEG", "small_sample.webp", "image/webp", "image/jpeg"},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *qt.C) {
			imageBytes, err := os.ReadFile("testdata/" + tc.filename)
			c.Assert(err, qt.IsNil)

			image, err := NewImageFromBytes(imageBytes, tc.contentType, tc.filename, true)
			c.Assert(err, qt.IsNil)

			convertedImage, err := image.Convert(tc.expectedFormat)
			c.Assert(err, qt.IsNil)
			c.Assert(convertedImage, qt.Not(qt.IsNil))
			c.Assert(convertedImage.ContentType().String(), qt.Equals, tc.expectedFormat)

			// Check that the converted image has the same dimensions as the original
			c.Assert(convertedImage.Width().Integer(), qt.Equals, image.Width().Integer())
			c.Assert(convertedImage.Height().Integer(), qt.Equals, image.Height().Integer())
		})
	}

	c.Run("Invalid target format", func(c *qt.C) {
		imageBytes, err := os.ReadFile("testdata/small_sample.png")
		c.Assert(err, qt.IsNil)

		image, err := NewImageFromBytes(imageBytes, "image/png", "small_sample.png", true)
		c.Assert(err, qt.IsNil)

		_, err = image.Convert("invalid_format")
		c.Assert(err, qt.Not(qt.IsNil))
	})
}

func TestAllSupportedImageFormats(t *testing.T) {
	t.Parallel()
	c := qt.New(t)

	// Test all supported image formats with their corresponding test files
	supportedFormats := []struct {
		name        string
		filename    string
		contentType string
		width       int
		height      int
	}{
		{"JPEG", "small_sample.jpeg", "image/jpeg", 320, 240},
		{"PNG", "small_sample.png", "image/png", 320, 240},
		{"GIF", "small_sample.gif", "image/gif", 320, 240},
		{"BMP", "small_sample.bmp", "image/bmp", 320, 240},
		{"WEBP", "small_sample.webp", "image/webp", 320, 240},
		{"TIFF", "small_sample.tiff", "image/tiff", 320, 240},
	}

	for _, format := range supportedFormats {
		c.Run(format.name, func(c *qt.C) {
			// Test reading from bytes
			imageBytes, err := os.ReadFile("testdata/" + format.filename)
			c.Assert(err, qt.IsNil)

			// Test non-unified (preserves original format)
			imageOriginal, err := NewImageFromBytes(imageBytes, format.contentType, format.filename, false)
			c.Assert(err, qt.IsNil)
			c.Assert(imageOriginal.ContentType().String(), qt.Equals, format.contentType)
			c.Assert(imageOriginal.Width().Integer(), qt.Equals, format.width)
			c.Assert(imageOriginal.Height().Integer(), qt.Equals, format.height)

			// Test unified (converts to PNG)
			imageUnified, err := NewImageFromBytes(imageBytes, format.contentType, format.filename, true)
			c.Assert(err, qt.IsNil)
			c.Assert(imageUnified.ContentType().String(), qt.Equals, "image/png")
			c.Assert(imageUnified.Width().Integer(), qt.Equals, format.width)
			c.Assert(imageUnified.Height().Integer(), qt.Equals, format.height)

			// Test conversion capabilities - try converting to PNG if not already PNG
			if format.contentType != "image/png" {
				convertedToPNG, err := imageOriginal.Convert("image/png")
				c.Assert(err, qt.IsNil)
				c.Assert(convertedToPNG.ContentType().String(), qt.Equals, "image/png")
				c.Assert(convertedToPNG.Width().Integer(), qt.Equals, format.width)
				c.Assert(convertedToPNG.Height().Integer(), qt.Equals, format.height)
			}

			// Test conversion to JPEG if not already JPEG
			if format.contentType != "image/jpeg" {
				convertedToJPEG, err := imageOriginal.Convert("image/jpeg")
				c.Assert(err, qt.IsNil)
				c.Assert(convertedToJPEG.ContentType().String(), qt.Equals, "image/jpeg")
				c.Assert(convertedToJPEG.Width().Integer(), qt.Equals, format.width)
				c.Assert(convertedToJPEG.Height().Integer(), qt.Equals, format.height)
			}
		})
	}
}
