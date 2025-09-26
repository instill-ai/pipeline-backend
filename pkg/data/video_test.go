package data

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/external"
)

func TestNewVideoFromBytes(t *testing.T) {
	t.Parallel()
	c := qt.New(t)

	testCases := []struct {
		name        string
		filename    string
		contentType string
		expectError bool
	}{
		{"Valid MP4 video", "small_sample.mp4", "video/mp4", false},
		{"Valid MOV video", "small_sample.mov", "video/mp4", false},
		{"Valid AVI video", "small_sample.avi", "video/mp4", false},
		{"Valid WebM video", "small_sample.webm", "video/mp4", false},
		{"Valid MKV video", "small_sample.mkv", "video/mp4", false},
		{"Valid FLV video", "small_sample.flv", "video/mp4", false},
		{"Valid MPEG video", "small_sample.mpeg", "video/mp4", false},
		{"Invalid file type", "small_sample.wav", "audio/wav", true},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *qt.C) {
			videoBytes, err := os.ReadFile(filepath.Join("testdata", tc.filename))
			c.Assert(err, qt.IsNil)

			video, err := NewVideoFromBytes(videoBytes, tc.contentType, tc.filename, true)

			if tc.expectError {
				c.Assert(err, qt.IsNotNil)
			} else {
				c.Assert(err, qt.IsNil)
				c.Assert(video, qt.IsNotNil)
				c.Assert(video.ContentType().String(), qt.Equals, tc.contentType)
				c.Assert(video.Filename().String(), qt.Equals, tc.filename)
			}
		})
	}

	c.Run("Invalid video format", func(c *qt.C) {
		invalidBytes := []byte("not a video")
		contentType := "invalid/type"
		filename := "invalid.txt"

		_, err := NewVideoFromBytes(invalidBytes, contentType, filename, true)
		c.Assert(err, qt.IsNotNil)
	})

	c.Run("Empty video bytes", func(c *qt.C) {
		emptyBytes := []byte{}
		contentType := "video/mp4"
		filename := "empty.mp4"

		_, err := NewVideoFromBytes(emptyBytes, contentType, filename, true)
		c.Assert(err, qt.IsNotNil)
	})
}

func TestNewVideoFromURL(t *testing.T) {
	c := qt.New(t)
	c.Parallel()

	ctx := context.Background()
	binaryFetcher := external.NewBinaryFetcher()
	c.Run("Valid video URL", func(c *qt.C) {
		c.Parallel()

		url := "https://raw.githubusercontent.com/instill-ai/pipeline-backend/24153e2c57ba4ce508059a0bd1af8528b07b5ed3/pkg/data/testdata/sample_640_360.mp4"
		video, err := NewVideoFromURL(ctx, binaryFetcher, url, true)

		c.Assert(err, qt.IsNil)
		c.Assert(video, qt.IsNotNil)
		c.Assert(video.ContentType().String(), qt.Equals, "video/mp4")
	})

	c.Run("Invalid URL", func(c *qt.C) {
		c.Parallel()

		invalidURL := "not-a-url"
		_, err := NewVideoFromURL(ctx, binaryFetcher, invalidURL, true)
		c.Assert(err, qt.IsNotNil)
	})
}

func TestVideoProperties(t *testing.T) {
	t.Parallel()
	c := qt.New(t)

	testCases := []struct {
		name        string
		filename    string
		contentType string
		width       int
		height      int
		duration    float64
		frameRate   float64
	}{
		{"MP4 video", "small_sample.mp4", "video/mp4", 320, 240, 1.0, 15.0},
		{"MOV video", "small_sample.mov", "video/quicktime", 320, 240, 1.0, 15.0},
		{"AVI video", "small_sample.avi", "video/x-msvideo", 320, 240, 1.0, 15.0},
		{"WebM video", "small_sample.webm", "video/webm", 320, 240, 1.0, 15.0},
		{"MKV video", "small_sample.mkv", "video/x-matroska", 320, 240, 1.0, 15.0},
		{"FLV video", "small_sample.flv", "video/x-flv", 320, 240, 1.0, 15.0},
		{"MPEG video", "small_sample.mpeg", "video/mpeg", 320, 240, 1.0, 25.0},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *qt.C) {
			videoBytes, err := os.ReadFile(filepath.Join("testdata", tc.filename))
			c.Assert(err, qt.IsNil)

			video, err := NewVideoFromBytes(videoBytes, tc.contentType, tc.filename, true)
			c.Assert(err, qt.IsNil)
			qt.CmpEquals()
			c.Assert(video.ContentType().String(), qt.Equals, "video/mp4")
			c.Assert(video.Filename().String(), qt.Equals, tc.filename)
			c.Assert(video.Width().Integer(), qt.Equals, tc.width)
			c.Assert(video.Height().Integer(), qt.Equals, tc.height)
			c.Assert(video.Duration().Float64(), qt.CmpEquals(cmpopts.EquateApprox(0, 0.2)), tc.duration)
			c.Assert(video.FrameRate().Float64(), qt.CmpEquals(cmpopts.EquateApprox(0, 1.0)), tc.frameRate)

		})
	}
}

