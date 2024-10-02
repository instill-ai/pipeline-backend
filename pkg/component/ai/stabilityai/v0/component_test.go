package stabilityai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"google.golang.org/protobuf/types/known/structpb"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/mock"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util/httpclient"
	"github.com/instill-ai/x/errmsg"
)

const (
	apiKey        = "123"
	instillSecret = "instill-credential-key"
	errResp       = `
{
  "id": "6e958442e7911ffb2e0bf89c6efe804f",
  "message": "Incorrect API key provided",
  "name": "unauthorized"
}`

	okResp = `
{
  "artifacts": [
    {
      "base64": "a",
      "seed": 1234,
      "finishReason": "SUCCESS"
    }
  ]
}
`
)

func TestComponent_ExecuteImageFromText(t *testing.T) {
	c := qt.New(t)
	ctx := context.Background()

	weight := 0.5
	text := "a cat and a dog"
	engine := "engine"

	bc := base.Component{}
	cmp := Init(bc).WithInstillCredentials(map[string]any{"apikey": instillSecret})

	testcases := []struct {
		name      string
		gotStatus int
		gotResp   string
		wantResp  TextToImageOutput
		wantErr   string
	}{
		{
			name:      "ok - 200",
			gotStatus: http.StatusOK,
			gotResp:   okResp,
			wantResp: TextToImageOutput{
				Images: []string{"data:image/png;base64,a"},
				Seeds:  []uint32{1234},
			},
		},
		{
			name:      "nok - 401",
			gotStatus: http.StatusUnauthorized,
			gotResp:   errResp,
			wantErr:   "Stability AI responded with a 401 status code. Incorrect API key provided",
		},
	}

	for _, tc := range testcases {
		c.Run(tc.name, func(c *qt.C) {
			h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				c.Check(r.Method, qt.Equals, http.MethodPost)
				c.Check(r.URL.Path, qt.Matches, `/v1/generation/.*/text-to-image`)

				c.Check(r.Header.Get("Authorization"), qt.Equals, "Bearer "+apiKey)
				c.Check(r.Header.Get("Content-Type"), qt.Equals, httpclient.MIMETypeJSON)

				w.Header().Set("Content-Type", httpclient.MIMETypeJSON)
				w.WriteHeader(tc.gotStatus)
				fmt.Fprintln(w, tc.gotResp)
			})

			srv := httptest.NewServer(h)
			c.Cleanup(srv.Close)

			setup, err := structpb.NewStruct(map[string]any{
				"base-path": srv.URL,
				"api-key":   apiKey,
			})
			c.Assert(err, qt.IsNil)

			exec, err := cmp.CreateExecution(base.ComponentExecution{
				Component: cmp,
				Setup:     setup,
				Task:      TextToImageTask,
			})
			c.Assert(err, qt.IsNil)

			weights := []float64{weight}
			pbIn, err := base.ConvertToStructpb(TextToImageInput{
				Engine:  engine,
				Prompts: []string{text},
				Weights: &weights,
			})
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
					c.Check(errmsg.Message(err), qt.Equals, tc.wantErr)
				}
			})

			err = exec.Execute(ctx, []*base.Job{job})
			c.Assert(err, qt.IsNil)

		})
	}

	c.Run("nok - unsupported task", func(c *qt.C) {
		task := "FOOBAR"
		exec, err := cmp.CreateExecution(base.ComponentExecution{
			Component: cmp,
			Setup:     new(structpb.Struct),
			Task:      task,
		})
		c.Assert(err, qt.IsNil)

		pbIn := new(structpb.Struct)
		ir, ow, eh, job := mock.GenerateMockJob(c)
		ir.ReadMock.Return(pbIn, nil)
		ow.WriteMock.Optional().Return(nil)
		eh.ErrorMock.Optional().Set(func(ctx context.Context, err error) {
			want := "FOOBAR task is not supported."
			c.Check(errmsg.Message(err), qt.Equals, want)
		})

		err = exec.Execute(ctx, []*base.Job{job})
		c.Check(err, qt.IsNil)

	})
}

