package image

import (
	"context"
	"image"
	"image/color"

	"github.com/fogleman/gg"
	"golang.org/x/image/font/opentype"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func drawOCRLabel(img *image.RGBA, bbox *boundingBox, text string) error {
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

	w, h := dc.MeasureString(text)

	// Set the rectangle padding
	padding := 2.0

	x := float64(bbox.Left)
	y := float64(bbox.Top)
	w += 4 * padding
	h += padding
	dc.SetRGBA(0, 0, 0, 128)
	dc.DrawRoundedRectangle(x, y, w, h, 4)
	dc.Fill()
	dc.SetColor(color.RGBA{255, 255, 255, 255})
	dc.DrawString(text, float64(bbox.Left)+2*padding, float64(bbox.Top)+h-4*padding)

	return nil
}

func drawOCR(ctx context.Context, job *base.Job) error {
	var inputStruct drawOCRInput
	if err := job.Input.ReadData(ctx, &inputStruct); err != nil {
		return err
	}

	img, err := decodeImage(inputStruct.Image)
	if err != nil {
		return err
	}

	imgRGBA := convertToRGBA(img)

	for _, obj := range inputStruct.Objects {
		if err := drawOCRLabel(imgRGBA, obj.BoundingBox, obj.Text); err != nil {
			return err
		}
	}

	imgData, err := encodeImage(imgRGBA)
	if err != nil {
		return err
	}

	outputData := drawOCROutput{
		Image: imgData,
	}

	if err := job.Output.WriteData(ctx, outputData); err != nil {
		return err
	}

	return nil
}
