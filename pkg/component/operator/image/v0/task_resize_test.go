package image

import (
	"context"
	"image"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func TestResize(t *testing.T) {
	// Create a sample 100x100 image
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	base64Img, err := encodeBase64Image(img)
	assert.NoError(t, err)

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
		t.Run(tc.name, func(t *testing.T) {
			inputStruct, err := base.ConvertToStructpb(tc.input)
			assert.NoError(t, err)

			output, err := resize(inputStruct, nil, context.Background())

			if tc.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
			} else {
				assert.NoError(t, err)

				var resizedOutput resizeOutput
				err = base.ConvertFromStructpb(output, &resizedOutput)
				assert.NoError(t, err)

				// Decode the output image
				decodedImg, err := decodeBase64Image(string(resizedOutput.Image)[22:]) // Remove "data:image/png;base64," prefix
				assert.NoError(t, err)

				// Check if the output image dimensions match the expected dimensions
				assert.Equal(t, tc.expectedWidth, decodedImg.Bounds().Dx())
				assert.Equal(t, tc.expectedHeight, decodedImg.Bounds().Dy())
			}
		})
	}
}
