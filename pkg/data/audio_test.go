package data

import (
	"context"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/external"
)

func TestNewAudioFromBytes(t *testing.T) {
	t.Parallel()
	c := qt.New(t)

	testCases := []struct {
		name        string
		filename    string
		contentType string
		duration    float64
	}{
		{"Valid WAV audio", "small_sample.wav", "audio/wav", 1.0},
		{"Valid MP3 audio", "small_sample.mp3", "audio/mpeg", 1.0},
		{"Valid OGG audio", "small_sample.ogg", "audio/ogg", 1.0},
		{"Valid AAC audio", "small_sample.aac", "audio/aac", 1.0},
		{"Valid FLAC audio", "small_sample.flac", "audio/flac", 1.0},
		{"Valid M4A audio", "small_sample.m4a", "audio/mp4", 1.0},
		{"Valid WMA audio", "small_sample.wma", "audio/x-ms-wma", 1.0},
		{"Valid AIFF audio", "small_sample.aiff", "audio/aiff", 1.0},
		{"Invalid file type", "sample_640_426.png", "", 0.0},
		{"Invalid audio format", "", "", 0.0},
		{"Empty audio bytes", "", "", 0.0},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *qt.C) {
			var audioBytes []byte
			var err error

			if tc.filename != "" {
				audioBytes, err = os.ReadFile("testdata/" + tc.filename)
				c.Assert(err, qt.IsNil)
			}

			audio, err := NewAudioFromBytes(audioBytes, tc.contentType, tc.filename, true)

			if tc.contentType == "" {
				c.Assert(err, qt.Not(qt.IsNil))
				return
			}

			c.Assert(err, qt.IsNil)
			c.Assert(audio.ContentType().String(), qt.Equals, "audio/ogg")
			c.Assert(audio.Duration().Float64(), qt.CmpEquals(cmpopts.EquateApprox(0, 0.1)), tc.duration)
		})
	}
}

func TestNewAudioFromURL(t *testing.T) {
	t.Parallel()
	c := qt.New(t)
	ctx := context.Background()

	binaryFetcher := external.NewBinaryFetcher()
	testCases := []struct {
		name string
		url  string
	}{
		{"Valid audio URL", "https://raw.githubusercontent.com/instill-ai/pipeline-backend/24153e2c57ba4ce508059a0bd1af8528b07b5ed3/pkg/data/testdata/sample1.wav"},
		{"Invalid URL", "https://invalid-url.com/audio.wav"},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *qt.C) {
			audio, err := NewAudioFromURL(ctx, binaryFetcher, tc.url, true)

			if tc.name == "Valid audio URL" {
				c.Assert(err, qt.IsNil)
				c.Assert(audio.ContentType().String(), qt.Equals, "audio/ogg")
				// Add more assertions for audio properties if needed
			} else {
				c.Assert(err, qt.Not(qt.IsNil))
			}
		})
	}
}

func TestAudioProperties(t *testing.T) {
	t.Parallel()
	c := qt.New(t)

	testCases := []struct {
		name        string
		filename    string
		contentType string
		duration    float64
	}{
		{"WAV audio", "small_sample.wav", "audio/wav", 1.0},
		{"MP3 audio", "small_sample.mp3", "audio/mpeg", 1.0},
		{"OGG audio", "small_sample.ogg", "audio/ogg", 1.0},
		{"AAC audio", "small_sample.aac", "audio/aac", 1.0},
		{"FLAC audio", "small_sample.flac", "audio/flac", 1.0},
		{"M4A audio", "small_sample.m4a", "audio/mp4", 1.0},
		{"WMA audio", "small_sample.wma", "audio/x-ms-wma", 1.0},
		{"AIFF audio", "small_sample.aiff", "audio/aiff", 1.0},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *qt.C) {
			audioBytes, err := os.ReadFile("testdata/" + tc.filename)
			c.Assert(err, qt.IsNil)

			audio, err := NewAudioFromBytes(audioBytes, tc.contentType, tc.filename, true)
			c.Assert(err, qt.IsNil)

			c.Assert(audio.ContentType().String(), qt.Equals, "audio/ogg")
			c.Assert(audio.Duration().Float64(), qt.CmpEquals(cmpopts.EquateApprox(0, 0.1)), tc.duration)

		})
	}
}

