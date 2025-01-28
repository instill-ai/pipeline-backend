package transformer

import (
	"fmt"
	"strings"

	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util"
)

type ConvertDocumentToMarkdownTransformerInput struct {
	Document            string `json:"document"`
	DisplayImageTag     bool   `json:"display-image-tag"`
	Filename            string `json:"filename"`
	DisplayAllPageImage bool   `json:"display-all-page-image"`
	Resolution          int    `json:"resolution"`
	UseDoclingConverter bool   `json:"use-docling-converter"`
}

type ConvertDocumentToMarkdownTransformerOutput struct {
	Body          string   `json:"body"`
	Filename      string   `json:"filename"`
	Images        []string `json:"images,omitempty"`
	Error         string   `json:"error,omitempty"`
	AllPageImages []string `json:"all-page-images,omitempty"`
	Markdowns     []string `json:"markdowns"`
}

func ConvertDocumentToMarkdown(inputStruct *ConvertDocumentToMarkdownTransformerInput, transformerGetter MarkdownTransformerGetterFunc) (*ConvertDocumentToMarkdownTransformerOutput, error) {
	contentType, err := util.GetContentTypeFromBase64(inputStruct.Document)
	if err != nil {
		return nil, err
	}

	fileExtension := util.TransformContentTypeToFileExtension(contentType)

	if fileExtension == "" {
		return nil, fmt.Errorf("unsupported file type")
	}

	var transformer MarkdownTransformer

	transformer, err = transformerGetter(fileExtension, inputStruct)
	if err != nil {
		return nil, err
	}
	converterOutput, err := transformer.Transform()
	if err != nil {
		return nil, err
	}

	outputStruct := &ConvertDocumentToMarkdownTransformerOutput{
		Body:          converterOutput.Body,
		Images:        converterOutput.Images,
		Error:         strings.Join(converterOutput.ParsingError, "\n"),
		AllPageImages: converterOutput.AllPageImages,
		Markdowns:     converterOutput.Markdowns,
	}

	if inputStruct.Filename != "" {
		filename := strings.Split(inputStruct.Filename, ".")[0] + ".md"
		outputStruct.Filename = filename
	}
	return outputStruct, nil
}

func GetMarkdownTransformer(fileExtension string, inputStruct *ConvertDocumentToMarkdownTransformerInput) (MarkdownTransformer, error) {
	switch fileExtension {
	case "pdf":
		pdfToMarkdownStruct := pdfToMarkdownInputStruct{
			Base64Text:          inputStruct.Document,
			DisplayImageTag:     inputStruct.DisplayImageTag,
			DisplayAllPageImage: inputStruct.DisplayAllPageImage,
			Resolution:          inputStruct.Resolution,
		}

		converter := "pdfplumber"
		if inputStruct.UseDoclingConverter {
			converter = "docling"
		}

		return PDFToMarkdownTransformer{
			FileExtension:       fileExtension,
			PDFToMarkdownStruct: pdfToMarkdownStruct,
			PDFConvertFunc:      getPDFConvertFunc(converter),
		}, nil
	case "doc", "docx":
		pdfToMarkdownStruct := pdfToMarkdownInputStruct{
			DisplayImageTag:     inputStruct.DisplayImageTag,
			DisplayAllPageImage: inputStruct.DisplayAllPageImage,
			Resolution:          inputStruct.Resolution,
		}
		return DocxDocToMarkdownTransformer{
			FileExtension:       fileExtension,
			Base64EncodedText:   inputStruct.Document,
			PDFToMarkdownStruct: pdfToMarkdownStruct,
			PDFConvertFunc:      getPDFConvertFunc("pdfplumber"),
		}, nil
	case "ppt", "pptx":
		pdfToMarkdownStruct := pdfToMarkdownInputStruct{
			DisplayImageTag:     inputStruct.DisplayImageTag,
			DisplayAllPageImage: inputStruct.DisplayAllPageImage,
			Resolution:          inputStruct.Resolution,
		}
		return PptPptxToMarkdownTransformer{
			FileExtension:       fileExtension,
			Base64EncodedText:   inputStruct.Document,
			PDFToMarkdownStruct: pdfToMarkdownStruct,
			PDFConvertFunc:      getPDFConvertFunc("pdfplumber"),
		}, nil
	case "html":
		return HTMLToMarkdownTransformer{
			Base64EncodedText: inputStruct.Document,
		}, nil
	case "xlsx":
		return XlsxToMarkdownTransformer{
			Base64EncodedText: inputStruct.Document,
		}, nil
	case "xls":
		return XlsToMarkdownTransformer{
			Base64EncodedText: inputStruct.Document,
		}, nil
	case "csv":
		return CSVToMarkdownTransformer{
			Base64EncodedText: inputStruct.Document,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported file type")
	}
}

type pdfToMarkdownInputStruct struct {
	Base64Text          string
	DisplayImageTag     bool
	DisplayAllPageImage bool
	Resolution          int
}

// We could provide more converters in the future. For now, we only have one.
func getPDFConvertFunc(converter string) func(pdfToMarkdownInputStruct) (converterOutput, error) {
	switch converter {
	case "docling":
		return convertPDFToMarkdownWithDocling
	default:
		return convertPDFToMarkdownWithPDFPlumber
	}
}
