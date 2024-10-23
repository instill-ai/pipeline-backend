package image

import (
	"context"
	"encoding/json"
	"testing"

	_ "embed"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/mock"
	"github.com/instill-ai/pipeline-backend/pkg/data"
)

//go:embed testdata/det-coco-1.json
var detCOCO1JSON []byte

//go:embed testdata/det-coco-2.json
var detCOCO2JSON []byte

//go:embed testdata/det-coco-1.jpeg
var detCOCO1JPEG []byte

//go:embed testdata/det-coco-2.jpeg
var detCOCO2JPEG []byte

// TestDrawDetection tests the drawDetection function
func TestDrawDetection(t *testing.T) {
	c := qt.New(t)

	testCases := []struct {
		name      string
		inputJPEG []byte
		inputJSON []byte

		expectedError  string
		expectedOutput bool
	}{
		{
			name:           "Detection COCO 1",
			inputJPEG:      detCOCO1JPEG,
			inputJSON:      detCOCO1JSON,
			expectedOutput: true,
		},
		{
			name:           "Detection COCO 2",
			inputJPEG:      detCOCO2JPEG,
			inputJSON:      detCOCO2JSON,
			expectedOutput: true,
		},
		{
			name:          "Invalid Image",
			inputJPEG:     []byte("invalid image data"),
			inputJSON:     detCOCO1JSON,
			expectedError: "convert image: failed to decode source image: invalid JPEG format: missing SOI marker",
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
					img, err := data.NewImageFromBytes(tc.inputJPEG, "image/jpeg", "test")
					if err != nil {
						return err
					}
					var detectionResult struct {
						Objects []*detectionObject `json:"objects"`
					}
					err = json.Unmarshal(tc.inputJSON, &detectionResult)
					if err != nil {
						return err
					}
					*input = drawDetectionInput{
						Image:   img,
						Objects: detectionResult.Objects,
					}
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
				output, ok := capturedOutput.(drawDetectionOutput)
				c.Assert(ok, qt.IsTrue)
				c.Assert(output.Image, qt.Not(qt.IsNil))
			}
		})
	}
}
