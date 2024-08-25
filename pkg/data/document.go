package data

import (
	"fmt"
	"strings"
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
	f, err := NewFileFromBytes(b, contentType, fileName, nil)
	if err != nil {
		return
	}
	return newDocument(f)
}

func NewDocumentFromURL(url string) (doc *Document, err error) {
	f, err := NewFileFromURL(url, nil)
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

func (d *Document) GetText() *String {
	return NewString("test")
}

func (d *Document) Get(path string) (v Value, err error) {
	v, err = d.File.Get(path)
	if err == nil {
		return
	}
	switch {
	case path == "":
		// TODO: we use data-url for now
		return d.GetDataURL(d.ContentType)
	case path == ".text":
		return d.GetText(), nil
	case strings.HasPrefix(path, ".base64"):
		return d.GetBase64(d.ContentType)
	case strings.HasPrefix(path, ".data-url"):
		return d.GetDataURL(d.ContentType)
	case strings.HasPrefix(path, ".byte-array"):
		return d.GetByteArray(d.ContentType)
	}
	return nil, fmt.Errorf("wrong path")
}
