package weaviate

import (
	"context"
	"encoding/json"
	"testing"

	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/mock"
)

func TestComponent_ExecuteInsertTask(t *testing.T) {
	c := qt.New(t)
	ctx := context.Background()
	bc := base.Component{Logger: zap.NewNop()}
	component := Init(bc)

	testcases := []struct {
		name     string
		input    InsertInput
		wantResp InsertOutput
		wantErr  string
	}{
		{
			name: "ok to insert",
			input: InsertInput{
				CollectionName: "test_coll",
				Vector:         []float32{0.1, 0.2},
				Metadata:       map[string]any{"name": "test"},
			},
			wantResp: InsertOutput{
				Status: "Successfully inserted 1 object",
			},
		},
	}

	for _, tc := range testcases {
		c.Run(tc.name, func(c *qt.C) {
			setup, err := structpb.NewStruct(map[string]any{
				"url":     "mock-url",
				"api-key": "mock-api-key",
			})
			c.Assert(err, qt.IsNil)

			e := &execution{
				ComponentExecution: base.ComponentExecution{Component: component, SystemVariables: nil, Setup: setup, Task: TaskInsert},
				mockClient:         &MockWeaviateClient{},
			}
			e.execute = e.insert

			pbIn, err := base.ConvertToStructpb(tc.input)
			c.Assert(err, qt.IsNil)

			ir, ow, eh, job := mock.GenerateMockJob(c)
			ir.ReadMock.Return(pbIn, nil)
			ow.WriteMock.Optional().Set(func(ctx context.Context, output *structpb.Struct) (err error) {
				wantJSON, err := json.Marshal(tc.wantResp)
				c.Assert(err, qt.IsNil)
				c.Check(wantJSON, qt.JSONEquals, output.AsMap())
				return nil
			})
			eh.ErrorMock.Optional().Set(func(ctx context.Context, err error) {
				if tc.wantErr != "" {
					c.Assert(err, qt.ErrorMatches, tc.wantErr)
				}
			})

			err = e.Execute(ctx, []*base.Job{job})
			c.Assert(err, qt.IsNil)

		})
	}
}

func TestComponent_ExecuteDeleteTask(t *testing.T) {
	c := qt.New(t)
	ctx := context.Background()
	bc := base.Component{Logger: zap.NewNop()}
	component := Init(bc)

	testcases := []struct {
		name       string
		input      DeleteInput
		wantResp   DeleteOutput
		wantErr    string
		Successful int
	}{
		{
			name: "ok to delete",
			input: DeleteInput{
				CollectionName: "test_coll",
				Filter:         map[string]any{"path": "text", "operator": "Equal", "valueText": "test"},
			},
			wantResp: DeleteOutput{
				Status: "Successfully deleted 1 objects",
			},
			Successful: 1,
		},
	}

	for _, tc := range testcases {
		c.Run(tc.name, func(c *qt.C) {
			setup, err := structpb.NewStruct(map[string]any{
				"url":     "mock-url",
				"api-key": "mock-api-key",
			})
			c.Assert(err, qt.IsNil)

			e := &execution{
				ComponentExecution: base.ComponentExecution{Component: component, SystemVariables: nil, Setup: setup, Task: TaskDelete},
				mockClient: &MockWeaviateClient{
					Successful: tc.Successful,
				},
			}
			e.execute = e.delete

			pbIn, err := base.ConvertToStructpb(tc.input)
			c.Assert(err, qt.IsNil)

			ir, ow, eh, job := mock.GenerateMockJob(c)
			ir.ReadMock.Return(pbIn, nil)
			ow.WriteMock.Optional().Set(func(ctx context.Context, output *structpb.Struct) (err error) {
				wantJSON, err := json.Marshal(tc.wantResp)
				c.Assert(err, qt.IsNil)
				c.Check(wantJSON, qt.JSONEquals, output.AsMap())
				return nil
			})
			eh.ErrorMock.Optional().Set(func(ctx context.Context, err error) {
				if tc.wantErr != "" {
					c.Assert(err, qt.ErrorMatches, tc.wantErr)
				}
			})

			err = e.Execute(ctx, []*base.Job{job})
			c.Assert(err, qt.IsNil)

		})
	}
}

