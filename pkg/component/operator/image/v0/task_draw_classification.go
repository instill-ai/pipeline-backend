package image

import (
	"context"
	"image"
	"image/color"

	"github.com/fogleman/gg"
	"golang.org/x/image/font/opentype"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func drawImageLabel(img *image.RGBA, category string) error {

	dc := gg.NewContextForRGBA(img)

	// Parse the font
	font, err := opentype.Parse(IBMPlexSansRegular)
	if err != nil {
		return err
	}

	// Create a font face
	face, err := opentype.NewFace(font, &opentype.FaceOptions{
		Size: 20,
		DPI:  72,
	})
	if err != nil {
		return err
	}

	// Set the font face
	dc.SetFontFace(face)

	w, h := dc.MeasureString(category)

	// Set the rectangle padding
	padding := 2.0

	x := padding
	y := padding
	w += 6 * padding
	h += padding
	dc.SetRGB(0, 0, 0)
	dc.DrawRoundedRectangle(x, y, w, h, 4)
	dc.Fill()
	dc.SetColor(color.RGBA{255, 255, 255, 255})
	dc.DrawString(category, 4*padding, 11*padding)
	return nil
}

func drawClassification(ctx context.Context, job *base.Job) error {

	var inputStruct drawClassificationInput
	if err := job.Input.ReadData(ctx, &inputStruct); err != nil {
		return err
	}

	img, err := decodeImage(inputStruct.Image)
	if err != nil {
		return err
	}

	category := inputStruct.Category
	imgRGBA := convertToRGBA(img)

	if err := drawImageLabel(imgRGBA, category); err != nil {
		return err
	}

	imgData, err := encodeImage(imgRGBA)
	if err != nil {
		return err
	}

	outputData := drawClassificationOutput{
		Image: imgData,
	}

	if err := job.Output.WriteData(ctx, outputData); err != nil {
		return err
	}

	return nil
}
