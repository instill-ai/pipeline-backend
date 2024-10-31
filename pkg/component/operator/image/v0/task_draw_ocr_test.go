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

//go:embed testdata/ocr-mm.json
var ocrMMJSON []byte

//go:embed testdata/ocr-mm.jpeg
var ocrMMJPEG []byte

// TestDrawOCR tests the drawOCR function
func TestDrawOCR(t *testing.T) {
	c := qt.New(t)

	testCases := []struct {
		name      string
		inputJPEG []byte
		inputJSON []byte

		expectedError  string
		expectedOutput bool
	}{
		{
			name:           "OCR MM",
			inputJPEG:      ocrMMJPEG,
			inputJSON:      ocrMMJSON,
			expectedOutput: true,
		},
		{
			name:          "Invalid Image",
			inputJPEG:     []byte("invalid image data"),
			inputJSON:     ocrMMJSON,
			expectedError: "convert image: failed to decode source image: invalid JPEG format: missing SOI marker",
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
					img, err := data.NewImageFromBytes(tc.inputJPEG, "image/jpeg", "test")
					if err != nil {
						return err
					}
					var ocrResult struct {
						Objects []*ocrObject `json:"objects"`
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

			err = execution.Execute(context.Background(), []*base.Job{job})

			if tc.expectedError == "" {
				c.Assert(err, qt.IsNil)
				output, ok := capturedOutput.(drawOCROutput)
				c.Assert(ok, qt.IsTrue)
				c.Assert(output.Image, qt.Not(qt.IsNil))
			}
		})
	}
}
