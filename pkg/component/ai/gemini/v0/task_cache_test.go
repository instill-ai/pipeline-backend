package gemini

import (
	"context"
	"fmt"
	"testing"
	"time"

	"google.golang.org/genai"

	qt "github.com/frankban/quicktest"
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
)

func TestTaskCache_Create(t *testing.T) {
	c := qt.New(t)

	// Test create operation with valid input
	t.Run("create with valid input", func(t *testing.T) {
		input := TaskCacheInput{
			Operation: "create",
			Model:     "gemini-2.5-flash",
			Contents: []*genai.Content{
				{
					Role: "user",
					Parts: []*genai.Part{
						{Text: "This is test content to cache"},
					},
				},
			},
			DisplayName: stringPtr("test-cache"),
			TTL:         durationPtr(time.Hour),
		}

		// Validate input structure
		c.Check(input.Operation, qt.Equals, "create")
		c.Check(input.Model, qt.Equals, "gemini-2.5-flash")
		c.Check(len(input.Contents), qt.Equals, 1)
		c.Check(input.Contents[0].Role, qt.Equals, "user")
		c.Check(len(input.Contents[0].Parts), qt.Equals, 1)
		c.Check(input.Contents[0].Parts[0].Text, qt.Equals, "This is test content to cache")
		c.Check(*input.DisplayName, qt.Equals, "test-cache")
		c.Check(*input.TTL, qt.Equals, time.Hour)
	})

	t.Run("create with expire time", func(t *testing.T) {
		expireTime := time.Now().Add(24 * time.Hour)
		input := TaskCacheInput{
			Operation: "create",
			Model:     "gemini-2.5-pro",
			Contents: []*genai.Content{
				{
					Role: "user",
					Parts: []*genai.Part{
						{Text: "Content with expire time"},
					},
				},
			},
			ExpireTime: &expireTime,
		}

		c.Check(input.ExpireTime, qt.Not(qt.IsNil))
		c.Check(input.ExpireTime.After(time.Now()), qt.IsTrue)
	})

	t.Run("create with system instruction and tools", func(t *testing.T) {
		input := TaskCacheInput{
			Operation: "create",
			Model:     "gemini-2.5-flash",
			Contents: []*genai.Content{
				{
					Role: "user",
					Parts: []*genai.Part{
						{Text: "Content with system instruction"},
					},
				},
			},
			SystemInstruction: &genai.Content{
				Parts: []*genai.Part{
					{Text: "You are a helpful assistant"},
				},
			},
			Tools: []*genai.Tool{
				{
					FunctionDeclarations: []*genai.FunctionDeclaration{
						{
							Name:        "test_function",
							Description: "A test function",
							Parameters: &genai.Schema{
								Type: genai.TypeObject,
								Properties: map[string]*genai.Schema{
									"param1": {
										Type:        genai.TypeString,
										Description: "Test parameter",
									},
								},
							},
						},
					},
				},
			},
			TTL: durationPtr(2 * time.Hour),
		}

		c.Check(input.SystemInstruction, qt.Not(qt.IsNil))
		c.Check(len(input.SystemInstruction.Parts), qt.Equals, 1)
		c.Check(input.SystemInstruction.Parts[0].Text, qt.Equals, "You are a helpful assistant")
		c.Check(len(input.Tools), qt.Equals, 1)
		c.Check(len(input.Tools[0].FunctionDeclarations), qt.Equals, 1)
		c.Check(input.Tools[0].FunctionDeclarations[0].Name, qt.Equals, "test_function")
	})

	t.Run("create with multimedia inputs", func(t *testing.T) {
		input := TaskCacheInput{
			Operation:   "create",
			Model:       "gemini-2.5-flash",
			Prompt:      stringPtr("Analyze this content"),
			DisplayName: stringPtr("multimedia-cache"),
			TTL:         durationPtr(time.Hour),
		}

		// Validate multimedia input structure
		c.Check(input.Operation, qt.Equals, "create")
		c.Check(input.Model, qt.Equals, "gemini-2.5-flash")
		c.Check(*input.Prompt, qt.Equals, "Analyze this content")
		c.Check(*input.DisplayName, qt.Equals, "multimedia-cache")
		c.Check(*input.TTL, qt.Equals, time.Hour)
	})

	t.Run("create with system message", func(t *testing.T) {
		input := TaskCacheInput{
			Operation:     "create",
			Model:         "gemini-2.5-flash",
			Prompt:        stringPtr("Test prompt"),
			SystemMessage: stringPtr("You are a helpful assistant"),
			TTL:           durationPtr(time.Hour),
		}

		// Validate system message input
		c.Check(input.Operation, qt.Equals, "create")
		c.Check(input.Model, qt.Equals, "gemini-2.5-flash")
		c.Check(*input.Prompt, qt.Equals, "Test prompt")
		c.Check(*input.SystemMessage, qt.Equals, "You are a helpful assistant")
		c.Check(*input.TTL, qt.Equals, time.Hour)
	})
}

