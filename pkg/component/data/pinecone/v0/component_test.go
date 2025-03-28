package pinecone

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pinecone-io/go-pinecone/pinecone"
	"google.golang.org/protobuf/types/known/structpb"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/mock"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util/httpclient"
	"github.com/instill-ai/pipeline-backend/pkg/data"
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
	"github.com/instill-ai/x/errmsg"
)

const (
	pineconeKey = "secret-key"
	namespace   = "pantone"
	threshold   = 0.9

	upsertOK = `{"upsertedCount": 2}`

	queryOK = `
{
	"namespace": "color-schemes",
	"matches": [
		{
			"id": "A",
			"values": [ 2.23 ],
			"metadata": { "color": "pumpkin" },
			"score": 0.99
		},
		{
			"id": "B",
			"values": [ 3.32 ],
			"metadata": { "color": "cerulean" },
			"score": 0.87
		}
	]
}`

	errResp = `

	{
	  "code": 3,
	  "message": "Cannot provide both ID and vector at the same time.",
	  "details": []
	}`
)

func newValue(in any) format.Value {
	v, _ := data.NewValue(in)
	return v
}

var (
	vectorA = vector{
		ID:       "A",
		Values:   []float32{2.23},
		Metadata: map[string]format.Value{"color": newValue("pumpkin")},
	}
	vectorB = vector{
		ID:       "B",
		Values:   []float32{3.32},
		Metadata: map[string]format.Value{"color": newValue("cerulean")},
	}
	queryByVector = queryInput{
		Namespace:       "color-schemes",
		TopK:            1,
		Vector:          vectorA.Values,
		IncludeValues:   true,
		IncludeMetadata: true,
		Filter: map[string]any{
			"color": map[string]any{
				"$in": []string{"green", "cerulean", "pumpkin"},
			},
		},
	}
	queryWithThreshold = func(q queryInput, th float64) queryInput {
		q.MinScore = th
		return q
	}
	queryByID = queryInput{
		Namespace:       "color-schemes",
		TopK:            1,
		Vector:          vectorA.Values,
		ID:              vectorA.ID,
		IncludeValues:   true,
		IncludeMetadata: true,
	}
)

