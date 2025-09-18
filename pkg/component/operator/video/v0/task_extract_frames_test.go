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

func TestExtractFrames(t *testing.T) {
	c := qt.New(t)

	testCases := []struct {
		name          string
		videoFile     string
		interval      float64
		timestamps    []float64
		wantFrames    []string
		expectedError string
	}{
		{
			name:      "ok - extract frames at interval",
			videoFile: "testdata/video.mp4",
			interval:  15,
			wantFrames: []string{
				"testdata/frame-0.png",
				"testdata/frame-1.png",
				"testdata/frame-2.png",
				"testdata/frame-3.png",
				"testdata/frame-4.png",
			},
		},
		{
			name:       "ok - extract frames at timestamps",
			videoFile:  "testdata/video.mp4",
			timestamps: []float64{0, 15, 30, 45, 60},
			wantFrames: []string{
				"testdata/frame-0.png",
				"testdata/frame-1.png",
				"testdata/frame-2.png",
				"testdata/frame-3.png",
				"testdata/frame-4.png",
			},
		},
		{
			name:          "nok - invalid video file",
			videoFile:     "invalid_video_data",
			expectedError: "reading input data: open invalid_video_data: no such file or directory",
		},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *qt.C) {
			c.Parallel()
			component := Init(base.Component{})
			c.Assert(component, qt.IsNotNil)

			execution, err := component.CreateExecution(base.ComponentExecution{
				Component: component,
				Task:      "TASK_EXTRACT_FRAMES",
			})
			c.Assert(err, qt.IsNil)
			c.Assert(execution, qt.IsNotNil)

			ir, ow, eh, job := mock.GenerateMockJob(c)

			ir.ReadDataMock.Set(func(ctx context.Context, input any) error {
				switch input := input.(type) {
				case *extractFramesInput:
					videoBytes, err := os.ReadFile(tc.videoFile)
					if err != nil {
						return err
					}
					video, err := data.NewVideoFromBytes(videoBytes, data.MP4, tc.videoFile, true)
					if err != nil {
						return err
					}
					*input = extractFramesInput{
						Video:      video,
						Interval:   tc.interval,
						Timestamps: tc.timestamps,
					}
				}
				return nil
			})

			var capturedOutput any
			ow.WriteDataMock.Set(func(ctx context.Context, output any) error {
				capturedOutput = output
				actualOutput := output.(extractFramesOutput)

				c.Assert(len(actualOutput.Frames), qt.Equals, len(tc.wantFrames))
				for i, wantFrame := range tc.wantFrames {
					expectedFrameBytes, err := os.ReadFile(wantFrame)
					if err != nil {
						return err
					}
					expectedFrame, err := data.NewImageFromBytes(expectedFrameBytes, data.PNG, wantFrame, true)
					if err != nil {
						return err
					}
					compareFrame(c, actualOutput.Frames[i], expectedFrame)
				}
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
				output, ok := capturedOutput.(extractFramesOutput)
				c.Assert(ok, qt.IsTrue)
				c.Assert(output.Frames, qt.Not(qt.IsNil))
			}
		})
	}
}
