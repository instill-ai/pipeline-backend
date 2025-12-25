package external

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// createTestServer creates a local HTTP server for testing filename extraction
func createTestServer() *httptest.Server {
	mux := http.NewServeMux()

	// Endpoint with attachment filename
	mux.HandleFunc("/file-attachment", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Disposition", `attachment; filename="test-document.pdf"`)
		w.Header().Set("Content-Type", "application/pdf")
		_, _ = w.Write([]byte("PDF content"))
	})

	// Endpoint with inline filename
	mux.HandleFunc("/file-inline", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Disposition", `inline; filename="image.png"`)
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte("PNG content"))
	})

	// Endpoint with filename* (RFC 5987)
	mux.HandleFunc("/file-encoded", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Disposition", `attachment; filename*=UTF-8''test%20file%20with%20spaces.txt`)
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("Text content with encoded filename"))
	})

	// Endpoint without Content-Disposition
	mux.HandleFunc("/file-no-disposition", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("Plain text content"))
	})

	// Endpoint with complex filename containing quotes and special chars
	mux.HandleFunc("/file-complex", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Disposition", `attachment; filename="report (final).xlsx"`)
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		_, _ = w.Write([]byte("Excel content"))
	})

	return httptest.NewServer(mux)
}

func TestNewBinaryFetcher(t *testing.T) {
	fetcher := NewBinaryFetcher()
	assert.NotNil(t, fetcher)
	assert.IsType(t, &binaryFetcher{}, fetcher)
}

func TestNewArtifactBinaryFetcher(t *testing.T) {
	// Test that we can create the fetcher (without mocking the complex interface)
	// This test just verifies the constructor doesn't panic with nil inputs
	fetcher := NewArtifactBinaryFetcher(nil, nil)
	assert.NotNil(t, fetcher)
	assert.IsType(t, &artifactBinaryFetcher{}, fetcher)
}

func TestBinaryFetcher_FetchFromURL_WithContentDisposition(t *testing.T) {
	server := createTestServer()
	defer server.Close()

	tests := []struct {
		name             string
		endpoint         string
		expectedFilename string
		expectedContent  string
	}{
		{
			name:             "attachment with filename",
			endpoint:         "/file-attachment",
			expectedFilename: "test-document.pdf",
			expectedContent:  "PDF content",
		},
		{
			name:             "inline with filename",
			endpoint:         "/file-inline",
			expectedFilename: "image.png",
			expectedContent:  "PNG content",
		},
		{
			name:             "encoded filename (RFC 5987)",
			endpoint:         "/file-encoded",
			expectedFilename: "test file with spaces.txt",
			expectedContent:  "Text content with encoded filename",
		},
		{
			name:             "no content disposition",
			endpoint:         "/file-no-disposition",
			expectedFilename: "",
			expectedContent:  "Plain text content",
		},
		{
			name:             "complex filename with special chars",
			endpoint:         "/file-complex",
			expectedFilename: "report (final).xlsx",
			expectedContent:  "Excel content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fetcher := NewBinaryFetcher()
			body, contentType, filename, err := fetcher.FetchFromURL(context.Background(), server.URL+tt.endpoint)

			assert.NoError(t, err)
			assert.Equal(t, []byte(tt.expectedContent), body)
			assert.Equal(t, "text/plain", contentType) // mimetype detection
			assert.Equal(t, tt.expectedFilename, filename)
		})
	}
}

func TestBinaryFetcher_FetchFromURL_DataURI(t *testing.T) {
	tests := []struct {
		name             string
		dataURI          string
		expectedContent  string
		expectedType     string
		expectedFilename string
		expectError      bool
	}{
		{
			name:             "simple base64",
			dataURI:          "data:text/plain;base64,SGVsbG8gV29ybGQ=",
			expectedContent:  "Hello World",
			expectedType:     "text/plain",
			expectedFilename: "",
			expectError:      false,
		},
		{
			name:             "with filename parameter",
			dataURI:          "data:text/plain;filename=test.txt;base64,SGVsbG8gV29ybGQ=",
			expectedContent:  "Hello World",
			expectedType:     "text/plain",
			expectedFilename: "test.txt",
			expectError:      false,
		},
		{
			name:             "with encoded filename",
			dataURI:          "data:text/plain;filename=test%20file.txt;base64,SGVsbG8gV29ybGQ=",
			expectedContent:  "Hello World",
			expectedType:     "text/plain",
			expectedFilename: "test file.txt",
			expectError:      false,
		},
		{
			name:        "invalid format",
			dataURI:     "invalid:data:uri",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fetcher := NewBinaryFetcher()
			body, contentType, filename, err := fetcher.FetchFromURL(context.Background(), tt.dataURI)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedContent, string(body))
			assert.Equal(t, tt.expectedType, contentType)
			assert.Equal(t, tt.expectedFilename, filename)
		})
	}
}

