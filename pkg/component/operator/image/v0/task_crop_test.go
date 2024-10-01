package image

import (
	"context"
	"image"
	"testing"

	"github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func TestCrop(t *testing.T) {
	c := quicktest.New(t)

	// Create a sample 100x100 image
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			img.Set(x, y, image.White)
		}
	}
	base64Img, err := encodeBase64Image(img)
	c.Assert(err, quicktest.IsNil)

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
				Image:        base64Image("data:image/png;base64," + base64Img),
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
				Image:        base64Image("data:image/png;base64," + base64Img),
				CircleRadius: 25,
			},
			expectedWidth:  100,
			expectedHeight: 100,
			checkCorners:   true,
		},
		{
			name: "Corner radius crop",
			input: cropInput{
				Image:        base64Image("data:image/png;base64," + base64Img),
				CornerRadius: 20,
			},
			expectedWidth:  100,
			expectedHeight: 100,
			checkCorners:   true,
		},
		{
			name: "No crop (all offsets 0)",
			input: cropInput{
				Image: base64Image("data:image/png;base64," + base64Img),
			},
			expectedWidth:  100,
			expectedHeight: 100,
		},
		{
			name: "Invalid crop dimensions",
			input: cropInput{
				Image:        base64Image("data:image/png;base64," + base64Img),
				TopOffset:    50,
				RightOffset:  50,
				BottomOffset: 51,
				LeftOffset:   50,
			},
			expectedError: "invalid crop dimensions",
		},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *quicktest.C) {
			inputStruct, err := base.ConvertToStructpb(tc.input)
			c.Assert(err, quicktest.IsNil)

			output, err := crop(inputStruct, nil, context.Background())

			if tc.expectedError != "" {
				c.Assert(err, quicktest.ErrorMatches, tc.expectedError)
			} else {
				c.Assert(err, quicktest.IsNil)

				var croppedOutput cropOutput
				err = base.ConvertFromStructpb(output, &croppedOutput)
				c.Assert(err, quicktest.IsNil)

				// Decode the output image as PNG
				decodedImg, err := decodeBase64Image(string(croppedOutput.Image)[22:]) // Remove "data:image/png;base64," prefix
				c.Assert(err, quicktest.IsNil)

				// Check if the output image dimensions match the expected dimensions
				c.Assert(decodedImg.Bounds().Dx(), quicktest.Equals, tc.expectedWidth)
				c.Assert(decodedImg.Bounds().Dy(), quicktest.Equals, tc.expectedHeight)

				// For circular and corner radius crop, check if corners are transparent
				if tc.checkCorners {
					checkCornerTransparency(c, decodedImg)
				}
			}
		})
	}
}

func checkCornerTransparency(c *quicktest.C, img image.Image) {
	bounds := img.Bounds()
	corners := []image.Point{
		{X: bounds.Min.X, Y: bounds.Min.Y},
		{X: bounds.Max.X - 1, Y: bounds.Min.Y},
		{X: bounds.Min.X, Y: bounds.Max.Y - 1},
		{X: bounds.Max.X - 1, Y: bounds.Max.Y - 1},
	}

	for _, corner := range corners {
		_, _, _, a := img.At(corner.X, corner.Y).RGBA()
		c.Assert(a, quicktest.Equals, uint32(0), quicktest.Commentf("Corner at (%d, %d) should be transparent", corner.X, corner.Y))
	}
}