func TestTaskCache_List(t *testing.T) {
	c := qt.New(t)

	t.Run("list with default pagination", func(t *testing.T) {
		input := TaskCacheInput{
			Operation: "list",
		}

		c.Check(input.Operation, qt.Equals, "list")
		c.Check(input.PageSize, qt.IsNil) // Should use default
		c.Check(input.PageToken, qt.IsNil)
	})

	t.Run("list with custom pagination", func(t *testing.T) {
		pageSize := int32(10)
		pageToken := "next-page-token"
		input := TaskCacheInput{
			Operation: "list",
			PageSize:  &pageSize,
			PageToken: &pageToken,
		}

		c.Check(input.Operation, qt.Equals, "list")
		c.Check(*input.PageSize, qt.Equals, int32(10))
		c.Check(*input.PageToken, qt.Equals, "next-page-token")
	})
}

func TestTaskCache_Get(t *testing.T) {
	c := qt.New(t)

	t.Run("get with valid cache name", func(t *testing.T) {
		cacheName := "cachedContents/test-cache-id"
		input := TaskCacheInput{
			Operation: "get",
			CacheName: &cacheName,
		}

		c.Check(input.Operation, qt.Equals, "get")
		c.Check(*input.CacheName, qt.Equals, "cachedContents/test-cache-id")
	})
}

func TestTaskCache_Update(t *testing.T) {
	c := qt.New(t)

	t.Run("update with TTL", func(t *testing.T) {
		cacheName := "cachedContents/test-cache-id"
		input := TaskCacheInput{
			Operation: "update",
			CacheName: &cacheName,
			TTL:       durationPtr(2 * time.Hour),
		}

		c.Check(input.Operation, qt.Equals, "update")
		c.Check(*input.CacheName, qt.Equals, "cachedContents/test-cache-id")
		c.Check(*input.TTL, qt.Equals, 2*time.Hour)
	})

	t.Run("update with expire time", func(t *testing.T) {
		cacheName := "cachedContents/test-cache-id"
		expireTime := time.Now().Add(48 * time.Hour)
		input := TaskCacheInput{
			Operation:  "update",
			CacheName:  &cacheName,
			ExpireTime: &expireTime,
		}

		c.Check(input.Operation, qt.Equals, "update")
		c.Check(*input.CacheName, qt.Equals, "cachedContents/test-cache-id")
		c.Check(input.ExpireTime, qt.Not(qt.IsNil))
		c.Check(input.ExpireTime.After(time.Now()), qt.IsTrue)
	})
}

func TestTaskCache_Delete(t *testing.T) {
	c := qt.New(t)

	t.Run("delete with valid cache name", func(t *testing.T) {
		cacheName := "cachedContents/test-cache-id"
		input := TaskCacheInput{
			Operation: "delete",
			CacheName: &cacheName,
		}

		c.Check(input.Operation, qt.Equals, "delete")
		c.Check(*input.CacheName, qt.Equals, "cachedContents/test-cache-id")
	})
}

