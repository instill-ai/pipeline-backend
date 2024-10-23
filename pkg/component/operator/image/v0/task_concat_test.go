package image

import (
	"context"
	"image/color"
	"testing"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/mock"
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
)

func TestConcat(t *testing.T) {
	c := qt.New(t)

	// Create sample images
	img1 := createTestImage(c, 50, 50, color.RGBA{255, 0, 0, 255})   // Red
	img2 := createTestImage(c, 50, 50, color.RGBA{0, 255, 0, 255})   // Green
	img3 := createTestImage(c, 50, 50, color.RGBA{0, 0, 255, 255})   // Blue
	img4 := createTestImage(c, 50, 50, color.RGBA{255, 255, 0, 255}) // Yellow

	testCases := []struct {
		name           string
		input          concatInput
		expectedWidth  int
		expectedHeight int
		expectedError  string
	}{
		{
			name: "2x2 grid with padding",
			input: concatInput{
				Images: []format.Image{
					img1,
					img2,
					img3,
					img4,
				},
				GridWidth: 2,
				Padding:   10,
			},
			expectedWidth:  110,
			expectedHeight: 110,
		},
		{
			name: "1x4 grid without padding",
			input: concatInput{
				Images: []format.Image{
					img1,
					img2,
					img3,
					img4,
				},
				GridHeight: 1,
			},
			expectedWidth:  200,
			expectedHeight: 50,
		},
		{
			name: "Default square grid",
			input: concatInput{
				Images: []format.Image{
					img1,
					img2,
					img3,
					img4,
				},
			},
			expectedWidth:  100,
			expectedHeight: 100,
		},
		{
			name: "Invalid input (no images)",
			input: concatInput{
				Images: []format.Image{},
			},
			expectedError: "no images provided",
		},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *qt.C) {
			component := Init(base.Component{})
			c.Assert(component, qt.IsNotNil)

			execution, err := component.CreateExecution(base.ComponentExecution{
				Component: component,
				Task:      "TASK_CONCAT",
			})
			c.Assert(err, qt.IsNil)
			c.Assert(execution, qt.IsNotNil)

			ir, ow, eh, job := mock.GenerateMockJob(c)
			ir.ReadDataMock.Set(func(ctx context.Context, input any) error {
				switch input := input.(type) {
				case *concatInput:
					*input = tc.input
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
				output, ok := capturedOutput.(concatOutput)
				c.Assert(ok, qt.IsTrue)
				c.Assert(output.Image, qt.Not(qt.IsNil))

				// Check the dimensions of the output image
				bounds := output.Image
				c.Assert(bounds.Width().Integer(), qt.Equals, tc.expectedWidth)
				c.Assert(bounds.Height().Integer(), qt.Equals, tc.expectedHeight)
			}
		})
	}
}
