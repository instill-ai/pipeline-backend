package image

import (
	"context"
	"image"

	"github.com/fogleman/gg"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

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

func drawKeypoint(ctx context.Context, job *base.Job) error {
	var inputStruct drawKeypointInput
	if err := job.Input.ReadData(ctx, &inputStruct); err != nil {
		return err
	}

	img, err := decodeImage(inputStruct.Image)
	if err != nil {
		return err
	}

	imgRGBA := convertToRGBA(img)

	for _, obj := range inputStruct.Objects {
		if err := drawSkeleton(imgRGBA, obj.Keypoints); err != nil {
			return err
		}
	}

	imgData, err := encodeImage(imgRGBA)
	if err != nil {
		return err
	}

	outputData := drawKeypointOutput{
		Image: imgData,
	}

	if err := job.Output.WriteData(ctx, outputData); err != nil {
		return err
	}

	return nil
}
