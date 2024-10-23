package document

import (
	"context"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/operator/document/v0/transformer"
	"github.com/instill-ai/pipeline-backend/pkg/data"
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
)

func (e *execution) convertDocumentToImages(ctx context.Context, job *base.Job) error {

	inputStruct := ConvertDocumentToImagesInput{}
	err := job.Input.ReadData(ctx, &inputStruct)
	if err != nil {
		return err
	}
	dataURI, err := inputStruct.Document.DataURI()
	if err != nil {
		return err
	}

	transformerInputStruct := transformer.ConvertDocumentToImagesTransformerInput{
		Document: dataURI.String(),
		Filename: inputStruct.Filename,
	}

	transformerOutputStruct, err := transformer.ConvertDocumentToImage(&transformerInputStruct)
	if err != nil {
		return err
	}
	outputStruct := ConvertDocumentToImagesOutput{
		Images: func() []format.Image {
			images := make([]format.Image, len(transformerOutputStruct.Images))
			for i, image := range transformerOutputStruct.Images {
				images[i], _ = data.NewImageFromURL(image)
			}
			return images
		}(),
		Filenames: transformerOutputStruct.Filenames,
	}

	return job.Output.WriteData(ctx, outputStruct)

}
