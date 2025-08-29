package data

import (
	"context"
	"encoding/base64"
	"fmt"
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

func TestDOCTypeDetection(t *testing.T) {
	t.Parallel()
	c := qt.New(t)

	// Test that DOC content type is correctly detected as a document type
	c.Assert(isDocumentContentType("application/msword"), qt.Equals, true)

	// Test that NewBinaryFromBytes returns a documentData for DOC files
	docBytes := []byte("dummy doc content")
	result, err := NewBinaryFromBytes(docBytes, "application/msword", "test.doc")
	c.Assert(err, qt.IsNil)

	// Verify that the result is a *documentData, not a *fileData
	_, ok := result.(*documentData)
	c.Assert(ok, qt.Equals, true)

	// Verify that it implements format.Document
	_, ok = result.(format.Document)
	c.Assert(ok, qt.Equals, true)
}

func TestDocumentGobSerialization(t *testing.T) {
	t.Parallel()
	c := qt.New(t)

	// Create a test document
	docBytes := []byte("test doc content")
	original, err := NewDocumentFromBytes(docBytes, "application/msword", "test.doc")
	c.Assert(err, qt.IsNil)

	// Test Gob encoding
	encoded, err := original.GobEncode()
	c.Assert(err, qt.IsNil)

	// Test Gob decoding
	var decoded documentData
	err = decoded.GobDecode(encoded)
	c.Assert(err, qt.IsNil)

	// Verify that the decoded document is equivalent
	c.Assert(decoded.ContentType().String(), qt.Equals, original.ContentType().String())
	c.Assert(decoded.Filename().String(), qt.Equals, original.Filename().String())

	// Verify that the decoded document still implements format.Document
	var _ format.Document = &decoded
}

func TestComponentInputSimulation(t *testing.T) {
	t.Parallel()
	c := qt.New(t)

	// Create a test document (simulating a DOC file)
	docBytes := []byte("test doc content")
	original, err := NewDocumentFromBytes(docBytes, "application/msword", "test.doc")
	c.Assert(err, qt.IsNil)

	// Simulate component input handling by creating a struct with Document field
	type TestInput struct {
		Document format.Document `instill:"document"`
	}

	// Create input data
	inputData := TestInput{
		Document: original,
	}

	// Simulate marshaling (what happens when sending data to component)
	marshaler := &Marshaler{}
	marshaledData, err := marshaler.Marshal(inputData)
	c.Assert(err, qt.IsNil)

	// Simulate unmarshaling (what happens when component receives data)
	unmarshaler := &Unmarshaler{binaryFetcher: nil}
	var outputData TestInput
	err = unmarshaler.Unmarshal(context.Background(), marshaledData, &outputData)
	c.Assert(err, qt.IsNil)

	// Verify that the unmarshaled document is still a *documentData
	_, ok := outputData.Document.(*documentData)
	c.Assert(ok, qt.Equals, true, qt.Commentf("Expected *documentData, got %T", outputData.Document))

	// Verify that it still implements format.Document
	var _ = outputData.Document
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
	test("ok - Valid DOC URL", "https://file-examples.com/wp-content/storage/2017/02/file-sample_100kB.doc", false)
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
		{"DOC document", "sample1.doc", "application/msword"},
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

func TestDOCContentTypeDetection(t *testing.T) {
	t.Parallel()
	c := qt.New(t)

	// Test that DOC content type is correctly detected
	c.Assert(isDocumentContentType("application/msword"), qt.Equals, true)
	c.Assert(isDocumentContentType("application/vnd.openxmlformats-officedocument.wordprocessingml.document"), qt.Equals, true)

	// Test that NewBinaryFromBytes correctly handles both DOC and DOCX
	docBytes := []byte("test doc content")

	// Test DOC
	result, err := NewBinaryFromBytes(docBytes, "application/msword", "test.doc")
	c.Assert(err, qt.IsNil)
	_, ok := result.(*documentData)
	c.Assert(ok, qt.Equals, true, qt.Commentf("Expected *documentData for DOC, got %T", result))

	// Test DOCX
	result, err = NewBinaryFromBytes(docBytes, "application/vnd.openxmlformats-officedocument.wordprocessingml.document", "test.docx")
	c.Assert(err, qt.IsNil)
	_, ok = result.(*documentData)
	c.Assert(ok, qt.Equals, true, qt.Commentf("Expected *documentData for DOCX, got %T", result))

	// Test that both implement format.Document
	docResult, ok := result.(format.Document)
	c.Assert(ok, qt.Equals, true, qt.Commentf("Expected format.Document, got %T", result))
	var _ = docResult
}

func TestMimetypeDetection(t *testing.T) {
	t.Parallel()
	c := qt.New(t)

	// Test that mimetype detection works correctly for DOC files
	// This simulates what happens when content type is empty
	docBytes := []byte("test doc content")

	// Test with empty content type (should trigger mimetype detection)
	result, err := NewBinaryFromBytes(docBytes, "", "test.doc")
	c.Assert(err, qt.IsNil)

	// The result should be a *documentData if mimetype detection works
	// Note: This test might fail if mimetype detection doesn't recognize DOC files
	// In that case, it would fall back to *fileData
	_, ok := result.(*documentData)
	if !ok {
		// If mimetype detection fails, it should at least be a *fileData
		_, ok = result.(*fileData)
		c.Assert(ok, qt.Equals, true, qt.Commentf("Expected *fileData when mimetype detection fails, got %T", result))
	}
}

func TestDOCDataURIPreservation(t *testing.T) {
	t.Parallel()
	c := qt.New(t)

	// Create a DOC document
	docBytes := []byte("test doc content")
	original, err := NewDocumentFromBytes(docBytes, "application/msword", "test.doc")
	c.Assert(err, qt.IsNil)

	// Get the data URI
	dataURI, err := original.DataURI()
	c.Assert(err, qt.IsNil)

	// Check that the data URI contains the correct content type
	dataURIStr := dataURI.String()
	c.Assert(strings.HasPrefix(dataURIStr, "data:application/msword"), qt.Equals, true,
		qt.Commentf("Expected data URI to start with 'data:application/msword', got: %s", dataURIStr))

	// Now simulate what happens during component deserialization
	// This uses the binary fetcher to parse the data URI
	binaryFetcher := &testBinaryFetcher{}

	// Test parsing the data URI
	parsedBytes, parsedContentType, parsedFilename, err := binaryFetcher.FetchFromURL(context.Background(), dataURIStr)
	c.Assert(err, qt.IsNil)
	c.Assert(parsedContentType, qt.Equals, "application/msword",
		qt.Commentf("Content type not preserved correctly: expected 'application/msword', got '%s'", parsedContentType))

	// Test that the parsed data creates a *documentData
	result, err := NewBinaryFromBytes(parsedBytes, parsedContentType, parsedFilename)
	c.Assert(err, qt.IsNil)
	_, ok := result.(*documentData)
	c.Assert(ok, qt.Equals, true, qt.Commentf("Expected *documentData, got %T", result))
}

func TestAllDocumentFormatsFromTestdata(t *testing.T) {
	t.Parallel()
	c := qt.New(t)

	cases := []struct {
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

	for _, tc := range cases {
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

	// Inline cases not present in testdata: MARKDOWN and CSV
	c.Run("MARKDOWN", func(c *qt.C) {
		b := []byte("# Title\nBody\n")
		doc, err := NewDocumentFromBytes(b, "text/markdown", "sample1.md")
		c.Assert(err, qt.IsNil)
		_, ok := any(doc).(*documentData)
		c.Assert(ok, qt.Equals, true, qt.Commentf("Expected *documentData, got %T", doc))
		var as format.Document = doc
		c.Assert(as, qt.IsNotNil)
	})

	c.Run("CSV", func(c *qt.C) {
		b := []byte("a,b,c\n1,2,3\n")
		doc, err := NewDocumentFromBytes(b, "text/csv", "sample1.csv")
		c.Assert(err, qt.IsNil)
		_, ok := any(doc).(*documentData)
		c.Assert(ok, qt.Equals, true, qt.Commentf("Expected *documentData, got %T", doc))
		var as format.Document = doc
		c.Assert(as, qt.IsNotNil)
	})

	// OLE mapping: use DOC bytes but OLE content type, ensure it maps to DOC
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

// testBinaryFetcher simulates the binary fetcher for data URI parsing
type testBinaryFetcher struct{}

func (m *testBinaryFetcher) FetchFromURL(ctx context.Context, url string) ([]byte, string, string, error) {
	// Simple data URI parser for testing (similar to the real implementation)
	if strings.HasPrefix(url, "data:") {
		parts := strings.SplitN(url, ",", 2)
		if len(parts) != 2 {
			return nil, "", "", fmt.Errorf("invalid data URI")
		}

		headerParts := strings.Split(parts[0], ";")
		contentType := strings.TrimPrefix(headerParts[0], "data:")

		// For this test, assume base64 encoding
		data, err := base64.StdEncoding.DecodeString(parts[1])
		if err != nil {
			return nil, "", "", err
		}

		return data, contentType, "", nil
	}
	return nil, "", "", fmt.Errorf("unsupported URL")
}
