package image

import (
	"context"
	"encoding/json"
	"testing"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/mock"
	"github.com/instill-ai/pipeline-backend/pkg/data"
)

// TestDrawKeypoint tests the drawKeypoint function
func TestDrawKeypoint(t *testing.T) {
	c := qt.New(t)

	simpleKeypointData := `{
		"objects": [
			{
				"keypoints": [
					{"x": 10, "y": 10, "v": 2},
					{"x": 15, "y": 15, "v": 2}
				],
				"bounding_box": {
					"top": 5,
					"left": 5,
					"width": 20,
					"height": 20
				}
			}
		]
	}`

	testCases := []struct {
		name          string
		inputJPEG     []byte
		inputJSON     []byte
		expectedError string
	}{
		{
			name:          "Invalid Image",
			inputJPEG:     []byte("invalid image data"),
			inputJSON:     []byte(simpleKeypointData),
			expectedError: "error decoding image: image: unknown format",
		},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *qt.C) {
			component := Init(base.Component{})
			c.Assert(component, qt.IsNotNil)

			execution, err := component.CreateExecution(base.ComponentExecution{
				Component: component,
				Task:      "TASK_DRAW_KEYPOINT",
			})
			c.Assert(err, qt.IsNil)
			c.Assert(execution, qt.IsNotNil)

			ir, ow, eh, job := mock.GenerateMockJob(c)
			ir.ReadDataMock.Set(func(ctx context.Context, input any) error {
				switch input := input.(type) {
				case *drawKeypointInput:
					img, err := data.NewImageFromBytes(tc.inputJPEG, data.PNG, "test", true)
					if err != nil {
						return err
					}
					var keypointResult struct {
						Objects []*keypointObject `instill:"objects"`
					}
					err = json.Unmarshal(tc.inputJSON, &keypointResult)
					if err != nil {
						return err
					}
					*input = drawKeypointInput{
						Image:     img,
						Objects:   keypointResult.Objects,
						ShowScore: false,
					}
				}
				return nil
			})

			var capturedOutput any
			ow.WriteDataMock.Set(func(ctx context.Context, output any) error {
				capturedOutput = output
				compareTestImage(c, output.(drawKeypointOutput).Image, "task_draw_keypoint")
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

			_ = execution.Execute(context.Background(), []*base.Job{job})

			if tc.expectedError == "" {
				c.Assert(err, qt.IsNil)
				output, ok := capturedOutput.(drawKeypointOutput)
				c.Assert(ok, qt.IsTrue)
				c.Assert(output.Image, qt.Not(qt.IsNil))
			}
		})
	}
}