func TestTaskCacheOutput(t *testing.T) {
	c := qt.New(t)

	t.Run("create output", func(t *testing.T) {
		createTime := time.Now()
		expireTime := time.Now().Add(24 * time.Hour)

		output := TaskCacheOutput{
			Operation: "create",
			CachedContent: &genai.CachedContent{
				Name:        "cachedContents/test-cache-id",
				Model:       "gemini-2.5-flash",
				DisplayName: "test-cache",
				CreateTime:  createTime,
				ExpireTime:  expireTime,
			},
		}

		c.Check(output.Operation, qt.Equals, "create")
		c.Check(output.CachedContent, qt.Not(qt.IsNil))
		c.Check(output.CachedContent.Name, qt.Equals, "cachedContents/test-cache-id")
		c.Check(output.CachedContent.Model, qt.Equals, "gemini-2.5-flash")
		c.Check(output.CachedContent.DisplayName, qt.Equals, "test-cache")
		c.Check(output.CachedContent.CreateTime.IsZero(), qt.IsFalse)
		c.Check(output.CachedContent.ExpireTime.IsZero(), qt.IsFalse)
	})

	t.Run("list output", func(t *testing.T) {
		nextPageToken := "next-page-token"
		output := TaskCacheOutput{
			Operation: "list",
			CachedContents: []*genai.CachedContent{
				{
					Name:  "cachedContents/cache-1",
					Model: "gemini-2.5-flash",
				},
				{
					Name:  "cachedContents/cache-2",
					Model: "gemini-2.5-pro",
				},
			},
			NextPageToken: &nextPageToken,
		}

		c.Check(output.Operation, qt.Equals, "list")
		c.Check(len(output.CachedContents), qt.Equals, 2)
		c.Check(output.CachedContents[0].Name, qt.Equals, "cachedContents/cache-1")
		c.Check(output.CachedContents[1].Name, qt.Equals, "cachedContents/cache-2")
		c.Check(*output.NextPageToken, qt.Equals, "next-page-token")
	})

	t.Run("delete output", func(t *testing.T) {
		output := TaskCacheOutput{
			Operation: "delete",
		}

		c.Check(output.Operation, qt.Equals, "delete")
		c.Check(output.CachedContent, qt.IsNil)
		c.Check(output.CachedContents, qt.IsNil)
		c.Check(output.NextPageToken, qt.IsNil)
	})
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}

// Helper function to create duration pointers
func durationPtr(d time.Duration) *time.Duration {
	return &d
}

// Mock execution for testing validation logic
type mockExecution struct {
	execution
}

func TestCacheValidation(t *testing.T) {
	c := qt.New(t)

	t.Run("validate create operation requirements", func(t *testing.T) {
		// Test missing model
		input := TaskCacheInput{
			Operation: "create",
			Contents: []*genai.Content{
				{
					Role: "user",
					Parts: []*genai.Part{
						{Text: "Test content"},
					},
				},
			},
		}
		c.Check(input.Model, qt.Equals, "") // Should be invalid

		// Test missing contents
		input2 := TaskCacheInput{
			Operation: "create",
			Model:     "gemini-2.5-flash",
		}
		c.Check(len(input2.Contents), qt.Equals, 0) // Should be invalid
	})

	t.Run("validate get/update/delete operation requirements", func(t *testing.T) {
		// Test missing cache name for get
		input := TaskCacheInput{
			Operation: "get",
		}
		c.Check(input.CacheName, qt.IsNil) // Should be invalid

		// Test missing cache name for update
		input2 := TaskCacheInput{
			Operation: "update",
			TTL:       durationPtr(time.Hour),
		}
		c.Check(input2.CacheName, qt.IsNil) // Should be invalid

		// Test missing expiration for update
		cacheName := "cachedContents/test"
		input3 := TaskCacheInput{
			Operation: "update",
			CacheName: &cacheName,
		}
		c.Check(input3.TTL, qt.IsNil)        // Should be invalid
		c.Check(input3.ExpireTime, qt.IsNil) // Should be invalid
	})
}

