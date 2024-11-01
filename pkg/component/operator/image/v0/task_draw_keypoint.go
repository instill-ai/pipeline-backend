package image

import (
	"context"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

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

		if err := drawBoundingBox(imgRGBA, obj.BoundingBox, 0); err != nil {
			return err
		}
		if err := drawObjectLabel(imgRGBA, obj.BoundingBox, "object", obj.Score, inputStruct.ShowScore, 0); err != nil {
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
