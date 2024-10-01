package cohere

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"

	cohereSDK "github.com/cohere-ai/cohere-go/v2"
	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/mock"
)

const (
	apiKey        = "cohere-api-key"
	instillSecret = "instill-credential-key"
)

func TestComponent_Execute(t *testing.T) {
	c := qt.New(t)

	bc := base.Component{Logger: zap.NewNop()}
	cmp := Init(bc).WithInstillCredentials(map[string]any{"apikey": instillSecret})

	c.Run("ok - supported task", func(c *qt.C) {
		task := TextGenerationTask

		_, err := cmp.CreateExecution(base.ComponentExecution{
			Component: cmp,
			Task:      task,
		})
		c.Check(err, qt.IsNil)
	})
	c.Run("ok - supported task", func(c *qt.C) {
		task := TextEmbeddingTask

		_, err := cmp.CreateExecution(base.ComponentExecution{
			Component: cmp,
			Task:      task,
		})
		c.Check(err, qt.IsNil)
	})
	c.Run("ok - supported task", func(c *qt.C) {
		task := TextRerankTask

		_, err := cmp.CreateExecution(base.ComponentExecution{
			Component: cmp,
			Task:      task,
		})
		c.Check(err, qt.IsNil)
	})

	c.Run("nok - unsupported task", func(c *qt.C) {
		task := "FOOBAR"

		_, err := cmp.CreateExecution(base.ComponentExecution{
			Component: cmp,
			Task:      task,
		})
		c.Check(err, qt.ErrorMatches, "unsupported task")
	})

}

