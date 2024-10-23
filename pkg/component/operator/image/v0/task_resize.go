package image

import (
	"context"
	"fmt"
	"image"

	nr "github.com/nfnt/resize"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func resize(ctx context.Context, job *base.Job) error {
	// Parse input
	var inputStruct resizeInput
	if err := job.Input.ReadData(ctx, &inputStruct); err != nil {
		return err
	}

	// Validate input parameters
	if err := validateInputParams(inputStruct); err != nil {
		return err
	}

	// Decode image
	img, err := decodeImage(inputStruct.Image)
	if err != nil {
		return fmt.Errorf("error decoding image: %v", err)
	}

	// Determine new dimensions
	width, height := calculateNewDimensions(inputStruct, img.Bounds())

	// Check if resizing is needed
	if width == img.Bounds().Dx() && height == img.Bounds().Dy() {
		return createOutput(img, job, ctx)
	}

	// Resize the image
	resizedImg := nr.Resize(uint(width), uint(height), img, nr.Lanczos3)
	return createOutput(resizedImg, job, ctx)
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

func createOutput(img image.Image, job *base.Job, ctx context.Context) error {

	imgData, err := encodeImage(img)
	if err != nil {
		return err
	}

	outputData := resizeOutput{
		Image: imgData,
	}

	if err := job.Output.WriteData(ctx, outputData); err != nil {
		return err
	}

	return nil
}