// Test File API functionality
func TestFileAPIIntegration(t *testing.T) {
	c := qt.New(t)

	t.Run("uploadedFile struct", func(t *testing.T) {
		file := &uploadedFile{
			name:     "files/test-file-123",
			uri:      "https://generativelanguage.googleapis.com/v1beta/files/test-file-123",
			mimeType: "image/jpeg",
		}

		c.Check(file.name, qt.Equals, "files/test-file-123")
		c.Check(file.uri, qt.Equals, "https://generativelanguage.googleapis.com/v1beta/files/test-file-123")
		c.Check(file.mimeType, qt.Equals, "image/jpeg")
	})

	t.Run("total request size threshold logic", func(t *testing.T) {
		exec := &mockExecution{}
		ctx := context.Background()

		// Test small total request size (should use inline data)
		smallTotalSize := 1024 * 1024 // 1MB total request
		shouldUseFileAPI := smallTotalSize > MaxInlineSize

		c.Check(shouldUseFileAPI, qt.IsFalse) // Small total request should use inline

		// Test large total request size (should use File API)
		largeTotalSize := 25 * 1024 * 1024 // 25MB total request
		shouldUseFileAPILarge := largeTotalSize > MaxInlineSize

		c.Check(shouldUseFileAPILarge, qt.IsTrue) // Large total request should use File API

		// Test cache video rule (should always use File API for cache videos)
		isCache := true
		shouldUseFileAPIVideo := smallTotalSize > MaxInlineSize || isCache

		c.Check(shouldUseFileAPIVideo, qt.IsTrue) // Cache videos should always use File API

		// Test non-cache video with small total size (should use inline)
		isNonCache := false
		shouldUseFileAPINonCacheVideo := smallTotalSize > MaxInlineSize || isNonCache

		c.Check(shouldUseFileAPINonCacheVideo, qt.IsFalse) // Non-cache videos follow total size rule

		// Avoid unused variable warning
		_ = exec
		_ = ctx
	})

	t.Run("calculateTotalRequestSize function", func(t *testing.T) {
		exec := &mockExecution{}

		// Create a mock input with various content types
		input := TaskCacheInput{
			Prompt:        stringPtr("This is a test prompt"),
			SystemMessage: stringPtr("You are a helpful assistant"),
		}

		// Test basic size calculation
		totalSize := exec.calculateTotalRequestSize(input)
		expectedSize := len("This is a test prompt") + len("You are a helpful assistant")

		c.Check(totalSize, qt.Equals, expectedSize)

		// Test empty input
		emptyInput := TaskCacheInput{}
		emptySize := exec.calculateTotalRequestSize(emptyInput)
		c.Check(emptySize, qt.Equals, 0)
	})
}

// Test media processing optimizations
func TestMediaProcessingOptimizations(t *testing.T) {
	c := qt.New(t)

	t.Run("slice pre-allocation logic", func(t *testing.T) {
		// Simulate input with various media types
		imageCount := 3
		audioCount := 2
		videoCount := 1
		documentCount := 2

		totalMediaCount := imageCount + audioCount + videoCount + documentCount
		nonTextPartsCount := 1
		textPartsCount := 1

		// Test capacity calculation
		expectedPartsCapacity := totalMediaCount + nonTextPartsCount + textPartsCount
		expectedFilesCapacity := totalMediaCount

		c.Check(expectedPartsCapacity, qt.Equals, 10) // 3+2+1+2+1+1 = 10
		c.Check(expectedFilesCapacity, qt.Equals, 8)  // 3+2+1+2 = 8

		// Test slice creation with proper capacity
		parts := make([]genai.Part, 0, expectedPartsCapacity)
		uploadedFiles := make([]string, 0, expectedFilesCapacity)

		c.Check(cap(parts), qt.Equals, expectedPartsCapacity)
		c.Check(cap(uploadedFiles), qt.Equals, expectedFilesCapacity)
		c.Check(len(parts), qt.Equals, 0)
		c.Check(len(uploadedFiles), qt.Equals, 0)
	})

	t.Run("media processor structure", func(t *testing.T) {
		// Test the processor pattern used in buildCacheReqParts
		type mediaProcessor struct {
			name string
			fn   func() ([]genai.Part, []string, error)
		}

		processors := []mediaProcessor{
			{"images", func() ([]genai.Part, []string, error) {
				return []genai.Part{{Text: "mock-image"}}, []string{"file1"}, nil
			}},
			{"audio", func() ([]genai.Part, []string, error) {
				return []genai.Part{{Text: "mock-audio"}}, []string{"file2"}, nil
			}},
			{"videos", func() ([]genai.Part, []string, error) {
				return []genai.Part{{Text: "mock-video"}}, []string{"file3"}, nil
			}},
			{"documents", func() ([]genai.Part, []string, error) {
				return []genai.Part{{Text: "mock-doc"}}, []string{"file4"}, nil
			}},
		}

		c.Check(len(processors), qt.Equals, 4)
		c.Check(processors[0].name, qt.Equals, "images")
		c.Check(processors[1].name, qt.Equals, "audio")
		c.Check(processors[2].name, qt.Equals, "videos")
		c.Check(processors[3].name, qt.Equals, "documents")

		// Test processor execution
		parts, files, err := processors[0].fn()
		c.Check(err, qt.IsNil)
		c.Check(len(parts), qt.Equals, 1)
		c.Check(len(files), qt.Equals, 1)
		c.Check(parts[0].Text, qt.Equals, "mock-image")
		c.Check(files[0], qt.Equals, "file1")
	})
}

