//go:build onnx
// +build onnx

package audio

import (
	"context"
	"encoding/json"
	"io"
	"math"
	"os"
	"testing"

	"github.com/go-audio/audio"
	"github.com/google/go-cmp/cmp"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/mock"
	"github.com/instill-ai/pipeline-backend/pkg/data"
	"github.com/instill-ai/pipeline-backend/pkg/external"
)

func TestDetectActivity(t *testing.T) {
	c := qt.New(t)

	testCases := []struct {
		name            string
		audioFile       string
		sampleRate      int
		threshold       float64
		silenceDuration int
		speechPad       int
		wantSegments    string
		expectedError   string
	}{
		{
			name:            "ok - detect voice activity (voice1)",
			audioFile:       "testdata/voice1.wav",
			sampleRate:      16000,
			threshold:       0.5,
			silenceDuration: 500,
			speechPad:       100,
			wantSegments:    "testdata/voice1-activity-segments.json",
		},
		{
			name:            "ok - detect voice activity (voice2)",
			audioFile:       "testdata/voice2.wav",
			sampleRate:      16000,
			threshold:       0.5,
			silenceDuration: 500,
			speechPad:       30,
			wantSegments:    "testdata/voice2-activity-segments.json",
		},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *qt.C) {
			component := Init(base.Component{})
			c.Assert(component, qt.IsNotNil)

			execution, err := component.CreateExecution(base.ComponentExecution{
				Component: component,
				Task:      taskDetectActivity,
			})
			c.Assert(err, qt.IsNil)
			c.Assert(execution, qt.IsNotNil)

			ir, ow, eh, job := mock.GenerateMockJob(c)

			// Load audio data
			audioFile, err := os.Open(tc.audioFile)
			c.Assert(err, qt.IsNil)
			defer audioFile.Close()
			audioData, err := io.ReadAll(audioFile)
			c.Assert(err, qt.IsNil)

			ir.ReadDataMock.Set(func(ctx context.Context, input any) error {
				switch input := input.(type) {
				case *detectActivityInput:
					audio, err := data.NewAudioFromBytes(audioData, "audio/wav", "input.wav")
					c.Assert(err, qt.IsNil)
					*input = detectActivityInput{
						Audio:              audio,
						MinSilenceDuration: tc.silenceDuration,
						SpeechPad:          tc.speechPad,
					}
				}
				return nil
			})

			var capturedOutput detectActivityOutput
			ow.WriteDataMock.Set(func(ctx context.Context, output any) error {
				capturedOutput = output.(detectActivityOutput)
				return nil
			})

			eh.ErrorMock.Set(func(ctx context.Context, err error) {
				c.Assert(err, qt.ErrorMatches, tc.expectedError)
			})

			if tc.expectedError != "" {
				ow.WriteDataMock.Optional()
			} else {
				eh.ErrorMock.Optional()
			}

			err = execution.Execute(context.Background(), []*base.Job{job})

			if tc.expectedError == "" {
				c.Assert(err, qt.IsNil)

				// Load expected segments
				expectedSegmentsJSONData, err := os.ReadFile(tc.wantSegments)
				c.Assert(err, qt.IsNil)
				var expectedSegmentsStruct struct {
					Segments []segmentData `instill:"segments"`
				}

				var segmentsMap map[string]interface{}
				err = json.Unmarshal(expectedSegmentsJSONData, &segmentsMap)
				c.Assert(err, qt.IsNil)

				jsonValue, err := data.NewJSONValue(segmentsMap)
				c.Assert(err, qt.IsNil)

				binaryFetcher := external.NewBinaryFetcher()
				unmarshaler := data.NewUnmarshaler(binaryFetcher)
				c.Assert(unmarshaler.Unmarshal(context.Background(), jsonValue, &expectedSegmentsStruct), qt.IsNil)
				expectedSegments := expectedSegmentsStruct.Segments

				c.Assert(capturedOutput.Segments, qt.HasLen, len(expectedSegments))

				for i, actual := range capturedOutput.Segments {
					expected := expectedSegments[i]
					c.Assert(actual.StartTime, floatEquals(0.1), expected.StartTime)
					c.Assert(actual.EndTime, floatEquals(0.1), expected.EndTime)
				}
			}
		})
	}
}

// floatEquals is a custom checker for comparing float64 values with an epsilon
func floatEquals(epsilon float64) qt.Checker {
	return qt.CmpEquals(cmp.Comparer(func(x, y float64) bool {
		return math.Abs(x-y) <= epsilon
	}))
}

func TestToMono(t *testing.T) {
	c := qt.New(t)

	stereoBuffer := &audio.IntBuffer{
		Data:   []int{1, 2, 3, 4, 5, 6},
		Format: &audio.Format{NumChannels: 2},
	}
	monoBuffer := toMono(stereoBuffer)
	c.Assert(monoBuffer.Format.NumChannels, qt.Equals, 1)
	c.Assert(monoBuffer.Data, qt.DeepEquals, []int{1, 3, 5})
}
