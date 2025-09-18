package image

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/mock"
	"github.com/instill-ai/pipeline-backend/pkg/data"
)

// TestDrawClassification tests the drawClassification function
func TestDrawClassification(t *testing.T) {
	c := quicktest.New(t)

	simpleClassificationData := `{
		"category": "test_class",
		"score": 0.95
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
			inputJSON:     []byte(simpleClassificationData),
			expectedError: "error decoding image: image: unknown format",
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
					img, err := data.NewImageFromBytes(tc.inputJPEG, data.PNG, "test", true)
					if err != nil {
						return err
					}
					var classificationResult struct {
						Category string  `instill:"category"`
						Score    float64 `instill:"score"`
					}
					err = json.Unmarshal(tc.inputJSON, &classificationResult)
					if err != nil {
						return err
					}
					*input = drawClassificationInput{
						Image:     img,
						Category:  classificationResult.Category,
						Score:     classificationResult.Score,
						ShowScore: true,
					}
				}
				return nil
			})

			var capturedOutput any
			ow.WriteDataMock.Set(func(ctx context.Context, output any) error {
				capturedOutput = output
				compareTestImage(c, output.(drawClassificationOutput).Image, "task_draw_classification")
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

			_ = execution.Execute(context.Background(), []*base.Job{job})

			if tc.expectedError == "" {
				c.Assert(err, quicktest.IsNil)
				output, ok := capturedOutput.(drawClassificationOutput)
				c.Assert(ok, quicktest.IsTrue)
				c.Assert(output.Image, quicktest.Not(quicktest.IsNil))
			}
		})
	}
}
