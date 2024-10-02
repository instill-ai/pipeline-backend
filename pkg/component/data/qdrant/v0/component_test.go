package qdrant

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/mock"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util/httpclient"
)

func TestComponent_ExecuteVectorSearchTask(t *testing.T) {
	c := qt.New(t)
	ctx := context.Background()
	bc := base.Component{Logger: zap.NewNop()}
	cmp := Init(bc)

	testcases := []struct {
		name     string
		input    VectorSearchInput
		wantResp VectorSearchOutput
		wantErr  string

		wantClientPath string
		wantClientReq  any
		clientResp     string
	}{
		{
			name: "ok to vector search",
			input: VectorSearchInput{
				CollectionName: "mock-collection",
				Vector:         []float64{0.1, 0.2},
				Limit:          2,
			},
			wantResp: VectorSearchOutput{
				Status: "Successfully vector searched 2 points",
				Result: Result{
					Ids: []string{"mockID1", "mockID2"},
					Points: []map[string]any{
						{"id": "mockID1", "version": 1, "score": 0.1, "name": "a", "vector": []float64{0.1, 0.2}},
						{"id": "mockID2", "version": 1, "score": 0.2, "name": "b", "vector": []float64{0.2, 0.3}},
					},
					Vectors: [][]float64{{0.1, 0.2}, {0.2, 0.3}},
					Metadata: []map[string]any{
						{"name": "a"},
						{"name": "b"},
					},
				},
			},
			wantClientPath: fmt.Sprintf(vectorSearchPath, "mock-collection"),
			wantClientReq: VectorSearchReq{
				Vector:     []float64{0.1, 0.2},
				Limit:      2,
				Payloads:   true,
				Filter:     map[string]any{},
				Params:     map[string]any{},
				MinScore:   0,
				WithVector: true,
			},
			clientResp: `{
				"time": 0.1,
				"status": "ok",
				"result": [
					{
						"id": "mockID1",
						"version": 1,
						"score": 0.1,
						"payload": {"name": "a"},
						"vector": [0.1, 0.2]
					},
					{
						"id": "mockID2",
						"version": 1,
						"score": 0.2,
						"payload": {"name": "b"},
						"vector": [0.2, 0.3]
					}
				]
			}`,
		},
	}

	for _, tc := range testcases {
		c.Run(tc.name, func(c *qt.C) {
			h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				c.Check(r.Method, qt.Equals, http.MethodPost)
				c.Check(r.URL.Path, qt.Equals, tc.wantClientPath)

				c.Check(r.Header.Get("Content-Type"), qt.Equals, httpclient.MIMETypeJSON)
				c.Check(r.Header.Get("Accept"), qt.Equals, httpclient.MIMETypeJSON)
				c.Check(r.Header.Get("api-key"), qt.Equals, "mock-api-key")

				c.Assert(r.Body, qt.IsNotNil)
				defer r.Body.Close()

				body, err := io.ReadAll(r.Body)
				c.Assert(err, qt.IsNil)
				c.Check(body, qt.JSONEquals, tc.wantClientReq)

				w.Header().Set("Content-Type", httpclient.MIMETypeJSON)
				fmt.Fprintln(w, tc.clientResp)
			})

			qdrantServer := httptest.NewServer(h)
			c.Cleanup(qdrantServer.Close)

			setup, _ := structpb.NewStruct(map[string]any{
				"api-key": "mock-api-key",
				"url":     qdrantServer.URL,
			})

			exec, err := cmp.CreateExecution(base.ComponentExecution{
				Component: cmp,
				Setup:     setup,
				Task:      TaskVectorSearch,
			})
			c.Assert(err, qt.IsNil)

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
			eh.ErrorMock.Optional()

			err = exec.Execute(ctx, []*base.Job{job})
			c.Check(err, qt.IsNil)

		})
	}
}

