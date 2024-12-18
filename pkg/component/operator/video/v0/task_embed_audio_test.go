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

func TestEmbedAudio(t *testing.T) {
	c := qt.New(t)

	testCases := []struct {
		name          string
		videoFile     string
		audioFile     string
		wantVideo     string
		expectedError string
	}{
		{
			name:      "ok - extract audio from video",
			videoFile: "testdata/video-sample-bunny.mp4",
			audioFile: "testdata/audio.ogg",
			wantVideo: "testdata/embed-video.mp4",
		},
		{
			name:          "nok - invalid video file",
			videoFile:     "invalid_video_data",
			expectedError: "reading input data: open invalid_video_data: no such file or directory",
		},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *qt.C) {
			component := Init(base.Component{})
			c.Assert(component, qt.IsNotNil)

			execution, err := component.CreateExecution(base.ComponentExecution{
				Component: component,
				Task:      "TASK_EMBED_AUDIO",
			})
			c.Assert(err, qt.IsNil)
			c.Assert(execution, qt.IsNotNil)

			ir, ow, eh, job := mock.GenerateMockJob(c)

			ir.ReadDataMock.Set(func(ctx context.Context, input any) error {
				switch input := input.(type) {
				case *embedAudioInput:
					videoBytes, err := os.ReadFile(tc.videoFile)
					if err != nil {
						return err
					}
					video, err := data.NewVideoFromBytes(videoBytes, "video/mp4", tc.videoFile)
					if err != nil {
						return err
					}
					audioBytes, err := os.ReadFile(tc.audioFile)
					if err != nil {
						return err
					}
					audio, err := data.NewAudioFromBytes(audioBytes, "audio/ogg", tc.audioFile)
					if err != nil {
						return err
					}
					*input = embedAudioInput{
						Video: video,
						Audio: audio,
					}
				}
				return nil
			})

			var capturedOutput any
			ow.WriteDataMock.Set(func(ctx context.Context, output any) error {
				capturedOutput = output
				actualOutput := output.(embedAudioOutput)

				expectedVideoBytes, err := os.ReadFile(tc.wantVideo)
				if err != nil {
					return err
				}
				expectedVideo, err := data.NewVideoFromBytes(expectedVideoBytes, "video/mp4", tc.wantVideo)
				if err != nil {
					return err
				}

				compareVideo(c, actualOutput.Video, expectedVideo)
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
				output, ok := capturedOutput.(embedAudioOutput)
				c.Assert(ok, qt.IsTrue)
				c.Assert(output.Video, qt.Not(qt.IsNil))
			}
		})
	}
}
