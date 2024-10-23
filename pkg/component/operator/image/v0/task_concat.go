package image

import (
	"context"
	"fmt"
	"image"
	"image/draw"
	"math"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func concat(ctx context.Context, job *base.Job) error {
	// Parse input
	var inputStruct concatInput
	if err := job.Input.ReadData(ctx, &inputStruct); err != nil {
		return err
	}

	// Check if images are provided
	if len(inputStruct.Images) == 0 {
		return fmt.Errorf("no images provided")
	}

	// Decode images
	images := make([]image.Image, len(inputStruct.Images))
	for i, img := range inputStruct.Images {
		img, err := decodeImage(img)
		if err != nil {
			return err
		}
		images[i] = img
	}

	// Determine grid dimensions
	gridWidth, gridHeight := determineGridDimensions(len(images), inputStruct.GridWidth, inputStruct.GridHeight)

	// Calculate output image dimensions
	sampleImg := images[0]
	imgWidth, imgHeight := sampleImg.Bounds().Dx(), sampleImg.Bounds().Dy()
	padding := inputStruct.Padding
	outputWidth := gridWidth*imgWidth + (gridWidth-1)*padding
	outputHeight := gridHeight*imgHeight + (gridHeight-1)*padding

	// Create output image
	output := image.NewRGBA(image.Rect(0, 0, outputWidth, outputHeight))

	// Place images in grid
	for i, img := range images {
		row := i / gridWidth
		col := i % gridWidth
		x := col * (imgWidth + padding)
		y := row * (imgHeight + padding)
		draw.Draw(output, image.Rect(x, y, x+imgWidth, y+imgHeight), img, image.Point{}, draw.Over)
	}

	imgData, err := encodeImage(output)
	if err != nil {
		return err
	}

	outputData := concatOutput{
		Image: imgData,
	}

	if err := job.Output.WriteData(ctx, outputData); err != nil {
		return err
	}

	return nil
}

func determineGridDimensions(imageCount, gridWidth, gridHeight int) (int, int) {
	if gridWidth > 0 {
		return gridWidth, int(math.Ceil(float64(imageCount) / float64(gridWidth)))
	}
	if gridHeight > 0 {
		return int(math.Ceil(float64(imageCount) / float64(gridHeight))), gridHeight
	}
	// Default to square grid
	side := int(math.Ceil(math.Sqrt(float64(imageCount))))
	return side, side
}
