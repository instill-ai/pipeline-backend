package document

import (
	"context"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util"
)

type ConvertDocumentToMarkdownInput struct {
	Document            string `json:"document"`
	DisplayImageTag     bool   `json:"display-image-tag"`
	Filename            string `json:"filename"`
	DisplayAllPageImage bool   `json:"display-all-page-image"`
}

type ConvertDocumentToMarkdownOutput struct {
	Body          string   `json:"body"`
	Filename      string   `json:"filename"`
	Images        []string `json:"images,omitempty"`
	Error         string   `json:"error,omitempty"`
	AllPageImages []string `json:"all-page-images,omitempty"`
	Markdowns     []string `json:"markdowns"`
}

func ConvertDocumentToMarkdown(inputStruct *ConvertDocumentToMarkdownInput, transformerGetter MarkdownTransformerGetterFunc) (*ConvertDocumentToMarkdownOutput, error) {
	contentType, err := util.GetContentTypeFromBase64(inputStruct.Document)
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
	converterOutput, err := transformer.Transform()
	if err != nil {
		return nil, err
	}

	outputStruct := &ConvertDocumentToMarkdownOutput{
		Body:          converterOutput.Body,
		Images:        converterOutput.Images,
		Error:         strings.Join(converterOutput.ParsingError, "\n"),
		AllPageImages: converterOutput.AllPageImages,
	}

	if inputStruct.Filename != "" {
		filename := strings.Split(inputStruct.Filename, ".")[0] + ".md"
		outputStruct.Filename = filename
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
	}

	err = job.Output.WriteData(ctx, outputStruct)
	if err != nil {
		return err
	}

	return nil
}
