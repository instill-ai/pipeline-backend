package document

import (
	"fmt"
	"strings"

	"google.golang.org/protobuf/types/known/structpb"

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
}

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
	}

	if inputStruct.Filename != "" {
		filename := strings.Split(inputStruct.Filename, ".")[0] + ".md"
		outputStruct.Filename = filename
	}
	return outputStruct, nil
}

func (e *execution) convertDocumentToMarkdown(input *structpb.Struct) (*structpb.Struct, error) {
	inputStruct := ConvertDocumentToMarkdownInput{}
	err := base.ConvertFromStructpb(input, &inputStruct)
	if err != nil {
		return nil, err
	}

	outputStruct, err := ConvertDocumentToMarkdown(&inputStruct, e.getMarkdownTransformer)
	if err != nil {
		return nil, err
	}

	output, err := base.ConvertToStructpb(outputStruct)
	if err != nil {
		return nil, err
	}

	return output, nil
}

func GetMarkdownTransformer(fileExtension string, inputStruct *ConvertDocumentToMarkdownInput) (MarkdownTransformer, error) {
	switch fileExtension {
	case "pdf":
		return PDFToMarkdownTransformer{
			Base64EncodedText:   inputStruct.Document,
			FileExtension:       fileExtension,
			DisplayImageTag:     inputStruct.DisplayImageTag,
			DisplayAllPageImage: inputStruct.DisplayAllPageImage,
			PDFConvertFunc:      getPDFConvertFunc("pdfplumber"),
		}, nil
	case "doc", "docx":
		return DocxDocToMarkdownTransformer{
			Base64EncodedText:   inputStruct.Document,
			FileExtension:       fileExtension,
			DisplayImageTag:     inputStruct.DisplayImageTag,
			DisplayAllPageImage: inputStruct.DisplayAllPageImage,
			PDFConvertFunc:      getPDFConvertFunc("pdfplumber"),
		}, nil
	case "ppt", "pptx":
		return PptPptxToMarkdownTransformer{
			Base64EncodedText:   inputStruct.Document,
			FileExtension:       fileExtension,
			DisplayImageTag:     inputStruct.DisplayImageTag,
			DisplayAllPageImage: inputStruct.DisplayAllPageImage,
			PDFConvertFunc:      getPDFConvertFunc("pdfplumber"),
		}, nil
	case "html":
		return HTMLToMarkdownTransformer{
			Base64EncodedText: inputStruct.Document,
			FileExtension:     fileExtension,
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

// We could provide more converters in the future. For now, we only have one.
func getPDFConvertFunc(converter string) func(string, bool, bool) (converterOutput, error) {
	switch converter {
	default:
		return convertPDFToMarkdownWithPDFPlumber
	}
}
