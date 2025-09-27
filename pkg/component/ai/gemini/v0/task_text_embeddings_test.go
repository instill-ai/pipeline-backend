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

	c.Run("ok - text embeddings", func(c *qt.C) {
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

		ir, _, eh, job := mock.GenerateMockJob(c)
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
			}
			return nil
		})

		eh.ErrorMock.Set(func(ctx context.Context, err error) {
			c.Fatal(err)
		})

		err = e.textEmbeddings(ctx, job)
		c.Assert(err, qt.IsNil)
	})

	c.Run("nok - empty text", func(c *qt.C) {
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

		ir, _, eh, job := mock.GenerateMockJob(c)
		ir.ReadDataMock.Set(func(ctx context.Context, input any) error {
			switch input := input.(type) {
			case *TaskTextEmbeddingsInput:
				*input = TaskTextEmbeddingsInput{
					Model:    "gemini-embedding-001",
					Text:     "", // Empty text
					TaskType: "CLASSIFICATION",
				}
			}
			return nil
		})

		var errorOccurred bool
		eh.ErrorMock.Set(func(ctx context.Context, err error) {
			errorOccurred = true
		})

		err = e.textEmbeddings(ctx, job)
		c.Assert(err, qt.IsNil)
		c.Assert(errorOccurred, qt.Equals, true, qt.Commentf("Expected error for empty text"))
	})

	c.Run("ok - different task types", func(c *qt.C) {
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

			ir, _, eh, job := mock.GenerateMockJob(c)
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
				}
				return nil
			})

			eh.ErrorMock.Set(func(ctx context.Context, err error) {
				c.Fatalf("Unexpected error for task type %s: %v", taskType, err)
			})

			err = e.textEmbeddings(ctx, job)
			c.Assert(err, qt.IsNil, qt.Commentf("Failed for task type: %s", taskType))
		}
	})
}
