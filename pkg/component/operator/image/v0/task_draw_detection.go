package image

import (
	"context"
	"image"
	"image/color"

	"github.com/fogleman/gg"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func drawBoundingBox(img *image.RGBA, bbox *boundingBox, colorSeed int) error {
	dc := gg.NewContextForRGBA(img)
	originalColor := img.At(bbox.Left, bbox.Top).(color.RGBA)
	blendedColor := blendColors(originalColor, randomColor(colorSeed, 255))
	dc.SetColor(blendedColor)
	dc.SetLineWidth(3)
	dc.DrawRoundedRectangle(float64(bbox.Left), float64(bbox.Top), float64(bbox.Width), float64(bbox.Height), 4)
	dc.Stroke()
	return nil
}

func drawDetection(ctx context.Context, job *base.Job) error {

	var inputStruct drawDetectionInput
	if err := job.Input.ReadData(ctx, &inputStruct); err != nil {
		return err
	}

	// Decode image
	img, err := decodeImage(inputStruct.Image)
	if err != nil {
		return err
	}

	catIdx := indexUniqueCategories(inputStruct.Objects)

	imgRGBA := convertToRGBA(img)

	for _, obj := range inputStruct.Objects {
		if err := drawBoundingBox(imgRGBA, obj.BoundingBox, catIdx[obj.Category]); err != nil {
			return err
		}
		if err := drawObjectLabel(imgRGBA, obj.BoundingBox, obj.Category, obj.Score, inputStruct.ShowScore, catIdx[obj.Category]); err != nil {
			return err
		}
	}

	imgData, err := encodeImage(imgRGBA)
	if err != nil {
		return err
	}

	outputData := drawDetectionOutput{
		Image: imgData,
	}

	if err := job.Output.WriteData(ctx, outputData); err != nil {
		return err
	}

	return nil
}
