package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"testing"

	"github.com/elastic/go-elasticsearch/v8/esapi"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/mock"
)

func MockESSearch(wantResp SearchOutput) *esapi.Response {
	var Hits []Hit
	documentsBytes, _ := json.Marshal(wantResp.Result.Documents)
	_ = json.Unmarshal(documentsBytes, &Hits)

	resp := SearchResponse{
		Shards: struct {
			Total      int `json:"total"`
			Successful int `json:"successful"`
			Skipped    int `json:"skipped"`
			Failed     int `json:"failed"`
		}{
			Total:      1,
			Successful: 1,
			Skipped:    0,
			Failed:     0,
		},
		Hits: struct {
			Total struct {
				Value    int    `json:"value"`
				Relation string `json:"relation"`
			} `json:"total"`
			MaxScore float64 `json:"max_score"`
			Hits     []Hit   `json:"hits"`
		}{
			Total: struct {
				Value    int    `json:"value"`
				Relation string `json:"relation"`
			}{
				Value:    len(wantResp.Result.Documents),
				Relation: "eq",
			},
			MaxScore: 2,
			Hits:     Hits,
		},
	}

	b, _ := json.Marshal(resp)
	return &esapi.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewReader(b)),
		Header:     make(map[string][]string),
	}
}

func MockESVectorSearch(wantResp VectorSearchOutput) *esapi.Response {
	var Hits []Hit
	documentsBytes, _ := json.Marshal(wantResp.Result.Documents)
	_ = json.Unmarshal(documentsBytes, &Hits)

	resp := SearchResponse{
		Shards: struct {
			Total      int `json:"total"`
			Successful int `json:"successful"`
			Skipped    int `json:"skipped"`
			Failed     int `json:"failed"`
		}{
			Total:      1,
			Successful: 1,
			Skipped:    0,
			Failed:     0,
		},
		Hits: struct {
			Total struct {
				Value    int    `json:"value"`
				Relation string `json:"relation"`
			} `json:"total"`
			MaxScore float64 `json:"max_score"`
			Hits     []Hit   `json:"hits"`
		}{
			Total: struct {
				Value    int    `json:"value"`
				Relation string `json:"relation"`
			}{
				Value:    len(wantResp.Result.Documents),
				Relation: "eq",
			},
			MaxScore: 2,
			Hits:     Hits,
		},
	}

	b, _ := json.Marshal(resp)
	return &esapi.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewReader(b)),
		Header:     make(map[string][]string),
	}
}

func MockESIndex(wantResp IndexOutput) *esapi.Response {
	resp := map[string]string{"status": wantResp.Status}
	b, _ := json.Marshal(resp)
	return &esapi.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewReader(b)),
		Header:     make(map[string][]string),
	}
}

func MockESMultiIndex(wantResp MultiIndexOutput) *esapi.Response {
	resp := MultiIndexResponse{
		Items: []any{"mockItem1", "mockItem2"},
	}
	b, _ := json.Marshal(resp)
	return &esapi.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewReader(b)),
		Header:     make(map[string][]string),
	}
}

func MockESUpdate(wantResp UpdateOutput) *esapi.Response {
	resp := DeleteUpdateResponse{
		Updated: 1,
	}
	b, _ := json.Marshal(resp)
	return &esapi.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewReader(b)),
		Header:     make(map[string][]string),
	}
}

func MockESDelete(wantResp DeleteOutput) *esapi.Response {
	resp := DeleteUpdateResponse{
		Deleted: 1,
	}
	b, _ := json.Marshal(resp)
	return &esapi.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewReader(b)),
		Header:     make(map[string][]string),
	}
}

func MockESCreateIndex(wantResp CreateIndexOutput) *esapi.Response {
	resp := map[string]string{"status": wantResp.Status}
	b, _ := json.Marshal(resp)
	return &esapi.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewReader(b)),
		Header:     make(map[string][]string),
	}
}

func MockESDeleteIndex(wantResp DeleteIndexOutput) *esapi.Response {
	resp := map[string]string{"status": wantResp.Status}
	b, _ := json.Marshal(resp)
	return &esapi.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewReader(b)),
		Header:     make(map[string][]string),
	}
}

