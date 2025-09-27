package gemini

import (
	"context"
	"fmt"

	"google.golang.org/genai"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func (e *execution) cache(ctx context.Context, job *base.Job) error {
	// Read input
	in := TaskCacheInput{}
	if err := job.Input.ReadData(ctx, &in); err != nil {
		job.Error.Error(ctx, err)
		return nil
	}

	// Check if operation is specified
	if in.Operation == "" {
		err := fmt.Errorf("caching operation is not specified")
		job.Error.Error(ctx, err)
		return nil
	}

	// Create Gemini client
	client, err := e.createGeminiClient(ctx)
	if err != nil {
		job.Error.Error(ctx, err)
		return nil
	}

	// Execute the appropriate cache operation
	switch in.Operation {
	case "create":
		return e.handleCreateCache(ctx, job, client, &in)
	case "list":
		return e.handleListCache(ctx, job, client, &in)
	case "get":
		return e.handleGetCache(ctx, job, client, &in)
	case "update":
		return e.handleUpdateCache(ctx, job, client, &in)
	case "delete":
		return e.handleDeleteCache(ctx, job, client, &in)
	default:
		err := fmt.Errorf("unsupported cache operation: %s", in.Operation)
		job.Error.Error(ctx, err)
		return nil
	}
}

// buildCacheRequestContents builds the complete cache contents from input using File API for large files and videos
func (e *execution) buildCacheRequestContents(ctx context.Context, client *genai.Client, in TaskCacheInput) ([]*genai.Content, []string, error) {
	// If Contents are provided directly, use them
	if len(in.Contents) > 0 {
		return in.Contents, nil, nil
	}

	// Build content parts from multimedia inputs, using File API based on total request size and cache video rule
	inParts, uploadedFileNames, err := e.buildReqPartsWithFileAPI(ctx, client, in, true) // isCache = true
	if err != nil {
		return nil, nil, err
	}
	if len(inParts) == 0 {
		return nil, nil, nil
	}

	// Convert parts to content
	partsPtrs := make([]*genai.Part, 0, len(inParts))
	for i := range inParts {
		p := inParts[i]
		partsPtrs = append(partsPtrs, &p)
	}
	contents := []*genai.Content{{Role: genai.RoleUser, Parts: partsPtrs}}

	return contents, uploadedFileNames, nil
}

// handleCreateCache creates a new cached content
func (e *execution) handleCreateCache(ctx context.Context, job *base.Job, client *genai.Client, in *TaskCacheInput) error {
	// Validate required fields for create operation
	if in.Model == "" {
		err := fmt.Errorf("model is required for create operation")
		job.Error.Error(ctx, err)
		return nil
	}

	// Build contents from multimedia inputs or direct Contents field, using File API for large files and videos
	contents, uploadedFileNames, err := e.buildCacheRequestContents(ctx, client, *in)
	if err != nil {
		job.Error.Error(ctx, err)
		return nil
	}
	if len(contents) == 0 {
		err := fmt.Errorf("contents or multimedia inputs (prompt, images, audio, videos, documents) are required for create operation")
		job.Error.Error(ctx, err)
		return nil
	}

	// Ensure uploaded files are cleaned up after cache creation
	defer func() {
		for _, fileName := range uploadedFileNames {
			if _, deleteErr := client.Files.Delete(ctx, fileName, nil); deleteErr != nil {
				// Log the error but don't fail the operation
				// The files will be automatically deleted after 48 hours anyway
				fmt.Printf("Warning: failed to delete uploaded file %s: %v\n", fileName, deleteErr)
			}
		}
	}()

	// Build create config
	config := &genai.CreateCachedContentConfig{}

	// Set expiration
	if in.TTL != nil {
		config.TTL = *in.TTL
	}
	if in.ExpireTime != nil {
		config.ExpireTime = *in.ExpireTime
	}

	// Set optional fields
	if in.DisplayName != nil {
		config.DisplayName = *in.DisplayName
	}

	// Handle system instruction - prioritize system-message over system-instruction
	systemMessage := extractSystemMessage(*in)
	if systemMessage != "" {
		config.SystemInstruction = &genai.Content{Parts: []*genai.Part{{Text: systemMessage}}}
	} else if in.SystemInstruction != nil {
		config.SystemInstruction = in.SystemInstruction
	}

	if len(in.Tools) > 0 {
		config.Tools = in.Tools
	}

	if in.ToolConfig != nil {
		config.ToolConfig = in.ToolConfig
	}

	// Set contents
	config.Contents = contents

	// Create cached content
	result, err := client.Caches.Create(ctx, in.Model, config)
	if err != nil {
		job.Error.Error(ctx, err)
		return nil
	}

	// Build output
	output := TaskCacheOutput{
		Operation:     "create",
		CachedContent: result,
	}

	if err := job.Output.WriteData(ctx, output); err != nil {
		job.Error.Error(ctx, err)
		return nil
	}

	return nil
}

// handleListCache lists cached contents with pagination
func (e *execution) handleListCache(ctx context.Context, job *base.Job, client *genai.Client, in *TaskCacheInput) error {
	// Build list config
	config := &genai.ListCachedContentsConfig{}

	if in.PageSize != nil {
		config.PageSize = *in.PageSize
	}

	if in.PageToken != nil {
		config.PageToken = *in.PageToken
	}

	// List cached contents
	page, err := client.Caches.List(ctx, config)
	if err != nil {
		job.Error.Error(ctx, err)
		return nil
	}

	// Build output
	output := TaskCacheOutput{
		Operation:      "list",
		CachedContents: page.Items,
	}

	if page.NextPageToken != "" {
		output.NextPageToken = &page.NextPageToken
	}

	if err := job.Output.WriteData(ctx, output); err != nil {
		job.Error.Error(ctx, err)
		return nil
	}

	return nil
}

// handleGetCache retrieves a specific cached content
func (e *execution) handleGetCache(ctx context.Context, job *base.Job, client *genai.Client, in *TaskCacheInput) error {
	// Validate required fields for get operation
	if in.CacheName == nil || *in.CacheName == "" {
		err := fmt.Errorf("cache-name is required for get operation")
		job.Error.Error(ctx, err)
		return nil
	}

	// Get cached content
	result, err := client.Caches.Get(ctx, *in.CacheName, &genai.GetCachedContentConfig{})
	if err != nil {
		job.Error.Error(ctx, err)
		return nil
	}

	// Build output
	output := TaskCacheOutput{
		Operation:     "get",
		CachedContent: result,
	}

	if err := job.Output.WriteData(ctx, output); err != nil {
		job.Error.Error(ctx, err)
		return nil
	}

	return nil
}

// handleUpdateCache updates an existing cached content (only expiration can be updated)
func (e *execution) handleUpdateCache(ctx context.Context, job *base.Job, client *genai.Client, in *TaskCacheInput) error {
	// Validate required fields for update operation
	if in.CacheName == nil || *in.CacheName == "" {
		err := fmt.Errorf("cache-name is required for update operation")
		job.Error.Error(ctx, err)
		return nil
	}

	if in.TTL == nil && in.ExpireTime == nil {
		err := fmt.Errorf("either ttl or expire-time must be specified for update operation")
		job.Error.Error(ctx, err)
		return nil
	}

	// Build update config
	config := &genai.UpdateCachedContentConfig{}

	// Set expiration
	if in.TTL != nil {
		config.TTL = *in.TTL
	}
	if in.ExpireTime != nil {
		config.ExpireTime = *in.ExpireTime
	}

	// Update cached content
	result, err := client.Caches.Update(ctx, *in.CacheName, config)
	if err != nil {
		job.Error.Error(ctx, err)
		return nil
	}

	// Build output
	output := TaskCacheOutput{
		Operation:     "update",
		CachedContent: result,
	}

	if err := job.Output.WriteData(ctx, output); err != nil {
		job.Error.Error(ctx, err)
		return nil
	}

	return nil
}

// handleDeleteCache deletes a cached content
func (e *execution) handleDeleteCache(ctx context.Context, job *base.Job, client *genai.Client, in *TaskCacheInput) error {
	// Validate required fields for delete operation
	if in.CacheName == nil || *in.CacheName == "" {
		err := fmt.Errorf("cache-name is required for delete operation")
		job.Error.Error(ctx, err)
		return nil
	}

	// Delete cached content
	_, err := client.Caches.Delete(ctx, *in.CacheName, &genai.DeleteCachedContentConfig{})
	if err != nil {
		job.Error.Error(ctx, err)
		return nil
	}

	// Build output (no cached content returned for delete)
	output := TaskCacheOutput{
		Operation: "delete",
	}

	if err := job.Output.WriteData(ctx, output); err != nil {
		job.Error.Error(ctx, err)
		return nil
	}

	return nil
}
