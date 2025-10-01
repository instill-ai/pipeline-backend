package gemini

import (
	"context"

	"google.golang.org/genai"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func (e *execution) textEmbeddings(ctx context.Context, job *base.Job) error {
	// Read input
	in := TaskTextEmbeddingsInput{}
	if err := job.Input.ReadData(ctx, &in); err != nil {
		job.Error.Error(ctx, err)
		return nil
	}

	// Create Gemini client
	client, err := e.createGeminiClient(ctx)
	if err != nil {
		job.Error.Error(ctx, err)
		return nil
	}

	// Create content from input text
	contents := []*genai.Content{
		genai.NewContentFromText(in.Text, genai.RoleUser),
	}

	// Use the task type from input, defaulting to SEMANTIC_SIMILARITY if empty
	taskType := in.TaskType
	if taskType == "" {
		taskType = "SEMANTIC_SIMILARITY"
	}

	// Generate embeddings using the Gemini API
	result, err := client.Models.EmbedContent(ctx, in.Model, contents, &genai.EmbedContentConfig{
		TaskType:             taskType,
		OutputDimensionality: in.OutputDimensionality,
		Title:                in.Title,
	})
	if err != nil {
		job.Error.Error(ctx, err)
		return nil
	}

	// Extract embeddings from the result
	if len(result.Embeddings) == 0 {
		job.Error.Error(ctx, err)
		return nil
	}

	embedding := result.Embeddings[0]
	if len(embedding.Values) == 0 {
		job.Error.Error(ctx, err)
		return nil
	}

	// Prepare output using the genai ContentEmbedding directly
	output := TaskTextEmbeddingsOutput{
		Embedding: embedding,
	}

	// Write output
	if err := job.Output.WriteData(ctx, output); err != nil {
		job.Error.Error(ctx, err)
		return nil
	}

	return nil
}