func TestComponent_ExecuteDeleteTask(t *testing.T) {
	c := qt.New(t)
	ctx := context.Background()
	bc := base.Component{Logger: zap.NewNop()}
	cmp := Init(bc)

	testcases := []struct {
		name     string
		input    DeleteInput
		wantResp DeleteOutput
		wantErr  string

		wantClientPath string
		wantClientReq  any
		clientResp     string
	}{
		{
			name: "ok to delete search",
			input: DeleteInput{
				CollectionName: "mock-collection",
				Ordering:       "weak",
				Filter: map[string]any{
					"must": map[string]any{
						"key":   "name",
						"match": map[string]any{"value": "a"},
					},
				},
			},
			wantResp: DeleteOutput{
				Status: "Successfully deleted points",
			},
			wantClientPath: fmt.Sprintf(deletePath, "mock-collection", "weak"),
			wantClientReq: DeleteReq{
				Filter: map[string]any{
					"must": map[string]any{
						"key":   "name",
						"match": map[string]any{"value": "a"},
					},
				},
			},
			clientResp: `{
				"time": 0.1,
				"status": "ok",
				"result": {
					"status": "COMPLETED",
					"total": 2
				}
			}`,
		},
	}

	for _, tc := range testcases {
		c.Run(tc.name, func(c *qt.C) {
			h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				c.Check(r.Method, qt.Equals, http.MethodPost)
				c.Check(r.URL.Path+"?wait=true&ordering=weak", qt.Equals, tc.wantClientPath)

				c.Check(r.Header.Get("Content-Type"), qt.Equals, httpclient.MIMETypeJSON)
				c.Check(r.Header.Get("Accept"), qt.Equals, httpclient.MIMETypeJSON)
				c.Check(r.Header.Get("api-key"), qt.Equals, "mock-api-key")

				c.Assert(r.Body, qt.IsNotNil)
				defer r.Body.Close()

				body, err := io.ReadAll(r.Body)
				c.Assert(err, qt.IsNil)
				c.Check(body, qt.JSONEquals, tc.wantClientReq)

				w.Header().Set("Content-Type", httpclient.MIMETypeJSON)
				fmt.Fprintln(w, tc.clientResp)
			})

			qdrantServer := httptest.NewServer(h)
			c.Cleanup(qdrantServer.Close)

			setup, _ := structpb.NewStruct(map[string]any{
				"api-key": "mock-api-key",
				"url":     qdrantServer.URL,
			})

			exec, err := cmp.CreateExecution(base.ComponentExecution{
				Component: cmp,
				Setup:     setup,
				Task:      TaskDelete,
			})

			c.Assert(err, qt.IsNil)

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
			eh.ErrorMock.Optional()

			err = exec.Execute(ctx, []*base.Job{job})
			c.Check(err, qt.IsNil)

		})
	}
}

func TestComponent_ExecuteUpsertTask(t *testing.T) {
	c := qt.New(t)
	ctx := context.Background()
	bc := base.Component{Logger: zap.NewNop()}
	cmp := Init(bc)

	testcases := []struct {
		name     string
		input    UpsertInput
		wantResp UpsertOutput
		wantErr  string

		wantClientPath string
		wantClientReq  any
		clientResp     string
	}{
		{
			name: "ok to upsert search",
			input: UpsertInput{
				CollectionName: "mock-collection",
				Metadata: map[string]any{
					"name": "a",
				},
				Vector: []float64{0.1, 0.2},
			},
			wantResp: UpsertOutput{
				Status: "Successfully upserted 1 point",
			},
			wantClientPath: fmt.Sprintf(batchUpsertPath, "mock-collection", "weak"),
			wantClientReq: BatchUpsertReq{
				Batch: Batch{
					IDs:     []string{""},
					Vectors: [][]float64{{0.1, 0.2}},
					Payloads: []map[string]any{
						{"name": "a"},
					},
				},
			},
			clientResp: `{
				"time": 0.1,
				"status": "ok",
				"result": {
					"total": 2,
					"success": 2,
					"fail": 0
				}
			}`,
		},
	}

	for _, tc := range testcases {
		c.Run(tc.name, func(c *qt.C) {
			h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				c.Check(r.Method, qt.Equals, http.MethodPut)
				c.Check(r.URL.Path+"?wait=true&ordering=weak", qt.Equals, tc.wantClientPath)

				c.Check(r.Header.Get("Content-Type"), qt.Equals, httpclient.MIMETypeJSON)
				c.Check(r.Header.Get("Accept"), qt.Equals, httpclient.MIMETypeJSON)
				c.Check(r.Header.Get("api-key"), qt.Equals, "mock-api-key")

				c.Assert(r.Body, qt.IsNotNil)
				defer r.Body.Close()

				body, err := io.ReadAll(r.Body)
				c.Assert(err, qt.IsNil)
				c.Check(body, qt.JSONEquals, tc.wantClientReq)

				w.Header().Set("Content-Type", httpclient.MIMETypeJSON)
				fmt.Fprintln(w, tc.clientResp)
			})

			qdrantServer := httptest.NewServer(h)
			c.Cleanup(qdrantServer.Close)

			setup, _ := structpb.NewStruct(map[string]any{
				"api-key": "mock-api-key",
				"url":     qdrantServer.URL,
			})

			exec, err := cmp.CreateExecution(base.ComponentExecution{
				Component: cmp,
				Setup:     setup,
				Task:      TaskUpsert,
			})

			c.Assert(err, qt.IsNil)

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
			eh.ErrorMock.Optional()

			err = exec.Execute(ctx, []*base.Job{job})
			c.Check(err, qt.IsNil)

		})
	}
}

