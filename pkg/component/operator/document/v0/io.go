package document

import "github.com/instill-ai/pipeline-backend/pkg/data/format"

// ConvertDocumentToMarkdownInput contains the public input parameters for
// `TASK_CONVERT_TO_MARKDOWN`.
type ConvertDocumentToMarkdownInput struct {
	Document            format.Document `instill:"document"`
	DisplayImageTag     bool            `instill:"display-image-tag,default=false"`
	Filename            string          `instill:"filename"`
	DisplayAllPageImage bool            `instill:"display-all-page-image,default=false"`
	Resolution          int             `instill:"resolution,default=300"`
	UseDoclingConverter bool            `instill:"use-docling-converter,default=false"`
}

type ConvertDocumentToMarkdownOutput struct {
	Body          string         `instill:"body"`
	Filename      string         `instill:"filename"`
	Images        []format.Image `instill:"images"`
	Error         string         `instill:"error"`
	AllPageImages []format.Image `instill:"all-page-images"`
	Markdowns     []string       `instill:"markdowns"`
}

type ConvertDocumentToImagesInput struct {
	Document   format.Document `instill:"document"`
	Filename   string          `instill:"filename"`
	Resolution int             `instill:"resolution,default=300"`
}

type ConvertDocumentToImagesOutput struct {
	Images    []format.Image `instill:"images"`
	Filenames []string       `instill:"filenames"`
}

// ConvertToTextInput defines the input for convert to text task
type ConvertToTextInput struct {
	// Document: Document to convert
	Document format.Document `instill:"document"`
	Filename string          `instill:"filename"`
}

// ConvertToTextOutput defines the output for convert to text task
type ConvertToTextOutput struct {
	// Body: Plain text converted from the document
	Body string `instill:"body"`
	// Meta: Metadata extracted from the document
	Meta map[string]string `instill:"meta"`
	// MSecs: Time taken to convert the document
	MSecs uint32 `instill:"msecs"`
	// Error: Error message if any during the conversion process
	Error    string `instill:"error"`
	Filename string `instill:"filename"`
}