func MockESSQLTranslate() *esapi.Response {
	resp := map[string]any{"query": map[string]any{}}
	b, _ := json.Marshal(resp)
	return &esapi.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewReader(b)),
	}
}

func TestComponent_ExecuteSearchTask(t *testing.T) {
	c := qt.New(t)
	ctx := context.Background()
	bc := base.Component{Logger: zap.NewNop()}
	component := Init(bc)

	testcases := []struct {
		name     string
		input    SearchInput
		wantResp SearchOutput
		wantErr  string
		task     string
		count    int
	}{
		{
			name: "ok to search",
			input: SearchInput{
				IndexName: "index_name",
				FilterSQL: "city = 'New York'",
				Size:      0,
			},
			wantResp: SearchOutput{
				Status: "Successfully searched 2 documents",
				Result: SearchResult{
					IDs: []string{"mockID1", "mockID2"},
					Documents: []map[string]any{
						{"_id": "mockID1", "_index": "index_name", "_score": 1, "_source": map[string]any{"name": "John Doe", "email": "john@example.com"}},
						{"_id": "mockID2", "_index": "index_name", "_score": 0.5, "_source": map[string]any{"name": "Jane Smith", "email": "jane@example.com"}},
					},
					Data: []map[string]any{
						{"name": "John Doe", "email": "john@example.com"},
						{"name": "Jane Smith", "email": "jane@example.com"},
					},
				},
			},
			task: TaskSearch,
		},
	}

	for _, tc := range testcases {
		c.Run(tc.name, func(c *qt.C) {
			setup, err := structpb.NewStruct(map[string]any{
				"api-key":  "mock-api-key",
				"cloud-id": "mock-cloud-id",
			})
			c.Assert(err, qt.IsNil)

			e := &execution{
				ComponentExecution: base.ComponentExecution{Component: component, SystemVariables: nil, Setup: setup, Task: tc.task},
				client: ESClient{
					searchClient: func(o ...func(*esapi.SearchRequest)) (*esapi.Response, error) {
						return MockESSearch(tc.wantResp), nil
					},
					sqlTranslateClient: func(body io.Reader, o ...func(*esapi.SQLTranslateRequest)) (*esapi.Response, error) {
						return MockESSQLTranslate(), nil
					},
				},
			}

			e.execute = e.search

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
		name     string
		input    VectorSearchInput
		wantResp VectorSearchOutput
		wantErr  string
		task     string
		count    int
	}{
		{
			name: "ok to vector search",
			input: VectorSearchInput{
				IndexName:   "index_name",
				FilterSQL:   "name = 'a'",
				QueryVector: []float64{0.1, 0.2},
				K:           2,
				Field:       "vector",
			},
			wantResp: VectorSearchOutput{
				Status: "Successfully vector searched 2 documents",
				Result: VectorResult{
					IDs: []string{"mockID1", "mockID2"},
					Documents: []map[string]any{
						{
							"_index":  "index_name",
							"_id":     "mockID1",
							"_score":  1,
							"_source": map[string]any{"name": "a", "vector": []float64{0.1, 0.2}},
						},
						{
							"_index":  "index_name",
							"_id":     "mockID2",
							"_score":  0.5,
							"_source": map[string]any{"name": "b", "vector": []float64{0.2, 0.3}},
						},
					},
					Vectors: [][]float64{{0.1, 0.2}, {0.2, 0.3}},
					Metadata: []map[string]any{
						{"name": "a"},
						{"name": "b"},
					},
				},
			},
			task: TaskVectorSearch,
		},
	}

	for _, tc := range testcases {
		c.Run(tc.name, func(c *qt.C) {
			setup, err := structpb.NewStruct(map[string]any{
				"api-key":  "mock-api-key",
				"cloud-id": "mock-cloud-id",
			})
			c.Assert(err, qt.IsNil)

			e := &execution{
				ComponentExecution: base.ComponentExecution{Component: component, SystemVariables: nil, Setup: setup, Task: tc.task},
				client: ESClient{
					searchClient: func(o ...func(*esapi.SearchRequest)) (*esapi.Response, error) {
						return MockESVectorSearch(tc.wantResp), nil
					},
					sqlTranslateClient: func(body io.Reader, o ...func(*esapi.SQLTranslateRequest)) (*esapi.Response, error) {
						return MockESSQLTranslate(), nil
					},
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

func TestComponent_ExecuteIndexTask(t *testing.T) {
	c := qt.New(t)
	ctx := context.Background()
	bc := base.Component{Logger: zap.NewNop()}
	component := Init(bc)

	testcases := []struct {
		name     string
		input    IndexInput
		wantResp IndexOutput
		wantErr  string
	}{
		{
			name: "ok to index",
			input: IndexInput{
				IndexName: "index_name",
				Data:      map[string]any{"name": "John Doe", "email": "john@example.com"},
			},
			wantResp: IndexOutput{
				Status: "Successfully indexed 1 document",
			},
		},
	}

	for _, tc := range testcases {
		c.Run(tc.name, func(c *qt.C) {
			setup, err := structpb.NewStruct(map[string]any{
				"api-key":  "mock-api-key",
				"cloud-id": "mock-cloud-id",
			})
			c.Assert(err, qt.IsNil)

			e := &execution{
				ComponentExecution: base.ComponentExecution{Component: component, SystemVariables: nil, Setup: setup, Task: TaskIndex},
				client: ESClient{
					indexClient: func(index string, body io.Reader, o ...func(*esapi.IndexRequest)) (*esapi.Response, error) {
						return MockESIndex(tc.wantResp), nil
					},
				},
			}

			e.execute = e.index

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

func TestComponent_ExecuteMultiIndexTask(t *testing.T) {
	c := qt.New(t)
	ctx := context.Background()
	bc := base.Component{Logger: zap.NewNop()}
	component := Init(bc)

	testcases := []struct {
		name     string
		input    MultiIndexInput
		wantResp MultiIndexOutput
		wantErr  string
	}{
		{
			name: "ok to multi index",
			input: MultiIndexInput{
				IndexName: "index_name",
				ArrayData: []map[string]any{
					{"name": "John Doe", "email": "john@example.com"},
					{"name": "Jane Smith", "email": "jane@example.com"},
				},
			},
			wantResp: MultiIndexOutput{
				Status: "Successfully indexed 2 documents",
			},
		},
	}

	for _, tc := range testcases {
		c.Run(tc.name, func(c *qt.C) {
			setup, err := structpb.NewStruct(map[string]any{
				"api-key":  "mock-api",
				"cloud-id": "mock-cloud-id",
			})
			c.Assert(err, qt.IsNil)

			e := &execution{
				ComponentExecution: base.ComponentExecution{Component: component, SystemVariables: nil, Setup: setup, Task: TaskMultiIndex},
				client: ESClient{
					bulkClient: func(body io.Reader, o ...func(*esapi.BulkRequest)) (*esapi.Response, error) {
						return MockESMultiIndex(tc.wantResp), nil
					},
				},
			}

			e.execute = e.multiIndex

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
				IndexName: "index_name",
				FilterSQL: "name = 'John Doe' AND city = 'New York'",
				Update:    map[string]any{"name": "Pablo Vereira", "city": "Los Angeles"},
			},
			wantResp: UpdateOutput{
				Status: "Successfully updated 1 documents",
			},
		},
	}

	for _, tc := range testcases {
		c.Run(tc.name, func(c *qt.C) {
			setup, err := structpb.NewStruct(map[string]any{
				"api-key":  "mock-api-key",
				"cloud-id": "mock-cloud-id",
			})
			c.Assert(err, qt.IsNil)

			e := &execution{
				ComponentExecution: base.ComponentExecution{Component: component, SystemVariables: nil, Setup: setup, Task: TaskUpdate},
				client: ESClient{
					updateClient: func(index []string, o ...func(*esapi.UpdateByQueryRequest)) (*esapi.Response, error) {
						return MockESUpdate(tc.wantResp), nil
					},
					sqlTranslateClient: func(body io.Reader, o ...func(*esapi.SQLTranslateRequest)) (*esapi.Response, error) {
						return MockESSQLTranslate(), nil
					},
				},
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

func TestComponent_ExecuteDeleteTask(t *testing.T) {
	c := qt.New(t)
	ctx := context.Background()
	bc := base.Component{Logger: zap.NewNop()}
	component := Init(bc)

	testcases := []struct {
		name     string
		input    DeleteInput
		wantResp DeleteOutput
		wantErr  string
	}{
		{
			name: "ok to delete",
			input: DeleteInput{
				IndexName: "index_name",
				FilterSQL: "name = 'John Doe' AND city = 'New York'",
			},
			wantResp: DeleteOutput{
				Status: "Successfully deleted 1 documents",
			},
		},
	}

	for _, tc := range testcases {
		c.Run(tc.name, func(c *qt.C) {
			setup, err := structpb.NewStruct(map[string]any{
				"api-key":  "mock-api-key",
				"cloud-id": "mock-cloud-id",
			})
			c.Assert(err, qt.IsNil)

			e := &execution{
				ComponentExecution: base.ComponentExecution{Component: component, SystemVariables: nil, Setup: setup, Task: TaskDelete},
				client: ESClient{
					deleteClient: func(index []string, body io.Reader, o ...func(*esapi.DeleteByQueryRequest)) (*esapi.Response, error) {
						return MockESDelete(tc.wantResp), nil
					},
					sqlTranslateClient: func(body io.Reader, o ...func(*esapi.SQLTranslateRequest)) (*esapi.Response, error) {
						return MockESSQLTranslate(), nil
					},
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

func TestComponent_ExecuteCreateIndexTask(t *testing.T) {
	c := qt.New(t)
	ctx := context.Background()
	bc := base.Component{Logger: zap.NewNop()}
	component := Init(bc)

	testcases := []struct {
		name     string
		input    CreateIndexInput
		wantResp CreateIndexOutput
		wantErr  string
	}{
		{
			name: "ok to create index",
			input: CreateIndexInput{
				IndexName: "index_name",
				Mappings:  map[string]any{"name": "text", "email": "text", "vector": map[string]any{"type": "dense_vector", "dims": 2}},
			},
			wantResp: CreateIndexOutput{
				Status: "Successfully created 1 index",
			},
		},
	}

	for _, tc := range testcases {
		c.Run(tc.name, func(c *qt.C) {
			setup, err := structpb.NewStruct(map[string]any{
				"api-key":  "mock-api-key",
				"cloud-id": "mock-cloud-id",
			})
			c.Assert(err, qt.IsNil)

			e := &execution{
				ComponentExecution: base.ComponentExecution{Component: component, SystemVariables: nil, Setup: setup, Task: TaskCreateIndex},
				client: ESClient{
					createIndexClient: func(index string, o ...func(*esapi.IndicesCreateRequest)) (*esapi.Response, error) {
						return MockESCreateIndex(tc.wantResp), nil
					},
				},
			}

			e.execute = e.createIndex

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

func TestComponent_ExecuteDeleteIndexTask(t *testing.T) {
	c := qt.New(t)
	ctx := context.Background()
	bc := base.Component{Logger: zap.NewNop()}
	component := Init(bc)

	testcases := []struct {
		name     string
		input    DeleteIndexInput
		wantResp DeleteIndexOutput
		wantErr  string
	}{
		{
			name: "ok to delete index",
			input: DeleteIndexInput{
				IndexName: "index_name",
			},
			wantResp: DeleteIndexOutput{
				Status: "Successfully deleted 1 index",
			},
		},
	}

	for _, tc := range testcases {
		c.Run(tc.name, func(c *qt.C) {
			setup, err := structpb.NewStruct(map[string]any{
				"api-key":  "mock-api-key",
				"cloud-id": "mock-cloud-id",
			})
			c.Assert(err, qt.IsNil)

			e := &execution{
				ComponentExecution: base.ComponentExecution{Component: component, SystemVariables: nil, Setup: setup, Task: TaskDeleteIndex},
				client: ESClient{
					deleteIndexClient: func(index []string, o ...func(*esapi.IndicesDeleteRequest)) (*esapi.Response, error) {
						return MockESDeleteIndex(tc.wantResp), nil
					},
				},
			}

			e.execute = e.deleteIndex

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
