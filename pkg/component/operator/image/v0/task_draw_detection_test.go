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

// TestDrawDetection tests the drawDetection function
func TestDrawDetection(t *testing.T) {
	c := qt.New(t)

	simpleDetectionData := `{
		"objects": [
			{
				"category": "test_object",
				"score": 0.9,
				"bounding_box": {
					"top": 5,
					"left": 5,
					"width": 15,
					"height": 15
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
			inputJSON:     []byte(simpleDetectionData),
			expectedError: "error decoding image: image: unknown format",
		},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *qt.C) {
			component := Init(base.Component{})
			c.Assert(component, qt.IsNotNil)

			execution, err := component.CreateExecution(base.ComponentExecution{
				Component: component,
				Task:      "TASK_DRAW_DETECTION",
			})
			c.Assert(err, qt.IsNil)
			c.Assert(execution, qt.IsNotNil)

			ir, ow, eh, job := mock.GenerateMockJob(c)
			ir.ReadDataMock.Set(func(ctx context.Context, input any) error {
				switch input := input.(type) {
				case *drawDetectionInput:
					img, err := data.NewImageFromBytes(tc.inputJPEG, data.PNG, "test", true)
					if err != nil {
						return err
					}
					var detectionResult struct {
						Objects []*detectionObject `instill:"objects"`
					}
					err = json.Unmarshal(tc.inputJSON, &detectionResult)
					if err != nil {
						return err
					}
					*input = drawDetectionInput{
						Image:     img,
						Objects:   detectionResult.Objects,
						ShowScore: true,
					}
				}
				return nil
			})

			ow.WriteDataMock.Set(func(ctx context.Context, output any) error {
				compareTestImage(c, output.(drawDetectionOutput).Image, "task_draw_detection")
				return nil
			})
			eh.ErrorMock.Set(func(ctx context.Context, err error) {
				c.Assert(err, qt.ErrorMatches, tc.expectedError)
			})
			ow.WriteDataMock.Optional()

			_ = execution.Execute(context.Background(), []*base.Job{job})
		})
	}
}