func TestComponent_Execute(t *testing.T) {
	c := qt.New(t)
	ctx := context.Background()

	pvA, err := vectorA.toPinecone()
	c.Assert(err, qt.IsNil)
	pvB, err := vectorB.toPinecone()
	c.Assert(err, qt.IsNil)

	testcases := []struct {
		name string

		task     string
		execIn   any
		wantExec any

		wantClientPath string
		wantClientReq  any
		clientResp     string
	}{
		{
			name: "ok - upsert",

			task: taskBatchUpsert,
			execIn: taskBatchUpsertInput{
				Vectors:   []vector{vectorA, vectorB},
				Namespace: namespace,
			},
			wantExec: taskUpsertOutput{UpsertedCount: 2},

			wantClientPath: upsertPath,
			wantClientReq:  upsertReq{Vectors: []*pinecone.Vector{pvA, pvB}, Namespace: namespace},
			clientResp:     upsertOK,
		},
		{
			name: "ok - query by vector",

			task:   taskQuery,
			execIn: queryByVector,
			wantExec: queryResp{
				Namespace: "color-schemes",
				Matches: []match{
					{
						Vector: pvA,
						Score:  0.99,
					},
					{
						Vector: pvB,
						Score:  0.87,
					},
				},
			},

			wantClientPath: queryPath,
			wantClientReq:  queryByVector.asRequest(),
			clientResp:     queryOK,
		},
		{
			name: "ok - filter out below threshold score",

			task:   taskQuery,
			execIn: queryWithThreshold(queryByVector, threshold),
			wantExec: queryResp{
				Namespace: "color-schemes",
				Matches: []match{
					{
						Vector: pvA,
						Score:  0.99,
					},
				},
			},

			wantClientPath: queryPath,
			wantClientReq:  queryByVector.asRequest(),
			clientResp:     queryOK,
		},
		{
			name: "ok - query by ID",

			task:   taskQuery,
			execIn: queryByID,
			wantExec: queryResp{
				Namespace: "color-schemes",
				Matches: []match{
					{
						Vector: pvA,
						Score:  0.99,
					},
					{
						Vector: pvB,
						Score:  0.87,
					},
				},
			},

			wantClientPath: queryPath,
			wantClientReq: queryReq{
				// Vector is wiped from the request.
				Namespace:       "color-schemes",
				TopK:            1,
				ID:              vectorA.ID,
				IncludeValues:   true,
				IncludeMetadata: true,
			},
			clientResp: queryOK,
		},
	}

	bc := base.Component{}
	cmp := Init(bc)

	for _, tc := range testcases {
		c.Run(tc.name, func(c *qt.C) {
			h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// For now only POST methods are considered. When this changes,
				// this will need to be asserted per-path.
				c.Check(r.Method, qt.Equals, http.MethodPost)
				c.Check(r.URL.Path, qt.Equals, tc.wantClientPath)

				c.Check(r.Header.Get("Content-Type"), qt.Equals, httpclient.MIMETypeJSON)
				c.Check(r.Header.Get("Accept"), qt.Equals, httpclient.MIMETypeJSON)
				c.Check(r.Header.Get("Api-Key"), qt.Equals, pineconeKey)

				c.Assert(r.Body, qt.IsNotNil)
				defer r.Body.Close()

				body, err := io.ReadAll(r.Body)
				c.Assert(err, qt.IsNil)
				c.Check(body, qt.JSONEquals, tc.wantClientReq)

				w.Header().Set("Content-Type", httpclient.MIMETypeJSON)
				fmt.Fprintln(w, tc.clientResp)
			})

			pineconeServer := httptest.NewServer(h)
			c.Cleanup(pineconeServer.Close)

			setup, _ := structpb.NewStruct(map[string]any{
				"api-key": pineconeKey,
				"url":     pineconeServer.URL,
			})

			exec, err := cmp.CreateExecution(base.ComponentExecution{
				Component: cmp,
				Setup:     setup,
				Task:      tc.task,
			})
			c.Assert(err, qt.IsNil)

			wantJSON, err := json.Marshal(tc.wantExec)

			ir, ow, eh, job := mock.GenerateMockJob(c)
			c.Assert(err, qt.IsNil)

			switch tc.task {
			case taskBatchUpsert:
				ir.ReadDataMock.Set(func(ctx context.Context, in any) error {
					switch in := in.(type) {
					case *taskBatchUpsertInput:
						*in = tc.execIn.(taskBatchUpsertInput)
					}
					return nil
				})

				ow.WriteDataMock.Optional().Set(func(ctx context.Context, output any) error {
					c.Check(wantJSON, qt.JSONEquals, output)
					return nil
				})
			default:
				pbIn, err := base.ConvertToStructpb(tc.execIn)
				c.Assert(err, qt.IsNil)

				ir.ReadMock.Return(pbIn, nil)
				ow.WriteMock.Optional().Set(func(ctx context.Context, output *structpb.Struct) (err error) {
					c.Check(wantJSON, qt.JSONEquals, output.AsMap())
					return nil
				})
			}

			eh.ErrorMock.Optional()
			err = exec.Execute(ctx, []*base.Job{job})
			c.Assert(err, qt.IsNil)
		})
	}

	c.Run("nok - 400", func(c *qt.C) {
		h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintln(w, errResp)
		})

		pineconeServer := httptest.NewServer(h)
		c.Cleanup(pineconeServer.Close)

		setup, _ := structpb.NewStruct(map[string]any{
			"url": pineconeServer.URL,
		})

		exec, err := cmp.CreateExecution(base.ComponentExecution{
			Component: cmp,
			Setup:     setup,
			Task:      taskQuery,
		})
		c.Assert(err, qt.IsNil)

		pbIn := new(structpb.Struct)
		ir, ow, eh, job := mock.GenerateMockJob(c)
		ir.ReadMock.Return(pbIn, nil)
		ow.WriteMock.Optional().Return(nil)
		eh.ErrorMock.Optional().Set(func(ctx context.Context, err error) {
			want := "Pinecone responded with a 400 status code. Cannot provide both ID and vector at the same time."
			c.Check(errmsg.Message(err), qt.Equals, want)
		})

		err = exec.Execute(ctx, []*base.Job{job})
		c.Check(err, qt.IsNil)

	})

	c.Run("nok - URL misconfiguration", func(c *qt.C) {
		setup, _ := structpb.NewStruct(map[string]any{
			"url": "http://no-such.host",
		})

		exec, err := cmp.CreateExecution(base.ComponentExecution{
			Component: cmp,
			Setup:     setup,
			Task:      taskQuery,
		})
		c.Assert(err, qt.IsNil)

		pbIn := new(structpb.Struct)
		ir, ow, eh, job := mock.GenerateMockJob(c)
		ir.ReadMock.Return(pbIn, nil)
		ow.WriteMock.Optional().Return(nil)
		eh.ErrorMock.Optional().Set(func(ctx context.Context, err error) {
			want := "Failed to call http://no-such.host/.*. Please check that the component configuration is correct."
			c.Check(errmsg.Message(err), qt.Matches, want)
		})

		err = exec.Execute(ctx, []*base.Job{job})
		c.Check(err, qt.IsNil)

	})
}