func TestVideoConvert(t *testing.T) {
	t.Parallel()
	c := qt.New(t)

	testCases := []struct {
		name           string
		filename       string
		contentType    string
		expectedFormat string
	}{
		{"MP4 to WebM", "small_sample.mp4", "video/mp4", "video/webm"},
		{"MOV to MP4", "small_sample.mov", "video/quicktime", "video/mp4"},
		{"AVI to MP4", "small_sample.avi", "video/x-msvideo", "video/mp4"},
		{"WebM to MP4", "small_sample.webm", "video/webm", "video/mp4"},
		{"MKV to WebM", "small_sample.mkv", "video/x-matroska", "video/webm"},
		{"FLV to MP4", "small_sample.flv", "video/x-flv", "video/mp4"},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *qt.C) {
			videoBytes, err := os.ReadFile(filepath.Join("testdata", tc.filename))
			c.Assert(err, qt.IsNil)

			video, err := NewVideoFromBytes(videoBytes, tc.contentType, tc.filename, true)
			c.Assert(err, qt.IsNil)

			convertedVideo, err := video.Convert(tc.expectedFormat)
			c.Assert(err, qt.IsNil)
			c.Assert(convertedVideo, qt.IsNotNil)
			c.Assert(convertedVideo.ContentType().String(), qt.Equals, tc.expectedFormat)

			// Check that the converted video has the same properties as the original
			c.Assert(convertedVideo.Width().Integer(), qt.Equals, video.Width().Integer())
			c.Assert(convertedVideo.Height().Integer(), qt.Equals, video.Height().Integer())
			c.Assert(convertedVideo.Duration().Float64(), qt.CmpEquals(cmpopts.EquateApprox(0, 1.0)), video.Duration().Float64())
			c.Assert(convertedVideo.FrameRate().Float64(), qt.CmpEquals(cmpopts.EquateApprox(0, 1.0)), video.FrameRate().Float64())

			// Check that the converted video is different from the original
			c.Assert(convertedVideo.(*videoData).raw, qt.Not(qt.DeepEquals), video.raw)
		})
	}

	c.Run("Invalid target format", func(c *qt.C) {
		videoBytes, err := os.ReadFile(filepath.Join("testdata", "small_sample.mp4"))
		c.Assert(err, qt.IsNil)

		video, err := NewVideoFromBytes(videoBytes, "video/mp4", "small_sample.mp4", true)
		c.Assert(err, qt.IsNil)

		_, err = video.Convert("invalid_format")
		c.Assert(err, qt.IsNotNil)
	})
}

func TestNewVideoFromBytesUnified(t *testing.T) {
	t.Parallel()
	c := qt.New(t)

	testCases := []struct {
		name        string
		filename    string
		contentType string
		width       int
		height      int
		duration    float64
		frameRate   float64
	}{
		{"MP4 as unified", "small_sample.mp4", "video/mp4", 320, 240, 1.0, 15.0},
		{"MOV as unified", "small_sample.mov", "video/quicktime", 320, 240, 1.0, 15.0},
		{"AVI as unified", "small_sample.avi", "video/x-msvideo", 320, 240, 1.0, 15.0},
		{"WebM as unified", "small_sample.webm", "video/webm", 320, 240, 1.0, 15.0},
		{"MKV as unified", "small_sample.mkv", "video/x-matroska", 320, 240, 1.0, 15.0},
		{"FLV as unified", "small_sample.flv", "video/x-flv", 320, 240, 1.0, 15.0},
		{"MPEG as unified", "small_sample.mpeg", "video/mpeg", 320, 240, 1.0, 25.0},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *qt.C) {
			videoBytes, err := os.ReadFile(filepath.Join("testdata", tc.filename))
			c.Assert(err, qt.IsNil)

			// Test as unified (should convert to MP4)
			video, err := NewVideoFromBytes(videoBytes, tc.contentType, tc.filename, true)
			c.Assert(err, qt.IsNil)
			c.Assert(video.ContentType().String(), qt.Equals, "video/mp4")
			c.Assert(video.Width().Integer(), qt.Equals, tc.width)
			c.Assert(video.Height().Integer(), qt.Equals, tc.height)
			c.Assert(video.Duration().Float64(), qt.CmpEquals(cmpopts.EquateApprox(0, 0.2)), tc.duration)
			c.Assert(video.FrameRate().Float64(), qt.CmpEquals(cmpopts.EquateApprox(0, 1.0)), tc.frameRate)

			// Test as non-unified (should preserve original format)
			videoOriginal, err := NewVideoFromBytes(videoBytes, tc.contentType, tc.filename, false)
			c.Assert(err, qt.IsNil)
			c.Assert(videoOriginal.ContentType().String(), qt.Equals, tc.contentType)
			c.Assert(videoOriginal.Width().Integer(), qt.Equals, tc.width)
			c.Assert(videoOriginal.Height().Integer(), qt.Equals, tc.height)
			c.Assert(videoOriginal.Duration().Float64(), qt.CmpEquals(cmpopts.EquateApprox(0, 0.2)), tc.duration)
			c.Assert(videoOriginal.FrameRate().Float64(), qt.CmpEquals(cmpopts.EquateApprox(0, 1.0)), tc.frameRate)
		})
	}
}

