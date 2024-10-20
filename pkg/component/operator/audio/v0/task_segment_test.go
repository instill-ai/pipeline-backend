package audio

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"os"
	"testing"

	"github.com/go-audio/wav"
	"google.golang.org/protobuf/types/known/structpb"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func TestSegment(t *testing.T) {
	c := qt.New(t)
	testCase := struct {
		audioFile     string
		segmentsFile  string
		expectedCount int
	}{
		audioFile:     "testdata/voice1.wav",
		segmentsFile:  "testdata/voice1-activity-segments.json",
		expectedCount: 15,
	}

	audioData := loadBase64Audio(testCase.audioFile)
	rawSegments := loadWantSegments(c, testCase.segmentsFile)
	segmentValues := make([]interface{}, len(rawSegments))
	for i, seg := range rawSegments {
		segmentValues[i] = seg
	}

	input, err := structpb.NewStruct(map[string]interface{}{
		"audio":    audioData,
		"segments": segmentValues,
	})
	c.Assert(err, qt.IsNil)

	output, err := segment(input, nil, context.Background())
	c.Assert(err, qt.IsNil)
	c.Assert(output, qt.Not(qt.IsNil))

	audioSegments := output.Fields["audio-segments"].GetListValue().Values
	c.Assert(len(audioSegments), qt.Equals, testCase.expectedCount)

	for i, segment := range audioSegments {
		segmentData, err := base64.StdEncoding.DecodeString(base.TrimBase64Mime(segment.GetStringValue()))
		c.Assert(err, qt.IsNil)

		dec := wav.NewDecoder(bytes.NewReader(segmentData))
		c.Assert(dec.IsValidFile(), qt.IsTrue, qt.Commentf("Segment %d is not a valid WAV file", i))
	}
}

func loadBase64Audio(filename string) string {
	file, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	buf, err := io.ReadAll(file)
	if err != nil {
		panic(err)
	}

	return base64.StdEncoding.EncodeToString(buf)
}

func loadWantSegments(c *qt.C, filename string) []map[string]interface{} {
	var expected map[string]interface{}
	data, err := os.ReadFile(filename)
	c.Assert(err, qt.IsNil)
	c.Assert(json.Unmarshal(data, &expected), qt.IsNil)

	segments := expected["segments"].([]interface{})
	result := make([]map[string]interface{}, len(segments))
	for i, v := range segments {
		result[i] = v.(map[string]interface{})
	}
	return result
}
