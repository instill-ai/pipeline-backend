package image

import (
	"context"
	"fmt"
	"image"
	"image/draw"
	"math"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

type ConcatInput struct {
	Images     []base64Image `json:"images"`
	GridWidth  int           `json:"grid-width"`
	GridHeight int           `json:"grid-height"`
	Padding    int           `json:"padding"`
}

type ConcatOutput struct {
	Image base64Image `json:"image"`
}

func concat(input *structpb.Struct, job *base.Job, ctx context.Context) (*structpb.Struct, error) {
	// Parse input
	var inputStruct ConcatInput
	if err := base.ConvertFromStructpb(input, &inputStruct); err != nil {
		return nil, fmt.Errorf("error converting input: %v", err)
	}

	// Check if images are provided
	if len(inputStruct.Images) == 0 {
		return nil, fmt.Errorf("no images provided")
	}

	// Decode images
	images := make([]image.Image, len(inputStruct.Images))
	for i, base64Img := range inputStruct.Images {
		img, err := decodeBase64Image(string(base64Img))
		if err != nil {
			return nil, fmt.Errorf("error decoding image %d: %v", i, err)
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

	// Encode output image
	base64Output, err := encodeBase64Image(output)
	if err != nil {
		return nil, fmt.Errorf("error encoding output image: %v", err)
	}

	// Prepare output
	outputStruct := ConcatOutput{
		Image: base64Image(fmt.Sprintf("data:image/png;base64,%s", base64Output)),
	}

	return base.ConvertToStructpb(outputStruct)
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
