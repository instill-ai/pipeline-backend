package audio

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"testing"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/mock"
	"github.com/instill-ai/pipeline-backend/pkg/data"
	"github.com/instill-ai/pipeline-backend/pkg/external"
)

func TestSegment(t *testing.T) {
	c := qt.New(t)

	testCases := []struct {
		name          string
		audioFile     string
		segmentsFile  string
		expectedCount int
		expectedError string
	}{
		{
			name:          "ok - valid segmentation",
			audioFile:     "testdata/voice1.wav",
			segmentsFile:  "testdata/voice1-activity-segments.json",
			expectedCount: 5,
		},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *qt.C) {
			component := Init(base.Component{})
			c.Assert(component, qt.IsNotNil)

			execution, err := component.CreateExecution(base.ComponentExecution{
				Component: component,
				Task:      taskSegment,
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

			// Load segments data
			segmentsJSONData, err := os.ReadFile(tc.segmentsFile)
			c.Assert(err, qt.IsNil)
			var segmentsStruct struct {
				Segments []segmentData `instill:"segments"`
			}

			var segmentsMap map[string]interface{}
			err = json.Unmarshal(segmentsJSONData, &segmentsMap)
			c.Assert(err, qt.IsNil)

			jsonValue, err := data.NewJSONValue(segmentsMap)
			c.Assert(err, qt.IsNil)

			binaryFetcher := external.NewBinaryFetcher()
			unmarshaler := data.NewUnmarshaler(context.Background(), binaryFetcher)
			c.Assert(unmarshaler.Unmarshal(jsonValue, &segmentsStruct), qt.IsNil)
			segments := segmentsStruct.Segments

			ir.ReadDataMock.Set(func(ctx context.Context, input any) error {
				switch input := input.(type) {
				case *segmentInput:
					audio, err := data.NewAudioFromBytes(audioData, "audio/wav", "input.wav")
					c.Assert(err, qt.IsNil)
					*input = segmentInput{
						Audio:    audio,
						Segments: segments,
					}
				}
				return nil
			})

			var capturedOutput segmentOutput
			ow.WriteDataMock.Set(func(ctx context.Context, output any) error {
				capturedOutput = output.(segmentOutput)
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
				c.Assert(capturedOutput.AudioSegments, qt.HasLen, tc.expectedCount)

				for i, segment := range capturedOutput.AudioSegments {
					c.Assert(segment, qt.Not(qt.IsNil), qt.Commentf("Segment %d is nil", i))
					c.Assert(segment.ContentType().String(), qt.Equals, "audio/ogg", qt.Commentf("Segment %d has incorrect MIME type", i))
				}
			}
		})
	}
}