func TestAudioConvert(t *testing.T) {
	t.Parallel()
	c := qt.New(t)

	testCases := []struct {
		name           string
		filename       string
		contentType    string
		expectedFormat string
	}{
		{"WAV to MP3", "small_sample.wav", "audio/wav", "audio/mpeg"},
		{"MP3 to OGG", "small_sample.mp3", "audio/mpeg", "audio/ogg"},
		{"OGG to WAV", "small_sample.ogg", "audio/ogg", "audio/wav"},
		{"AAC to MP3", "small_sample.aac", "audio/aac", "audio/mpeg"},
		{"FLAC to OGG", "small_sample.flac", "audio/flac", "audio/ogg"},
		{"M4A to WAV", "small_sample.m4a", "audio/mp4", "audio/wav"},
		{"AIFF to MP3", "small_sample.aiff", "audio/aiff", "audio/mpeg"},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *qt.C) {
			audioBytes, err := os.ReadFile("testdata/" + tc.filename)
			c.Assert(err, qt.IsNil)

			audio, err := NewAudioFromBytes(audioBytes, tc.contentType, tc.filename, true)
			c.Assert(err, qt.IsNil)

			convertedAudio, err := audio.Convert(tc.expectedFormat)
			c.Assert(err, qt.IsNil)
			c.Assert(convertedAudio, qt.Not(qt.IsNil))
			c.Assert(convertedAudio.ContentType().String(), qt.Equals, tc.expectedFormat)

			// Check that the converted audio has the same duration as the original
			c.Assert(convertedAudio.Duration().Float64(), qt.CmpEquals(cmpopts.EquateApprox(0, 0.1)), audio.Duration().Float64())

			// Check that the converted audio is different from the original
			c.Assert(convertedAudio.(*audioData).raw, qt.Not(qt.DeepEquals), audio.raw)
		})
	}

	c.Run("Invalid target format", func(c *qt.C) {
		audioBytes, err := os.ReadFile("testdata/small_sample.wav")
		c.Assert(err, qt.IsNil)

		audio, err := NewAudioFromBytes(audioBytes, "audio/wav", "small_sample.wav", true)
		c.Assert(err, qt.IsNil)

		_, err = audio.Convert("invalid_format")
		c.Assert(err, qt.Not(qt.IsNil))
	})
}

func TestNewAudioFromBytesUnified(t *testing.T) {
	t.Parallel()
	c := qt.New(t)

	testCases := []struct {
		name        string
		filename    string
		contentType string
		duration    float64
	}{
		{"WAV as unified", "small_sample.wav", "audio/wav", 1.0},
		{"MP3 as unified", "small_sample.mp3", "audio/mpeg", 1.0},
		{"OGG as unified", "small_sample.ogg", "audio/ogg", 1.0},
		{"AAC as unified", "small_sample.aac", "audio/aac", 1.0},
		{"FLAC as unified", "small_sample.flac", "audio/flac", 1.0},
		{"M4A as unified", "small_sample.m4a", "audio/mp4", 1.0},
		{"WMA as unified", "small_sample.wma", "audio/x-ms-wma", 1.0},
		{"AIFF as unified", "small_sample.aiff", "audio/aiff", 1.0},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *qt.C) {
			audioBytes, err := os.ReadFile("testdata/" + tc.filename)
			c.Assert(err, qt.IsNil)

			// Test as unified (should convert to OGG)
			audio, err := NewAudioFromBytes(audioBytes, tc.contentType, tc.filename, true)
			c.Assert(err, qt.IsNil)
			c.Assert(audio.ContentType().String(), qt.Equals, "audio/ogg")
			c.Assert(audio.Duration().Float64(), qt.CmpEquals(cmpopts.EquateApprox(0, 0.1)), tc.duration)

			// Test as non-unified (should preserve original format)
			audioOriginal, err := NewAudioFromBytes(audioBytes, tc.contentType, tc.filename, false)
			c.Assert(err, qt.IsNil)
			c.Assert(audioOriginal.ContentType().String(), qt.Equals, tc.contentType)
			c.Assert(audioOriginal.Duration().Float64(), qt.CmpEquals(cmpopts.EquateApprox(0, 0.1)), tc.duration)
		})
	}
}

