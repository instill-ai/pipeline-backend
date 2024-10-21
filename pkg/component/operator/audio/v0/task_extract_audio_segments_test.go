package audio

import (
	"bytes"
	"context"
	"encoding/base64"
	"testing"

	"github.com/frankban/quicktest"
	"github.com/go-audio/wav"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func TestExtractAudioSegments(t *testing.T) {
	c := quicktest.New(t)
	testCase := struct {
		audioFile     string
		segmentsFile  string
		expectedCount int
	}{
		audioFile:     "testdata/voice.wav",
		segmentsFile:  "testdata/voice-activity-segments.json",
		expectedCount: 15,
	}

	// Load audio file
	audioData := loadBase64Audio(testCase.audioFile)

	// Load segments
	rawSegments := loadExpectedSegments(c, testCase.segmentsFile)
	segmentValues := make([]interface{}, len(rawSegments))
	for i, seg := range rawSegments {
		segmentValues[i] = seg
	}

	input, err := structpb.NewStruct(map[string]interface{}{
		"audio":    audioData,
		"segments": segmentValues,
	})
	c.Assert(err, quicktest.IsNil)

	output, err := extractAudioSegments(input, nil, context.Background())
	c.Assert(err, quicktest.IsNil)
	c.Assert(output, quicktest.Not(quicktest.IsNil))

	audioSegments := output.Fields["audio-segments"].GetListValue().Values
	c.Assert(len(audioSegments), quicktest.Equals, testCase.expectedCount)

	// Validate each extracted segment
	for i, segment := range audioSegments {
		segmentData, err := base64.StdEncoding.DecodeString(base.TrimBase64Mime(segment.GetStringValue()))
		c.Assert(err, quicktest.IsNil)

		// Verify that the extracted segment is a valid WAV file
		dec := wav.NewDecoder(bytes.NewReader(segmentData))
		c.Assert(dec.IsValidFile(), quicktest.IsTrue, quicktest.Commentf("Segment %d is not a valid WAV file", i))
	}
}
