package document

import (
	"context"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/operator/document/v0/transformer"
	"github.com/instill-ai/pipeline-backend/pkg/data"
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
)

func (e *execution) convertDocumentToMarkdown(ctx context.Context, job *base.Job) error {
	inputStruct := ConvertDocumentToMarkdownInput{}

	err := job.Input.ReadData(ctx, &inputStruct)
	if err != nil {
		return err
	}

	dataURI, err := inputStruct.Document.DataURI()
	if err != nil {
		return err
	}
	transformerInputStruct := transformer.ConvertDocumentToMarkdownTransformerInput{
		Document:            dataURI.String(),
		DisplayImageTag:     inputStruct.DisplayImageTag,
		Filename:            inputStruct.Filename,
		DisplayAllPageImage: inputStruct.DisplayAllPageImage,
		Resolution:          inputStruct.Resolution,
	}

	transformerOutputStruct, err := transformer.ConvertDocumentToMarkdown(&transformerInputStruct, e.getMarkdownTransformer)
	if err != nil {
		return err
	}

	outputStruct := ConvertDocumentToMarkdownOutput{
		Body:     transformerOutputStruct.Body,
		Filename: transformerOutputStruct.Filename,
		Images: func() []format.Image {
			images := make([]format.Image, len(transformerOutputStruct.Images))
			for i, image := range transformerOutputStruct.Images {
				images[i], _ = data.NewImageFromURL(image)
				// TODO: handle error
			}
			return images
		}(),
		Error: transformerOutputStruct.Error,
		AllPageImages: func() []format.Image {
			images := make([]format.Image, len(transformerOutputStruct.AllPageImages))
			for i, image := range transformerOutputStruct.AllPageImages {
				images[i], _ = data.NewImageFromURL(image)
				// TODO: handle error
			}
			return images
		}(),
		Markdowns: transformerOutputStruct.Markdowns,
	}

	err = job.Output.WriteData(ctx, outputStruct)
	if err != nil {
		return err
	}

	return nil
}