func TestComponent_ExecuteDeleteCollectionTask(t *testing.T) {
	c := qt.New(t)
	ctx := context.Background()
	bc := base.Component{Logger: zap.NewNop()}
	component := Init(bc)

	testcases := []struct {
		name     string
		input    DeleteCollectionInput
		wantResp DeleteCollectionOutput
		wantErr  string
	}{
		{
			name: "ok to delete collection",
			input: DeleteCollectionInput{
				CollectionName: "test_coll",
			},
			wantResp: DeleteCollectionOutput{
				Status: "Successfully deleted 1 collection",
			},
		},
	}

	for _, tc := range testcases {
		c.Run(tc.name, func(c *qt.C) {
			setup, err := structpb.NewStruct(map[string]any{
				"url":     "mock-url",
				"api-key": "mock-api-key",
			})
			c.Assert(err, qt.IsNil)

			e := &execution{
				ComponentExecution: base.ComponentExecution{Component: component, SystemVariables: nil, Setup: setup, Task: TaskDeleteCollection},
				mockClient:         &MockWeaviateClient{},
			}
			e.execute = e.deleteCollection

			pbIn, err := base.ConvertToStructpb(tc.input)
			c.Assert(err, qt.IsNil)

			ir, ow, eh, job := mock.GenerateMockJob(c)
			ir.ReadMock.Return(pbIn, nil)
			ow.WriteMock.Optional().Set(func(ctx context.Context, output *structpb.Struct) (err error) {
				wantJSON, err := json.Marshal(tc.wantResp)
				c.Assert(err, qt.IsNil)
				c.Check(wantJSON, qt.JSONEquals, output.AsMap())
				return nil
			})
			eh.ErrorMock.Optional().Set(func(ctx context.Context, err error) {
				if tc.wantErr != "" {
					c.Assert(err, qt.ErrorMatches, tc.wantErr)
				}
			})

			err = e.Execute(ctx, []*base.Job{job})
			c.Assert(err, qt.IsNil)

		})
	}
}

func TestComponent_ExecuteVectorSearchTask(t *testing.T) {
	c := qt.New(t)
	ctx := context.Background()
	bc := base.Component{Logger: zap.NewNop()}
	component := Init(bc)

	testcases := []struct {
		name       string
		input      VectorSearchInput
		wantResp   VectorSearchOutput
		wantErr    string
		Successful int
	}{
		{
			name: "ok to vector search",
			input: VectorSearchInput{
				CollectionName: "test_coll",
				Vector:         []float32{0.1, 0.2},
				Limit:          1,
				Filter: map[string]any{
					"path":     "age",
					"operator": "Equal",
					"valueInt": 20,
				},
			},
			wantResp: VectorSearchOutput{
				Status: "Successfully found 1 objects",
				Result: Result{
					Vectors:  [][]float32{{0.1, 0.2}},
					Metadata: []map[string]any{{"name": "test"}},
					Objects:  []map[string]any{{"name": "test", "_additional": map[string]any{"vector": []float32{0.1, 0.2}}}},
				},
			},
			Successful: 1,
		},
	}

	for _, tc := range testcases {
		c.Run(tc.name, func(c *qt.C) {
			setup, err := structpb.NewStruct(map[string]any{
				"url":     "mock-url",
				"api-key": "mock-api-key",
			})
			c.Assert(err, qt.IsNil)

			e := &execution{
				ComponentExecution: base.ComponentExecution{Component: component, SystemVariables: nil, Setup: setup, Task: TaskVectorSearch},
				mockClient: &MockWeaviateClient{
					VectorSearch: tc.wantResp.Result,
					Successful:   tc.Successful,
				},
			}
			e.execute = e.vectorSearch

			pbIn, err := base.ConvertToStructpb(tc.input)
			c.Assert(err, qt.IsNil)

			ir, ow, eh, job := mock.GenerateMockJob(c)
			ir.ReadMock.Return(pbIn, nil)
			ow.WriteMock.Optional().Set(func(ctx context.Context, output *structpb.Struct) (err error) {
				wantJSON, err := json.Marshal(tc.wantResp)
				c.Assert(err, qt.IsNil)
				c.Check(wantJSON, qt.JSONEquals, output.AsMap())
				return nil
			})
			eh.ErrorMock.Optional().Set(func(ctx context.Context, err error) {
				if tc.wantErr != "" {
					c.Assert(err, qt.ErrorMatches, tc.wantErr)
				}
			})

			err = e.Execute(ctx, []*base.Job{job})
			c.Assert(err, qt.IsNil)

		})
	}
}

