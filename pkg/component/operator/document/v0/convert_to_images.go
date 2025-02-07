package document

import (
	"context"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/operator/document/v0/transformer"
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

	converter := transformer.NewDocumentToImageConverter(e.logger)

	conversionParams := &transformer.ConvertDocumentToImagesInput{
		Document:   dataURI.String(),
		Filename:   inputStruct.Filename,
		Resolution: inputStruct.Resolution,
	}
	transformerOutputStruct, err := converter.Convert(conversionParams)
	if err != nil {
		return err
	}
	outputStruct := ConvertDocumentToImagesOutput{
		Images:    e.parseImages(transformerOutputStruct.Images),
		Filenames: transformerOutputStruct.Filenames,
	}

	return job.Output.WriteData(ctx, outputStruct)

}
