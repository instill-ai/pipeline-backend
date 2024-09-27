package milvus

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
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util/httpclient"
)

func TestComponent_ExecuteVectorSearchTask(t *testing.T) {
	c := qt.New(t)
	ctx := context.Background()
	bc := base.Component{Logger: zap.NewNop()}
	cmp := Init(bc)

	testcases := []struct {
		name     string
		input    SearchInput
		wantResp SearchOutput
		wantErr  string

		wantClientPath string
		wantClientReq  any
		clientResp     string
	}{
		{
			name: "ok to vector search",
			input: SearchInput{
				CollectionName: "mock-collection",
				Vector:         []float32{0.1, 0.2},
				VectorField:    "vector",
				Limit:          2,
			},
			wantResp: SearchOutput{
				Status: "Successfully searched 2 data",
				Result: Result{
					Ids:      []string{"mockID1", "mockID2"},
					Data:     []map[string]any{{"distance": 1, "id": "mockID1", "name": "a", "vector": []float32{0.1, 0.2}}, {"distance": 1, "id": "mockID2", "name": "b", "vector": []float32{0.2, 0.3}}},
					Vectors:  [][]float32{{0.1, 0.2}, {0.2, 0.3}},
					Metadata: []map[string]any{{"id": "mockID1", "name": "a", "vector": []float64{0.1, 0.2}}, {"id": "mockID2", "name": "b", "vector": []float64{0.2, 0.3}}},
				},
			},
			wantClientPath: searchPath,
			wantClientReq: SearchReq{
				CollectionName: "mock-collection",
				Data:           [][]float32{{0.1, 0.2}},
				AnnsField:      "vector",
				Limit:          2,
				OutputFields:   []string{"id", "name", "vector"},
			},
			clientResp: `{
				"code": 200,
				"data": [
					{"distance":1, "id": "mockID1", "name": "a", "vector": [0.1, 0.2]},
					{"distance":1, "id": "mockID2", "name": "b", "vector": [0.2, 0.3]}
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
				c.Check(r.Header.Get("Authorization"), qt.Equals, "Bearer mock-root:Milvus")

				c.Assert(r.Body, qt.IsNotNil)
				defer r.Body.Close()

				body, err := io.ReadAll(r.Body)
				c.Assert(err, qt.IsNil)
				c.Check(body, qt.JSONEquals, tc.wantClientReq)

				w.Header().Set("Content-Type", httpclient.MIMETypeJSON)
				fmt.Fprintln(w, tc.clientResp)
			})

			milvusServer := httptest.NewServer(h)
			c.Cleanup(milvusServer.Close)

			setup, _ := structpb.NewStruct(map[string]any{
				"username": "mock-root",
				"password": "Milvus",
				"url":      milvusServer.URL,
			})

			exec, err := cmp.CreateExecution(base.ComponentExecution{
				Component: cmp,
				Setup:     setup,
				Task:      TaskVectorSearch,
			})
			c.Assert(err, qt.IsNil)

			pbIn, err := base.ConvertToStructpb(tc.input)
			c.Assert(err, qt.IsNil)

			ir, ow, eh, job := base.GenerateMockJob(c)
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

			err = exec.Execute(ctx, []*base.Job{job})
			c.Assert(err, qt.IsNil)

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
				Filter:         "id like 'mockID1'",
			},
			wantResp: DeleteOutput{
				Status: "Successfully deleted data",
			},
			wantClientPath: deletePath,
			wantClientReq: DeleteReq{
				CollectionNameReq: "mock-collection",
				FilterReq:         "id like 'mockID1'",
			},
			clientResp: `{
				"code": 200,
				"data": {}
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
				c.Check(r.Header.Get("Authorization"), qt.Equals, "Bearer mock-root:Milvus")

				c.Assert(r.Body, qt.IsNotNil)
				defer r.Body.Close()

				body, err := io.ReadAll(r.Body)
				c.Assert(err, qt.IsNil)
				c.Check(body, qt.JSONEquals, tc.wantClientReq)

				w.Header().Set("Content-Type", httpclient.MIMETypeJSON)
				fmt.Fprintln(w, tc.clientResp)
			})

			milvusServer := httptest.NewServer(h)
			c.Cleanup(milvusServer.Close)

			setup, _ := structpb.NewStruct(map[string]any{
				"username": "mock-root",
				"password": "Milvus",
				"url":      milvusServer.URL,
			})

			exec, err := cmp.CreateExecution(base.ComponentExecution{
				Component: cmp,
				Setup:     setup,
				Task:      TaskDelete,
			})

			c.Assert(err, qt.IsNil)

			pbIn, err := base.ConvertToStructpb(tc.input)
			c.Assert(err, qt.IsNil)

			ir, ow, eh, job := base.GenerateMockJob(c)
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

			err = exec.Execute(ctx, []*base.Job{job})
			c.Assert(err, qt.IsNil)
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
				Data: map[string]any{
					"name":   "a",
					"vector": []float32{0.1, 0.2},
				},
			},
			wantResp: UpsertOutput{
				Status: "Successfully upserted 1 data",
			},
			wantClientPath: upsertPath,
			wantClientReq: UpsertReq{
				CollectionNameReq: "mock-collection",
				DataReq: []map[string]any{
					{"name": "a", "vector": []float32{0.1, 0.2}},
				},
			},
			clientResp: `{
				"code": 200,
				"data": {
					"upsertCount": 1,
					"upsertIds": ["mockID1"]
				}
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
				c.Check(r.Header.Get("Authorization"), qt.Equals, "Bearer mock-root:Milvus")

				c.Assert(r.Body, qt.IsNotNil)
				defer r.Body.Close()

				body, err := io.ReadAll(r.Body)
				c.Assert(err, qt.IsNil)
				c.Check(body, qt.JSONEquals, tc.wantClientReq)

				w.Header().Set("Content-Type", httpclient.MIMETypeJSON)
				fmt.Fprintln(w, tc.clientResp)
			})

			milvusServer := httptest.NewServer(h)
			c.Cleanup(milvusServer.Close)

			setup, _ := structpb.NewStruct(map[string]any{
				"username": "mock-root",
				"password": "Milvus",
				"url":      milvusServer.URL,
			})

			exec, err := cmp.CreateExecution(base.ComponentExecution{
				Component: cmp,
				Setup:     setup,
				Task:      TaskUpsert,
			})

			c.Assert(err, qt.IsNil)

			pbIn, err := base.ConvertToStructpb(tc.input)
			c.Assert(err, qt.IsNil)

			ir, ow, eh, job := base.GenerateMockJob(c)
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

			err = exec.Execute(ctx, []*base.Job{job})
			c.Assert(err, qt.IsNil)

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
				ArrayData: []map[string]any{
					{"vector": []float64{0.1, 0.2}, "name": "a"},
					{"vector": []float64{0.2, 0.3}, "name": "b"},
				},
			},
			wantResp: BatchUpsertOutput{
				Status: "Successfully batch upserted 2 data",
			},
			wantClientPath: upsertPath,
			wantClientReq: UpsertReq{
				CollectionNameReq: "mock-collection",
				DataReq: []map[string]any{
					{"vector": []float64{0.1, 0.2}, "name": "a"},
					{"vector": []float64{0.2, 0.3}, "name": "b"},
				},
			},
			clientResp: `{
				"code": 200,
				"data": {
					"upsertCount": 2,
					"upsertIds": ["mockID1", "mockID2"]
				}
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
				c.Check(r.Header.Get("Authorization"), qt.Equals, "Bearer mock-root:Milvus")

				c.Assert(r.Body, qt.IsNotNil)
				defer r.Body.Close()

				body, err := io.ReadAll(r.Body)
				c.Assert(err, qt.IsNil)
				c.Check(body, qt.JSONEquals, tc.wantClientReq)

				w.Header().Set("Content-Type", httpclient.MIMETypeJSON)
				fmt.Fprintln(w, tc.clientResp)
			})

			milvusServer := httptest.NewServer(h)
			c.Cleanup(milvusServer.Close)

			setup, _ := structpb.NewStruct(map[string]any{
				"username": "mock-root",
				"password": "Milvus",
				"url":      milvusServer.URL,
			})

			exec, err := cmp.CreateExecution(base.ComponentExecution{
				Component: cmp,
				Setup:     setup,
				Task:      TaskBatchUpsert,
			})

			c.Assert(err, qt.IsNil)

			pbIn, err := base.ConvertToStructpb(tc.input)
			c.Assert(err, qt.IsNil)

			ir, ow, eh, job := base.GenerateMockJob(c)
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

			err = exec.Execute(ctx, []*base.Job{job})
			c.Assert(err, qt.IsNil)

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
				Dimension:      2,
			},
			wantResp: CreateCollectionOutput{
				Status: "Successfully created 1 collection",
			},
			wantClientPath: createCollectionPath,
			wantClientReq: CreateCollectionReq{
				CollectionNameReq: "mock-collection",
				DimensionReq:      2,
			},
			clientResp: `{
				"code": 200,
				"data": {}
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
				c.Check(r.Header.Get("Authorization"), qt.Equals, "Bearer mock-root:Milvus")

				c.Assert(r.Body, qt.IsNotNil)
				defer r.Body.Close()

				body, err := io.ReadAll(r.Body)
				c.Assert(err, qt.IsNil)
				c.Check(body, qt.JSONEquals, tc.wantClientReq)

				w.Header().Set("Content-Type", httpclient.MIMETypeJSON)
				fmt.Fprintln(w, tc.clientResp)
			})

			milvusServer := httptest.NewServer(h)
			c.Cleanup(milvusServer.Close)

			setup, _ := structpb.NewStruct(map[string]any{
				"username": "mock-root",
				"password": "Milvus",
				"url":      milvusServer.URL,
			})

			exec, err := cmp.CreateExecution(base.ComponentExecution{
				Component: cmp,
				Setup:     setup,
				Task:      TaskCreateCollection,
			})

			c.Assert(err, qt.IsNil)

			pbIn, err := base.ConvertToStructpb(tc.input)
			c.Assert(err, qt.IsNil)

			ir, ow, eh, job := base.GenerateMockJob(c)
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

			err = exec.Execute(ctx, []*base.Job{job})
			c.Assert(err, qt.IsNil)

		})
	}
}

func TestComponent_ExecuteDropCollectionTask(t *testing.T) {
	c := qt.New(t)
	ctx := context.Background()
	bc := base.Component{Logger: zap.NewNop()}
	cmp := Init(bc)

	testcases := []struct {
		name     string
		input    DropCollectionInput
		wantResp DropCollectionOutput
		wantErr  string

		wantClientPath string
		wantClientReq  DropCollectionReq
		clientResp     string
	}{
		{
			name: "ok to delete collection",
			input: DropCollectionInput{
				CollectionName: "mock-collection",
			},
			wantResp: DropCollectionOutput{
				Status: "Successfully dropped 1 collection",
			},
			wantClientPath: dropCollectionPath,
			wantClientReq: DropCollectionReq{
				CollectionNameReq: "mock-collection",
			},
			clientResp: `{
				"code": 200,
				"data": {}
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
				c.Check(r.Header.Get("Authorization"), qt.Equals, "Bearer mock-root:Milvus")

				w.Header().Set("Content-Type", httpclient.MIMETypeJSON)
				fmt.Fprintln(w, tc.clientResp)
			})

			milvusServer := httptest.NewServer(h)
			c.Cleanup(milvusServer.Close)

			setup, _ := structpb.NewStruct(map[string]any{
				"username": "mock-root",
				"password": "Milvus",
				"url":      milvusServer.URL,
			})

			exec, err := cmp.CreateExecution(base.ComponentExecution{
				Component: cmp,
				Setup:     setup,
				Task:      TaskDropCollection,
			})

			c.Assert(err, qt.IsNil)

			pbIn, err := base.ConvertToStructpb(tc.input)
			c.Assert(err, qt.IsNil)

			ir, ow, eh, job := base.GenerateMockJob(c)
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

			err = exec.Execute(ctx, []*base.Job{job})
			c.Assert(err, qt.IsNil)

		})
	}
}

