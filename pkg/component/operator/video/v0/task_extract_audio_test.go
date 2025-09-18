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

func TestExtractAudio(t *testing.T) {
	c := qt.New(t)

	testCases := []struct {
		name          string
		videoFile     string
		wantAudio     string
		expectedError string
	}{
		{
			name:      "ok - extract audio from video",
			videoFile: "testdata/video.mp4",
			wantAudio: "testdata/audio.ogg",
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
				Task:      "TASK_EXTRACT_AUDIO",
			})
			c.Assert(err, qt.IsNil)
			c.Assert(execution, qt.IsNotNil)

			ir, ow, eh, job := mock.GenerateMockJob(c)

			ir.ReadDataMock.Set(func(ctx context.Context, input any) error {
				switch input := input.(type) {
				case *extractAudioInput:
					videoBytes, err := os.ReadFile(tc.videoFile)
					if err != nil {
						return err
					}
					video, err := data.NewVideoFromBytes(videoBytes, data.MP4, tc.videoFile, true)
					if err != nil {
						return err
					}
					*input = extractAudioInput{
						Video: video,
					}
				}
				return nil
			})

			var capturedOutput any
			ow.WriteDataMock.Set(func(ctx context.Context, output any) error {
				capturedOutput = output
				actualOutput := output.(extractAudioOutput)

				expectedAudioBytes, err := os.ReadFile(tc.wantAudio)
				if err != nil {
					return err
				}
				expectedAudio, err := data.NewAudioFromBytes(expectedAudioBytes, data.OGG, tc.wantAudio, true)
				if err != nil {
					return err
				}

				compareAudio(c, actualOutput.Audio, expectedAudio)
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
				output, ok := capturedOutput.(extractAudioOutput)
				c.Assert(ok, qt.IsTrue)
				c.Assert(output.Audio, qt.Not(qt.IsNil))
			}
		})
	}
}
