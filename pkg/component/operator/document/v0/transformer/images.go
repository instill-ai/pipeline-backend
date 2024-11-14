package transformer

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util"
)

type ConvertDocumentToImagesTransformerInput struct {
	Document   string `json:"document"`
	Filename   string `json:"filename"`
	Resolution int    `json:"resolution"`
}

type ConvertDocumentToImagesTransformerOutput struct {
	Images    []string `json:"images"`
	Filenames []string `json:"filenames"`
}

type pageNumbers struct {
	PageNumbers int    `json:"page_numbers"`
	Error       string `json:"error"`
}

type pageImage struct {
	Image    string `json:"image"`
	Filename string `json:"filename"`
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

	getNumberJSON := map[string]interface{}{
		"PDF": base64PDFWithoutMime,
	}

	pageNumbersBytes, err := util.ExecutePythonCode(getPageNumbersExecution, getNumberJSON)
	if err != nil {
		return nil, fmt.Errorf("get page numbers: %w", err)
	}

	var pageNumbers pageNumbers
	err = json.Unmarshal(pageNumbersBytes, &pageNumbers)
	if err != nil || pageNumbers.Error != "" {
		return nil, fmt.Errorf("unmarshal page numbers: %w, %s", err, pageNumbers.Error)
	}

	if pageNumbers.PageNumbers == 0 {
		return &ConvertDocumentToImagesTransformerOutput{
			Images:    []string{},
			Filenames: []string{},
		}, nil
	}

	pythonCode := imageProcessor + pdfTransformer + taskConvertToImagesExecution

	// We will make this number tunable & configurable in the future.
	maxWorkers := 5

	jobs := make(chan int, pageNumbers.PageNumbers)
	output := ConvertDocumentToImagesTransformerOutput{
		Images:    make([]string, pageNumbers.PageNumbers),
		Filenames: make([]string, pageNumbers.PageNumbers),
	}

	// Create workers
	wg := sync.WaitGroup{}
	for w := 0; w < maxWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for pageIdx := range jobs {
				paramsJSON := map[string]interface{}{
					"PDF":        base64PDFWithoutMime,
					"filename":   inputStruct.Filename,
					"resolution": inputStruct.Resolution,
					"page_idx":   pageIdx,
				}
				outputBytes, err := util.ExecutePythonCode(pythonCode, paramsJSON)
				if err != nil {
					continue
				}
				var pageImage pageImage
				err = json.Unmarshal(outputBytes, &pageImage)
				if err != nil {
					continue
				}
				output.Images[pageIdx] = pageImage.Image
				output.Filenames[pageIdx] = pageImage.Filename
			}
		}()
	}

	// Send jobs to workers
	for i := 0; i < pageNumbers.PageNumbers; i++ {
		jobs <- i
	}
	close(jobs)

	// Wait for all workers to complete
	wg.Wait()

	if len(output.Filenames) == 0 {
		output.Filenames = []string{}
	}
	return &output, nil
}
