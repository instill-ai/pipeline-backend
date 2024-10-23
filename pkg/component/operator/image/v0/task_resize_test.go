package image

import (
	"context"
	"image"
	"testing"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/mock"
)

func TestResize(t *testing.T) {
	c := qt.New(t)

	// Create a sample 100x100 image
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	imgData, err := encodeImage(img)
	c.Assert(err, qt.IsNil)

	testCases := []struct {
		name           string
		input          resizeInput
		expectedWidth  int
		expectedHeight int
		expectedError  string
	}{
		{
			name: "Resize by width and height",
			input: resizeInput{
				Image:  imgData,
				Width:  80,
				Height: 20,
			},
			expectedWidth:  80,
			expectedHeight: 20,
		},
		{
			name: "Resize by ratio",
			input: resizeInput{
				Image: imgData,
				Ratio: 0.2,
			},
			expectedWidth:  20,
			expectedHeight: 20,
		},
		{
			name: "Resize by ratio 0",
			input: resizeInput{
				Image: imgData,
				Ratio: 0,
			},
			expectedWidth:  100,
			expectedHeight: 100,
		},
		{
			name: "Resize by width and height 0",
			input: resizeInput{
				Image:  imgData,
				Width:  0,
				Height: 0,
			},
			expectedWidth:  100,
			expectedHeight: 100,
		},
		{
			name: "Resize by width 0",
			input: resizeInput{
				Image:  imgData,
				Width:  0,
				Height: 100,
			},
			expectedWidth:  100,
			expectedHeight: 100,
		},
		{
			name: "Negative ratio",
			input: resizeInput{
				Image: imgData,
				Ratio: -0.5,
			},
			expectedError: "ratio must be between 0 and 1",
		},
		{
			name: "Negative width",
			input: resizeInput{
				Image:  imgData,
				Width:  -100,
				Height: 100,
			},
			expectedError: "width and height must be greater than or equal to 0",
		},
		{
			name: "Negative height",
			input: resizeInput{
				Image:  imgData,
				Width:  100,
				Height: -100,
			},
			expectedError: "width and height must be greater than or equal to 0",
		},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *qt.C) {
			component := Init(base.Component{})
			c.Assert(component, qt.IsNotNil)

			execution, err := component.CreateExecution(base.ComponentExecution{
				Component: component,
				Task:      "TASK_RESIZE",
			})
			c.Assert(err, qt.IsNil)
			c.Assert(execution, qt.IsNotNil)

			ir, ow, eh, job := mock.GenerateMockJob(c)
			ir.ReadDataMock.Set(func(ctx context.Context, input any) error {
				switch input := input.(type) {
				case *resizeInput:
					*input = tc.input
				}
				return nil
			})

			var capturedOutput any
			ow.WriteDataMock.Set(func(ctx context.Context, output any) error {
				capturedOutput = output
				compareTestImage(c, output.(resizeOutput).Image, "task_resize")
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
				output, ok := capturedOutput.(resizeOutput)
				c.Assert(ok, qt.IsTrue)
				c.Assert(output.Image, qt.Not(qt.IsNil))

				// Decode the output image
				decodedImg, err := decodeImage(output.Image)
				c.Assert(err, qt.IsNil)

				// Check if the output image dimensions match the expected dimensions
				c.Assert(decodedImg.Bounds().Dx(), qt.Equals, tc.expectedWidth)
				c.Assert(decodedImg.Bounds().Dy(), qt.Equals, tc.expectedHeight)
			}
		})
	}
}
