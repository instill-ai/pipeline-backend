package openai

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"google.golang.org/protobuf/types/known/structpb"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util/httpclient"
	"github.com/instill-ai/x/errmsg"
)

const (
	apiKey        = "123"
	instillSecret = "instill-credential-key"
	org           = "org1"
	errResp       = `
{
  "error": {
    "message": "Incorrect API key provided."
  }
}`
)

func TestComponent_Execute(t *testing.T) {
	c := qt.New(t)
	ctx := context.Background()

	bc := base.Component{}
	cmp := Init(bc).WithInstillCredentials(map[string]any{"apikey": instillSecret})

	testcases := []struct {
		name        string
		task        string
		path        string
		contentType string
	}{
		// The response uses `text/event-stream`, which requires different error
		// handling. We need to fix this.
		// {
		// 	name:        "text generation",
		// 	task:        ,
		// 	path:        completionsPath,
		// 	contentType: httpclient.MIMETypeJSON,
		// },
		{
			name:        "text embeddings",
			task:        TextEmbeddingsTask,
			path:        embeddingsPath,
			contentType: httpclient.MIMETypeJSON,
		},
		{
			name:        "speech recognition",
			task:        SpeechRecognitionTask,
			path:        transcriptionsPath,
			contentType: "multipart/form-data; boundary=.*",
		},
		{
			name:        "text to speech",
			task:        TextToSpeechTask,
			path:        createSpeechPath,
			contentType: httpclient.MIMETypeJSON,
		},
		{
			name:        "text to image",
			task:        TextToImageTask,
			path:        imgGenerationPath,
			contentType: httpclient.MIMETypeJSON,
		},
	}

	// TODO we'll likely want to have a test function per task and test at
	// least OK, NOK. For now, only errors are tested in order to verify
	// end-user messages.
	for _, tc := range testcases {
		c.Run("nok - "+tc.name+" 401", func(c *qt.C) {
			h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				c.Check(r.Method, qt.Equals, http.MethodPost)
				c.Check(r.URL.Path, qt.Equals, tc.path)

				c.Check(r.Header.Get("OpenAI-Organization"), qt.Equals, org)
				c.Check(r.Header.Get("Authorization"), qt.Equals, "Bearer "+apiKey)

				c.Check(r.Header.Get("Content-Type"), qt.Matches, tc.contentType)

				w.Header().Set("Content-Type", httpclient.MIMETypeJSON)
				w.WriteHeader(http.StatusUnauthorized)
				fmt.Fprintln(w, errResp)
			})

			openAIServer := httptest.NewServer(h)
			c.Cleanup(openAIServer.Close)

			setup, err := structpb.NewStruct(map[string]any{
				"base-path":    openAIServer.URL,
				"api-key":      apiKey,
				"organization": org,
			})
			c.Assert(err, qt.IsNil)

			x, err := cmp.CreateExecution(base.ComponentExecution{
				Component: cmp,
				Setup:     setup,
				Task:      tc.task,
			})
			c.Assert(err, qt.IsNil)

			pbIn := new(structpb.Struct)
			ir, ow, eh, job := base.GenerateMockJob(c)
			ir.ReadMock.Return(pbIn, nil)
			ow.WriteMock.Optional().Return(nil)

			eh.ErrorMock.Optional().Set(func(ctx context.Context, err error) {
				want := "OpenAI responded with a 401 status code. Incorrect API key provided."
				c.Check(errmsg.Message(err), qt.Equals, want)
			})

			err = x.Execute(ctx, []*base.Job{job})
			c.Check(err, qt.IsNil)

		})
	}

	c.Run("nok - unsupported task", func(c *qt.C) {
		task := "FOOBAR"
		exec, err := cmp.CreateExecution(base.ComponentExecution{
			Component: cmp,
			Task:      task,
		})
		c.Assert(err, qt.IsNil)

		pbIn := new(structpb.Struct)
		ir, ow, eh, job := base.GenerateMockJob(c)
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
			c.Check(r.URL.Path, qt.Equals, listModelsPath)

			w.Header().Set("Content-Type", httpclient.MIMETypeJSON)
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprintln(w, errResp)
		})

		openAIServer := httptest.NewServer(h)
		c.Cleanup(openAIServer.Close)

		setup, err := structpb.NewStruct(map[string]any{
			"base-path": openAIServer.URL,
		})
		c.Assert(err, qt.IsNil)

		err = cmp.Test(nil, setup)
		c.Check(err, qt.IsNotNil)

		wantMsg := "OpenAI responded with a 401 status code. Incorrect API key provided."
		c.Check(errmsg.Message(err), qt.Equals, wantMsg)
	})

	c.Run("ok - disconnected", func(c *qt.C) {
		h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c.Check(r.Method, qt.Equals, http.MethodGet)
			c.Check(r.URL.Path, qt.Equals, listModelsPath)

			w.Header().Set("Content-Type", httpclient.MIMETypeJSON)
			fmt.Fprintln(w, `{}`)
		})

		openAIServer := httptest.NewServer(h)
		c.Cleanup(openAIServer.Close)

		setup, err := structpb.NewStruct(map[string]any{
			"base-path": openAIServer.URL,
		})
		c.Assert(err, qt.IsNil)

		err = cmp.Test(nil, setup)
		c.Check(err, qt.IsNotNil)
	})

	c.Run("ok - connected", func(c *qt.C) {
		h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c.Check(r.Method, qt.Equals, http.MethodGet)
			c.Check(r.URL.Path, qt.Equals, listModelsPath)

			w.Header().Set("Content-Type", httpclient.MIMETypeJSON)
			fmt.Fprintln(w, `{"data": [{}]}`)
		})

		openAIServer := httptest.NewServer(h)
		c.Cleanup(openAIServer.Close)

		setup, err := structpb.NewStruct(map[string]any{
			"base-path": openAIServer.URL,
		})
		c.Assert(err, qt.IsNil)

		err = cmp.Test(nil, setup)
		c.Check(err, qt.IsNil)
	})
}

