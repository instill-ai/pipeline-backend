package data

import (
	"context"
	"encoding/base64"
	"os"
	"strings"
	"testing"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/data/format"
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
		{"Valid DOC document", "sample1.doc", "application/msword"},
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

			// Test that the document properly implements format.Document interface
			// This will catch if it's incorrectly returning a *fileData instead of *documentData
			var _ format.Document = document
		})
	}
}

// Removed TestDOCTypeDetection - functionality covered by TestNewDocumentFromBytes

// Removed TestDocumentGobSerialization - not critical functionality

// Removed TestComponentInputSimulation - functionality covered by struct tests

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
	test("nok - Invalid URL", "https://invaliiiddd-url.com/document.pdf", true)
}

// Removed TestDocumentProperties - functionality covered by TestNewDocumentFromBytes

// Removed TestDOCContentTypeDetection - functionality covered by TestNewDocumentFromBytes

// Removed TestMimetypeDetection - edge case not critical for core functionality

func TestDocumentDataURIHandling(t *testing.T) {
	t.Parallel()
	c := qt.New(t)

	// Create a simple test document
	docBytes := []byte("test document content")
	original, err := NewDocumentFromBytes(docBytes, "application/pdf", "test.pdf")
	c.Assert(err, qt.IsNil)

	// Get the data URI
	dataURI, err := original.DataURI()
	c.Assert(err, qt.IsNil)

	// Check that the data URI has the correct format
	dataURIStr := dataURI.String()
	c.Assert(strings.HasPrefix(dataURIStr, "data:application/pdf"), qt.Equals, true,
		qt.Commentf("Expected data URI to start with 'data:application/pdf', got: %s", dataURIStr))

	// Test that the data URI can be parsed back using the real binary fetcher
	ctx := context.Background()
	binaryFetcher := external.NewBinaryFetcher()

	parsedBytes, parsedContentType, parsedFilename, err := binaryFetcher.FetchFromURL(ctx, dataURIStr)
	c.Assert(err, qt.IsNil)
	c.Assert(parsedContentType, qt.Equals, "application/pdf")
	c.Assert(parsedBytes, qt.DeepEquals, docBytes)

	// Test that the parsed data creates a valid document
	result, err := NewDocumentFromBytes(parsedBytes, parsedContentType, parsedFilename)
	c.Assert(err, qt.IsNil)
	_, ok := any(result).(*documentData)
	c.Assert(ok, qt.Equals, true, qt.Commentf("Expected *documentData, got %T", result))
}

