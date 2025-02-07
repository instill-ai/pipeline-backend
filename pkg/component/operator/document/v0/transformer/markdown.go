package transformer

import (
	"fmt"
	"strings"

	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util"
	"go.uber.org/zap"
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

// DocumentToMarkdownConverter transforms documents to Markdown.
type DocumentToMarkdownConverter struct {
	logger *zap.Logger
}

// NewDocumentToMarkdownConverter initializes a DocumentToMarkdownConverter.
func NewDocumentToMarkdownConverter(l *zap.Logger) *DocumentToMarkdownConverter {
	if l == nil {
		l = zap.NewNop()
	}

	return &DocumentToMarkdownConverter{logger: l}
}

// Convert transforms a document to Markdown format. In PDF-to-Markdown
// conversion, the converter can be selected (between Docling and pdfplumber).
// For the moment, the rest of extensions don't allow for such selection.
func (c *DocumentToMarkdownConverter) Convert(in *ConvertDocumentToMarkdownInput) (*ConvertDocumentToMarkdownOutput, error) {
	contentType, err := util.GetContentTypeFromBase64(in.Document)
	if err != nil {
		return nil, err
	}

	fileExtension := util.TransformContentTypeToFileExtension(contentType)
	if fileExtension == "" {
		return nil, fmt.Errorf("unsupported file type")
	}

	transformer, err := c.getMarkdownTransformer(fileExtension, in)
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

func (c *DocumentToMarkdownConverter) getMarkdownTransformer(fileExtension string, inputStruct *ConvertDocumentToMarkdownInput) (markdownTransformer, error) {
	switch fileExtension {
	case "html":
		return &htmlToMarkdownTransformer{base64EncodedText: inputStruct.Document}, nil
	case "xlsx":
		return &xlsxToMarkdownTransformer{base64EncodedText: inputStruct.Document}, nil
	case "xls":
		return &xlsToMarkdownTransformer{base64EncodedText: inputStruct.Document}, nil
	case "csv":
		return &csvToMarkdownTransformer{base64EncodedText: inputStruct.Document}, nil
	}

	pdfToMarkdownStruct := pdfToMarkdownInputStruct{
		displayImageTag:     inputStruct.DisplayImageTag,
		displayAllPageImage: inputStruct.DisplayAllPageImage,
		resolution:          inputStruct.Resolution,
	}
	pdfTransformer := &pdfToMarkdownTransformer{
		fileExtension:       fileExtension,
		engine:              inputStruct.Converter,
		pdfToMarkdownStruct: pdfToMarkdownStruct,
		logger:              c.logger,
	}

	switch fileExtension {
	case "pdf":
		pdfTransformer.pdfToMarkdownStruct.base64Text = inputStruct.Document
		return pdfTransformer, nil
	case "doc", "docx":
		return &docToMarkdownTransformer{
			pdfToMarkdownTransformer: pdfTransformer,
			base64EncodedText:        inputStruct.Document,
		}, nil
	case "ppt", "pptx":
		return &pptToMarkdownTransformer{
			pdfToMarkdownTransformer: pdfTransformer,
			base64EncodedText:        inputStruct.Document,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported file type")
	}
}
