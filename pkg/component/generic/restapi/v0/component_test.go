package restapi

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"google.golang.org/protobuf/types/known/structpb"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util/httpclient"
	"github.com/instill-ai/x/errmsg"
)

const (
	errResp = `{"message": "Bad request"}`
	okResp  = `{"title": "Be the wheel"}`
)

var (
	path = "/good-songs/10"
)

func TestComponent_Execute(t *testing.T) {
	c := qt.New(t)
	ctx := context.Background()

	bc := base.Component{}
	cmp := Init(bc)
	reqBody := map[string]any{
		"title": "Be the wheel",
	}

	c.Run("nok - unsupported task", func(c *qt.C) {
		task := "FOOBAR"

		exec, err := cmp.CreateExecution(base.ComponentExecution{
			Component: cmp,
			Setup:     cfg(noAuthType),
			Task:      task,
		})
		c.Assert(err, qt.IsNil)

		pbIn := new(structpb.Struct)
		ir, ow, eh, job := base.GenerateMockJob(c)
		ir.ReadMock.Optional().Return(pbIn, nil)
		ow.WriteMock.Optional().Return(nil)
		eh.ErrorMock.Optional()

		err = exec.Execute(ctx, []*base.Job{job})
		c.Check(err, qt.IsNotNil)

		want := "FOOBAR task is not supported."
		c.Check(errmsg.Message(err), qt.Equals, want)
	})

	c.Run("ok - POST, 400, basic auth", func(c *qt.C) {
		h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c.Check(r.Method, qt.Equals, http.MethodPost)
			c.Check(r.URL.Path, qt.Matches, path)

			auth := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
			c.Check(r.Header.Get("Authorization"), qt.Equals, "Basic "+auth)

			c.Assert(r.Body, qt.IsNotNil)
			defer r.Body.Close()

			body, err := io.ReadAll(r.Body)
			c.Assert(err, qt.IsNil)
			c.Check(body, qt.JSONEquals, reqBody)

			w.Header().Set("Content-Type", httpclient.MIMETypeJSON)
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintln(w, errResp)
		})

		srv := httptest.NewServer(h)
		c.Cleanup(srv.Close)

		exec, err := cmp.CreateExecution(base.ComponentExecution{
			Component: cmp,
			Setup:     cfg(basicAuthType),
			Task:      taskPost,
		})
		c.Assert(err, qt.IsNil)

		pbIn, err := base.ConvertToStructpb(TaskInput{
			EndpointURL: srv.URL + path,
			Body:        reqBody,
		})
		c.Assert(err, qt.IsNil)

		ir, ow, eh, job := base.GenerateMockJob(c)
		ir.ReadMock.Return(pbIn, nil)
		ow.WriteMock.Optional().Set(func(ctx context.Context, output *structpb.Struct) (err error) {
			resp := output.AsMap()
			c.Check(resp["status-code"], qt.Equals, float64(http.StatusBadRequest))
			c.Check(resp["body"], qt.ContentEquals, map[string]any{"message": "Bad request"})
			return nil
		})
		eh.ErrorMock.Optional()

		err = exec.Execute(ctx, []*base.Job{job})
		c.Check(err, qt.IsNil)

	})

	c.Run("ok - PUT + query auth", func(c *qt.C) {
		h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c.Check(r.Method, qt.Equals, http.MethodPut)
			c.Check(r.URL.Path, qt.Matches, path)

			c.Check(r.FormValue(authKey), qt.Equals, authValue)
			w.Header().Set("Content-Type", httpclient.MIMETypeJSON)
			fmt.Fprintln(w, okResp)
		})

		srv := httptest.NewServer(h)
		c.Cleanup(srv.Close)

		exec, err := cmp.CreateExecution(base.ComponentExecution{
			Component: cmp,
			Setup:     cfg(apiKeyType),
			Task:      taskPut,
		})
		c.Assert(err, qt.IsNil)

		pbIn, err := base.ConvertToStructpb(TaskInput{
			EndpointURL: srv.URL + path,
			Body:        reqBody,
		})

		c.Assert(err, qt.IsNil)

		ir, ow, eh, job := base.GenerateMockJob(c)
		ir.ReadMock.Return(pbIn, nil)
		ow.WriteMock.Optional().Set(func(ctx context.Context, output *structpb.Struct) (err error) {
			resp := output.AsMap()
			c.Check(resp["status-code"], qt.Equals, float64(http.StatusOK))
			c.Check(resp["body"], qt.ContentEquals, map[string]any{"title": "Be the wheel"})
			return nil
		})
		eh.ErrorMock.Optional()

		err = exec.Execute(ctx, []*base.Job{job})
		c.Check(err, qt.IsNil)

	})

	c.Run("ok - GET + bearer auth", func(c *qt.C) {
		h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c.Check(r.Method, qt.Equals, http.MethodGet)
			c.Check(r.URL.Path, qt.Matches, path)

			c.Check(r.Header.Get("Authorization"), qt.Equals, "Bearer "+token)

			w.Header().Set("Content-Type", httpclient.MIMETypeJSON)
			fmt.Fprintln(w, okResp)
		})

		srv := httptest.NewServer(h)
		c.Cleanup(srv.Close)

		exec, err := cmp.CreateExecution(base.ComponentExecution{
			Component: cmp,
			Setup:     cfg(bearerTokenType),
			Task:      taskGet,
		})
		c.Assert(err, qt.IsNil)

		pbIn, err := base.ConvertToStructpb(TaskInput{
			EndpointURL: srv.URL + path,
			Body:        reqBody,
		})
		c.Assert(err, qt.IsNil)

		ir, ow, eh, job := base.GenerateMockJob(c)
		ir.ReadMock.Return(pbIn, nil)
		ow.WriteMock.Optional().Set(func(ctx context.Context, output *structpb.Struct) (err error) {
			resp := output.AsMap()
			c.Check(resp["status-code"], qt.Equals, float64(http.StatusOK))
			c.Check(resp["body"], qt.ContentEquals, map[string]any{"title": "Be the wheel"})
			return nil
		})
		eh.ErrorMock.Optional()

		err = exec.Execute(ctx, []*base.Job{job})
		c.Check(err, qt.IsNil)

	})
}

func TestComponent_Test(t *testing.T) {
	c := qt.New(t)

	bc := base.Component{}
	cmp := Init(bc)

	c.Run("ok - connected (even with non-2xx status", func(c *qt.C) {
		h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c.Check(r.Method, qt.Equals, http.MethodGet)

			w.WriteHeader(http.StatusNotFound)
			w.Header().Set("Content-Type", "plain/text")
			fmt.Fprintln(w, "Not Found")
		})

		srv := httptest.NewServer(h)
		c.Cleanup(srv.Close)

		err := cmp.Test(nil, cfg(noAuthType))
		c.Check(err, qt.IsNil)
	})
}

const (
	username  = "foo"
	password  = "bar"
	token     = "123"
	authKey   = "api-key"
	authValue = "321"
)

var testAuth = map[authType]map[string]any{
	noAuthType: map[string]any{},
	basicAuthType: map[string]any{
		"username": username,
		"password": password,
	},
	bearerTokenType: map[string]any{"token": token},
	apiKeyType: map[string]any{
		"auth-location": string(query),
		"key":           authKey,
		"value":         authValue,
	},
}

func cfg(atype authType) *structpb.Struct {
	auth := testAuth[atype]
	auth["auth-type"] = string(atype)
	setup, _ := structpb.NewStruct(map[string]any{
		"authentication": auth,
	})

	return setup
}