func TestNewAudioFromURLUnified(t *testing.T) {
	t.Parallel()
	c := qt.New(t)

	ctx := context.Background()
	binaryFetcher := external.NewBinaryFetcher()
	validURL := "https://raw.githubusercontent.com/instill-ai/pipeline-backend/24153e2c57ba4ce508059a0bd1af8528b07b5ed3/pkg/data/testdata/sample1.wav"

	c.Run("Unified converts to OGG", func(c *qt.C) {
		audio, err := NewAudioFromURL(ctx, binaryFetcher, validURL, true)
		c.Assert(err, qt.IsNil)
		// Should convert to OGG (internal unified format)
		c.Assert(audio.ContentType().String(), qt.Equals, "audio/ogg")
	})

	c.Run("Non-unified preserves original format", func(c *qt.C) {
		audio, err := NewAudioFromURL(ctx, binaryFetcher, validURL, false)
		c.Assert(err, qt.IsNil)
		// Should preserve original format (WAV in this case)
		c.Assert(audio.ContentType().String(), qt.Equals, "audio/wav")
	})
}

func TestAllSupportedAudioFormats(t *testing.T) {
	t.Parallel()
	c := qt.New(t)

	// Test all supported audio formats with their corresponding test files
	supportedFormats := []struct {
		name        string
		filename    string
		contentType string
		duration    float64
		sampleRate  int
	}{
		{"WAV", "small_sample.wav", "audio/wav", 1.0, 22050},
		{"MP3", "small_sample.mp3", "audio/mpeg", 1.0, 22050},
		{"OGG", "small_sample.ogg", "audio/ogg", 1.0, 22050},
		{"AAC", "small_sample.aac", "audio/aac", 1.0, 22050},
		{"FLAC", "small_sample.flac", "audio/flac", 1.0, 22050},
		{"M4A", "small_sample.m4a", "audio/mp4", 1.0, 22050},
		{"WMA", "small_sample.wma", "audio/x-ms-wma", 1.0, 22050},
		{"AIFF", "small_sample.aiff", "audio/aiff", 1.0, 22050},
	}

	for _, format := range supportedFormats {
		c.Run(format.name, func(c *qt.C) {
			// Test reading from bytes
			audioBytes, err := os.ReadFile("testdata/" + format.filename)
			c.Assert(err, qt.IsNil)

			// Test non-unified (preserves original format)
			audioOriginal, err := NewAudioFromBytes(audioBytes, format.contentType, format.filename, false)
			c.Assert(err, qt.IsNil)
			c.Assert(audioOriginal.ContentType().String(), qt.Equals, format.contentType)
			c.Assert(audioOriginal.Duration().Float64(), qt.CmpEquals(cmpopts.EquateApprox(0, 0.1)), format.duration)
			c.Assert(audioOriginal.SampleRate().Integer(), qt.Equals, format.sampleRate)

			// Test unified (converts to OGG)
			audioUnified, err := NewAudioFromBytes(audioBytes, format.contentType, format.filename, true)
			c.Assert(err, qt.IsNil)
			c.Assert(audioUnified.ContentType().String(), qt.Equals, "audio/ogg")
			c.Assert(audioUnified.Duration().Float64(), qt.CmpEquals(cmpopts.EquateApprox(0, 0.1)), format.duration)
			// Note: Sample rate might change during unified conversion

			// Test conversion capabilities - try converting to MP3 if not already MP3
			if format.contentType != "audio/mpeg" {
				convertedToMP3, err := audioOriginal.Convert("audio/mpeg")
				c.Assert(err, qt.IsNil)
				c.Assert(convertedToMP3.ContentType().String(), qt.Equals, "audio/mpeg")
				c.Assert(convertedToMP3.Duration().Float64(), qt.CmpEquals(cmpopts.EquateApprox(0, 0.1)), format.duration)
			}

			// Test conversion to OGG if not already OGG
			if format.contentType != "audio/ogg" {
				convertedToOGG, err := audioOriginal.Convert("audio/ogg")
				c.Assert(err, qt.IsNil)
				c.Assert(convertedToOGG.ContentType().String(), qt.Equals, "audio/ogg")
				c.Assert(convertedToOGG.Duration().Float64(), qt.CmpEquals(cmpopts.EquateApprox(0, 0.1)), format.duration)
			}

			// Test conversion to WAV if not already WAV
			if format.contentType != "audio/wav" {
				convertedToWAV, err := audioOriginal.Convert("audio/wav")
				c.Assert(err, qt.IsNil)
				c.Assert(convertedToWAV.ContentType().String(), qt.Equals, "audio/wav")
				c.Assert(convertedToWAV.Duration().Float64(), qt.CmpEquals(cmpopts.EquateApprox(0, 0.1)), format.duration)
			}
		})
	}
}
