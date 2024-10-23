package image

import (
	"context"
	"encoding/json"
	"testing"

	_ "embed"

	"github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/mock"
	"github.com/instill-ai/pipeline-backend/pkg/data"
)

//go:embed testdata/cls-dog.json
var clsDogJSON []byte

//go:embed testdata/cls-dog.jpeg
var clsDogJPEG []byte

// TestDrawClassification tests the drawClassification function
func TestDrawClassification(t *testing.T) {
	c := quicktest.New(t)

	testCases := []struct {
		name      string
		inputJPEG []byte
		inputJSON []byte

		expectedError  string
		expectedOutput bool
	}{
		{
			name:           "Classification Dog",
			inputJPEG:      clsDogJPEG,
			inputJSON:      clsDogJSON,
			expectedOutput: true,
		},
		{
			name:          "Invalid Image",
			inputJPEG:     []byte("invalid image data"),
			inputJSON:     nil,
			expectedError: "convert image: failed to decode source image: invalid JPEG format: missing SOI marker",
		},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *quicktest.C) {
			component := Init(base.Component{})
			c.Assert(component, quicktest.IsNotNil)

			execution, err := component.CreateExecution(base.ComponentExecution{
				Component: component,
				Task:      "TASK_DRAW_CLASSIFICATION",
			})
			c.Assert(err, quicktest.IsNil)
			c.Assert(execution, quicktest.IsNotNil)

			ir, ow, eh, job := mock.GenerateMockJob(c)
			ir.ReadDataMock.Set(func(ctx context.Context, input any) error {
				switch input := input.(type) {
				case *drawClassificationInput:
					img, err := data.NewImageFromBytes(tc.inputJPEG, "image/jpeg", "test")
					if err != nil {
						return err
					}
					var classificationResult struct {
						Category string  `json:"category"`
						Score    float64 `json:"score"`
					}
					err = json.Unmarshal(tc.inputJSON, &classificationResult)
					if err != nil {
						return err
					}
					*input = drawClassificationInput{
						Image:    img,
						Category: classificationResult.Category,
						Score:    classificationResult.Score,
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
				c.Assert(err, quicktest.ErrorMatches, tc.expectedError)
			})
			if tc.expectedError != "" {
				ow.WriteDataMock.Optional()
			} else {
				eh.ErrorMock.Optional()
			}

			err = execution.Execute(context.Background(), []*base.Job{job})

			if tc.expectedError == "" {
				c.Assert(err, quicktest.IsNil)
				output, ok := capturedOutput.(drawClassificationOutput)
				c.Assert(ok, quicktest.IsTrue)
				c.Assert(output.Image, quicktest.Not(quicktest.IsNil))
			}
		})
	}
}
