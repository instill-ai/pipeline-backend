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
	// Converter selects the conversion engine for the transformation. For the
	// moment, it only applies to PDF-to-Markdown conversion. The allowed
	// values are:
	// - "pdfplumber"
	// - "docling"
	// Any other value will default to "pdfplumber" for backwards
	// compatibility.
	Converter string
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

// ConvertDocumentToMarkdown transforms a document to Markdown format. In
// PDF-to-Markdown conversion, the converter can be selected (between Docling
// and pdfplumber).  The rest of extensions use a deterministic converter.
func ConvertDocumentToMarkdown(in *ConvertDocumentToMarkdownInput) (*ConvertDocumentToMarkdownOutput, error) {
	contentType, err := util.GetContentTypeFromBase64(in.Document)
	if err != nil {
		return nil, err
	}

	fileExtension := util.TransformContentTypeToFileExtension(contentType)
	if fileExtension == "" {
		return nil, fmt.Errorf("unsupported file type")
	}

	transformer, err := getMarkdownTransformer(fileExtension, in)
	if err != nil {
		return nil, err
	}
	converterOutput, err := transformer.transform()
	if err != nil {
		return nil, err
	}

	out := &ConvertDocumentToMarkdownOutput{
		Body:          converterOutput.Body,
		Images:        converterOutput.Images,
		Error:         strings.Join(converterOutput.ParsingError, "\n"),
		AllPageImages: converterOutput.AllPageImages,
		Markdowns:     converterOutput.Markdowns,
	}

	if in.Filename != "" {
		filename := strings.Split(in.Filename, ".")[0] + ".md"
		out.Filename = filename
	}

	return out, nil
}

func getMarkdownTransformer(fileExtension string, inputStruct *ConvertDocumentToMarkdownInput) (markdownTransformer, error) {
	switch fileExtension {
	case "pdf":
		pdfToMarkdownStruct := pdfToMarkdownInputStruct{
			base64Text:          inputStruct.Document,
			displayImageTag:     inputStruct.DisplayImageTag,
			displayAllPageImage: inputStruct.DisplayAllPageImage,
			resolution:          inputStruct.Resolution,
		}

		return pdfToMarkdownTransformer{
			fileExtension:       fileExtension,
			pdfToMarkdownStruct: pdfToMarkdownStruct,
			pdfConvertFunc:      getPDFConvertFunc(inputStruct.Converter),
		}, nil
	case "doc", "docx":

		t := docToMarkdownTransformer{base64EncodedText: inputStruct.Document}
		t.fileExtension = fileExtension
		t.pdfToMarkdownStruct = pdfToMarkdownInputStruct{
			displayImageTag:     inputStruct.DisplayImageTag,
			displayAllPageImage: inputStruct.DisplayAllPageImage,
			resolution:          inputStruct.Resolution,
		}
		t.pdfConvertFunc = getPDFConvertFunc("pdfplumber")

		return t, nil
	case "ppt", "pptx":
		t := pptToMarkdownTransformer{base64EncodedText: inputStruct.Document}
		t.fileExtension = fileExtension
		t.pdfToMarkdownStruct = pdfToMarkdownInputStruct{
			displayImageTag:     inputStruct.DisplayImageTag,
			displayAllPageImage: inputStruct.DisplayAllPageImage,
			resolution:          inputStruct.Resolution,
		}
		t.pdfConvertFunc = getPDFConvertFunc("pdfplumber")

		return t, nil
	case "html":
		return htmlToMarkdownTransformer{base64EncodedText: inputStruct.Document}, nil
	case "xlsx":
		return xlsxToMarkdownTransformer{base64EncodedText: inputStruct.Document}, nil
	case "xls":
		return xlsToMarkdownTransformer{base64EncodedText: inputStruct.Document}, nil
	case "csv":
		return csvToMarkdownTransformer{base64EncodedText: inputStruct.Document}, nil
	default:
		return nil, fmt.Errorf("unsupported file type")
	}
}

type pdfToMarkdownInputStruct struct {
	base64Text          string
	displayImageTag     bool
	displayAllPageImage bool
	resolution          int
}

func getPDFConvertFunc(converter string) func(pdfToMarkdownInputStruct) (converterOutput, error) {
	switch converter {
	case "docling":
		return convertPDFToMarkdown(doclingPDFToMDConverter)
	default:
		return convertPDFToMarkdown(pageImageProcessor + pdfTransformer + pdfPlumberPDFToMDConverter)
	}
}
