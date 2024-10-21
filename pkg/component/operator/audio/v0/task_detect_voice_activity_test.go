package audio

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"math"
	"os"
	"testing"

	"github.com/frankban/quicktest"
	"github.com/go-audio/audio"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestToMono(t *testing.T) {
	c := quicktest.New(t)
	stereoBuffer := &audio.IntBuffer{
		Data:   []int{1, 2, 3, 4, 5, 6},
		Format: &audio.Format{NumChannels: 2},
	}
	monoBuffer := toMono(stereoBuffer)
	c.Assert(monoBuffer.Format.NumChannels, quicktest.Equals, 1)
	c.Assert(monoBuffer.Data, quicktest.DeepEquals, []int{1, 3, 5})
}

func TestResample(t *testing.T) {
	c := quicktest.New(t)
	resampled := resample([]float64{1, 2, 3, 4, 5, 6, 7, 8}, 44100, 22050)
	c.Assert(len(resampled), quicktest.Equals, 4)
}

func TestDetectVoiceActivity(t *testing.T) {
	c := quicktest.New(t)
	testCase := struct {
		audioFile, expectedFile    string
		sampleRate                 int
		threshold                  float64
		silenceDuration, speechPad int
	}{
		audioFile:       "testdata/voice.wav",
		sampleRate:      16000,
		threshold:       0.5,
		silenceDuration: 100,
		speechPad:       30,
		expectedFile:    "testdata/voice-activity-segments.json",
	}

	input, err := structpb.NewStruct(map[string]interface{}{
		"audio":                   loadBase64Audio(testCase.audioFile),
		"sample-rate":             testCase.sampleRate,
		"threshold":               testCase.threshold,
		"min-silence-duration-ms": testCase.silenceDuration,
		"speech-pad-ms":           testCase.speechPad,
	})
	c.Assert(err, quicktest.IsNil)

	output, err := detectVoiceActivity(input, nil, context.Background())
	c.Assert(err, quicktest.IsNil)
	c.Assert(output, quicktest.Not(quicktest.IsNil))

	actualSegments := output.Fields["segments"].GetListValue().Values
	expectedSegments := loadExpectedSegments(c, testCase.expectedFile)

	c.Assert(len(actualSegments), quicktest.Equals, len(expectedSegments))

	for i, expected := range expectedSegments {
		actual := actualSegments[i].GetStructValue().Fields
		compareSegments(c, actual, expected, i)
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

func loadExpectedSegments(c *quicktest.C, filename string) []map[string]interface{} {
	var expected map[string]interface{}
	data, err := os.ReadFile(filename)
	c.Assert(err, quicktest.IsNil)
	c.Assert(json.Unmarshal(data, &expected), quicktest.IsNil)

	segments := expected["segments"].([]interface{})
	result := make([]map[string]interface{}, len(segments))
	for i, v := range segments {
		result[i] = v.(map[string]interface{})
	}
	return result
}

func compareSegments(c *quicktest.C, actual map[string]*structpb.Value, expected interface{}, index int) {
	expectedMap := expected.(map[string]interface{})
	approxEquals := func(got, want float64) bool {
		return math.Abs(got-want) < 0.1
	}

	for _, key := range []string{"start-time", "end-time"} {
		c.Assert(
			approxEquals(actual[key].GetNumberValue(), expectedMap[key].(float64)),
			quicktest.IsTrue,
			quicktest.Commentf("%s mismatch at index %d: got %v, want %v", key, index, actual[key].GetNumberValue(), expectedMap[key]),
		)
	}
}
