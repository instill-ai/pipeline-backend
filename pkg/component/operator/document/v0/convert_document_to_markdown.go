package document

import (
	"context"
	"encoding/base64"

	"go.uber.org/zap"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util"
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

	converter := transformer.NewDocumentToMarkdownConverter(e.logger)

	conversionParams := &transformer.ConvertDocumentToMarkdownInput{
		Document:            dataURI.String(),
		DisplayImageTag:     inputStruct.DisplayImageTag,
		Filename:            inputStruct.Filename,
		DisplayAllPageImage: inputStruct.DisplayAllPageImage,
		Resolution:          inputStruct.Resolution,
		Converter:           inputStruct.Converter,
	}
	transformerOutputStruct, err := converter.Convert(conversionParams)
	if err != nil {
		return err
	}

	outputStruct := ConvertDocumentToMarkdownOutput{
		Body:          transformerOutputStruct.Body,
		Filename:      transformerOutputStruct.Filename,
		Images:        e.parseImages(transformerOutputStruct.Images),
		Error:         transformerOutputStruct.Error,
		AllPageImages: e.parseImages(transformerOutputStruct.AllPageImages),
		Markdowns:     transformerOutputStruct.Markdowns,
	}

	err = job.Output.WriteData(ctx, outputStruct)
	if err != nil {
		return err
	}

	return nil
}

func (e *execution) parseImages(strImages []string) []format.Image {
	images := make([]format.Image, len(strImages))
	for i, image := range strImages {
		b, err := base64.StdEncoding.DecodeString(util.TrimBase64Mime(image))
		if err != nil {
			e.logger.Error("Failed to decode image from document", zap.Error(err))
		}
		images[i], err = data.NewImageFromBytes(b, data.PNG, "")
		if err != nil {
			e.logger.Error("Failed to create image data from document", zap.Error(err))
		}
	}

	return images
}
