package image

import (
	"context"
	"image"
	"image/color"
	"testing"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/mock"
)

func TestCrop(t *testing.T) {
	c := qt.New(t)

	// Create a sample 100x100 image
	img := createTestImage(c, 100, 100, color.White)

	testCases := []struct {
		name           string
		input          cropInput
		expectedWidth  int
		expectedHeight int
		expectedError  string
		checkCorners   bool
	}{
		{
			name: "Rectangular crop",
			input: cropInput{
				Image:        img,
				TopOffset:    10,
				RightOffset:  20,
				BottomOffset: 30,
				LeftOffset:   40,
			},
			expectedWidth:  40,
			expectedHeight: 60,
		},
		{
			name: "Circular crop",
			input: cropInput{
				Image:        img,
				CircleRadius: 25,
			},
			expectedWidth:  100,
			expectedHeight: 100,
			checkCorners:   true,
		},
		{
			name: "Corner radius crop",
			input: cropInput{
				Image:        img,
				CornerRadius: 20,
			},
			expectedWidth:  100,
			expectedHeight: 100,
			checkCorners:   true,
		},
		{
			name: "No crop (all offsets 0)",
			input: cropInput{
				Image: img,
			},
			expectedWidth:  100,
			expectedHeight: 100,
		},
		{
			name: "Invalid crop dimensions",
			input: cropInput{
				Image:        img,
				TopOffset:    50,
				RightOffset:  50,
				BottomOffset: 51,
				LeftOffset:   50,
			},
			expectedError: "invalid crop dimensions",
		},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *qt.C) {
			component := Init(base.Component{})
			c.Assert(component, qt.IsNotNil)

			execution, err := component.CreateExecution(base.ComponentExecution{
				Component: component,
				Task:      "TASK_CROP",
			})
			c.Assert(err, qt.IsNil)
			c.Assert(execution, qt.IsNotNil)

			ir, ow, eh, job := mock.GenerateMockJob(c)
			ir.ReadDataMock.Set(func(ctx context.Context, input any) error {
				switch input := input.(type) {
				case *cropInput:
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
				output, ok := capturedOutput.(cropOutput)
				c.Assert(ok, qt.IsTrue)
				c.Assert(output.Image, qt.Not(qt.IsNil))

				// Check the dimensions of the output image
				bounds := output.Image
				c.Assert(bounds.Width().Integer(), qt.Equals, tc.expectedWidth)
				c.Assert(bounds.Height().Integer(), qt.Equals, tc.expectedHeight)

				// For circular and corner radius crop, check if corners are transparent
				if tc.checkCorners {
					decodedImg, err := decodeImage(output.Image)
					c.Assert(err, qt.IsNil)
					checkCornerTransparency(c, decodedImg)
				}
			}
		})
	}
}

func checkCornerTransparency(c *qt.C, img image.Image) {
	bounds := img.Bounds()
	corners := []image.Point{
		{X: bounds.Min.X, Y: bounds.Min.Y},
		{X: bounds.Max.X - 1, Y: bounds.Min.Y},
		{X: bounds.Min.X, Y: bounds.Max.Y - 1},
		{X: bounds.Max.X - 1, Y: bounds.Max.Y - 1},
	}

	for _, corner := range corners {
		_, _, _, a := img.At(corner.X, corner.Y).RGBA()
		c.Assert(a, qt.Equals, uint32(0), qt.Commentf("Corner at (%d, %d) should be transparent", corner.X, corner.Y))
	}
}
