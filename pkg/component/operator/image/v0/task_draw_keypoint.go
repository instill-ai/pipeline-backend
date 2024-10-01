package image

import (
	"context"
	"fmt"
	"image"

	"github.com/fogleman/gg"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

type keypoint struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	V float64 `json:"v"`
}

type keypointObject struct {
	BoundingBox *boundingBox `json:"bounding-box"`
	Keypoints   []*keypoint  `json:"keypoints"`
	Score       float64      `json:"score"`
}

type drawKeypointInput struct {
	Image     base64Image       `json:"image"`
	Objects   []*keypointObject `json:"objects"`
	ShowScore bool              `json:"show-score"`
}

type drawKeypointOutput struct {
	Image base64Image `json:"image"`
}

var skeleton = [][]int{{16, 14}, {14, 12}, {17, 15}, {15, 13}, {12, 13}, {6, 12},
	{7, 13}, {6, 7}, {6, 8}, {7, 9}, {8, 10}, {9, 11}, {2, 3}, {1, 2}, {1, 3}, {2, 4}, {3, 5}, {4, 6}, {5, 7},
}

var keypointLimbColorIdx = []int{9, 9, 9, 9, 7, 7, 7, 0, 0, 0, 0, 0, 16, 16, 16, 16, 16, 16, 16}
var keypointColorIdx = []int{16, 16, 16, 16, 16, 0, 0, 0, 0, 0, 0, 9, 9, 9, 9, 9, 9}

func drawSkeleton(img *image.RGBA, kpts []*keypoint) error {
	dc := gg.NewContextForRGBA(img)
	for idx, kpt := range kpts {
		if kpt.V > 0.5 {
			dc.SetColor(palette[keypointColorIdx[idx]])
			dc.DrawPoint(kpt.X, kpt.Y, 2)
			dc.Fill()
		}
	}
	for idx, sk := range skeleton {
		if kpts[sk[0]-1].V > 0.5 && kpts[sk[1]-1].V > 0.5 {
			dc.SetColor(palette[keypointLimbColorIdx[idx]])
			dc.SetLineWidth(2)
			dc.DrawLine(kpts[sk[0]-1].X, kpts[sk[0]-1].Y, kpts[sk[1]-1].X, kpts[sk[1]-1].Y)
			dc.Stroke()
		}
	}
	return nil
}

func drawKeypoint(input *structpb.Struct, job *base.Job, ctx context.Context) (*structpb.Struct, error) {

	inputStruct := drawKeypointInput{}

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
		if err := drawSkeleton(imgRGBA, obj.Keypoints); err != nil {
			return nil, err
		}
	}

	base64Img, err := encodeBase64Image(imgRGBA)
	if err != nil {
		return nil, err
	}

	output := drawKeypointOutput{
		Image: base64Image(fmt.Sprintf("data:image/png;base64,%s", base64Img)),
	}

	return base.ConvertToStructpb(output)
}
