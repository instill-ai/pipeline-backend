//go:build onnx
// +build onnx

// This task requires ONNX Runtime to be installed. Follow these steps to set it up:
//
// 1. Download ONNX Runtime:
//   - Visit the official repository: https://github.com/microsoft/onnxruntime/releases
//   - Choose the latest version compatible with your OS architecture
//
// 2. Install ONNX Runtime:
//   - Extract the downloaded tar file to a directory (referred to as ONNXRUNTIME_ROOT_PATH)
//   - Set up the environment:
//     a. Set the C_INCLUDE_PATH environment variable:
//     export C_INCLUDE_PATH=$ONNXRUNTIME_ROOT_PATH/include
//     b. Create a symlink to the ONNX Runtime libraries:
//     sudo ln -s ${ONNXRUNTIME_ROOT_PATH}/lib/libonnxruntime.so* /usr/lib/
//
// Note: Ensure you have the necessary permissions to create symlinks in /usr/lib/

package audio

import (
	"context"
	"math"
	"testing"

	"github.com/go-audio/audio"
	"google.golang.org/protobuf/types/known/structpb"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/mock"
)

func TestDetectActivity(t *testing.T) {
	c := qt.New(t)
	ctx := context.Background()

	testCases := []struct {
		name string

		task            string
		audioFile       string
		sampleRate      int
		threshold       float64
		silenceDuration int
		speechPad       int
		wantSegments    string
	}{
		{
			name: "ok - detect voice activity (voice1)",

			task:            taskDetectActivity,
			audioFile:       "testdata/voice1.wav",
			sampleRate:      16000,
			threshold:       0.5,
			silenceDuration: 100,
			speechPad:       30,
			wantSegments:    "testdata/voice1-activity-segments.json",
		},
		{
			name: "ok - detect voice activity (voice2)",

			task:            taskDetectActivity,
			audioFile:       "testdata/voice2.wav",
			sampleRate:      16000,
			threshold:       0.5,
			silenceDuration: 500,
			speechPad:       30,
			wantSegments:    "testdata/voice2-activity-segments.json",
		},
	}

	bo := base.Component{}
	cmp := Init(bo)

	for _, tc := range testCases {
		c.Run(tc.name, func(c *qt.C) {
			exec, err := cmp.CreateExecution(base.ComponentExecution{
				Component: cmp,
				Task:      tc.task,
			})
			c.Assert(err, qt.IsNil)

			input, err := structpb.NewStruct(map[string]interface{}{
				"audio":                loadBase64Audio(tc.audioFile),
				"sample-rate":          tc.sampleRate,
				"threshold":            tc.threshold,
				"min-silence-duration": tc.silenceDuration,
				"speech-pad":           tc.speechPad,
			})
			c.Assert(err, qt.IsNil)

			ir, ow, eh, job := mock.GenerateMockJob(c)
			ir.ReadMock.Return(input, nil)
			ow.WriteMock.Optional().Set(func(ctx context.Context, output *structpb.Struct) error {

				actualSegments := output.Fields["segments"].GetListValue().Values
				expectedSegments := loadWantSegments(c, tc.wantSegments)

				c.Assert(len(actualSegments), qt.Equals, len(expectedSegments))

				for i, expected := range expectedSegments {
					actual := actualSegments[i].GetStructValue().Fields
					compareSegments(c, actual, expected, i)
				}

				return nil
			})
			eh.ErrorMock.Optional().Set(func(ctx context.Context, err error) {
				c.Assert(err, qt.IsNil)
			})

			err = exec.Execute(ctx, []*base.Job{job})
			c.Assert(err, qt.IsNil)
		})
	}
}

func compareSegments(c *qt.C, actual map[string]*structpb.Value, expected interface{}, index int) {
	expectedMap := expected.(map[string]interface{})
	approxEquals := func(got, want float64) bool {
		return math.Abs(got-want) < 0.1
	}

	for _, key := range []string{"start-time", "end-time"} {
		c.Assert(
			approxEquals(actual[key].GetNumberValue(), expectedMap[key].(float64)),
			qt.IsTrue,
			qt.Commentf("%s mismatch at index %d: got %v, want %v", key, index, actual[key].GetNumberValue(), expectedMap[key]),
		)
	}
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

func TestResample(t *testing.T) {
	c := qt.New(t)
	resampled := resample([]float64{1, 2, 3, 4, 5, 6, 7, 8}, 44100, 22050)
	c.Assert(len(resampled), qt.Equals, 4)
}
