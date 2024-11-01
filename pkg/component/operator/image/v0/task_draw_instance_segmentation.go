package image

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"sort"
	"strconv"
	"strings"

	"github.com/fogleman/gg"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func drawInstanceMask(img *image.RGBA, bbox *boundingBox, rle string, colorSeed int) error {

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

	// Decode the RLE mask for the full image size.
	mask := rleDecode(rleInts, bbox.Width, bbox.Height)

	// Iterate over the bounding box and draw the mask onto the image.
	for y := 0; y < bbox.Height; y++ {
		for x := 0; x < bbox.Width; x++ {
			if mask[y][x] {
				// The mask is present for this pixel, so draw it on the image.
				// Here you could set a specific color or just use the mask value.
				// For example, let's paint the mask as a red semi-transparent overlay:
				originalColor := img.At(x+bbox.Left, y+bbox.Top).(color.RGBA)
				// Blend the original color with the mask color.
				blendedColor := blendColors(originalColor, randomColor(colorSeed, 156))
				img.Set(x+bbox.Left, y+bbox.Top, blendedColor)
			}
		}
	}

	dc := gg.NewContextForRGBA(img)
	dc.SetColor(randomColor(colorSeed, 255))
	contourPoints := findContour(mask)
	for _, pt := range contourPoints {
		dc.DrawPoint(float64(pt.X+bbox.Left), float64(pt.Y+bbox.Top), 0.5)
		dc.Fill()
	}

	return nil
}

func drawInstanceSegmentation(ctx context.Context, job *base.Job) error {
	var inputStruct drawInstanceSegmentationInput
	if err := job.Input.ReadData(ctx, &inputStruct); err != nil {
		return err
	}

	img, err := decodeImage(inputStruct.Image)
	if err != nil {
		return err
	}

	imgRGBA := convertToRGBA(img)

	// Sort the objects by size.
	sort.Slice(inputStruct.Objects, func(i, j int) bool {
		bbox1 := inputStruct.Objects[i].BoundingBox
		bbox2 := inputStruct.Objects[j].BoundingBox
		return bbox1.Size() > bbox2.Size()
	})

	for instIdx, obj := range inputStruct.Objects {
		bbox := obj.BoundingBox
		if err := drawInstanceMask(imgRGBA, bbox, obj.RLE, instIdx); err != nil {
			return err
		}
	}

	for instIdx, obj := range inputStruct.Objects {
		bbox := obj.BoundingBox
		text := obj.Category
		if err := drawObjectLabel(imgRGBA, bbox, text, obj.Score, inputStruct.ShowScore, instIdx); err != nil {
			return err
		}
	}

	imgData, err := encodeImage(imgRGBA)
	if err != nil {
		return err
	}

	outputData := drawInstanceSegmentationOutput{
		Image: imgData,
	}

	if err := job.Output.WriteData(ctx, outputData); err != nil {
		return err
	}

	return nil
}