func TestComponent_ExecuteBatchUpsertTask(t *testing.T) {
	c := qt.New(t)
	ctx := context.Background()
	bc := base.Component{Logger: zap.NewNop()}
	cmp := Init(bc)

	testcases := []struct {
		name     string
		input    BatchUpsertInput
		wantResp BatchUpsertOutput
		wantErr  string

		wantClientPath string
		wantClientReq  any
		clientResp     string
	}{
		{
			name: "ok to batch upsert search",
			input: BatchUpsertInput{
				CollectionName: "mock-collection",
				ArrayMetadata: []map[string]any{
					{"name": "a"},
					{"name": "b"},
				},
				ArrayVector: [][]float64{{0.1, 0.2}, {0.2, 0.3}},
			},
			wantResp: BatchUpsertOutput{
				Status: "Successfully batch upserted 2 points",
			},
			wantClientPath: fmt.Sprintf(batchUpsertPath, "mock-collection", "weak"),
			wantClientReq: BatchUpsertReq{
				Batch: Batch{
					IDs:     nil,
					Vectors: [][]float64{{0.1, 0.2}, {0.2, 0.3}},
					Payloads: []map[string]any{
						{"name": "a"},
						{"name": "b"},
					},
				},
			},
			clientResp: `{
				"time": 0.1,
				"status": "ok",
				"result": {
					"total": 2,
					"success": 2,
					"fail": 0
				}
			}`,
		},
	}

	for _, tc := range testcases {
		c.Run(tc.name, func(c *qt.C) {
			h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				c.Check(r.Method, qt.Equals, http.MethodPut)
				c.Check(r.URL.Path+"?wait=true&ordering=weak", qt.Equals, tc.wantClientPath)

				c.Check(r.Header.Get("Content-Type"), qt.Equals, httpclient.MIMETypeJSON)
				c.Check(r.Header.Get("Accept"), qt.Equals, httpclient.MIMETypeJSON)
				c.Check(r.Header.Get("api-key"), qt.Equals, "mock-api-key")

				c.Assert(r.Body, qt.IsNotNil)
				defer r.Body.Close()

				body, err := io.ReadAll(r.Body)
				c.Assert(err, qt.IsNil)
				c.Check(body, qt.JSONEquals, tc.wantClientReq)

				w.Header().Set("Content-Type", httpclient.MIMETypeJSON)
				fmt.Fprintln(w, tc.clientResp)
			})

			qdrantServer := httptest.NewServer(h)
			c.Cleanup(qdrantServer.Close)

			setup, _ := structpb.NewStruct(map[string]any{
				"api-key": "mock-api-key",
				"url":     qdrantServer.URL,
			})

			exec, err := cmp.CreateExecution(base.ComponentExecution{
				Component: cmp,
				Setup:     setup,
				Task:      TaskBatchUpsert,
			})

			c.Assert(err, qt.IsNil)

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
			eh.ErrorMock.Optional()

			err = exec.Execute(ctx, []*base.Job{job})
			c.Check(err, qt.IsNil)

		})
	}
}