func TestComponent_ExecuteCreatePartitionTask(t *testing.T) {
	c := qt.New(t)
	ctx := context.Background()
	bc := base.Component{Logger: zap.NewNop()}
	cmp := Init(bc)

	testcases := []struct {
		name     string
		input    CreatePartitionInput
		wantResp CreatePartitionOutput
		wantErr  string

		wantClientPath string
		wantClientReq  any
		clientResp     string
	}{
		{
			name: "ok to create partition",
			input: CreatePartitionInput{
				CollectionName: "mock-collection",
				PartitionName:  "mock-partition",
			},
			wantResp: CreatePartitionOutput{
				Status: "Successfully created 1 partition",
			},
			wantClientPath: createPartitionPath,
			wantClientReq: CreatePartitionReq{
				CollectionNameReq: "mock-collection",
				PartitionNameReq:  "mock-partition",
			},
			clientResp: `{
				"code": 200,
				"data": {}
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
				c.Check(r.Header.Get("Authorization"), qt.Equals, "Bearer mock-root:Milvus")

				c.Assert(r.Body, qt.IsNotNil)
				defer r.Body.Close()

				body, err := io.ReadAll(r.Body)
				c.Assert(err, qt.IsNil)
				c.Check(body, qt.JSONEquals, tc.wantClientReq)

				w.Header().Set("Content-Type", httpclient.MIMETypeJSON)
				fmt.Fprintln(w, tc.clientResp)
			})

			milvusServer := httptest.NewServer(h)
			c.Cleanup(milvusServer.Close)

			setup, _ := structpb.NewStruct(map[string]any{
				"username": "mock-root",
				"password": "Milvus",
				"url":      milvusServer.URL,
			})

			exec, err := cmp.CreateExecution(base.ComponentExecution{
				Component: cmp,
				Setup:     setup,
				Task:      TaskCreatePartition,
			})

			c.Assert(err, qt.IsNil)

			pbIn, err := base.ConvertToStructpb(tc.input)
			c.Assert(err, qt.IsNil)

			ir, ow, eh, job := base.GenerateMockJob(c)
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

			err = exec.Execute(ctx, []*base.Job{job})
			c.Assert(err, qt.IsNil)

		})
	}
}

func TestComponent_ExecuteDropPartitionTask(t *testing.T) {
	c := qt.New(t)
	ctx := context.Background()
	bc := base.Component{Logger: zap.NewNop()}
	cmp := Init(bc)

	testcases := []struct {
		name     string
		input    DropPartitionInput
		wantResp DropPartitionOutput
		wantErr  string

		wantClientPath string
		wantClientReq  any
		clientResp     string
	}{
		{
			name: "ok to delete partition",
			input: DropPartitionInput{
				CollectionName: "mock-collection",
				PartitionName:  "mock-partition",
			},
			wantResp: DropPartitionOutput{
				Status: "Successfully dropped 1 partition",
			},
			wantClientPath: dropPartitionPath,
			wantClientReq: DropPartitionReq{
				CollectionNameReq: "mock-collection",
				PartitionNameReq:  "mock-partition",
			},
			clientResp: `{
				"code": 200,
				"data": {}
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
				c.Check(r.Header.Get("Authorization"), qt.Equals, "Bearer mock-root:Milvus")

				c.Assert(r.Body, qt.IsNotNil)
				defer r.Body.Close()

				body, err := io.ReadAll(r.Body)
				c.Assert(err, qt.IsNil)
				c.Check(body, qt.JSONEquals, tc.wantClientReq)

				w.Header().Set("Content-Type", httpclient.MIMETypeJSON)
				fmt.Fprintln(w, tc.clientResp)
			})

			milvusServer := httptest.NewServer(h)
			c.Cleanup(milvusServer.Close)

			setup, _ := structpb.NewStruct(map[string]any{
				"username": "mock-root",
				"password": "Milvus",
				"url":      milvusServer.URL,
			})

			exec, err := cmp.CreateExecution(base.ComponentExecution{
				Component: cmp,
				Setup:     setup,
				Task:      TaskDropPartition,
			})

			c.Assert(err, qt.IsNil)

			pbIn, err := base.ConvertToStructpb(tc.input)
			c.Assert(err, qt.IsNil)

			ir, ow, eh, job := base.GenerateMockJob(c)
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

			err = exec.Execute(ctx, []*base.Job{job})
			c.Assert(err, qt.IsNil)

		})
	}
}

