package document

import (
	"context"
	"encoding/base64"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util"
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
		Document:   dataURI.String(),
		Filename:   inputStruct.Filename,
		Resolution: inputStruct.Resolution,
	}

	transformerOutputStruct, err := transformer.ConvertDocumentToImage(&transformerInputStruct)
	if err != nil {
		return err
	}
	outputStruct := ConvertDocumentToImagesOutput{
		Images: func() []format.Image {
			images := make([]format.Image, len(transformerOutputStruct.Images))
			for i, image := range transformerOutputStruct.Images {
				b, _ := base64.StdEncoding.DecodeString(util.TrimBase64Mime(image))
				images[i], _ = data.NewImageFromBytes(b, data.PNG, "")
				// TODO: handle error
			}
			return images
		}(),
		Filenames: transformerOutputStruct.Filenames,
	}

	return job.Output.WriteData(ctx, outputStruct)

}
