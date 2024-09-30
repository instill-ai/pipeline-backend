package image

import (
	"context"
	"fmt"
	"image"

	"google.golang.org/protobuf/types/known/structpb"

	nr "github.com/nfnt/resize"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

type resizeInput struct {
	Image  base64Image `json:"image"`
	Width  int         `json:"width"`
	Height int         `json:"height"`
	Ratio  float64     `json:"ratio"`
}

type resizeOutput struct {
	Image base64Image `json:"image"`
}

func resize(input *structpb.Struct, job *base.Job, ctx context.Context) (*structpb.Struct, error) {
	var inputStruct resizeInput
	if err := base.ConvertFromStructpb(input, &inputStruct); err != nil {
		return nil, fmt.Errorf("error converting input to struct: %w", err)
	}

	// Validate input parameters
	if err := validateInputParams(inputStruct); err != nil {
		return nil, err
	}

	img, err := decodeBase64Image(string(inputStruct.Image))
	if err != nil {
		return nil, fmt.Errorf("error decoding image: %w", err)
	}

	// Determine new dimensions
	width, height := calculateNewDimensions(inputStruct, img.Bounds())

	// Check if resizing is needed
	if width == img.Bounds().Dx() && height == img.Bounds().Dy() {
		return createOutput(img)
	}

	// Resize the image
	resizedImg := nr.Resize(uint(width), uint(height), img, nr.Lanczos3)
	return createOutput(resizedImg)
}

func validateInputParams(input resizeInput) error {
	if input.Ratio < 0 || input.Ratio > 1 {
		return fmt.Errorf("ratio must be between 0 and 1")
	}
	if input.Width < 0 || input.Height < 0 {
		return fmt.Errorf("width and height must be greater than or equal to 0")
	}
	return nil
}

func calculateNewDimensions(input resizeInput, bounds image.Rectangle) (int, int) {
	if input.Ratio > 0 {
		return int(float64(bounds.Dx()) * input.Ratio),
			int(float64(bounds.Dy()) * input.Ratio)
	}

	aspectRatio := float64(bounds.Dx()) / float64(bounds.Dy())

	switch {
	case input.Width > 0 && input.Height > 0:
		return input.Width, input.Height
	case input.Width > 0:
		return input.Width, int(float64(input.Width) / aspectRatio)
	case input.Height > 0:
		return int(float64(input.Height) * aspectRatio), input.Height
	default:
		return bounds.Dx(), bounds.Dy()
	}
}

func createOutput(img image.Image) (*structpb.Struct, error) {
	base64Img, err := encodeBase64Image(img)
	if err != nil {
		return nil, err
	}
	output := resizeOutput{
		Image: base64Image(fmt.Sprintf("data:image/png;base64,%s", base64Img)),
	}
	return base.ConvertToStructpb(output)
}