func TestComponent_ExecuteBatchInsertTask(t *testing.T) {
	c := qt.New(t)
	ctx := context.Background()
	bc := base.Component{Logger: zap.NewNop()}
	component := Init(bc)

	testcases := []struct {
		name       string
		input      BatchInsertInput
		wantResp   BatchInsertOutput
		wantErr    string
		Successful int
	}{
		{
			name: "ok to insert many",
			input: BatchInsertInput{
				ArrayMetadata: []map[string]any{
					{"name": "test1", "email": "test1@example.com"},
					{"name": "test2", "email": "test2@example.com"},
				},
				ArrayVector: [][]float32{
					{0.1, 0.2},
					{0.3, 0.4},
				},
				CollectionName: "test_coll",
			},
			wantResp: BatchInsertOutput{
				Status: "Successfully batch inserted 2 objects",
			},
			Successful: 2,
		},
	}

	for _, tc := range testcases {
		c.Run(tc.name, func(c *qt.C) {
			setup, err := structpb.NewStruct(map[string]any{
				"url":     "mock-url",
				"api-key": "mock-api-key",
			})
			c.Assert(err, qt.IsNil)

			e := &execution{
				ComponentExecution: base.ComponentExecution{Component: component, SystemVariables: nil, Setup: setup, Task: TaskBatchInsert},
				mockClient: &MockWeaviateClient{
					Successful: tc.Successful,
				},
			}
			e.execute = e.batchInsert

			pbIn, err := base.ConvertToStructpb(tc.input)
			c.Assert(err, qt.IsNil)

			ir, ow, eh, job := mock.GenerateMockJob(c)
			ir.ReadMock.Return(pbIn, nil)
			ow.WriteMock.Optional().Set(func(ctx context.Context, output *structpb.Struct) (err error) {
				wantJSON, err := json.Marshal(tc.wantResp)
				c.Assert(err, qt.IsNil)
				c.Check(wantJSON, qt.JSONEquals, output.AsMap())
				return nil
			})
			eh.ErrorMock.Optional().Set(func(ctx context.Context, err error) {
				if tc.wantErr != "" {
					c.Assert(err, qt.ErrorMatches, tc.wantErr)
				}
			})

			err = e.Execute(ctx, []*base.Job{job})
			c.Assert(err, qt.IsNil)

		})
	}
}

func TestComponent_ExecuteUpdateTask(t *testing.T) {
	c := qt.New(t)
	ctx := context.Background()
	bc := base.Component{Logger: zap.NewNop()}
	component := Init(bc)

	testcases := []struct {
		name     string
		input    UpdateInput
		wantResp UpdateOutput
		wantErr  string
	}{
		{
			name: "ok to update",
			input: UpdateInput{
				CollectionName: "test_coll",
				ID:             "test-id",
				Vector:         []float32{0.1, 0.2},
				Metadata:       map[string]any{"name": "test"},
			},
			wantResp: UpdateOutput{
				Status: "Successfully updated 1 object",
			},
		},
	}

	for _, tc := range testcases {
		c.Run(tc.name, func(c *qt.C) {
			setup, err := structpb.NewStruct(map[string]any{
				"url":     "mock-url",
				"api-key": "mock-api-key",
			})
			c.Assert(err, qt.IsNil)

			e := &execution{
				ComponentExecution: base.ComponentExecution{Component: component, SystemVariables: nil, Setup: setup, Task: TaskUpdate},
				mockClient:         &MockWeaviateClient{},
			}
			e.execute = e.update

			pbIn, err := base.ConvertToStructpb(tc.input)
			c.Assert(err, qt.IsNil)

			ir, ow, eh, job := mock.GenerateMockJob(c)
			ir.ReadMock.Return(pbIn, nil)
			ow.WriteMock.Optional().Set(func(ctx context.Context, output *structpb.Struct) (err error) {
				wantJSON, err := json.Marshal(tc.wantResp)
				c.Assert(err, qt.IsNil)
				c.Check(wantJSON, qt.JSONEquals, output.AsMap())
				return nil
			})
			eh.ErrorMock.Optional().Set(func(ctx context.Context, err error) {
				if tc.wantErr != "" {
					c.Assert(err, qt.ErrorMatches, tc.wantErr)
				}
			})

			err = e.Execute(ctx, []*base.Job{job})
			c.Assert(err, qt.IsNil)

		})
	}
}
