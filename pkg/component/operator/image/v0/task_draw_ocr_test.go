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

// TestDrawOCR tests the drawOCR function
func TestDrawOCR(t *testing.T) {
	c := qt.New(t)

	simpleOCRData := `{
		"objects": [
			{
				"text": "Test",
				"bounding_box": {
					"top": 5,
					"left": 5,
					"width": 20,
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
			inputJSON:     []byte(simpleOCRData),
			expectedError: "error decoding image: image: unknown format",
		},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *qt.C) {
			component := Init(base.Component{})
			c.Assert(component, qt.IsNotNil)

			execution, err := component.CreateExecution(base.ComponentExecution{
				Component: component,
				Task:      "TASK_DRAW_OCR",
			})
			c.Assert(err, qt.IsNil)
			c.Assert(execution, qt.IsNotNil)

			ir, ow, eh, job := mock.GenerateMockJob(c)
			ir.ReadDataMock.Set(func(ctx context.Context, input any) error {
				switch input := input.(type) {
				case *drawOCRInput:
					img, err := data.NewImageFromBytes(tc.inputJPEG, data.PNG, "test", true)
					if err != nil {
						return err
					}
					var ocrResult struct {
						Objects []*ocrObject `instill:"objects"`
					}
					err = json.Unmarshal(tc.inputJSON, &ocrResult)
					if err != nil {
						return err
					}
					*input = drawOCRInput{
						Image:   img,
						Objects: ocrResult.Objects,
					}
				}
				return nil
			})

			var capturedOutput any
			ow.WriteDataMock.Set(func(ctx context.Context, output any) error {
				capturedOutput = output
				compareTestImage(c, output.(drawOCROutput).Image, "task_draw_ocr")
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
				output, ok := capturedOutput.(drawOCROutput)
				c.Assert(ok, qt.IsTrue)
				c.Assert(output.Image, qt.Not(qt.IsNil))
			}
		})
	}
}