func TestComponent_ExecuteCreateIndexTask(t *testing.T) {
	c := qt.New(t)
	ctx := context.Background()
	bc := base.Component{Logger: zap.NewNop()}
	cmp := Init(bc)

	testcases := []struct {
		name     string
		input    CreateIndexInput
		wantResp CreateIndexOutput
		wantErr  string

		wantClientPath string
		wantClientReq  any
		clientResp     string
	}{
		{
			name: "ok to create index",
			input: CreateIndexInput{
				CollectionName: "mock-collection",
				IndexParams: map[string]any{
					"metricType": "L2",
					"fieldName":  "my_vector",
					"indexName":  "my_vector",
					"indexConfig": map[string]any{
						"index_type": "IVF_FLAT",
						"nlist":      "1024",
					},
				},
			},
			wantResp: CreateIndexOutput{
				Status: "Successfully created 1 index",
			},
			wantClientPath: createIndexPath,
			wantClientReq: CreateIndexReq{
				CollectionName: "mock-collection",
				IndexParams: []map[string]any{{"metricType": "L2",
					"fieldName": "my_vector",
					"indexName": "my_vector",
					"indexConfig": map[string]any{
						"index_type": "IVF_FLAT",
						"nlist":      "1024",
					}}},
			},
			clientResp: `{
				"code": 200,
				"data": {}
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
				c.Check(r.Header.Get("Authorization"), qt.Equals, "Bearer mock-root:Milvus")

				c.Assert(r.Body, qt.IsNotNil)
				defer r.Body.Close()

				body, err := io.ReadAll(r.Body)
				c.Assert(err, qt.IsNil)
				c.Check(body, qt.JSONEquals, tc.wantClientReq)

				w.Header().Set("Content-Type", httpclient.MIMETypeJSON)
				fmt.Fprintln(w, tc.clientResp)
			})

			milvusServer := httptest.NewServer(h)
			c.Cleanup(milvusServer.Close)

			setup, _ := structpb.NewStruct(map[string]any{
				"username": "mock-root",
				"password": "Milvus",
				"url":      milvusServer.URL,
			})

			exec, err := cmp.CreateExecution(base.ComponentExecution{
				Component: cmp,
				Setup:     setup,
				Task:      TaskCreateIndex,
			})

			c.Assert(err, qt.IsNil)

			pbIn, err := base.ConvertToStructpb(tc.input)
			c.Assert(err, qt.IsNil)

			ir, ow, eh, job := base.GenerateMockJob(c)
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

			err = exec.Execute(ctx, []*base.Job{job})
			c.Assert(err, qt.IsNil)

		})
	}
}

func TestComponent_ExecuteDropIndexTask(t *testing.T) {
	c := qt.New(t)
	ctx := context.Background()
	bc := base.Component{Logger: zap.NewNop()}
	cmp := Init(bc)

	testcases := []struct {
		name     string
		input    DropIndexInput
		wantResp DropIndexOutput
		wantErr  string

		wantClientPath string
		wantClientReq  any
		clientResp     string
	}{
		{
			name: "ok to delete index",
			input: DropIndexInput{
				CollectionName: "mock-collection",
				IndexName:      "mock-index",
			},
			wantResp: DropIndexOutput{
				Status: "Successfully dropped 1 index",
			},
			wantClientPath: dropIndexPath,
			wantClientReq: DropIndexReq{
				CollectionNameReq: "mock-collection",
				IndexNameReq:      "mock-index",
			},
			clientResp: `{
				"code": 200,
				"data": {}
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
				c.Check(r.Header.Get("Authorization"), qt.Equals, "Bearer mock-root:Milvus")

				c.Assert(r.Body, qt.IsNotNil)
				defer r.Body.Close()

				body, err := io.ReadAll(r.Body)
				c.Assert(err, qt.IsNil)
				c.Check(body, qt.JSONEquals, tc.wantClientReq)

				w.Header().Set("Content-Type", httpclient.MIMETypeJSON)
				fmt.Fprintln(w, tc.clientResp)
			})

			milvusServer := httptest.NewServer(h)
			c.Cleanup(milvusServer.Close)

			setup, _ := structpb.NewStruct(map[string]any{
				"username": "mock-root",
				"password": "Milvus",
				"url":      milvusServer.URL,
			})

			exec, err := cmp.CreateExecution(base.ComponentExecution{
				Component: cmp,
				Setup:     setup,
				Task:      TaskDropIndex,
			})

			c.Assert(err, qt.IsNil)

			pbIn, err := base.ConvertToStructpb(tc.input)
			c.Assert(err, qt.IsNil)

			ir, ow, eh, job := base.GenerateMockJob(c)
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

			err = exec.Execute(ctx, []*base.Job{job})
			c.Assert(err, qt.IsNil)
		})
	}
}