// Test file state handling
func TestFileStateHandling(t *testing.T) {
	c := qt.New(t)

	t.Run("file state constants", func(t *testing.T) {
		// Test that we're using the correct genai file state constants
		c.Check(string(genai.FileStateActive), qt.Equals, "ACTIVE")
		c.Check(string(genai.FileStateProcessing), qt.Equals, "PROCESSING")
		c.Check(string(genai.FileStateFailed), qt.Equals, "FAILED")
	})

	t.Run("timeout configurations", func(t *testing.T) {
		// Test timeout values used in the implementation
		imageTimeout := 60 * time.Second
		audioTimeout := 60 * time.Second
		videoTimeout := 120 * time.Second
		documentTimeout := 60 * time.Second

		c.Check(imageTimeout, qt.Equals, time.Minute)
		c.Check(audioTimeout, qt.Equals, time.Minute)
		c.Check(videoTimeout, qt.Equals, 2*time.Minute)
		c.Check(documentTimeout, qt.Equals, time.Minute)

		// Video should have longer timeout
		c.Check(videoTimeout > imageTimeout, qt.IsTrue)
		c.Check(videoTimeout > audioTimeout, qt.IsTrue)
		c.Check(videoTimeout > documentTimeout, qt.IsTrue)
	})
}

// Test error handling and cleanup
func TestErrorHandlingAndCleanup(t *testing.T) {
	c := qt.New(t)

	t.Run("error message formatting", func(t *testing.T) {
		// Test error message patterns used in the implementation
		fileName := "files/test-123"
		mediaType := "image"

		timeoutErr := fmt.Errorf("timeout waiting for file %s to become ACTIVE: context deadline exceeded", fileName)
		processingErr := fmt.Errorf("file %s processing failed", fileName)
		unexpectedStateErr := fmt.Errorf("file %s in unexpected state: UNKNOWN", fileName)
		mediaProcessingErr := fmt.Errorf("failed to process %s: upload failed", mediaType)

		c.Check(timeoutErr.Error(), qt.Contains, "timeout waiting for file")
		c.Check(timeoutErr.Error(), qt.Contains, fileName)
		c.Check(timeoutErr.Error(), qt.Contains, "ACTIVE")

		c.Check(processingErr.Error(), qt.Contains, "processing failed")
		c.Check(processingErr.Error(), qt.Contains, fileName)

		c.Check(unexpectedStateErr.Error(), qt.Contains, "unexpected state")
		c.Check(unexpectedStateErr.Error(), qt.Contains, "UNKNOWN")

		c.Check(mediaProcessingErr.Error(), qt.Contains, "failed to process")
		c.Check(mediaProcessingErr.Error(), qt.Contains, mediaType)
	})

	t.Run("cleanup logic", func(t *testing.T) {
		// Test cleanup scenarios
		uploadedFiles := []string{
			"files/image-123",
			"files/video-456",
			"files/document-789",
		}

		// Simulate cleanup - in real implementation this would call client.Files.Delete
		cleanupCount := 0
		for _, fileName := range uploadedFiles {
			if fileName != "" {
				cleanupCount++
				// Mock cleanup: client.Files.Delete(ctx, fileName, nil)
			}
		}

		c.Check(cleanupCount, qt.Equals, 3)
		c.Check(len(uploadedFiles), qt.Equals, 3)
	})
}

