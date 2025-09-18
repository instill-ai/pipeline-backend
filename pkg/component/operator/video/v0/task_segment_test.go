package video

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/mock"
	"github.com/instill-ai/pipeline-backend/pkg/data"
)

func TestSegment(t *testing.T) {
	c := qt.New(t)

	testCases := []struct {
		name          string
		videoFile     string
		segmentsFile  string
		expectedError string
	}{
		{
			name:         "ok - segment video with provided segments",
			videoFile:    "testdata/video.mp4",
			segmentsFile: "testdata/video-segments.json",
		},
		{
			name:      "ok - segment video without segments",
			videoFile: "testdata/video.mp4",
		},
		{
			name:          "nok - invalid video file",
			videoFile:     "invalid_video.mp4",
			expectedError: "reading input data: open invalid_video.mp4: no such file or directory",
		},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *qt.C) {
			c.Parallel()
			component := Init(base.Component{})
			c.Assert(component, qt.IsNotNil)

			execution, err := component.CreateExecution(base.ComponentExecution{
				Component: component,
				Task:      "TASK_SEGMENT",
			})
			c.Assert(err, qt.IsNil)
			c.Assert(execution, qt.IsNotNil)

			ir, ow, eh, job := mock.GenerateMockJob(c)

			ir.ReadDataMock.Set(func(ctx context.Context, input any) error {
				switch input := input.(type) {
				case *segmentInput:
					videoBytes, err := os.ReadFile(tc.videoFile)
					if err != nil {
						return err
					}
					video, err := data.NewVideoFromBytes(videoBytes, data.MP4, tc.videoFile, true)
					if err != nil {
						return err
					}
					*input = segmentInput{Video: video}

					if tc.segmentsFile != "" {
						segmentsData, err := os.ReadFile(tc.segmentsFile)
						if err != nil {
							return err
						}
						var data struct {
							Segments []*segmentData `json:"segments"`
						}
						if err := json.Unmarshal(segmentsData, &data); err != nil {
							return err
						}
						input.Segments = data.Segments
					}
				}
				return nil
			})

			var capturedOutput any
			ow.WriteDataMock.Set(func(ctx context.Context, output any) error {
				capturedOutput = output
				actualOutput := output.(segmentOutput)

				if tc.segmentsFile == "" {
					c.Assert(len(actualOutput.VideoSegments), qt.Equals, 1)
				} else {
					for i := 0; i < len(actualOutput.VideoSegments); i++ {
						expectedFile := fmt.Sprintf("testdata/video-segment-%d.mp4", i)
						expectedBytes, err := os.ReadFile(expectedFile)
						c.Assert(err, qt.IsNil)
						expectedVideo, err := data.NewVideoFromBytes(expectedBytes, data.MP4, expectedFile, true)
						c.Assert(err, qt.IsNil)
						compareVideo(c, actualOutput.VideoSegments[i], expectedVideo)
					}
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
				output, ok := capturedOutput.(segmentOutput)
				c.Assert(ok, qt.IsTrue)
				c.Assert(output.VideoSegments, qt.Not(qt.IsNil))
			}
		})
	}
}
