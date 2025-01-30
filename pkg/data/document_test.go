package data

import (
	"context"
	"os"
	"testing"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/external"
)

func TestNewDocumentFromBytes(t *testing.T) {
	t.Parallel()
	c := qt.New(t)

	testCases := []struct {
		name        string
		filename    string
		contentType string
	}{
		{"Valid PDF document", "sample2.pdf", "application/pdf"},
		{"Valid TXT document", "sample2.txt", "text/plain"},
		{"Valid DOCX document", "sample1.docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document"},
		{"Invalid file type", "sample_640_426.png", ""},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *qt.C) {
			var documentBytes []byte
			var err error

			if tc.filename != "" {
				documentBytes, err = os.ReadFile("testdata/" + tc.filename)
				c.Assert(err, qt.IsNil)
			}

			document, err := NewDocumentFromBytes(documentBytes, tc.contentType, tc.filename)

			if tc.name == "Invalid file type" || tc.name == "Invalid document format" || tc.name == "Empty document bytes" {
				c.Assert(err, qt.Not(qt.IsNil))
				return
			}

			c.Assert(err, qt.IsNil)
			c.Assert(document.ContentType().String(), qt.Equals, tc.contentType)
		})
	}
}

func TestNewDocumentFromURL(t *testing.T) {
	c := qt.New(t)
	c.Parallel()

	ctx := context.Background()
	binaryFetcher := external.NewBinaryFetcher()

	test := func(name, url string, hasErr bool) {
		c.Run(name, func(c *qt.C) {
			c.Parallel()

			document, err := NewDocumentFromURL(ctx, binaryFetcher, url)
			if hasErr {
				c.Assert(err, qt.IsNotNil)
				return
			}

			c.Assert(err, qt.IsNil)
			c.Assert(document.ContentType().String(), qt.Not(qt.Equals), "")
		})
	}

	test("ok - Valid PDF URL", "https://raw.githubusercontent.com/instill-ai/pipeline-backend/24153e2c57ba4ce508059a0bd1af8528b07b5ed3/pkg/data/testdata/sample2.pdf", false)
	test("ok - Valid TXT URL", "https://raw.githubusercontent.com/instill-ai/pipeline-backend/24153e2c57ba4ce508059a0bd1af8528b07b5ed3/pkg/data/testdata/sample2.txt", false)
	test("ok - Valid DOCX URL", "https://filesamples.com/samples/document/docx/sample1.docx", false)
	test("nok - Invalid URL", "https://invaliiiddd-url.com/document.pdf", true)
}

func TestDocumentProperties(t *testing.T) {
	t.Parallel()
	c := qt.New(t)

	testCases := []struct {
		name        string
		filename    string
		contentType string
	}{
		{"PDF document", "sample2.pdf", "application/pdf"},
		{"TXT document", "sample2.txt", "text/plain"},
		{"DOCX document", "sample1.docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document"},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *qt.C) {
			documentBytes, err := os.ReadFile("testdata/" + tc.filename)
			c.Assert(err, qt.IsNil)

			document, err := NewDocumentFromBytes(documentBytes, tc.contentType, tc.filename)
			c.Assert(err, qt.IsNil)

			c.Assert(document.ContentType().String(), qt.Equals, tc.contentType)
		})
	}
}
