package image

import (
	"context"
	"image"
	"image/color"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func TestConcat(t *testing.T) {
	// Create sample images
	img1 := createTestImage(50, 50, color.RGBA{255, 0, 0, 255})   // Red
	img2 := createTestImage(50, 50, color.RGBA{0, 255, 0, 255})   // Green
	img3 := createTestImage(50, 50, color.RGBA{0, 0, 255, 255})   // Blue
	img4 := createTestImage(50, 50, color.RGBA{255, 255, 0, 255}) // Yellow

	base64Img1, _ := encodeBase64Image(img1)
	base64Img2, _ := encodeBase64Image(img2)
	base64Img3, _ := encodeBase64Image(img3)
	base64Img4, _ := encodeBase64Image(img4)

	testCases := []struct {
		name           string
		input          ConcatInput
		expectedWidth  int
		expectedHeight int
		expectedError  string
	}{
		{
			name: "2x2 grid with padding",
			input: ConcatInput{
				Images: []base64Image{
					base64Image(base64Img1),
					base64Image(base64Img2),
					base64Image(base64Img3),
					base64Image(base64Img4),
				},
				GridWidth: 2,
				Padding:   10,
			},
			expectedWidth:  110,
			expectedHeight: 110,
		},
		{
			name: "1x4 grid without padding",
			input: ConcatInput{
				Images: []base64Image{
					base64Image(base64Img1),
					base64Image(base64Img2),
					base64Image(base64Img3),
					base64Image(base64Img4),
				},
				GridHeight: 1,
			},
			expectedWidth:  200,
			expectedHeight: 50,
		},
		{
			name: "Default square grid",
			input: ConcatInput{
				Images: []base64Image{
					base64Image(base64Img1),
					base64Image(base64Img2),
					base64Image(base64Img3),
					base64Image(base64Img4),
				},
			},
			expectedWidth:  100,
			expectedHeight: 100,
		},
		{
			name: "Invalid input (no images)",
			input: ConcatInput{
				Images: []base64Image{},
			},
			expectedError: "no images provided",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			inputStruct, err := base.ConvertToStructpb(tc.input)
			assert.NoError(t, err)

			output, err := concat(inputStruct, nil, context.Background())

			if tc.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
			} else {
				assert.NoError(t, err)

				var concatOutput ConcatOutput
				err = base.ConvertFromStructpb(output, &concatOutput)
				assert.NoError(t, err)

				// Decode the output image
				decodedImg, err := decodeBase64Image(string(concatOutput.Image)[22:]) // Remove "data:image/png;base64," prefix
				assert.NoError(t, err)

				// Check if the output image dimensions match the expected dimensions
				assert.Equal(t, tc.expectedWidth, decodedImg.Bounds().Dx())
				assert.Equal(t, tc.expectedHeight, decodedImg.Bounds().Dy())

				// Additional checks can be added here, such as verifying the colors of specific pixels
			}
		})
	}
}

// Helper function to create a test image with a solid color
func createTestImage(width, height int, c color.Color) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, c)
		}
	}
	return img
}
