package image

import (
	"context"
	"image"
	"testing"

	"github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func TestResize(t *testing.T) {
	c := quicktest.New(t)

	// Create a sample 100x100 image
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	base64Img, err := encodeBase64Image(img)
	c.Assert(err, quicktest.IsNil)

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
				Image:  base64Image("data:image/png;base64," + base64Img),
				Width:  80,
				Height: 20,
			},
			expectedWidth:  80,
			expectedHeight: 20,
		},
		{
			name: "Resize by ratio",
			input: resizeInput{
				Image: base64Image("data:image/png;base64," + base64Img),
				Ratio: 0.2,
			},
			expectedWidth:  20,
			expectedHeight: 20,
		},
		{
			name: "Resize by ratio 0",
			input: resizeInput{
				Image: base64Image("data:image/png;base64," + base64Img),
				Ratio: 0,
			},
			expectedWidth:  100,
			expectedHeight: 100,
		},
		{
			name: "Resize by width and height 0",
			input: resizeInput{
				Image:  base64Image("data:image/png;base64," + base64Img),
				Width:  0,
				Height: 0,
			},
			expectedWidth:  100,
			expectedHeight: 100,
		},
		{
			name: "Resize by width 0",
			input: resizeInput{
				Image:  base64Image("data:image/png;base64," + base64Img),
				Width:  0,
				Height: 100,
			},
			expectedWidth:  100,
			expectedHeight: 100,
		},
		{
			name: "Negative ratio",
			input: resizeInput{
				Image: base64Image("data:image/png;base64," + base64Img),
				Ratio: -0.5,
			},
			expectedError: "ratio must be between 0 and 1",
		},
		{
			name: "Negative width",
			input: resizeInput{
				Image:  base64Image("data:image/png;base64," + base64Img),
				Width:  -100,
				Height: 100,
			},
			expectedError: "width and height must be greater than or equal to 0",
		},
		{
			name: "Negative height",
			input: resizeInput{
				Image:  base64Image("data:image/png;base64," + base64Img),
				Width:  100,
				Height: -100,
			},
			expectedError: "width and height must be greater than or equal to 0",
		},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *quicktest.C) {
			inputStruct, err := base.ConvertToStructpb(tc.input)
			c.Assert(err, quicktest.IsNil)

			output, err := resize(inputStruct, nil, context.Background())

			if tc.expectedError != "" {
				c.Assert(err, quicktest.ErrorMatches, tc.expectedError)
			} else {
				c.Assert(err, quicktest.IsNil)

				var resizedOutput resizeOutput
				err = base.ConvertFromStructpb(output, &resizedOutput)
				c.Assert(err, quicktest.IsNil)

				// Decode the output image
				decodedImg, err := decodeBase64Image(string(resizedOutput.Image)[22:]) // Remove "data:image/png;base64," prefix
				c.Assert(err, quicktest.IsNil)

				// Check if the output image dimensions match the expected dimensions
				c.Assert(decodedImg.Bounds().Dx(), quicktest.Equals, tc.expectedWidth)
				c.Assert(decodedImg.Bounds().Dy(), quicktest.Equals, tc.expectedHeight)
			}
		})
	}
}
