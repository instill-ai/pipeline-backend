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
		{"Valid WAV audio", "sample1.wav", "audio/wav", 122.093},
		{"Valid MP3 audio", "sample1.mp3", "audio/mpeg", 122.093},
		{"Valid OGG audio", "sample1.ogg", "audio/ogg", 122.093},
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

			audio, err := NewAudioFromBytes(audioBytes, tc.contentType, tc.filename)

			if tc.contentType == "" {
				c.Assert(err, qt.Not(qt.IsNil))
				return
			}

			c.Assert(err, qt.IsNil)
			c.Assert(audio.ContentType().String(), qt.Equals, "audio/ogg")
			c.Assert(audio.Duration().Float64(), qt.CmpEquals(cmpopts.EquateApprox(0, 0.01)), tc.duration)
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
		{"Non-existent URL", "https://filesamples.com/samples/audio/wav/non_existent.wav"},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *qt.C) {
			audio, err := NewAudioFromURL(ctx, binaryFetcher, tc.url)

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
		{"WAV audio", "sample1.wav", "audio/wav", 122.093},
		{"MP3 audio", "sample1.mp3", "audio/mpeg", 122.093},
		{"OGG audio", "sample1.ogg", "audio/ogg", 122.093},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *qt.C) {
			audioBytes, err := os.ReadFile("testdata/" + tc.filename)
			c.Assert(err, qt.IsNil)

			audio, err := NewAudioFromBytes(audioBytes, tc.contentType, tc.filename)
			c.Assert(err, qt.IsNil)

			c.Assert(audio.ContentType().String(), qt.Equals, "audio/ogg")
			c.Assert(audio.Duration().Float64(), qt.CmpEquals(cmpopts.EquateApprox(0, 0.01)), tc.duration)

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
		{"WAV to MP3", "sample1.wav", "audio/wav", "audio/mpeg"},
		{"MP3 to OGG", "sample1.mp3", "audio/mpeg", "audio/ogg"},
		{"OGG to WAV", "sample1.ogg", "audio/ogg", "audio/wav"},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *qt.C) {
			audioBytes, err := os.ReadFile("testdata/" + tc.filename)
			c.Assert(err, qt.IsNil)

			audio, err := NewAudioFromBytes(audioBytes, tc.contentType, tc.filename)
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
		audioBytes, err := os.ReadFile("testdata/sample1.wav")
		c.Assert(err, qt.IsNil)

		audio, err := NewAudioFromBytes(audioBytes, "audio/wav", "sample1.wav")
		c.Assert(err, qt.IsNil)

		_, err = audio.Convert("invalid_format")
		c.Assert(err, qt.Not(qt.IsNil))
	})
}