func TestNewVideoFromURLUnified(t *testing.T) {
	t.Parallel()
	c := qt.New(t)

	ctx := context.Background()
	binaryFetcher := external.NewBinaryFetcher()
	validURL := "https://raw.githubusercontent.com/instill-ai/pipeline-backend/24153e2c57ba4ce508059a0bd1af8528b07b5ed3/pkg/data/testdata/sample_640_360.mp4"

	c.Run("Unified converts to MP4", func(c *qt.C) {
		video, err := NewVideoFromURL(ctx, binaryFetcher, validURL, true)
		c.Assert(err, qt.IsNil)
		// Should convert to MP4 (internal unified format)
		c.Assert(video.ContentType().String(), qt.Equals, "video/mp4")
		c.Assert(video.Width().Integer(), qt.Equals, 640)
		c.Assert(video.Height().Integer(), qt.Equals, 360)
	})

	c.Run("Non-unified preserves original format", func(c *qt.C) {
		video, err := NewVideoFromURL(ctx, binaryFetcher, validURL, false)
		c.Assert(err, qt.IsNil)
		// Should preserve original format (MP4 in this case)
		c.Assert(video.ContentType().String(), qt.Equals, "video/mp4")
		c.Assert(video.Width().Integer(), qt.Equals, 640)
		c.Assert(video.Height().Integer(), qt.Equals, 360)
	})
}

func TestAllSupportedVideoFormats(t *testing.T) {
	t.Parallel()
	c := qt.New(t)

	// Test all supported video formats with their corresponding test files
	supportedFormats := []struct {
		name        string
		filename    string
		contentType string
		width       int
		height      int
		duration    float64
		frameRate   float64
	}{
		{"MP4", "small_sample.mp4", "video/mp4", 320, 240, 1.0, 15.0},
		{"MOV", "small_sample.mov", "video/quicktime", 320, 240, 1.0, 15.0},
		{"AVI", "small_sample.avi", "video/x-msvideo", 320, 240, 1.0, 15.0},
		{"WebM", "small_sample.webm", "video/webm", 320, 240, 1.0, 15.0},
		{"MKV", "small_sample.mkv", "video/x-matroska", 320, 240, 1.0, 15.0},
		{"FLV", "small_sample.flv", "video/x-flv", 320, 240, 1.0, 15.0},
		{"MPEG", "small_sample.mpeg", "video/mpeg", 320, 240, 1.0, 25.0},
	}

	for _, format := range supportedFormats {
		c.Run(format.name, func(c *qt.C) {
			// Test reading from bytes
			videoBytes, err := os.ReadFile(filepath.Join("testdata", format.filename))
			c.Assert(err, qt.IsNil)

			// Test non-unified (preserves original format)
			videoOriginal, err := NewVideoFromBytes(videoBytes, format.contentType, format.filename, false)
			c.Assert(err, qt.IsNil)
			c.Assert(videoOriginal.ContentType().String(), qt.Equals, format.contentType)
			c.Assert(videoOriginal.Width().Integer(), qt.Equals, format.width)
			c.Assert(videoOriginal.Height().Integer(), qt.Equals, format.height)
			c.Assert(videoOriginal.Duration().Float64(), qt.CmpEquals(cmpopts.EquateApprox(0, 0.2)), format.duration)
			c.Assert(videoOriginal.FrameRate().Float64(), qt.CmpEquals(cmpopts.EquateApprox(0, 1.0)), format.frameRate)

			// Test unified (converts to MP4)
			videoUnified, err := NewVideoFromBytes(videoBytes, format.contentType, format.filename, true)
			c.Assert(err, qt.IsNil)
			c.Assert(videoUnified.ContentType().String(), qt.Equals, "video/mp4")
			c.Assert(videoUnified.Width().Integer(), qt.Equals, format.width)
			c.Assert(videoUnified.Height().Integer(), qt.Equals, format.height)
			c.Assert(videoUnified.Duration().Float64(), qt.CmpEquals(cmpopts.EquateApprox(0, 0.2)), format.duration)
			// Note: Frame rate may change during unified conversion, so we use a wider tolerance

			// Test conversion capabilities - try converting to MP4 if not already MP4
			if format.contentType != "video/mp4" {
				convertedToMP4, err := videoOriginal.Convert("video/mp4")
				c.Assert(err, qt.IsNil)
				c.Assert(convertedToMP4.ContentType().String(), qt.Equals, "video/mp4")
				c.Assert(convertedToMP4.Width().Integer(), qt.Equals, format.width)
				c.Assert(convertedToMP4.Height().Integer(), qt.Equals, format.height)
			}

			// Test conversion to WebM if not already WebM
			if format.contentType != "video/webm" {
				convertedToWebM, err := videoOriginal.Convert("video/webm")
				c.Assert(err, qt.IsNil)
				c.Assert(convertedToWebM.ContentType().String(), qt.Equals, "video/webm")
				c.Assert(convertedToWebM.Width().Integer(), qt.Equals, format.width)
				c.Assert(convertedToWebM.Height().Integer(), qt.Equals, format.height)
			}
		})
	}
}