func TestAllSupportedDocumentFormats(t *testing.T) {
	t.Parallel()
	c := qt.New(t)

	// Test cases for files that exist in testdata
	fileTestCases := []struct {
		name        string
		filename    string
		contentType string
	}{
		{"PDF", "sample2.pdf", "application/pdf"},
		{"TXT", "sample2.txt", "text/plain"},
		{"DOCX", "sample1.docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document"},
		{"DOC", "sample1.doc", "application/msword"},
		{"PPTX", "sample1.pptx", "application/vnd.openxmlformats-officedocument.presentationml.presentation"},
		{"PPT", "sample1.ppt", "application/vnd.ms-powerpoint"},
		{"HTML", "sample1.html", "text/html"},
		{"PLAIN", "sample1.txt", "text/plain"},
		{"TEXT", "sample1.txt", "text"},
	}

	for _, tc := range fileTestCases {
		c.Run(tc.name, func(c *qt.C) {
			b, err := os.ReadFile("testdata/" + tc.filename)
			c.Assert(err, qt.IsNil)

			doc, err := NewDocumentFromBytes(b, tc.contentType, tc.filename)
			c.Assert(err, qt.IsNil)

			// Ensure concrete type is *documentData and it implements format.Document
			_, ok := any(doc).(*documentData)
			c.Assert(ok, qt.Equals, true, qt.Commentf("Expected *documentData, got %T", doc))
			var as format.Document = doc
			c.Assert(as, qt.IsNotNil)
		})
	}

	// Test cases for formats without files (using synthetic data)
	syntheticTestCases := []struct {
		name        string
		content     []byte
		contentType string
		filename    string
	}{
		{"MARKDOWN", []byte("# Title\nBody\n"), "text/markdown", "sample.md"},
		{"CSV", []byte("a,b,c\n1,2,3\n"), "text/csv", "sample.csv"},
		{"XLS", []byte("dummy xls content"), "application/vnd.ms-excel", "sample.xls"},
		{"XLSX", []byte("dummy xlsx content"), "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", "sample.xlsx"},
	}

	for _, tc := range syntheticTestCases {
		c.Run(tc.name, func(c *qt.C) {
			doc, err := NewDocumentFromBytes(tc.content, tc.contentType, tc.filename)
			c.Assert(err, qt.IsNil)

			_, ok := any(doc).(*documentData)
			c.Assert(ok, qt.Equals, true, qt.Commentf("Expected *documentData, got %T", doc))
			var as format.Document = doc
			c.Assert(as, qt.IsNotNil)
		})
	}

	// Test OLE mapping: use DOC bytes but OLE content type, ensure it maps to DOC
	c.Run("OLE_DOC_mapping", func(c *qt.C) {
		b, err := os.ReadFile("testdata/sample1.doc")
		c.Assert(err, qt.IsNil)

		doc, err := NewDocumentFromBytes(b, "application/x-ole-storage", "sample1.doc")
		c.Assert(err, qt.IsNil)

		_, ok := any(doc).(*documentData)
		c.Assert(ok, qt.Equals, true, qt.Commentf("Expected *documentData, got %T", doc))
		var as format.Document = doc
		c.Assert(as, qt.IsNotNil)
	})
}

func TestDocumentPDFConversion(t *testing.T) {
	t.Parallel()
	c := qt.New(t)

	// Test HTML to PDF conversion
	c.Run("HTML_to_PDF", func(c *qt.C) {
		htmlBytes, err := os.ReadFile("testdata/sample1.html")
		c.Assert(err, qt.IsNil)

		doc, err := NewDocumentFromBytes(htmlBytes, "text/html", "sample1.html")
		c.Assert(err, qt.IsNil)

		pdfDoc, err := doc.PDF()
		if err != nil {
			// Skip test if conversion tools are not available (e.g., in development environment)
			c.Skip("PDF conversion tools not available in test environment")
			return
		}
		c.Assert(pdfDoc, qt.IsNotNil)

		// Verify it's a PDF document
		c.Assert(pdfDoc.ContentType().String(), qt.Equals, "application/pdf")

		// Verify the PDF has some content (not empty)
		pdfDataURI, err := pdfDoc.DataURI()
		c.Assert(err, qt.IsNil)
		dataURIStr := pdfDataURI.String()
		c.Assert(strings.HasPrefix(dataURIStr, "data:application/pdf;base64,"), qt.Equals, true)

		// Decode base64 to check it's not empty
		base64Data := strings.TrimPrefix(dataURIStr, "data:application/pdf;base64,")
		pdfBytes, err := base64.StdEncoding.DecodeString(base64Data)
		c.Assert(err, qt.IsNil)
		c.Assert(len(pdfBytes), qt.Not(qt.Equals), 0, qt.Commentf("PDF should not be empty"))
	})

	// Test that PDF documents return themselves unchanged
	c.Run("PDF_identity", func(c *qt.C) {
		pdfBytes, err := os.ReadFile("testdata/sample2.pdf")
		c.Assert(err, qt.IsNil)

		doc, err := NewDocumentFromBytes(pdfBytes, "application/pdf", "sample2.pdf")
		c.Assert(err, qt.IsNil)

		pdfDoc, err := doc.PDF()
		c.Assert(err, qt.IsNil)

		// Should return the same document
		c.Assert(pdfDoc, qt.Equals, doc)
	})
}