func TestComponent_Tasks(t *testing.T) {
	c := qt.New(t)

	bc := base.Component{Logger: zap.NewNop()}
	cmp := Init(bc).WithInstillCredentials(map[string]any{"apikey": instillSecret})
	ctx := context.Background()

	commandTc := struct {
		input    map[string]any
		wantResp TextGenerationOutput
	}{
		input:    map[string]any{"model-name": "command-r-plus"},
		wantResp: TextGenerationOutput{Text: "Hi! My name is command-r-plus.", Citations: []citation{}, Usage: commandUsage{InputTokens: 20, OutputTokens: 30}},
	}

	c.Run("ok - task command", func(c *qt.C) {
		setup, err := structpb.NewStruct(map[string]any{
			"api-key": apiKey,
		})
		c.Assert(err, qt.IsNil)
		exec := &execution{
			ComponentExecution: base.ComponentExecution{Component: cmp, SystemVariables: nil, Setup: setup, Task: TextGenerationTask},
			client:             &MockCohereClient{},
		}
		exec.execute = exec.taskTextGeneration

		pbIn, err := base.ConvertToStructpb(commandTc.input)
		c.Assert(err, qt.IsNil)

		ir, ow, eh, job := mock.GenerateMockJob(c)
		ir.ReadMock.Return(pbIn, nil)
		ow.WriteMock.Optional().Set(func(ctx context.Context, output *structpb.Struct) (err error) {
			wantJSON, err := json.Marshal(commandTc.wantResp)
			c.Assert(err, qt.IsNil)
			c.Check(wantJSON, qt.JSONEquals, output.AsMap())
			return nil
		})
		eh.ErrorMock.Optional()

		err = exec.Execute(ctx, []*base.Job{job})
		c.Assert(err, qt.IsNil)

	})

	embedFloatTc := struct {
		input    map[string]any
		wantResp EmbeddingFloatOutput
	}{
		input:    map[string]any{"text": "abcde"},
		wantResp: EmbeddingFloatOutput{Embedding: []float64{0.1, 0.2, 0.3, 0.4, 0.5}, Usage: embedUsage{Tokens: 20}},
	}

	c.Run("ok - task float embed", func(c *qt.C) {
		setup, err := structpb.NewStruct(map[string]any{
			"api-key": apiKey,
		})
		c.Assert(err, qt.IsNil)
		exec := &execution{
			ComponentExecution: base.ComponentExecution{Component: cmp, SystemVariables: nil, Setup: setup, Task: TextEmbeddingTask},
			client:             &MockCohereClient{},
		}
		exec.execute = exec.taskEmbedding

		pbIn, err := base.ConvertToStructpb(embedFloatTc.input)
		c.Assert(err, qt.IsNil)

		ir, ow, eh, job := mock.GenerateMockJob(c)
		ir.ReadMock.Return(pbIn, nil)
		ow.WriteMock.Optional().Set(func(ctx context.Context, output *structpb.Struct) (err error) {
			wantJSON, err := json.Marshal(embedFloatTc.wantResp)
			c.Assert(err, qt.IsNil)
			c.Check(wantJSON, qt.JSONEquals, output.AsMap())
			return nil
		})
		eh.ErrorMock.Optional()

		err = exec.Execute(ctx, []*base.Job{job})
		c.Assert(err, qt.IsNil)

	})

	embedIntTc := struct {
		input    map[string]any
		wantResp EmbeddingIntOutput
	}{
		input:    map[string]any{"text": "abcde", "embedding-type": "int8"},
		wantResp: EmbeddingIntOutput{Embedding: []int{1, 2, 3, 4, 5}, Usage: embedUsage{Tokens: 20}},
	}

	c.Run("ok - task int embed", func(c *qt.C) {
		setup, err := structpb.NewStruct(map[string]any{
			"api-key": apiKey,
		})
		c.Assert(err, qt.IsNil)
		exec := &execution{
			ComponentExecution: base.ComponentExecution{Component: cmp, SystemVariables: nil, Setup: setup, Task: TextEmbeddingTask},
			client:             &MockCohereClient{},
		}
		exec.execute = exec.taskEmbedding

		pbIn, err := base.ConvertToStructpb(embedIntTc.input)
		c.Assert(err, qt.IsNil)

		ir, ow, eh, job := mock.GenerateMockJob(c)
		ir.ReadMock.Return(pbIn, nil)
		ow.WriteMock.Optional().Set(func(ctx context.Context, output *structpb.Struct) (err error) {
			wantJSON, err := json.Marshal(embedIntTc.wantResp)
			c.Assert(err, qt.IsNil)
			c.Check(wantJSON, qt.JSONEquals, output.AsMap())
			return nil
		})
		eh.ErrorMock.Optional()

		err = exec.Execute(ctx, []*base.Job{job})
		c.Assert(err, qt.IsNil)

	})

	rerankTc := struct {
		input    map[string]any
		wantResp RerankOutput
	}{
		input:    map[string]any{"documents": []string{"a", "b", "c", "d"}},
		wantResp: RerankOutput{Ranking: []string{"d", "c", "b", "a"}, Usage: rerankUsage{Search: 5}, Relevance: []float64{10, 9, 8, 7}},
	}
	c.Run("ok - task rerank", func(c *qt.C) {
		setup, err := structpb.NewStruct(map[string]any{
			"api-key": apiKey,
		})
		c.Assert(err, qt.IsNil)
		exec := &execution{
			ComponentExecution: base.ComponentExecution{Component: cmp, SystemVariables: nil, Setup: setup, Task: TextRerankTask},
			client:             &MockCohereClient{},
		}
		exec.execute = exec.taskRerank

		pbIn, err := base.ConvertToStructpb(rerankTc.input)
		c.Assert(err, qt.IsNil)

		ir, ow, eh, job := mock.GenerateMockJob(c)
		ir.ReadMock.Return(pbIn, nil)
		ow.WriteMock.Optional().Set(func(ctx context.Context, output *structpb.Struct) (err error) {
			wantJSON, err := json.Marshal(rerankTc.wantResp)
			c.Assert(err, qt.IsNil)
			c.Check(wantJSON, qt.JSONEquals, output.AsMap())
			return nil
		})
		eh.ErrorMock.Optional()

		err = exec.Execute(ctx, []*base.Job{job})
		c.Assert(err, qt.IsNil)

	})

}

type MockCohereClient struct{}