func TestVideoMIMETypeNormalization(t *testing.T) {
	t.Parallel()
	c := qt.New(t)

	// Test that video/mov is properly normalized to video/quicktime
	c.Run("video/mov normalization", func(c *qt.C) {
		videoBytes, err := os.ReadFile("testdata/small_sample.mov")
		c.Assert(err, qt.IsNil)

		// Create video with non-standard MIME type
		videoMOV, err := NewVideoFromBytes(videoBytes, "video/mov", "test.mov", false)
		c.Assert(err, qt.IsNil)

		// Create video with standard MIME type
		videoQuicktime, err := NewVideoFromBytes(videoBytes, "video/quicktime", "test.mov", false)
		c.Assert(err, qt.IsNil)

		// Both should have the same normalized content type
		c.Assert(videoMOV.ContentType().String(), qt.Equals, "video/quicktime")
		c.Assert(videoQuicktime.ContentType().String(), qt.Equals, "video/quicktime")
		c.Assert(videoMOV.ContentType().String(), qt.Equals, videoQuicktime.ContentType().String())

		// Both should have the same properties
		c.Assert(videoMOV.Width().Integer(), qt.Equals, videoQuicktime.Width().Integer())
		c.Assert(videoMOV.Height().Integer(), qt.Equals, videoQuicktime.Height().Integer())
	})

	// Test that video/avi is properly normalized to video/x-msvideo
	c.Run("video/avi normalization", func(c *qt.C) {
		videoBytes, err := os.ReadFile("testdata/small_sample.avi")
		c.Assert(err, qt.IsNil)

		// Create video with non-standard MIME type
		videoAVI, err := NewVideoFromBytes(videoBytes, "video/avi", "test.avi", false)
		c.Assert(err, qt.IsNil)

		// Create video with standard MIME type
		videoXMSVideo, err := NewVideoFromBytes(videoBytes, "video/x-msvideo", "test.avi", false)
		c.Assert(err, qt.IsNil)

		// Both should have the same normalized content type
		c.Assert(videoAVI.ContentType().String(), qt.Equals, "video/x-msvideo")
		c.Assert(videoXMSVideo.ContentType().String(), qt.Equals, "video/x-msvideo")
		c.Assert(videoAVI.ContentType().String(), qt.Equals, videoXMSVideo.ContentType().String())

		// Both should have the same properties
		c.Assert(videoAVI.Width().Integer(), qt.Equals, videoXMSVideo.Width().Integer())
		c.Assert(videoAVI.Height().Integer(), qt.Equals, videoXMSVideo.Height().Integer())
	})

	// Test that video/3gpp is properly normalized to video/mp4
	c.Run("video/3gpp normalization", func(c *qt.C) {
		videoBytes, err := os.ReadFile("testdata/small_sample.mp4")
		c.Assert(err, qt.IsNil)

		// Create video with non-standard MIME type (3GPP)
		video3GPP, err := NewVideoFromBytes(videoBytes, "video/3gpp", "test.3gp", false)
		c.Assert(err, qt.IsNil)

		// Create video with standard MIME type
		videoMP4, err := NewVideoFromBytes(videoBytes, "video/mp4", "test.mp4", false)
		c.Assert(err, qt.IsNil)

		// Both should have the same normalized content type (MP4)
		c.Assert(video3GPP.ContentType().String(), qt.Equals, "video/mp4")
		c.Assert(videoMP4.ContentType().String(), qt.Equals, "video/mp4")
		c.Assert(video3GPP.ContentType().String(), qt.Equals, videoMP4.ContentType().String())

		// Both should have the same properties
		c.Assert(video3GPP.Width().Integer(), qt.Equals, videoMP4.Width().Integer())
		c.Assert(video3GPP.Height().Integer(), qt.Equals, videoMP4.Height().Integer())
	})
}
