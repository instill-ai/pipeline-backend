package image

import (
	"context"
	"image"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func TestCrop(t *testing.T) {
	// Create a sample 100x100 image
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			img.Set(x, y, image.White)
		}
	}
	base64Img, err := encodeBase64Image(img)
	assert.NoError(t, err)

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
		t.Run(tc.name, func(t *testing.T) {
			inputStruct, err := base.ConvertToStructpb(tc.input)
			assert.NoError(t, err)

			output, err := crop(inputStruct, nil, context.Background())

			if tc.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
			} else {
				assert.NoError(t, err)

				var croppedOutput cropOutput
				err = base.ConvertFromStructpb(output, &croppedOutput)
				assert.NoError(t, err)

				// Decode the output image as PNG
				decodedImg, err := decodeBase64Image(string(croppedOutput.Image)[22:]) // Remove "data:image/png;base64," prefix
				assert.NoError(t, err)

				// Check if the output image dimensions match the expected dimensions
				assert.Equal(t, tc.expectedWidth, decodedImg.Bounds().Dx())
				assert.Equal(t, tc.expectedHeight, decodedImg.Bounds().Dy())

				// For circular and corner radius crop, check if corners are transparent
				if tc.checkCorners {
					checkCornerTransparency(t, decodedImg)
				}
			}
		})
	}
}

func checkCornerTransparency(t *testing.T, img image.Image) {
	bounds := img.Bounds()
	corners := []image.Point{
		{X: bounds.Min.X, Y: bounds.Min.Y},
		{X: bounds.Max.X - 1, Y: bounds.Min.Y},
		{X: bounds.Min.X, Y: bounds.Max.Y - 1},
		{X: bounds.Max.X - 1, Y: bounds.Max.Y - 1},
	}

	for _, corner := range corners {
		_, _, _, a := img.At(corner.X, corner.Y).RGBA()
		assert.Equal(t, uint32(0), a, "Corner at (%d, %d) should be transparent", corner.X, corner.Y)
	}
}
