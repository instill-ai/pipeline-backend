package image

import (
	"context"
	"fmt"
	"image"
	"image/color"

	"github.com/fogleman/gg"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

type detectionObject struct {
	BoundingBox *boundingBox `json:"bounding-box"`
	Category    string       `json:"category"`
	Score       float64      `json:"score"`
}

type drawDetectionInput struct {
	Image     base64Image        `json:"image"`
	Objects   []*detectionObject `json:"objects"`
	ShowScore bool               `json:"show-score"`
}

type drawDetectionOutput struct {
	Image base64Image `json:"image"`
}

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

func drawDetection(input *structpb.Struct, job *base.Job, ctx context.Context) (*structpb.Struct, error) {

	inputStruct := drawDetectionInput{}

	err := base.ConvertFromStructpb(input, &inputStruct)
	if err != nil {
		return nil, fmt.Errorf("error converting input to struct: %v", err)
	}

	catIdx := indexUniqueCategories(inputStruct.Objects)

	img, err := decodeBase64Image(string(inputStruct.Image))
	if err != nil {
		return nil, fmt.Errorf("error decoding image: %v", err)
	}

	imgRGBA := convertToRGBA(img)

	for _, obj := range inputStruct.Objects {
		if err := drawBoundingBox(imgRGBA, obj.BoundingBox, catIdx[obj.Category]); err != nil {
			return nil, err
		}
	}

	for _, obj := range inputStruct.Objects {
		if err := drawObjectLabel(imgRGBA, obj.BoundingBox, obj.Category, false, catIdx[obj.Category]); err != nil {
			return nil, err
		}
	}

	base64Img, err := encodeBase64Image(imgRGBA)
	if err != nil {
		return nil, err
	}

	output := drawDetectionOutput{
		Image: base64Image(fmt.Sprintf("data:image/png;base64,%s", base64Img)),
	}

	return base.ConvertToStructpb(output)
}