func TestBinaryFetcher_ConvertDataURIToBytes(t *testing.T) {
	fetcher := &binaryFetcher{}

	tests := []struct {
		name             string
		dataURI          string
		expectedContent  string
		expectedType     string
		expectedFilename string
		expectError      bool
	}{
		{
			name:             "basic data URI",
			dataURI:          "data:text/plain;base64,SGVsbG8gV29ybGQ=",
			expectedContent:  "Hello World",
			expectedType:     "text/plain",
			expectedFilename: "",
			expectError:      false,
		},
		{
			name:             "with filename",
			dataURI:          "data:image/png;filename=image.png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8/5+hHgAHggJ/PchI7wAAAABJRU5ErkJggg==",
			expectedType:     "image/png",
			expectedFilename: "image.png",
			expectError:      false,
		},
		{
			name:        "invalid format - no data prefix",
			dataURI:     "text/plain;base64,SGVsbG8gV29ybGQ=",
			expectError: true,
		},
		{
			name:        "invalid base64",
			dataURI:     "data:text/plain;base64,invalid_base64!@#$",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, contentType, filename, err := fetcher.convertDataURIToBytes(tt.dataURI)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			if tt.expectedContent != "" {
				assert.Equal(t, tt.expectedContent, string(body))
			}
			assert.Equal(t, tt.expectedType, contentType)
			assert.Equal(t, tt.expectedFilename, filename)
		})
	}
}

func TestArtifactBinaryFetcher_FetchFromURL_DataURI(t *testing.T) {
	fetcher := NewArtifactBinaryFetcher(nil, nil)

	dataURI := "data:text/plain;base64,SGVsbG8gV29ybGQ="
	body, contentType, filename, err := fetcher.FetchFromURL(context.Background(), dataURI)

	assert.NoError(t, err)
	assert.Equal(t, "Hello World", string(body))
	assert.Equal(t, "text/plain", contentType)
	assert.Equal(t, "", filename)
}

func TestArtifactBinaryFetcher_FetchFromURL_PresignedURL(t *testing.T) {
	server := createTestServer()
	defer server.Close()

	fetcher := NewArtifactBinaryFetcher(nil, nil)

	// Use our test server's attachment endpoint as the presigned URL
	presignedURL := server.URL + "/file-attachment"
	encodedTestURL := base64.URLEncoding.EncodeToString([]byte(presignedURL))
	testURL := fmt.Sprintf("https://example.com/v1alpha/blob-urls/%s", encodedTestURL)

	body, contentType, filename, err := fetcher.FetchFromURL(context.Background(), testURL)

	assert.NoError(t, err)
	assert.Equal(t, []byte("PDF content"), body)
	assert.Equal(t, "text/plain", contentType)
	assert.Equal(t, "test-document.pdf", filename)
}

func TestArtifactBinaryFetcher_FetchFromURL_RegularURL(t *testing.T) {
	server := createTestServer()
	defer server.Close()

	fetcher := NewArtifactBinaryFetcher(nil, nil)

	body, contentType, filename, err := fetcher.FetchFromURL(context.Background(), server.URL+"/file-complex")

	assert.NoError(t, err)
	assert.Equal(t, []byte("Excel content"), body)
	assert.Equal(t, "text/plain", contentType)
	assert.Equal(t, "report (final).xlsx", filename)
}

func TestMinioURLPatterns(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		shouldMatch bool
		expectedUID string
	}{
		{
			name:        "presigned pattern match",
			url:         "https://example.com/v1alpha/blob-urls/aHR0cHM6Ly9leGFtcGxlLmNvbS9maWxl",
			shouldMatch: true,
			expectedUID: "aHR0cHM6Ly9leGFtcGxlLmNvbS9maWxl",
		},
		{
			name:        "presigned pattern with different domain",
			url:         "http://localhost:8080/v1alpha/blob-urls/dGVzdC1wcmVzaWduZWQtdXJs",
			shouldMatch: true,
			expectedUID: "dGVzdC1wcmVzaWduZWQtdXJs",
		},
		{
			name:        "no match - different path",
			url:         "https://example.com/some/other/path",
			shouldMatch: false,
		},
		{
			name:        "no match - old deprecated format not supported",
			url:         "https://example.com/v1alpha/namespaces/test/blob-urls/123e4567-e89b-12d3-a456-426614174000",
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := minioURLPresignedPattern.FindStringSubmatch(tt.url)

			if tt.shouldMatch {
				assert.NotNil(t, matches)
				if len(matches) > 1 {
					assert.Equal(t, tt.expectedUID, matches[1])
				}
			} else {
				assert.Nil(t, matches)
			}
		})
	}
}

func TestBinaryFetcher_FetchFromURL_ErrorHandling(t *testing.T) {
	fetcher := NewBinaryFetcher()

	// Test with invalid URL
	_, _, _, err := fetcher.FetchFromURL(context.Background(), "invalid-url")
	assert.Error(t, err)

	// Test with unreachable URL
	_, _, _, err = fetcher.FetchFromURL(context.Background(), "http://localhost:99999/nonexistent")
	assert.Error(t, err)
}

func TestArtifactBinaryFetcher_RegularURL_Fallback(t *testing.T) {
	server := createTestServer()
	defer server.Close()

	fetcher := NewArtifactBinaryFetcher(nil, nil)

	// Test regular URL fallback (should work with nil clients since it uses binaryFetcher)
	body, contentType, filename, err := fetcher.FetchFromURL(context.Background(), server.URL+"/file-inline")
	assert.NoError(t, err)
	assert.Equal(t, []byte("PNG content"), body)
	assert.Equal(t, "text/plain", contentType)
	assert.Equal(t, "image.png", filename)
}
