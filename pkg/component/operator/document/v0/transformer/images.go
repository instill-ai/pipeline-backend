package transformer

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util"
)

type ConvertDocumentToImagesTransformerInput struct {
	Document string `json:"document"`
	Filename string `json:"filename"`
}

type ConvertDocumentToImagesTransformerOutput struct {
	Images    []string `json:"images"`
	Filenames []string `json:"filenames"`
}

func ConvertDocumentToImage(inputStruct *ConvertDocumentToImagesTransformerInput) (*ConvertDocumentToImagesTransformerOutput, error) {

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

	var base64PDFWithoutMime string
	if RequiredToRepair(base64PDF) {
		base64PDFWithoutMime, err = RepairPDF(base64PDF)
		if err != nil {
			return nil, fmt.Errorf("failed to repair PDF: %w", err)
		}
	} else {
		base64PDFWithoutMime = util.TrimBase64Mime(base64PDF)
	}

	paramsJSON := map[string]interface{}{
		"PDF":      base64PDFWithoutMime,
		"filename": inputStruct.Filename,
	}

	pythonCode := imageProcessor + pdfTransformer + taskConvertToImagesExecution

	outputBytes, err := util.ExecutePythonCode(pythonCode, paramsJSON)

	if err != nil {
		return nil, fmt.Errorf("failed to run python script: %w", err)
	}

	output := ConvertDocumentToImagesTransformerOutput{}

	err = json.Unmarshal(outputBytes, &output)

	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal output: %w", err)
	}

	if len(output.Filenames) == 0 {
		output.Filenames = []string{}
	}
	return &output, nil
}
