package ollama

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/gojuno/minimock/v3"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func TestComponent_Tasks(t *testing.T) {
	mc := minimock.NewController(t)
	c := qt.New(t)
	bc := base.Component{Logger: zap.NewNop()}
	component := Init(bc)
	ctx := context.Background()

	OllamaClientMock := NewOllamaClientInterfaceMock(mc)
	OllamaClientMock.ChatMock.
		When(ChatRequest{
			Model:    "moondream",
			Options:  OllamaOptions{Seed: 0, Temperature: 0, TopK: 0},
			Messages: []OllamaChatMessage{{Role: "user", Content: "Tell me a joke", Images: []string{}}},
		}).
		Then(ChatResponse{
			Model:              "moondream",
			CreatedAt:          "2024-07-19T10:54:31.448690295Z",
			Message:            OllamaChatMessage{Role: "assistant", Content: "\nWhy did the tomato turn red?\nAnswer: Because it saw the salad dressing"},
			Done:               true,
			DoneReason:         "stop",
			TotalDuration:      3393091575,
			LoadDuration:       3125721807,
			PromptEvalCount:    10,
			PromptEvalDuration: 34202000,
			EvalCount:          18,
			EvalDuration:       141520000,
		}, nil)
	OllamaClientMock.ChatMock.
		When(ChatRequest{
			Model:    "gemini",
			Options:  OllamaOptions{Seed: 0, Temperature: 0, TopK: 0},
			Messages: []OllamaChatMessage{{Role: "user", Content: "Tell me a joke", Images: []string{}}},
		}).
		Then(ChatResponse{}, fmt.Errorf("error when sending chat request %s", `model "gemini" not found, try pulling it first`))
	OllamaClientMock.EmbedMock.
		When(EmbedRequest{
			Model:  "snowflake-arctic-embed:22m",
			Prompt: "The United Kingdom, made up of England, Scotland, Wales and Northern Ireland, is an island nation in northwestern Europe.",
		}).
		Then(EmbedResponse{Embedding: []float32{0.1, 0.2, 0.3, 0.4, 0.5}}, nil)
	OllamaClientMock.EmbedMock.
		When(EmbedRequest{
			Model:  "snowflake-arctic-embed:23m",
			Prompt: "The United Kingdom, made up of England, Scotland, Wales and Northern Ireland, is an island nation in northwestern Europe.",
		}).
		Then(EmbedResponse{}, fmt.Errorf("error when sending embeddings request %s", `model "snowflake-arctic-embed:23m" not found, try pulling it first`))

	c.Run("ok - task text generation", func(c *qt.C) {
		setup, err := structpb.NewStruct(map[string]any{
			"endpoint":  "http://localhost:8080",
			"auto-pull": true,
		})
		c.Assert(err, qt.IsNil)
		e := &execution{
			ComponentExecution: base.ComponentExecution{Component: component, SystemVariables: nil, Setup: setup, Task: TaskTextGenerationChat},
			client:             OllamaClientMock,
		}
		e.execute = e.TaskTextGenerationChat

		pbIn, err := base.ConvertToStructpb(map[string]any{"model": "moondream", "prompt": "Tell me a joke"})
		c.Assert(err, qt.IsNil)

		ir, ow, eh, job := base.GenerateMockJob(c)
		ir.ReadMock.Return(pbIn, nil)
		ow.WriteMock.Optional().Set(func(ctx context.Context, output *structpb.Struct) (err error) {
			wantJSON, err := json.Marshal(TaskTextGenerationChatOuput{Text: "\nWhy did the tomato turn red?\nAnswer: Because it saw the salad dressing"})
			c.Assert(err, qt.IsNil)
			c.Check(wantJSON, qt.JSONEquals, output.AsMap())
			return nil
		})
		eh.ErrorMock.Optional()

		err = e.Execute(ctx, []*base.Job{job})
		c.Assert(err, qt.IsNil)

	})

	c.Run("nok - task text generation", func(c *qt.C) {
		setup, err := structpb.NewStruct(map[string]any{
			"endpoint":  "http://localhost:8080",
			"auto-pull": true,
		})
		c.Assert(err, qt.IsNil)
		e := &execution{
			ComponentExecution: base.ComponentExecution{Component: component, SystemVariables: nil, Setup: setup, Task: TaskTextGenerationChat},
			client:             OllamaClientMock,
		}
		e.execute = e.TaskTextGenerationChat

		pbIn, err := base.ConvertToStructpb(map[string]any{"model": "gemini", "prompt": "Tell me a joke"})
		c.Assert(err, qt.IsNil)

		ir, ow, eh, job := base.GenerateMockJob(c)
		ir.ReadMock.Return(pbIn, nil)
		ow.WriteMock.Optional().Return(nil)
		eh.ErrorMock.Optional().Set(func(ctx context.Context, err error) {
			c.Assert(err, qt.ErrorMatches, `error when sending chat request model "gemini" not found, try pulling it first`)
		})

		err = e.Execute(ctx, []*base.Job{job})
		c.Assert(err, qt.IsNil)

	})

	c.Run("ok - task embedding", func(c *qt.C) {
		setup, err := structpb.NewStruct(map[string]any{
			"endpoint":  "http://localhost:8080",
			"auto-pull": true,
		})
		c.Assert(err, qt.IsNil)
		e := &execution{
			ComponentExecution: base.ComponentExecution{Component: component, SystemVariables: nil, Setup: setup, Task: TaskTextEmbeddings},
			client:             OllamaClientMock,
		}
		e.execute = e.TaskTextEmbeddings

		pbIn, err := base.ConvertToStructpb(map[string]any{"model": "snowflake-arctic-embed:22m", "text": "The United Kingdom, made up of England, Scotland, Wales and Northern Ireland, is an island nation in northwestern Europe."})
		c.Assert(err, qt.IsNil)

		ir, ow, eh, job := base.GenerateMockJob(c)
		ir.ReadMock.Return(pbIn, nil)
		ow.WriteMock.Optional().Set(func(ctx context.Context, output *structpb.Struct) (err error) {
			wantJSON, err := json.Marshal(TaskTextEmbeddingsOutput{Embedding: []float32{0.1, 0.2, 0.3, 0.4, 0.5}})
			c.Assert(err, qt.IsNil)
			c.Check(wantJSON, qt.JSONEquals, output.AsMap())
			return nil
		})
		eh.ErrorMock.Optional()

		err = e.Execute(ctx, []*base.Job{job})
		c.Assert(err, qt.IsNil)

	})

	c.Run("nok - task embedding", func(c *qt.C) {
		setup, err := structpb.NewStruct(map[string]any{
			"endpoint":  "http://localhost:8080",
			"auto-pull": true,
		})
		c.Assert(err, qt.IsNil)
		e := &execution{
			ComponentExecution: base.ComponentExecution{Component: component, SystemVariables: nil, Setup: setup, Task: TaskTextEmbeddings},
			client:             OllamaClientMock,
		}
		e.execute = e.TaskTextEmbeddings

		pbIn, err := base.ConvertToStructpb(map[string]any{"model": "snowflake-arctic-embed:23m", "text": "The United Kingdom, made up of England, Scotland, Wales and Northern Ireland, is an island nation in northwestern Europe."})
		c.Assert(err, qt.IsNil)

		ir, ow, eh, job := base.GenerateMockJob(c)
		ir.ReadMock.Return(pbIn, nil)
		ow.WriteMock.Optional().Return(nil)
		eh.ErrorMock.Optional().Set(func(ctx context.Context, err error) {
			c.Assert(err, qt.ErrorMatches, `error when sending embeddings request model "snowflake-arctic-embed:23m" not found, try pulling it first`)
		})

		err = e.Execute(ctx, []*base.Job{job})
		c.Assert(err, qt.IsNil)

	})

}