func TestComponent_WithConfig(t *testing.T) {
	c := qt.New(t)
	cleanupConn := func() { once = sync.Once{} }

	task := TextGenerationTask
	bc := base.Component{}

	c.Run("ok - without secret", func(c *qt.C) {
		c.Cleanup(cleanupConn)

		cmp := Init(bc).WithInstillCredentials(map[string]any{"apikey": instillSecret})

		setup, err := structpb.NewStruct(map[string]any{
			"base-path": "foo/bar",
			"api-key":   apiKey,
		})
		c.Assert(err, qt.IsNil)

		x, err := cmp.CreateExecution(base.ComponentExecution{
			Component: cmp,
			Setup:     setup,
			Task:      task,
		})
		c.Assert(err, qt.IsNil)
		c.Check(x.UsesInstillCredentials(), qt.IsFalse)
	})

	c.Run("ok - with secret", func(c *qt.C) {
		c.Cleanup(cleanupConn)

		secrets := map[string]any{"apikey": apiKey}
		cmp := Init(bc).WithInstillCredentials(secrets)

		setup, err := structpb.NewStruct(map[string]any{
			"base-path": "foo/bar",
			"api-key":   "__INSTILL_SECRET",
		})
		c.Assert(err, qt.IsNil)

		x, err := cmp.CreateExecution(base.ComponentExecution{
			Component: cmp,
			Setup:     setup,
			Task:      task,
		})
		c.Assert(err, qt.IsNil)
		c.Check(x.UsesInstillCredentials(), qt.IsTrue)
	})

	c.Run("nok - secret not injected", func(c *qt.C) {
		c.Cleanup(cleanupConn)

		cmp := Init(bc)
		setup, err := structpb.NewStruct(map[string]any{
			"api-key": "__INSTILL_SECRET",
		})
		c.Assert(err, qt.IsNil)

		_, err = cmp.CreateExecution(base.ComponentExecution{
			Component: cmp,
			Setup:     setup,
			Task:      task,
		})
		c.Check(err, qt.IsNotNil)
		c.Check(err, qt.ErrorMatches, "unresolved global credential")
		c.Check(errmsg.Message(err), qt.Matches, "The configuration field api-key references a global secret but.*")
	})
}
