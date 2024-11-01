package image

import (
	"context"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func drawClassification(ctx context.Context, job *base.Job) error {

	var inputStruct drawClassificationInput
	if err := job.Input.ReadData(ctx, &inputStruct); err != nil {
		return err
	}

	img, err := decodeImage(inputStruct.Image)
	if err != nil {
		return err
	}

	imgRGBA := convertToRGBA(img)

	if err := drawImageLabel(imgRGBA, inputStruct.Category, inputStruct.Score, inputStruct.ShowScore); err != nil {
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
