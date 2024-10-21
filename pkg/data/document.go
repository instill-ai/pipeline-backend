package data

import (
	"fmt"
	"strings"

	"github.com/instill-ai/pipeline-backend/pkg/component/operator/document/v0"
	"github.com/instill-ai/pipeline-backend/pkg/data/value"
)

type documentData struct {
	fileData
}

const DOC = "application/msword"
const DOCX = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
const PPT = "application/vnd.ms-powerpoint"
const PPTX = "application/vnd.openxmlformats-officedocument.presentationml.presentation"
const HTML = "text/html"
const PDF = "application/pdf"

func (documentData) IsValue() {}

func NewDocumentFromBytes(b []byte, contentType, fileName string) (doc *documentData, err error) {
	f, err := NewFileFromBytes(b, contentType, fileName)
	if err != nil {
		return
	}
	return newDocument(f)
}

func NewDocumentFromURL(url string) (doc *documentData, err error) {
	f, err := NewFileFromURL(url)
	if err != nil {
		return
	}
	return newDocument(f)
}

func newDocument(f *fileData) (doc *documentData, err error) {
	return &documentData{
		fileData: *f,
	}, nil
}

func (d *documentData) Text() (val *stringData, err error) {

	dataURI, err := d.DataURI(d.contentType)
	if err != nil {
		return nil, err
	}

	res, err := document.ConvertDocumentToMarkdown(
		&document.ConvertDocumentToMarkdownInput{
			Document: dataURI.String(),
			Filename: d.fileName,
		}, document.GetMarkdownTransformer)
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

func (d *documentData) PDF() (val *documentData, err error) {
	dataURI, err := d.DataURI(d.contentType)
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
	case PDF:
		return d, nil
	}

	s, err := document.ConvertToPDF(dataURI.String(), ext)
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

	dataURI, err := pdf.DataURI("application/pdf")
	if err != nil {
		return nil, err
	}
	res, err := document.ConvertDocumentToImage(&document.ConvertDocumentToImagesInput{
		Document: dataURI.String(),
		Filename: d.fileName,
	})
	if err != nil {
		return nil, err
	}

	images := make([]value.Value, len(res.Images))

	for idx := range res.Images {

		img := strings.Split(res.Images[idx], ",")[1]
		images[idx], err = NewImageFromURL(fmt.Sprintf("data:image/jpeg;filename=%s;base64,%s", res.Filenames[idx], img))
		if err != nil {
			return nil, fmt.Errorf("NewImageFromBytes: %w", err)
		}
	}
	return images, nil
}

func (d *documentData) Get(path string) (v value.Value, err error) {
	v, err = d.fileData.Get(path)
	if err == nil {
		return
	}
	switch {
	case comparePath(path, ""):
		return d, nil
	case comparePath(path, ".text"):
		return d.Text()
	case comparePath(path, ".pdf"):
		return d.PDF()
	case comparePath(path, ".images"):
		return d.Images()
	case matchPathPrefix(path, ".images"):
		// TODO: we should only convert the required pages.
		images, err := d.Images()
		if err != nil {
			return nil, err
		}
		_, path, err = trimFirstKeyFromPath(path)
		if err != nil {
			return nil, err
		}
		return images.Get(path)

	case comparePath(path, ".base64"):
		return d.GetBase64(d.contentType)
	case comparePath(path, ".data-uri"):
		return d.DataURI(d.contentType)
	case comparePath(path, ".byte-array"):
		return d.Binary(d.contentType)
	}
	return nil, fmt.Errorf("wrong path")
}
