package transformer

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util"
	"go.uber.org/zap"
)

// ConvertDocumentToImagesInput ...
type ConvertDocumentToImagesInput struct {
	Document   string `json:"document"`
	Filename   string `json:"filename"`
	Resolution int    `json:"resolution"`
}

// ConvertDocumentToImagesOutput ...
type ConvertDocumentToImagesOutput struct {
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

// DocumentToImageConverter transforms documents to images.
type DocumentToImageConverter struct {
	logger *zap.Logger
}

// NewDocumentToImageConverter initializes a DocumentToImageConverter.
func NewDocumentToImageConverter(l *zap.Logger) *DocumentToImageConverter {
	if l == nil {
		l = zap.NewNop()
	}

	return &DocumentToImageConverter{logger: l}
}

// Convert transforms a document to images.
func (c *DocumentToImageConverter) Convert(inputStruct *ConvertDocumentToImagesInput) (*ConvertDocumentToImagesOutput, error) {
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
	shouldRepair, err := requiresRepair(base64PDF)
	if err != nil { // Non-blocking error
		c.logger.Error("Failed to check PDF state", zap.Error(err))
	}
	if shouldRepair {
		base64PDFWithoutMime, err = repairPDF(base64PDF, c.logger)
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
		return &ConvertDocumentToImagesOutput{
			Images:    []string{},
			Filenames: []string{},
		}, nil
	}

	pythonCode := pageImageProcessor + pdfTransformer + imageConverter

	// We will make this number tunable & configurable in the future.
	maxWorkers := 5

	jobs := make(chan int, pageNumbers.PageNumbers)
	output := ConvertDocumentToImagesOutput{
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
