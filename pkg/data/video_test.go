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
		{"Valid MP4 video", "sample_640_360.mp4", "video/mp4", false},
		{"Valid MOV video", "sample_640_360.mov", "video/mp4", false},
		{"Valid WMV video", "sample_640_360.wmv", "video/mp4", false},
		{"Invalid file type", "sample1.wav", "audio/wav", true},
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

	c.Run("Non-existent URL", func(c *qt.C) {
		c.Parallel()

		nonExistentURL := "https://filesamples.com/non-existent-video.mp4"
		_, err := NewVideoFromURL(ctx, binaryFetcher, nonExistentURL, true)
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
		{"MP4 video", "sample_640_360.mp4", "video/mp4", 640, 360, 13.346, 30.0},
		{"MOV video", "sample_640_360.mov", "video/quicktime", 640, 360, 13.346, 30.0},
		{"WMV video", "sample_640_360.wmv", "video/x-ms-wmv", 640, 360, 13.346, 30.0},
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
			c.Assert(video.Duration().Float64(), qt.CmpEquals(cmpopts.EquateApprox(0, 0.001)), tc.duration)
			c.Assert(video.FrameRate().Float64(), qt.CmpEquals(cmpopts.EquateApprox(0, 0.1)), tc.frameRate)

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
		{"MP4 to WebM", "sample_640_360.mp4", "video/mp4", "video/webm"},
		{"MOV to MP4", "sample_640_360.mov", "video/quicktime", "video/mp4"},
		{"WMV to WebM", "sample_640_360.wmv", "video/x-ms-wmv", "video/webm"},
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
			c.Assert(convertedVideo.Duration().Float64(), qt.CmpEquals(cmpopts.EquateApprox(0, 0.1)), video.Duration().Float64())
			c.Assert(convertedVideo.FrameRate().Float64(), qt.CmpEquals(cmpopts.EquateApprox(0, 0.1)), video.FrameRate().Float64())

			// Check that the converted video is different from the original
			c.Assert(convertedVideo.(*videoData).raw, qt.Not(qt.DeepEquals), video.raw)
		})
	}

	c.Run("Invalid target format", func(c *qt.C) {
		videoBytes, err := os.ReadFile(filepath.Join("testdata", "sample_640_360.mp4"))
		c.Assert(err, qt.IsNil)

		video, err := NewVideoFromBytes(videoBytes, "video/mp4", "sample_640_360.mp4", true)
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
		{"MP4 as unified", "sample_640_360.mp4", "video/mp4", 640, 360, 13.346, 30.0},
		{"MOV as unified", "sample_640_360.mov", "video/quicktime", 640, 360, 13.346, 30.0},
		{"WMV as unified", "sample_640_360.wmv", "video/x-ms-wmv", 640, 360, 13.346, 30.0},
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
			c.Assert(video.Duration().Float64(), qt.CmpEquals(cmpopts.EquateApprox(0, 0.001)), tc.duration)
			c.Assert(video.FrameRate().Float64(), qt.CmpEquals(cmpopts.EquateApprox(0, 0.1)), tc.frameRate)

			// Test as non-unified (should preserve original format)
			videoOriginal, err := NewVideoFromBytes(videoBytes, tc.contentType, tc.filename, false)
			c.Assert(err, qt.IsNil)
			c.Assert(videoOriginal.ContentType().String(), qt.Equals, tc.contentType)
			c.Assert(videoOriginal.Width().Integer(), qt.Equals, tc.width)
			c.Assert(videoOriginal.Height().Integer(), qt.Equals, tc.height)
			c.Assert(videoOriginal.Duration().Float64(), qt.CmpEquals(cmpopts.EquateApprox(0, 0.001)), tc.duration)
			c.Assert(videoOriginal.FrameRate().Float64(), qt.CmpEquals(cmpopts.EquateApprox(0, 0.1)), tc.frameRate)
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
