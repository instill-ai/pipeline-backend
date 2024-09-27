package document

import (
	"encoding/json"
	"fmt"
	"strings"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util"
)

type ConvertDocumentToImagesInput struct {
	Document string `json:"document"`
	Filename string `json:"filename"`
}

type ConvertDocumentToImagesOutput struct {
	Images    []string `json:"images"`
	Filenames []string `json:"filenames"`
}

func ConvertDocumentToImage(inputStruct *ConvertDocumentToImagesInput) (*ConvertDocumentToImagesOutput, error) {

	contentType, err := util.GetContentTypeFromBase64(inputStruct.Document)
	if err != nil {
		return nil, err
	}

	fileExtension := util.TransformContentTypeToFileExtension(contentType)

	if fileExtension == "" {
		return nil, fmt.Errorf("unsupported file type")
	}

	var base64PDF string
	if fileExtension != "pdf" {
		base64PDF, err = ConvertToPDF(inputStruct.Document, fileExtension)

		if err != nil {
			return nil, fmt.Errorf("failed to encode file to base64: %w", err)
		}
	} else {
		base64PDF = strings.Split(inputStruct.Document, ",")[1]
	}

	paramsJSON := map[string]interface{}{
		"PDF":      base.TrimBase64Mime(base64PDF),
		"filename": inputStruct.Filename,
	}

	pythonCode := imageProcessor + pdfTransformer + taskConvertToImagesExecution

	outputBytes, err := util.ExecutePythonCode(pythonCode, paramsJSON)

	if err != nil {
		return nil, fmt.Errorf("failed to run python script: %w", err)
	}

	output := ConvertDocumentToImagesOutput{}

	err = json.Unmarshal(outputBytes, &output)

	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal output: %w", err)
	}

	if len(output.Filenames) == 0 {
		output.Filenames = []string{}
	}
	return &output, nil
}

func (e *execution) convertDocumentToImages(input *structpb.Struct) (*structpb.Struct, error) {

	inputStruct := ConvertDocumentToImagesInput{}
	err := base.ConvertFromStructpb(input, &inputStruct)
	if err != nil {
		return nil, fmt.Errorf("failed to convert input struct: %w", err)
	}

	outputStruct, err := ConvertDocumentToImage(&inputStruct)
	if err != nil {
		return nil, err
	}

	return base.ConvertToStructpb(outputStruct)

}
