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

type ocrObject struct {
	BoundingBox *boundingBox `json:"bounding-box"`
	Text        string       `json:"text"`
	Score       float64      `json:"score"`
}

type drawOCRInput struct {
	Image     base64Image  `json:"image"`
	Objects   []*ocrObject `json:"objects"`
	ShowScore bool         `json:"show-score"`
}

type drawOCROutput struct {
	Image base64Image `json:"image"`
}

func draOCRLabel(img *image.RGBA, bbox *boundingBox, text string) error {

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

func drawOCR(input *structpb.Struct, job *base.Job, ctx context.Context) (*structpb.Struct, error) {

	inputStruct := drawOCRInput{}

	err := base.ConvertFromStructpb(input, &inputStruct)
	if err != nil {
		return nil, fmt.Errorf("error converting input to struct: %v", err)
	}

	img, err := decodeBase64Image(string(inputStruct.Image))
	if err != nil {
		return nil, fmt.Errorf("error decoding image: %v", err)
	}

	imgRGBA := convertToRGBA(img)

	for _, obj := range inputStruct.Objects {
		bbox := obj.BoundingBox
		if err := draOCRLabel(imgRGBA, bbox, obj.Text); err != nil {
			return nil, err
		}
	}

	base64Img, err := encodeBase64Image(imgRGBA)
	if err != nil {
		return nil, err
	}

	output := drawOCROutput{
		Image: base64Image(fmt.Sprintf("data:image/png;base64,%s", base64Img)),
	}

	return base.ConvertToStructpb(output)
}
