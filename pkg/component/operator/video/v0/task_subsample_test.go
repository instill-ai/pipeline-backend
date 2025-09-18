package video

import (
	"context"
	"os"
	"testing"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/mock"
	"github.com/instill-ai/pipeline-backend/pkg/data"
)

func TestSubsample(t *testing.T) {
	c := qt.New(t)

	testCases := []struct {
		name          string
		input         subsampleInput
		expectedFile  string
		expectedError string
	}{
		{
			name: "ok - subsample with FPS",
			input: subsampleInput{
				FPS: 15,
			},
			expectedFile: "testdata/video-subsample-fps-15.mp4",
		},
		{
			name: "ok - subsample with video bitrate",
			input: subsampleInput{
				VideoBitrate: 1000,
			},
			expectedFile: "testdata/video-subsample-vbr-1000.mp4",
		},
		{
			name: "ok - subsample with audio bitrate",
			input: subsampleInput{
				AudioBitrate: 128,
			},
			expectedFile: "testdata/video-subsample-abr-128.mp4",
		},
		{
			name: "ok - subsample with dimensions",
			input: subsampleInput{
				Width: 280,
			},
			expectedFile: "testdata/video-subsample-280x0.mp4",
		},
		{
			name:          "nok - no parameters provided",
			input:         subsampleInput{},
			expectedError: "at least one of video-bitrate, audio-bitrate, fps, width or height must be provided",
		},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *qt.C) {
			// Note: Removed c.Parallel() for subsample tests to reduce resource contention in CI
			component := Init(base.Component{})
			c.Assert(component, qt.IsNotNil)

			execution, err := component.CreateExecution(base.ComponentExecution{
				Component: component,
				Task:      "TASK_SUBSAMPLE",
			})
			c.Assert(err, qt.IsNil)
			c.Assert(execution, qt.IsNotNil)

			ir, ow, eh, job := mock.GenerateMockJob(c)

			ir.ReadDataMock.Set(func(ctx context.Context, input any) error {
				switch input := input.(type) {
				case *subsampleInput:
					videoBytes, err := os.ReadFile("testdata/video.mp4")
					if err != nil {
						return err
					}
					video, err := data.NewVideoFromBytes(videoBytes, data.MP4, "testdata/video.mp4", true)
					if err != nil {
						return err
					}
					tc.input.Video = video
					*input = tc.input
				}
				return nil
			})

			var capturedOutput any
			ow.WriteDataMock.Set(func(ctx context.Context, output any) error {
				capturedOutput = output

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
				output, ok := capturedOutput.(subsampleOutput)
				c.Assert(ok, qt.IsTrue)
				c.Assert(output.Video, qt.Not(qt.IsNil))

				// Read expected file
				expectedBytes, err := os.ReadFile(tc.expectedFile)
				c.Assert(err, qt.IsNil)
				expectedVideo, err := data.NewVideoFromBytes(expectedBytes, data.MP4, tc.expectedFile, true)
				c.Assert(err, qt.IsNil)

				// Compare videos with approximate matching
				compareVideo(c, output.Video, expectedVideo)
			}
		})
	}
}
