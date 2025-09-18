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

// TestDrawInstanceSegmentation tests the drawInstanceSegmentation function
func TestDrawInstanceSegmentation(t *testing.T) {
	c := qt.New(t)

	simpleInstanceData := `{
		"objects": [
			{
				"category": "test_object",
				"rle": "0,100,100,100,100,0",
				"bounding_box": {
					"top": 5,
					"left": 5,
					"width": 10,
					"height": 10
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
			inputJSON:     []byte(simpleInstanceData),
			expectedError: "error decoding image: image: unknown format",
		},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *qt.C) {
			component := Init(base.Component{})
			c.Assert(component, qt.IsNotNil)

			execution, err := component.CreateExecution(base.ComponentExecution{
				Component: component,
				Task:      "TASK_DRAW_INSTANCE_SEGMENTATION",
			})
			c.Assert(err, qt.IsNil)
			c.Assert(execution, qt.IsNotNil)

			ir, ow, eh, job := mock.GenerateMockJob(c)
			ir.ReadDataMock.Set(func(ctx context.Context, input any) error {
				switch input := input.(type) {
				case *drawInstanceSegmentationInput:
					img, err := data.NewImageFromBytes(tc.inputJPEG, data.PNG, "test", true)
					if err != nil {
						return err
					}
					var segmentationResult struct {
						Objects []*instanceSegmentationObject `instill:"objects"`
					}
					err = json.Unmarshal(tc.inputJSON, &segmentationResult)
					if err != nil {
						return err
					}
					*input = drawInstanceSegmentationInput{
						Image:     img,
						Objects:   segmentationResult.Objects,
						ShowScore: true,
					}
				}
				return nil
			})

			var capturedOutput any
			ow.WriteDataMock.Set(func(ctx context.Context, output any) error {
				capturedOutput = output
				compareTestImage(c, output.(drawInstanceSegmentationOutput).Image, "task_draw_instance_segmentation")
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
				output, ok := capturedOutput.(drawInstanceSegmentationOutput)
				c.Assert(ok, qt.IsTrue)
				c.Assert(output.Image, qt.Not(qt.IsNil))
			}
		})
	}
}