func (m *MockCohereClient) generateTextChat(request cohereSDK.ChatRequest) (cohereSDK.NonStreamedChatResponse, error) {
	tx := fmt.Sprintf("Hi! My name is %s.", *request.Model)
	cia := []*cohereSDK.ChatCitation{}
	inputToken := float64(20)
	outputToken := float64(30)
	bill := cohereSDK.ApiMetaBilledUnits{InputTokens: &inputToken, OutputTokens: &outputToken}
	meta := cohereSDK.ApiMeta{BilledUnits: &bill}
	return cohereSDK.NonStreamedChatResponse{
		Citations: cia,
		Text:      tx,
		Meta:      &meta,
	}, nil
}

func (m *MockCohereClient) generateEmbedding(request cohereSDK.EmbedRequest) (cohereSDK.EmbedResponse, error) {
	inputToken := float64(20)
	bill := cohereSDK.ApiMetaBilledUnits{InputTokens: &inputToken}
	meta := cohereSDK.ApiMeta{BilledUnits: &bill}
	if len(request.EmbeddingTypes) != 0 {
		tp := request.EmbeddingTypes[0]
		embedding := cohereSDK.EmbedByTypeResponse{}
		switch tp {
		case cohereSDK.EmbeddingTypeFloat:
			embedding = cohereSDK.EmbedByTypeResponse{
				Embeddings: &cohereSDK.EmbedByTypeResponseEmbeddings{
					Float: [][]float64{{0.1, 0.2, 0.3, 0.4, 0.5}},
				},
				Meta: &meta,
			}
		case cohereSDK.EmbeddingTypeInt8:
			embedding = cohereSDK.EmbedByTypeResponse{
				Embeddings: &cohereSDK.EmbedByTypeResponseEmbeddings{
					Int8: [][]int{{1, 2, 3, 4, 5}},
				},
				Meta: &meta,
			}
		case cohereSDK.EmbeddingTypeUint8:
			embedding = cohereSDK.EmbedByTypeResponse{
				Embeddings: &cohereSDK.EmbedByTypeResponseEmbeddings{
					Uint8: [][]int{{1, 2, 3, 4, 5}},
				},
				Meta: &meta,
			}
		case cohereSDK.EmbeddingTypeBinary:
			embedding = cohereSDK.EmbedByTypeResponse{
				Embeddings: &cohereSDK.EmbedByTypeResponseEmbeddings{
					Binary: [][]int{{1, 2, 3, 4, 5}},
				},
				Meta: &meta,
			}
		case cohereSDK.EmbeddingTypeUbinary:
			embedding = cohereSDK.EmbedByTypeResponse{
				Embeddings: &cohereSDK.EmbedByTypeResponseEmbeddings{
					Ubinary: [][]int{{1, 2, 3, 4, 5}},
				},
				Meta: &meta,
			}
		}

		return cohereSDK.EmbedResponse{
			EmbeddingsByType: &embedding,
		}, nil
	} else {
		embedding := cohereSDK.EmbedFloatsResponse{
			Embeddings: [][]float64{{0.1, 0.2, 0.3, 0.4, 0.5}},
			Meta:       &meta,
		}
		return cohereSDK.EmbedResponse{
			EmbeddingsFloats: &embedding,
		}, nil
	}
}

func (m *MockCohereClient) generateRerank(request cohereSDK.RerankRequest) (cohereSDK.RerankResponse, error) {
	documents := []cohereSDK.RerankResponseResultsItemDocument{
		{Text: request.Documents[3].String},
		{Text: request.Documents[2].String},
		{Text: request.Documents[1].String},
		{Text: request.Documents[0].String},
	}
	result := []*cohereSDK.RerankResponseResultsItem{
		{Document: &documents[0], RelevanceScore: 10},
		{Document: &documents[1], RelevanceScore: 9},
		{Document: &documents[2], RelevanceScore: 8},
		{Document: &documents[3], RelevanceScore: 7},
	}
	searchCnt := float64(5)
	bill := cohereSDK.ApiMetaBilledUnits{SearchUnits: &searchCnt}
	meta := cohereSDK.ApiMeta{BilledUnits: &bill}
	return cohereSDK.RerankResponse{
		Results: result,
		Meta:    &meta,
	}, nil
}
