package stabilityai

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "embed"

	"google.golang.org/protobuf/types/known/structpb"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/mock"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util/httpclient"
	"github.com/instill-ai/pipeline-backend/pkg/data"
	"github.com/instill-ai/pipeline-backend/pkg/data/format"

	errorsx "github.com/instill-ai/x/errors"
)

//go:embed testdata/dog.png
var dog []byte

const (
	apiKey        = "123"
	instillSecret = "instill-credential-key"
	errResp       = `
{
  "id": "6e958442e7911ffb2e0bf89c6efe804f",
  "message": "Incorrect API key provided",
  "name": "unauthorized"
}`
)

func TestComponent_ExecuteImageFromText(t *testing.T) {
	c := qt.New(t)
	ctx := context.Background()

	weight := 0.5
	text := "a cat and a dog"
	engine := "engine"

	bc := base.Component{}
	cmp := Init(bc).WithInstillCredentials(map[string]any{"apikey": instillSecret})

	img, err := data.NewImageFromBytes(dog, "image/png", "")
	c.Assert(err, qt.IsNil)

	okResp := fmt.Sprintf(`
	{
		"artifacts": [
			{
				"base64": "%s",
				"seed": 1234,
				"finishReason": "SUCCESS"
			}
		]
	}
	`, base64.StdEncoding.EncodeToString(dog))

	testcases := []struct {
		name      string
		gotStatus int
		gotResp   string
		wantResp  taskOutput
		wantErr   string
	}{
		{
			name:      "ok - 200",
			gotStatus: http.StatusOK,
			gotResp:   okResp,
			wantResp: taskOutput{
				Images: []format.Image{img},
				Seeds:  []int{1234},
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

			// Generate mock job
			ir, ow, eh, job := mock.GenerateMockJob(c)

			// Set up input mock
			ir.ReadDataMock.Set(func(ctx context.Context, input any) error {
				switch input := input.(type) {
				case *taskTextToImageInput:
					*input = taskTextToImageInput{
						Engine:  engine,
						Prompts: []string{text},
						Weights: []float64{weight},
					}
				}
				return nil
			})

			// Set up output capture
			var capturedOutput taskOutput
			ow.WriteDataMock.Set(func(ctx context.Context, output any) error {
				switch output := output.(type) {
				case *taskOutput:
					capturedOutput = *output
					for _, image := range capturedOutput.Images {
						imgBae64, err := image.Base64()
						c.Assert(err, qt.IsNil)
						wantImgBae64, err := tc.wantResp.Images[0].Base64()
						c.Assert(err, qt.IsNil)
						c.Check(imgBae64.String(), qt.Equals, wantImgBae64.String())
					}
				}
				return nil
			})

			// Set up error handling
			var executionErr error
			eh.ErrorMock.Set(func(ctx context.Context, err error) {
				executionErr = err
			})

			if tc.wantErr == "" {
				eh.ErrorMock.Optional()
			} else {
				ow.WriteDataMock.Optional()
			}

			err = exec.Execute(ctx, []*base.Job{job})
			c.Assert(err, qt.IsNil)

			if tc.wantErr != "" {
				c.Assert(executionErr, qt.Not(qt.IsNil))
				c.Check(errorsx.Message(executionErr), qt.Equals, tc.wantErr)
			}
		})
	}

	c.Run("nok - unsupported task", func(c *qt.C) {
		task := "FOOBAR"
		_, err := cmp.CreateExecution(base.ComponentExecution{
			Component: cmp,
			Setup:     new(structpb.Struct),
			Task:      task,
		})
		c.Check(err.Error(), qt.Equals, "unsupported task: FOOBAR")
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

	img, err := data.NewImageFromBytes(dog, "image/png", "")
	c.Assert(err, qt.IsNil)

	okResp := fmt.Sprintf(`
	{
		"artifacts": [
			{
				"base64": "%s",
				"seed": 1234,
				"finishReason": "SUCCESS"
			}
		]
	}
	`, base64.StdEncoding.EncodeToString(dog))

	testcases := []struct {
		name      string
		gotStatus int
		gotResp   string
		wantResp  taskOutput
		wantErr   string
	}{
		{
			name:      "ok - 200",
			gotStatus: http.StatusOK,
			gotResp:   okResp,
			wantResp: taskOutput{
				Images: []format.Image{img},
				Seeds:  []int{1234},
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

			// Generate mock job
			ir, ow, eh, job := mock.GenerateMockJob(c)

			// Set up input mock
			ir.ReadDataMock.Set(func(ctx context.Context, input any) error {
				switch input := input.(type) {
				case *taskImageToImageInput:
					*input = taskImageToImageInput{
						Engine:    engine,
						Prompts:   []string{text},
						Weights:   []float64{weight},
						InitImage: img,
					}
				}
				return nil
			})

			// Set up output capture
			var capturedOutput taskOutput
			ow.WriteDataMock.Set(func(ctx context.Context, output any) error {
				switch output := output.(type) {
				case *taskOutput:
					capturedOutput = *output
					for _, image := range capturedOutput.Images {
						imgBae64, err := image.Base64()
						c.Assert(err, qt.IsNil)
						wantImgBae64, err := tc.wantResp.Images[0].Base64()
						c.Assert(err, qt.IsNil)
						c.Check(imgBae64.String(), qt.Equals, wantImgBae64.String())
					}

				}
				return nil
			})

			// Set up error handling
			var executionErr error
			eh.ErrorMock.Set(func(ctx context.Context, err error) {
				executionErr = err
			})

			if tc.wantErr == "" {
				eh.ErrorMock.Optional()
			} else {
				ow.WriteDataMock.Optional()
			}

			err = exec.Execute(ctx, []*base.Job{job})
			c.Assert(err, qt.IsNil)

			if tc.wantErr != "" {
				c.Assert(executionErr, qt.Not(qt.IsNil))
				c.Check(errorsx.Message(executionErr), qt.Equals, tc.wantErr)
			}
		})
	}

	c.Run("nok - unsupported task", func(c *qt.C) {
		task := "FOOBAR"
		_, err := cmp.CreateExecution(base.ComponentExecution{
			Component: cmp,
			Setup:     new(structpb.Struct),
			Task:      task,
		})
		c.Check(err.Error(), qt.Equals, "unsupported task: FOOBAR")
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
		c.Check(errorsx.Message(err), qt.Equals, wantMsg)
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
