package transformer

import (
	"fmt"
	"strings"

	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util"
)

// ConvertDocumentToMarkdownInput ...
type ConvertDocumentToMarkdownInput struct {
	Document            string
	DisplayImageTag     bool
	Filename            string
	DisplayAllPageImage bool
	Resolution          int
	UseDoclingConverter bool
}

// ConvertDocumentToMarkdownOutput ...
type ConvertDocumentToMarkdownOutput struct {
	Body          string
	Filename      string
	Images        []string
	Error         string
	AllPageImages []string
	Markdowns     []string
}

// ConvertDocumentToMarkdown transforms a document to Markdown format.
func ConvertDocumentToMarkdown(inputStruct *ConvertDocumentToMarkdownInput, transformerGetter MarkdownTransformerGetterFunc) (*ConvertDocumentToMarkdownOutput, error) {
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

	outputStruct := &ConvertDocumentToMarkdownOutput{
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

func GetMarkdownTransformer(fileExtension string, inputStruct *ConvertDocumentToMarkdownInput) (MarkdownTransformer, error) {
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

func getPDFConvertFunc(converter string) func(pdfToMarkdownInputStruct) (converterOutput, error) {
	switch converter {
	case "docling":
		return convertPDFToMarkdownWithDocling
	default:
		return convertPDFToMarkdownWithPDFPlumber
	}
}
