package data

import (
	"fmt"
	"strings"

	"github.com/instill-ai/component/operator/document/v0"
	//  "github.com/instill-ai/component/store"
)

type Document struct {
	File
}

const DOC = "application/msword"
const DOCX = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
const PPT = "application/vnd.ms-powerpoint"
const PPTX = "application/vnd.openxmlformats-officedocument.presentationml.presentation"
const HTML = "text/html"
const PDF = "application/pdf"

func (Document) isValue() {}

func NewDocumentFromBytes(b []byte, contentType, fileName string) (doc *Document, err error) {
	f, err := NewFileFromBytes(b, contentType, fileName)
	if err != nil {
		return
	}
	return newDocument(f)
}

func NewDocumentFromURL(url string) (doc *Document, err error) {
	f, err := NewFileFromURL(url)
	if err != nil {
		return
	}
	return newDocument(f)
}

func newDocument(f *File) (doc *Document, err error) {
	return &Document{
		File: *f,
	}, nil
}

func (d *Document) GetText() (val *String, err error) {

	dataURL, err := d.GetDataURL(d.ContentType)
	if err != nil {
		return nil, err
	}

	res, err := document.ConvertToText(document.ConvertToTextInput{
		Document: dataURL.GetString(),
		Filename: d.FileName,
	})
	if err != nil {
		return nil, err
	}

	return NewString(res.Body), nil
}

func (d *Document) GetMarkdown() (val *String, err error) {

	dataURL, err := d.GetDataURL(d.ContentType)
	if err != nil {
		return nil, err
	}

	res, err := document.ConvertDocumentToMarkdown(
		&document.ConvertDocumentToMarkdownInput{
			Document: dataURL.GetString(),
			Filename: d.FileName,
		}, document.GetMarkdownTransformer)
	if err != nil {
		return nil, err
	}

	return NewString(res.Body), nil
}

func (d *Document) GetPDF() (val *Document, err error) {

	dataURL, err := d.GetDataURL(d.ContentType)
	if err != nil {
		return nil, err
	}

	ext := ""
	switch d.ContentType {
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

	s, err := document.ConvertToPDF(dataURL.GetString(), ext)
	if err != nil {
		return nil, err
	}

	return NewDocumentFromURL(fmt.Sprintf("data:application/pdf;base64,%s", s))
}

func (d *Document) GetImages() (mp *Array, err error) {

	pdf, err := d.GetPDF()
	if err != nil {
		return nil, err
	}
	fmt.Println(d.FileName)

	dataURL, err := pdf.GetDataURL("application/pdf")
	if err != nil {
		return nil, err
	}
	fmt.Println("cccc", dataURL.GetString()[:40])
	res, err := document.ConvertPDFToImage(&document.ConvertPDFToImagesInput{
		PDF:      dataURL.GetString(),
		Filename: d.FileName,
	})
	if err != nil {
		return nil, err
	}

	images := NewArray(make([]Value, len(res.Images)))

	for idx := range res.Images {

		img := strings.Split(res.Images[idx], ",")[1]
		images.Values[idx], err = NewImageFromURL(fmt.Sprintf("data:image/jpeg;filename=%s;base64,%s", res.Filenames[idx], img))
		if err != nil {
			return nil, fmt.Errorf("NewImageFromBytes: %w", err)
		}
	}
	return images, nil
}

func (d *Document) Get(path string) (v Value, err error) {
	v, err = d.File.Get(path)
	if err == nil {
		return
	}
	switch {
	case comparePath(path, ""):
		// TODO: we use data-url for now
		return d.GetDataURL(d.ContentType)
	case comparePath(path, ".text"):
		return d.GetText()
	case comparePath(path, ".markdown"):
		return d.GetMarkdown()
	case comparePath(path, ".pdf"):
		return d.GetPDF()
	case comparePath(path, ".images"):
		return d.GetImages()
	case matchPathPrefix(path, ".images"):
		// TODO: we should only convert the required pages.
		images, err := d.GetImages()
		if err != nil {
			return nil, err
		}
		_, path, err = trimFirstKeyFromPath(path)
		if err != nil {
			return nil, err
		}
		return images.Get(path)

	case comparePath(path, ".base64"):
		return d.GetBase64(d.ContentType)
	case comparePath(path, ".data-url"):
		return d.GetDataURL(d.ContentType)
	case comparePath(path, ".byte-array"):
		return d.GetByteArray(d.ContentType)
	}
	return nil, fmt.Errorf("wrong path")
}
