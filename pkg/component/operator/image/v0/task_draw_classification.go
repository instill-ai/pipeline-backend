package image

import (
	"context"
	"fmt"
	"image"
	"image/color"

	"github.com/fogleman/gg"
	"golang.org/x/image/font/opentype"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

type drawClassificationInput struct {
	Image     base64Image `json:"image"`
	Category  string      `json:"category"`
	Score     float64     `json:"score"`
	ShowScore bool        `json:"show-score"`
}

type drawClassificationOutput struct {
	Image base64Image `json:"image"`
}

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

func drawClassification(input *structpb.Struct, job *base.Job, ctx context.Context) (*structpb.Struct, error) {

	inputStruct := drawClassificationInput{}

	err := base.ConvertFromStructpb(input, &inputStruct)
	if err != nil {
		return nil, fmt.Errorf("error converting input to struct: %v", err)
	}

	category := inputStruct.Category

	img, err := decodeBase64Image(string(inputStruct.Image))
	if err != nil {
		return nil, fmt.Errorf("error decoding image: %v", err)
	}

	imgRGBA := convertToRGBA(img)

	if err := drawImageLabel(imgRGBA, category); err != nil {
		return nil, err
	}

	base64Img, err := encodeBase64Image(imgRGBA)
	if err != nil {
		return nil, err
	}

	output := drawClassificationOutput{
		Image: base64Image(fmt.Sprintf("data:image/png;base64,%s", base64Img)),
	}

	return base.ConvertToStructpb(output)
}