func TestComponent_ExecuteCreateCollectionTask(t *testing.T) {
	c := qt.New(t)
	ctx := context.Background()
	bc := base.Component{Logger: zap.NewNop()}
	cmp := Init(bc)

	testcases := []struct {
		name     string
		input    CreateCollectionInput
		wantResp CreateCollectionOutput
		wantErr  string

		wantClientPath string
		wantClientReq  any
		clientResp     string
	}{
		{
			name: "ok to create collection",
			input: CreateCollectionInput{
				CollectionName: "mock-collection",
				Config: map[string]any{
					"vector": map[string]any{
						"size":     2,
						"distance": "cosine",
					},
				},
			},
			wantResp: CreateCollectionOutput{
				Status: "Successfully created 1 collection",
			},
			wantClientPath: fmt.Sprintf(createCollectionPath, "mock-collection"),
			wantClientReq: map[string]any{
				"vector": map[string]any{
					"size":     2,
					"distance": "cosine",
				},
			},
			clientResp: `{
				"time": 0.1,
				"status": "ok",
				"result": true
			}`,
		},
	}

	for _, tc := range testcases {
		c.Run(tc.name, func(c *qt.C) {
			h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				c.Check(r.Method, qt.Equals, http.MethodPut)
				c.Check(r.URL.Path, qt.Equals, tc.wantClientPath)

				c.Check(r.Header.Get("Content-Type"), qt.Equals, httpclient.MIMETypeJSON)
				c.Check(r.Header.Get("Accept"), qt.Equals, httpclient.MIMETypeJSON)
				c.Check(r.Header.Get("api-key"), qt.Equals, "mock-api-key")

				c.Assert(r.Body, qt.IsNotNil)
				defer r.Body.Close()

				body, err := io.ReadAll(r.Body)
				c.Assert(err, qt.IsNil)
				c.Check(body, qt.JSONEquals, tc.wantClientReq)

				w.Header().Set("Content-Type", httpclient.MIMETypeJSON)
				fmt.Fprintln(w, tc.clientResp)
			})

			qdrantServer := httptest.NewServer(h)
			c.Cleanup(qdrantServer.Close)

			setup, _ := structpb.NewStruct(map[string]any{
				"api-key": "mock-api-key",
				"url":     qdrantServer.URL,
			})

			exec, err := cmp.CreateExecution(base.ComponentExecution{
				Component: cmp,
				Setup:     setup,
				Task:      TaskCreateCollection,
			})

			c.Assert(err, qt.IsNil)

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
			eh.ErrorMock.Optional()

			err = exec.Execute(ctx, []*base.Job{job})
			c.Check(err, qt.IsNil)

		})
	}
}

func TestComponent_ExecuteDeleteCollectionTask(t *testing.T) {
	c := qt.New(t)
	ctx := context.Background()
	bc := base.Component{Logger: zap.NewNop()}
	cmp := Init(bc)

	testcases := []struct {
		name     string
		input    DeleteCollectionInput
		wantResp DeleteCollectionOutput
		wantErr  string

		wantClientPath string
		wantClientReq  map[string]any
		clientResp     string
	}{
		{
			name: "ok to delete collection",
			input: DeleteCollectionInput{
				CollectionName: "mock-collection",
			},
			wantResp: DeleteCollectionOutput{
				Status: "Successfully deleted 1 collection",
			},
			wantClientPath: fmt.Sprintf(deleteCollectionPath, "mock-collection"),
			wantClientReq:  map[string]any{},
			clientResp: `{
				"time": 0.1,
				"status": "ok",
				"result": true
			}`,
		},
	}

	for _, tc := range testcases {
		c.Run(tc.name, func(c *qt.C) {
			h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				c.Check(r.Method, qt.Equals, http.MethodDelete)
				c.Check(r.URL.Path, qt.Equals, tc.wantClientPath)

				c.Check(r.Header.Get("Content-Type"), qt.Equals, httpclient.MIMETypeJSON)
				c.Check(r.Header.Get("Accept"), qt.Equals, httpclient.MIMETypeJSON)
				c.Check(r.Header.Get("api-key"), qt.Equals, "mock-api-key")

				w.Header().Set("Content-Type", httpclient.MIMETypeJSON)
				fmt.Fprintln(w, tc.clientResp)
			})

			qdrantServer := httptest.NewServer(h)
			c.Cleanup(qdrantServer.Close)

			setup, _ := structpb.NewStruct(map[string]any{
				"api-key": "mock-api-key",
				"url":     qdrantServer.URL,
			})

			exec, err := cmp.CreateExecution(base.ComponentExecution{
				Component: cmp,
				Setup:     setup,
				Task:      TaskDeleteCollection,
			})

			c.Assert(err, qt.IsNil)

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
			eh.ErrorMock.Optional()

			err = exec.Execute(ctx, []*base.Job{job})
			c.Check(err, qt.IsNil)

		})
	}
}
