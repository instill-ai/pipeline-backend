package image

import (
	"context"
	"encoding/json"
	"image/color"
	"testing"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/mock"
	"github.com/instill-ai/pipeline-backend/pkg/data"
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
)

// Removed unused embedded semantic segmentation data for performance optimization

// TestDrawSemanticSegmentation tests the drawSemanticSegmentation function
func TestDrawSemanticSegmentation(t *testing.T) {
	c := qt.New(t)

	// Create a simple test image and segmentation data for faster testing
	simpleTestImage := createTestImage(c, 10, 10, color.White)
	simpleSegmentationData := `{
		"stuffs": [
			{
				"category": "test_category",
				"rle": "0,50,50,0"
			}
		]
	}`

	testCases := []struct {
		name      string
		inputJPEG []byte
		inputJSON []byte
		useSimple bool

		expectedError  string
		expectedOutput bool
	}{
		{
			name:           "Simple Semantic Segmentation",
			useSimple:      true,
			expectedOutput: true,
		},
		{
			name:          "Invalid Image",
			inputJPEG:     []byte("invalid image data"),
			inputJSON:     []byte(simpleSegmentationData),
			expectedError: "error decoding image: image: unknown format",
		},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *qt.C) {
			component := Init(base.Component{})
			c.Assert(component, qt.IsNotNil)

			execution, err := component.CreateExecution(base.ComponentExecution{
				Component: component,
				Task:      "TASK_DRAW_SEMANTIC_SEGMENTATION",
			})
			c.Assert(err, qt.IsNil)
			c.Assert(execution, qt.IsNotNil)

			ir, ow, eh, job := mock.GenerateMockJob(c)
			ir.ReadDataMock.Set(func(ctx context.Context, input any) error {
				switch input := input.(type) {
				case *drawSemanticSegmentationInput:
					var img format.Image
					var segmentationData []byte

					if tc.useSimple {
						img = simpleTestImage
						segmentationData = []byte(simpleSegmentationData)
					} else {
						var err error
						img, err = data.NewImageFromBytes(tc.inputJPEG, data.PNG, "test", true)
						if err != nil {
							return err
						}
						segmentationData = tc.inputJSON
					}

					var segmentationResult struct {
						Stuffs []*semanticSegmentationStuff `instill:"stuffs"`
					}
					err := json.Unmarshal(segmentationData, &segmentationResult)
					if err != nil {
						return err
					}
					*input = drawSemanticSegmentationInput{
						Image:  img,
						Stuffs: segmentationResult.Stuffs,
					}
				}
				return nil
			})

			var capturedOutput any
			ow.WriteDataMock.Set(func(ctx context.Context, output any) error {
				capturedOutput = output
				compareTestImage(c, output.(drawSemanticSegmentationOutput).Image, "task_draw_semantic_segmentation")
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
				output, ok := capturedOutput.(drawSemanticSegmentationOutput)
				c.Assert(ok, qt.IsTrue)
				c.Assert(output.Image, qt.Not(qt.IsNil))
			}
		})
	}
}
