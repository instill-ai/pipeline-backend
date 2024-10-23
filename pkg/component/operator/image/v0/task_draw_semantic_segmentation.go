package image

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"strconv"
	"strings"

	"github.com/fogleman/gg"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func drawSemanticMask(img *image.RGBA, rle string, colorSeed int) error {
	// Split the string by commas to get the individual number strings.
	numberStrings := strings.Split(rle, ",")

	// Allocate an array of integers with the same length as the number of numberStrings.
	rleInts := make([]int, len(numberStrings))

	// Convert each number string to an integer.
	for i, s := range numberStrings {
		n, err := strconv.Atoi(strings.TrimSpace(s))
		if err != nil {
			return fmt.Errorf("failed to convert RLE string to int: %s, error: %v", s, err)
		}
		rleInts[i] = n
	}

	bound := img.Bounds()

	// Decode the RLE mask for the full image size.
	mask := rleDecode(rleInts, bound.Dx(), bound.Dy())

	// Iterate over the bounding box and draw the mask onto the image.
	for y := 0; y < bound.Dy(); y++ {
		for x := 0; x < bound.Dx(); x++ {
			if mask[y][x] {
				// The mask is present for this pixel, so draw it on the image.
				// Here you could set a specific color or just use the mask value.
				// For example, let's paint the mask as a red semi-transparent overlay:
				originalColor := img.At(x, y).(color.RGBA)
				// Blend the original color with the mask color.
				blendedColor := blendColors(originalColor, randomColor(colorSeed, 128))
				img.Set(x, y, blendedColor)
			}
		}
	}

	dc := gg.NewContextForRGBA(img)
	dc.SetColor(color.RGBA{255, 255, 255, 255})

	// Find contour points
	contourPoints := findContour(mask)

	// Draw the contour
	for _, pt := range contourPoints {
		// Scale points as needed for your canvas size
		dc.DrawPoint(float64(pt.X), float64(pt.Y), 0.5)
		dc.Fill()
	}

	return nil
}

func drawSemanticSegmentation(ctx context.Context, job *base.Job) error {
	var inputStruct drawSemanticSegmentationInput
	if err := job.Input.ReadData(ctx, &inputStruct); err != nil {
		return err
	}

	img, err := decodeImage(inputStruct.Image)
	if err != nil {
		return err
	}

	imgRGBA := convertToRGBA(img)

	for idx, stuff := range inputStruct.Stuffs {
		if err := drawSemanticMask(imgRGBA, stuff.RLE, idx); err != nil {
			return err
		}
	}

	imgData, err := encodeImage(imgRGBA)
	if err != nil {
		return err
	}

	outputData := drawSemanticSegmentationOutput{
		Image: imgData,
	}

	if err := job.Output.WriteData(ctx, outputData); err != nil {
		return err
	}

	return nil
}
