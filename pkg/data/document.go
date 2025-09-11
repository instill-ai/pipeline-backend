package data

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/instill-ai/pipeline-backend/pkg/component/operator/document/v0/transformer"
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
	"github.com/instill-ai/pipeline-backend/pkg/data/path"
	"github.com/instill-ai/pipeline-backend/pkg/external"
)

type documentData struct {
	fileData
}

// Document types
const (
	DOC      = "application/msword"
	DOCX     = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	PPT      = "application/vnd.ms-powerpoint"
	PPTX     = "application/vnd.openxmlformats-officedocument.presentationml.presentation"
	XLS      = "application/vnd.ms-excel"
	XLSX     = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	HTML     = "text/html"
	PLAIN    = "text/plain"
	TEXT     = "text"
	MARKDOWN = "text/markdown"
	CSV      = "text/csv"
	PDF      = "application/pdf"
	OLE      = "application/x-ole-storage"
)

var documentGetters = map[string]func(*documentData) (format.Value, error){
	"text":   func(d *documentData) (format.Value, error) { return d.Text() },
	"pdf":    func(d *documentData) (format.Value, error) { return d.PDF() },
	"images": func(d *documentData) (format.Value, error) { return d.Images() },
}

func (documentData) IsValue() {}

// NewDocumentFromBytes creates a new documentData from a byte slice
func NewDocumentFromBytes(b []byte, contentType, filename string) (*documentData, error) {
	return createDocumentData(b, contentType, filename)
}

// NewDocumentFromURL creates a new documentData from a URL
func NewDocumentFromURL(ctx context.Context, binaryFetcher external.BinaryFetcher, url string) (*documentData, error) {
	b, contentType, filename, err := binaryFetcher.FetchFromURL(ctx, url)
	if err != nil {
		return nil, err
	}
	return createDocumentData(b, contentType, filename)
}

func createDocumentData(b []byte, contentType, filename string) (*documentData, error) {
	// Normalize provided content type
	if contentType != "" {
		contentType = strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	}
	// If upstream reports generic/legacy Office container types, infer from filename
	if contentType == OLE || contentType == OCTETSTREAM {
		lf := strings.ToLower(filename)
		switch {
		case strings.HasSuffix(lf, ".doc"):
			contentType = DOC
		case strings.HasSuffix(lf, ".docx"):
			contentType = DOCX
		case strings.HasSuffix(lf, ".ppt"):
			contentType = PPT
		case strings.HasSuffix(lf, ".pptx"):
			contentType = PPTX
		case strings.HasSuffix(lf, ".xls"):
			contentType = XLS
		case strings.HasSuffix(lf, ".xlsx"):
			contentType = XLSX
		default:
			// Fallback: assume legacy DOC if we cannot infer by extension
			contentType = DOC
		}
	}
	f, err := NewFileFromBytes(b, contentType, filename)
	if err != nil {
		return nil, err
	}
	return newDocument(f)
}

func newDocument(f *fileData) (*documentData, error) {
	supportedTypes := []string{DOC, DOCX, PPT, PPTX, XLS, XLSX, HTML, PDF, PLAIN, MARKDOWN, CSV, TEXT, OCTETSTREAM}
	isSupported := false
	for _, supportedType := range supportedTypes {
		if strings.HasPrefix(f.contentType, TEXT) {
			isSupported = true
			break
		}
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

	if strings.HasPrefix(d.contentType, TEXT) {
		return NewString(string(d.raw)), nil
	}
	dataURI, err := d.DataURI()
	if err != nil {
		return nil, err
	}

	res, err := transformer.NewDocumentToMarkdownConverter(nil).Convert(&transformer.ConvertDocumentToMarkdownInput{
		Document: dataURI.String(),
		Filename: d.filename,
	})
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

	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}

	return NewDocumentFromBytes(b, PDF, d.filename)
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
	res, err := transformer.NewDocumentToImageConverter(nil).Convert(&transformer.ConvertDocumentToImagesInput{
		Document: dataURI.String(),
		Filename: d.filename,
	})
	if err != nil {
		return nil, err
	}

	images := make([]format.Value, len(res.Images))

	for idx := range res.Images {
		b, err := base64.StdEncoding.DecodeString(res.Images[idx])
		if err != nil {
			return nil, err
		}
		images[idx], err = NewImageFromBytes(b, PNG, d.filename, false)
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