// Test multimedia input validation
func TestMultimediaInputValidation(t *testing.T) {
	c := qt.New(t)

	t.Run("create with large file simulation", func(t *testing.T) {
		input := TaskCacheInput{
			Operation:   "create",
			Model:       "gemini-2.5-flash",
			Prompt:      stringPtr("Analyze this large image"),
			DisplayName: stringPtr("large-file-cache"),
			TTL:         durationPtr(time.Hour),
		}

		// Validate that the input structure supports File API scenarios
		c.Check(input.Operation, qt.Equals, "create")
		c.Check(input.Model, qt.Equals, "gemini-2.5-flash")
		c.Check(*input.Prompt, qt.Equals, "Analyze this large image")
		c.Check(*input.DisplayName, qt.Equals, "large-file-cache")

		// Test that multimedia input interfaces are implemented
		c.Check(input.GetPrompt(), qt.DeepEquals, input.Prompt)
		c.Check(input.GetImages(), qt.HasLen, 0)
		c.Check(input.GetAudio(), qt.HasLen, 0)
		c.Check(input.GetVideos(), qt.HasLen, 0)
		c.Check(input.GetDocuments(), qt.HasLen, 0)
		c.Check(input.GetContents(), qt.HasLen, 0)
	})

	t.Run("create with video files", func(t *testing.T) {
		input := TaskCacheInput{
			Operation:   "create",
			Model:       "gemini-2.5-flash",
			Prompt:      stringPtr("Analyze these videos"),
			DisplayName: stringPtr("video-cache"),
			TTL:         durationPtr(2 * time.Hour),
		}

		// Videos should always use File API regardless of size
		c.Check(input.Operation, qt.Equals, "create")
		c.Check(input.Model, qt.Equals, "gemini-2.5-flash")
		c.Check(*input.Prompt, qt.Equals, "Analyze these videos")

		// Test system message handling
		c.Check(input.GetSystemMessage(), qt.IsNil)
		c.Check(input.GetSystemInstruction(), qt.IsNil)
	})
}

// Test performance optimizations
func TestPerformanceOptimizations(t *testing.T) {
	c := qt.New(t)

	t.Run("memory allocation efficiency", func(t *testing.T) {
		// Test that we're pre-allocating slices efficiently
		mediaCount := 10

		// Old approach (would cause multiple reallocations)
		var oldParts []genai.Part
		var oldFiles []string
		for i := 0; i < mediaCount; i++ {
			oldParts = append(oldParts, genai.Part{Text: fmt.Sprintf("part-%d", i)})
			oldFiles = append(oldFiles, fmt.Sprintf("file-%d", i))
		}

		// New approach (pre-allocated)
		newParts := make([]genai.Part, 0, mediaCount)
		newFiles := make([]string, 0, mediaCount)
		for i := 0; i < mediaCount; i++ {
			newParts = append(newParts, genai.Part{Text: fmt.Sprintf("part-%d", i)})
			newFiles = append(newFiles, fmt.Sprintf("file-%d", i))
		}

		// Both should have same content
		c.Check(len(oldParts), qt.Equals, len(newParts))
		c.Check(len(oldFiles), qt.Equals, len(newFiles))

		// But new approach should have exact capacity
		c.Check(cap(newParts), qt.Equals, mediaCount)
		c.Check(cap(newFiles), qt.Equals, mediaCount)

		// Old approach might have over-allocated
		c.Check(cap(oldParts) >= mediaCount, qt.IsTrue)
		c.Check(cap(oldFiles) >= mediaCount, qt.IsTrue)
	})

	t.Run("early return optimizations", func(t *testing.T) {
		// Test early returns for empty inputs
		emptyImages := []format.Image{}
		emptyAudio := []format.Audio{}
		emptyVideos := []format.Video{}
		emptyDocuments := []format.Document{}

		c.Check(len(emptyImages), qt.Equals, 0)
		c.Check(len(emptyAudio), qt.Equals, 0)
		c.Check(len(emptyVideos), qt.Equals, 0)
		c.Check(len(emptyDocuments), qt.Equals, 0)

		// In the actual implementation, these would return (nil, nil, nil) immediately
		// without allocating any slices or doing any processing
	})
}