func TestComponent_ExecuteImageFromImage(t *testing.T) {
	c := qt.New(t)
	ctx := context.Background()

	weight := 0.5
	text := "a cat and a dog"
	engine := "engine"

	bc := base.Component{}
	cmp := Init(bc).WithInstillCredentials(map[string]any{"apikey": instillSecret})

	testcases := []struct {
		name      string
		gotStatus int
		gotResp   string
		wantResp  ImageToImageOutput
		wantErr   string
	}{
		{
			name:      "ok - 200",
			gotStatus: http.StatusOK,
			gotResp:   okResp,
			wantResp: ImageToImageOutput{
				Images: []string{"data:image/png;base64,a"},
				Seeds:  []uint32{1234},
			},
		},
		{
			name:      "nok - 401",
			gotStatus: http.StatusUnauthorized,
			gotResp:   errResp,
			wantErr:   "Stability AI responded with a 401 status code. Incorrect API key provided",
		},
	}

	for _, tc := range testcases {
		c.Run(tc.name, func(c *qt.C) {
			h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				c.Check(r.Method, qt.Equals, http.MethodPost)
				c.Check(r.URL.Path, qt.Matches, `/v1/generation/.*/image-to-image`)

				c.Check(r.Header.Get("Authorization"), qt.Equals, "Bearer "+apiKey)
				c.Check(r.Header.Get("Content-Type"), qt.Matches, "multipart/form-data; boundary=.*")

				w.Header().Set("Content-Type", httpclient.MIMETypeJSON)
				w.WriteHeader(tc.gotStatus)
				fmt.Fprintln(w, tc.gotResp)
			})

			srv := httptest.NewServer(h)
			c.Cleanup(srv.Close)

			setup, err := structpb.NewStruct(map[string]any{
				"base-path": srv.URL,
				"api-key":   apiKey,
			})
			c.Assert(err, qt.IsNil)

			exec, err := cmp.CreateExecution(base.ComponentExecution{
				Component: cmp,
				Setup:     setup,
				Task:      ImageToImageTask,
			})
			c.Assert(err, qt.IsNil)

			weights := []float64{weight}
			pbIn, err := base.ConvertToStructpb(ImageToImageInput{
				Engine:  engine,
				Prompts: []string{text},
				Weights: &weights,
			})
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
					c.Check(errmsg.Message(err), qt.Equals, tc.wantErr)
				}
			})

			err = exec.Execute(ctx, []*base.Job{job})
			c.Assert(err, qt.IsNil)

		})
	}

	c.Run("nok - unsupported task", func(c *qt.C) {
		task := "FOOBAR"
		exec, err := cmp.CreateExecution(base.ComponentExecution{
			Component: cmp,
			Setup:     new(structpb.Struct),
			Task:      task,
		})
		c.Assert(err, qt.IsNil)

		pbIn := new(structpb.Struct)
		ir, ow, eh, job := mock.GenerateMockJob(c)
		ir.ReadMock.Return(pbIn, nil)
		ow.WriteMock.Optional().Return(nil)
		eh.ErrorMock.Optional().Set(func(ctx context.Context, err error) {
			want := "FOOBAR task is not supported."
			c.Check(errmsg.Message(err), qt.Equals, want)
		})

		err = exec.Execute(ctx, []*base.Job{job})
		c.Check(err, qt.IsNil)

	})
}

func TestComponent_Test(t *testing.T) {
	c := qt.New(t)

	bc := base.Component{}
	cmp := Init(bc).WithInstillCredentials(map[string]any{"apikey": instillSecret})

	c.Run("nok - error", func(c *qt.C) {
		h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c.Check(r.Method, qt.Equals, http.MethodGet)
			c.Check(r.URL.Path, qt.Equals, listEnginesPath)

			w.Header().Set("Content-Type", httpclient.MIMETypeJSON)
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprintln(w, errResp)
		})

		srv := httptest.NewServer(h)
		c.Cleanup(srv.Close)

		setup, err := structpb.NewStruct(map[string]any{
			"base-path": srv.URL,
		})
		c.Assert(err, qt.IsNil)

		err = cmp.Test(nil, setup)
		c.Check(err, qt.IsNotNil)

		wantMsg := "Stability AI responded with a 401 status code. Incorrect API key provided"
		c.Check(errmsg.Message(err), qt.Equals, wantMsg)
	})

	c.Run("ok - disconnected", func(c *qt.C) {
		h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c.Check(r.Method, qt.Equals, http.MethodGet)
			c.Check(r.URL.Path, qt.Equals, listEnginesPath)

			w.Header().Set("Content-Type", httpclient.MIMETypeJSON)
			fmt.Fprintln(w, `[]`)
		})

		srv := httptest.NewServer(h)
		c.Cleanup(srv.Close)

		setup, err := structpb.NewStruct(map[string]any{
			"base-path": srv.URL,
		})
		c.Assert(err, qt.IsNil)

		err = cmp.Test(nil, setup)
		c.Check(err, qt.IsNotNil)
	})

	c.Run("ok - connected", func(c *qt.C) {
		h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c.Check(r.Method, qt.Equals, http.MethodGet)
			c.Check(r.URL.Path, qt.Equals, listEnginesPath)

			w.Header().Set("Content-Type", httpclient.MIMETypeJSON)
			fmt.Fprintln(w, `[{}]`)
		})

		srv := httptest.NewServer(h)
		c.Cleanup(srv.Close)

		setup, err := structpb.NewStruct(map[string]any{
			"base-path": srv.URL,
		})
		c.Assert(err, qt.IsNil)

		err = cmp.Test(nil, setup)
		c.Check(err, qt.IsNil)
	})
}
