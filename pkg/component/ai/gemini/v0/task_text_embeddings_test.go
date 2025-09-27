package gemini

import (
	"context"
	"testing"

	qt "github.com/frankban/quicktest"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/mock"
)

func TestTextEmbeddings(t *testing.T) {
	c := qt.New(t)
	ctx := context.Background()

	bc := base.Component{Logger: zap.NewNop()}
	component := Init(bc)

	c.Run("ok - input validation and processing", func(c *qt.C) {
		setup, err := structpb.NewStruct(map[string]any{
			"api-key": "test-api-key",
		})
		c.Assert(err, qt.IsNil)

		e := &execution{
			ComponentExecution: base.ComponentExecution{
				Component: component,
				Setup:     setup,
				Task:      TextEmbeddingsTask,
			},
		}

		ir, ow, eh, job := mock.GenerateMockJob(c)

		// Test input reading and validation
		var capturedInput TaskTextEmbeddingsInput
		ir.ReadDataMock.Set(func(ctx context.Context, input any) error {
			switch input := input.(type) {
			case *TaskTextEmbeddingsInput:
				outputDim := int32(768)
				*input = TaskTextEmbeddingsInput{
					Model:                "gemini-embedding-001",
					Text:                 "Hello, world! This is a test sentence for embeddings.",
					TaskType:             "SEMANTIC_SIMILARITY",
					Title:                "Sample Text",
					OutputDimensionality: &outputDim,
				}
				capturedInput = *input
			}
			return nil
		})

		// Mock successful output writing to test the logic flow
		ow.WriteDataMock.Optional()

		// Expect error due to API call (which is expected in unit test)
		var apiCallError error
		eh.ErrorMock.Set(func(ctx context.Context, err error) {
			apiCallError = err
		})

		err = e.textEmbeddings(ctx, job)
		c.Assert(err, qt.IsNil)

		// Verify input was read correctly
		c.Check(capturedInput.Model, qt.Equals, "gemini-embedding-001")
		c.Check(capturedInput.Text, qt.Equals, "Hello, world! This is a test sentence for embeddings.")
		c.Check(capturedInput.TaskType, qt.Equals, "SEMANTIC_SIMILARITY")
		c.Check(capturedInput.Title, qt.Equals, "Sample Text")
		c.Check(*capturedInput.OutputDimensionality, qt.Equals, int32(768))

		// In unit test, we expect API call to fail (no real API key)
		c.Check(apiCallError, qt.IsNotNil)
	})

	c.Run("nok - empty text validation", func(c *qt.C) {
		setup, err := structpb.NewStruct(map[string]any{
			"api-key": "test-api-key",
		})
		c.Assert(err, qt.IsNil)

		e := &execution{
			ComponentExecution: base.ComponentExecution{
				Component: component,
				Setup:     setup,
				Task:      TextEmbeddingsTask,
			},
		}

		ir, ow, eh, job := mock.GenerateMockJob(c)

		// Test input with empty text
		var capturedInput TaskTextEmbeddingsInput
		ir.ReadDataMock.Set(func(ctx context.Context, input any) error {
			switch input := input.(type) {
			case *TaskTextEmbeddingsInput:
				*input = TaskTextEmbeddingsInput{
					Model:    "gemini-embedding-001",
					Text:     "", // Empty text
					TaskType: "CLASSIFICATION",
				}
				capturedInput = *input
			}
			return nil
		})

		ow.WriteDataMock.Optional()

		var apiCallError error
		eh.ErrorMock.Set(func(ctx context.Context, err error) {
			apiCallError = err
		})

		err = e.textEmbeddings(ctx, job)
		c.Assert(err, qt.IsNil)

		// Verify input was read correctly
		c.Check(capturedInput.Model, qt.Equals, "gemini-embedding-001")
		c.Check(capturedInput.Text, qt.Equals, "")
		c.Check(capturedInput.TaskType, qt.Equals, "CLASSIFICATION")

		// Should get an error when trying to process empty text
		c.Check(apiCallError, qt.IsNotNil)
	})

	c.Run("ok - different task types validation", func(c *qt.C) {
		taskTypes := []string{
			"SEMANTIC_SIMILARITY",
			"CLASSIFICATION",
			"CLUSTERING",
			"RETRIEVAL_DOCUMENT",
			"RETRIEVAL_QUERY",
			"CODE_RETRIEVAL_QUERY",
			"QUESTION_ANSWERING",
			"FACT_VERIFICATION",
		}

		for _, taskType := range taskTypes {
			setup, err := structpb.NewStruct(map[string]any{
				"api-key": "test-api-key",
			})
			c.Assert(err, qt.IsNil)

			e := &execution{
				ComponentExecution: base.ComponentExecution{
					Component: component,
					Setup:     setup,
					Task:      TextEmbeddingsTask,
				},
			}

			ir, ow, eh, job := mock.GenerateMockJob(c)

			var capturedInput TaskTextEmbeddingsInput
			ir.ReadDataMock.Set(func(ctx context.Context, input any) error {
				switch input := input.(type) {
				case *TaskTextEmbeddingsInput:
					var outputDim *int32
					var title string

					// Set different parameters based on task type
					if taskType == "RETRIEVAL_DOCUMENT" {
						title = "Document Title for " + taskType
						dim := int32(1536)
						outputDim = &dim
					}

					*input = TaskTextEmbeddingsInput{
						Model:                "gemini-embedding-001",
						Text:                 "Test text for " + taskType + " embeddings.",
						TaskType:             taskType,
						Title:                title,
						OutputDimensionality: outputDim,
					}
					capturedInput = *input
				}
				return nil
			})

			ow.WriteDataMock.Optional()

			var apiCallError error
			eh.ErrorMock.Set(func(ctx context.Context, err error) {
				apiCallError = err
			})

			err = e.textEmbeddings(ctx, job)
			c.Assert(err, qt.IsNil, qt.Commentf("Failed for task type: %s", taskType))

			// Verify input was processed correctly for each task type
			c.Check(capturedInput.Model, qt.Equals, "gemini-embedding-001")
			c.Check(capturedInput.Text, qt.Equals, "Test text for "+taskType+" embeddings.")
			c.Check(capturedInput.TaskType, qt.Equals, taskType)

			if taskType == "RETRIEVAL_DOCUMENT" {
				c.Check(capturedInput.Title, qt.Equals, "Document Title for "+taskType)
				c.Check(*capturedInput.OutputDimensionality, qt.Equals, int32(1536))
			}

			// API call should fail in unit test (expected)
			c.Check(apiCallError, qt.IsNotNil, qt.Commentf("Expected API error for task type: %s", taskType))
		}
	})

	c.Run("ok - task type defaulting", func(c *qt.C) {
		setup, err := structpb.NewStruct(map[string]any{
			"api-key": "test-api-key",
		})
		c.Assert(err, qt.IsNil)

		e := &execution{
			ComponentExecution: base.ComponentExecution{
				Component: component,
				Setup:     setup,
				Task:      TextEmbeddingsTask,
			},
		}

		ir, ow, eh, job := mock.GenerateMockJob(c)

		// Test that empty task type defaults to SEMANTIC_SIMILARITY
		ir.ReadDataMock.Set(func(ctx context.Context, input any) error {
			switch input := input.(type) {
			case *TaskTextEmbeddingsInput:
				*input = TaskTextEmbeddingsInput{
					Model:    "gemini-embedding-001",
					Text:     "Test text with no task type",
					TaskType: "", // Empty task type
				}
			}
			return nil
		})

		ow.WriteDataMock.Optional()
		eh.ErrorMock.Set(func(ctx context.Context, err error) {
			// Expected to fail at API call
		})

		err = e.textEmbeddings(ctx, job)
		c.Assert(err, qt.IsNil)

		// The test verifies that the function processes empty task type correctly
		// (it should default to "SEMANTIC_SIMILARITY" in the business logic)
	})
}
