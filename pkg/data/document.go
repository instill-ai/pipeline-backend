package data

import (
	"fmt"

	"github.com/instill-ai/pipeline-backend/pkg/component/operator/document/v0/transformer"
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
	"github.com/instill-ai/pipeline-backend/pkg/data/path"
)

type documentData struct {
	fileData
}

const DOC = "application/msword"
const DOCX = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
const PPT = "application/vnd.ms-powerpoint"
const PPTX = "application/vnd.openxmlformats-officedocument.presentationml.presentation"
const XLS = "application/vnd.ms-excel"
const XLSX = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
const HTML = "text/html"
const PLAIN = "text/plain"
const MARKDOWN = "text/markdown"
const CSV = "text/csv"
const PDF = "application/pdf"

var documentGetters = map[string]func(*documentData) (format.Value, error){
	"text":   func(d *documentData) (format.Value, error) { return d.Text() },
	"pdf":    func(d *documentData) (format.Value, error) { return d.PDF() },
	"images": func(d *documentData) (format.Value, error) { return d.Images() },
}

func (documentData) IsValue() {}

func NewDocumentFromBytes(b []byte, contentType, fileName string) (*documentData, error) {
	return createDocumentData(b, contentType, fileName)
}

func NewDocumentFromURL(url string) (*documentData, error) {
	b, contentType, fileName, err := convertURLToBytes(url)
	if err != nil {
		return nil, err
	}

	return createDocumentData(b, contentType, fileName)
}

func createDocumentData(b []byte, contentType, fileName string) (*documentData, error) {
	f, err := NewFileFromBytes(b, contentType, fileName)
	if err != nil {
		return nil, err
	}
	return newDocument(f)
}

func newDocument(f *fileData) (*documentData, error) {
	supportedTypes := []string{DOC, DOCX, PPT, PPTX, XLS, XLSX, HTML, PDF, PLAIN, MARKDOWN, CSV}
	isSupported := false
	for _, supportedType := range supportedTypes {
		if f.contentType == supportedType {
			isSupported = true
			break
		}
	}
	if !isSupported {
		return nil, fmt.Errorf("unsupported document type: %s", f.contentType)
	}
	d := &documentData{
		fileData: *f,
	}

	return d, nil
}

func (d *documentData) Text() (val format.String, err error) {
	dataURI, err := d.DataURI()
	if err != nil {
		return nil, err
	}

	res, err := transformer.ConvertDocumentToMarkdown(
		&transformer.ConvertDocumentToMarkdownTransformerInput{
			Document: dataURI.String(),
			Filename: d.fileName,
		}, transformer.GetMarkdownTransformer)
	if err != nil {
		return nil, err
	}

	return NewString(res.Body), nil
}

func (d *documentData) String() (val string) {
	text, err := d.Text()
	if err != nil {
		return ""
	}
	return text.String()
}

func (d *documentData) PDF() (val format.Document, err error) {

	dataURI, err := d.DataURI()
	if err != nil {
		return nil, err
	}

	ext := ""
	switch d.contentType {
	case DOC:
		ext = "doc"
	case DOCX:
		ext = "docx"
	case PPT:
		ext = "ppt"
	case PPTX:
		ext = "pptx"
	case HTML:
		ext = "html"
	case MARKDOWN:
		ext = "md"
	case XLS:
		ext = "xls"
	case XLSX:
		ext = "xlsx"
	case PDF:
		return d, nil
	}

	s, err := transformer.ConvertToPDF(dataURI.String(), ext)
	if err != nil {
		return nil, err
	}

	return NewDocumentFromURL(fmt.Sprintf("data:application/pdf;base64,%s", s))
}

func (d *documentData) Images() (mp Array, err error) {

	pdf, err := d.PDF()
	if err != nil {
		return nil, err
	}

	dataURI, err := pdf.DataURI()
	if err != nil {
		return nil, err
	}
	res, err := transformer.ConvertDocumentToImage(&transformer.ConvertDocumentToImagesTransformerInput{
		Document: dataURI.String(),
		Filename: d.fileName,
	})
	if err != nil {
		return nil, err
	}

	images := make([]format.Value, len(res.Images))

	for idx := range res.Images {
		// img := strings.Split(res.Images[idx], ",")[1]
		images[idx], err = NewImageFromURL(res.Images[idx])
		if err != nil {
			return nil, fmt.Errorf("NewImageFromBytes: %w", err)
		}
	}
	return images, nil
}

func (d *documentData) Get(p *path.Path) (v format.Value, err error) {
	if p == nil || p.IsEmpty() {
		return d, nil
	}

	firstSeg, remainingPath, err := p.TrimFirst()
	if err != nil {
		return nil, err
	}

	if firstSeg.SegmentType != path.AttributeSegment {
		return nil, fmt.Errorf("path not found: %s", p)
	}

	getter, exists := documentGetters[firstSeg.Attribute]
	if !exists {
		return d.fileData.Get(p)
	}

	result, err := getter(d)
	if err != nil {
		return nil, err
	}

	if remainingPath.IsEmpty() {
		return result, nil
	}

	return result.Get(remainingPath)
}
