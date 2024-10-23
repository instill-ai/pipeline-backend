package document

import "github.com/instill-ai/pipeline-backend/pkg/data/format"

type ConvertDocumentToMarkdownInput struct {
	Document            format.Document `key:"document"`
	DisplayImageTag     bool            `key:"display-image-tag"`
	Filename            string          `key:"filename"`
	DisplayAllPageImage bool            `key:"display-all-page-image"`
}

type ConvertDocumentToMarkdownOutput struct {
	Body          string         `key:"body"`
	Filename      string         `key:"filename"`
	Images        []format.Image `key:"images"`
	Error         string         `key:"error"`
	AllPageImages []format.Image `key:"all-page-images"`
}

type ConvertDocumentToImagesInput struct {
	Document format.Document `key:"document"`
	Filename string          `key:"filename"`
}

type ConvertDocumentToImagesOutput struct {
	Images    []format.Image `key:"images"`
	Filenames []string       `key:"filenames"`
}

// ConvertToTextInput defines the input for convert to text task
type ConvertToTextInput struct {
	// Document: Document to convert
	Document format.Document `key:"document"`
	Filename string          `key:"filename"`
}

// ConvertToTextOutput defines the output for convert to text task
type ConvertToTextOutput struct {
	// Body: Plain text converted from the document
	Body string `key:"body"`
	// Meta: Metadata extracted from the document
	Meta map[string]string `key:"meta"`
	// MSecs: Time taken to convert the document
	MSecs uint32 `key:"msecs"`
	// Error: Error message if any during the conversion process
	Error    string `key:"error"`
	Filename string `key:"filename"`
}
