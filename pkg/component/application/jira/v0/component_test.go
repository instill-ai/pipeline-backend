package jira

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/mock"
)

const (
	email = "testemail@gmail.com"
	token = "testToken"
)

type TaskCase[inType any, outType any] struct {
	_type      string
	name       string
	input      inType
	wantOutput outType
	wantErr    string
}

func TestAuth_nok(t *testing.T) {
	c := qt.New(t)
	bc := base.Component{Logger: zap.NewNop()}
	cmp := Init(bc)
	c.Run("nok-empty token", func(c *qt.C) {
		setup, err := structpb.NewStruct(map[string]any{
			"token":    "",
			"email":    email,
			"base-url": "url",
		})
		c.Assert(err, qt.IsNil)
		_, err = cmp.CreateExecution(base.ComponentExecution{
			Component: cmp,
			Setup:     setup,
			Task:      "invalid",
		})
		c.Assert(err, qt.ErrorMatches, "token not provided")
	})
	c.Run("nok-empty email", func(c *qt.C) {
		setup, err := structpb.NewStruct(map[string]any{
			"token":    token,
			"email":    "",
			"base-url": "url",
		})
		c.Assert(err, qt.IsNil)
		_, err = cmp.CreateExecution(base.ComponentExecution{
			Component: cmp,
			Setup:     setup,
			Task:      "invalid",
		})
		c.Assert(err, qt.ErrorMatches, "email not provided")
	})
}

func taskTesting[inType any, outType any](testCases []TaskCase[inType, outType], task string, t *testing.T) {
	c := qt.New(t)
	cmp := Init(base.Component{Logger: zap.NewNop()})

	for _, tc := range testCases {
		c.Run(tc._type+`-`+tc.name, func(c *qt.C) {
			authenticationMiddleware := func(next http.Handler) http.Handler {
				fn := func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path != "/_edge/tenant_info" {
						auth := base64.StdEncoding.EncodeToString([]byte(email + ":" + token))
						c.Check(r.Header.Get("Authorization"), qt.Equals, "Basic "+auth)
					}
					next.ServeHTTP(w, r)
				}
				return http.HandlerFunc(fn)
			}
			setContentTypeMiddleware := func(next http.Handler) http.Handler {
				fn := func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					next.ServeHTTP(w, r)
				}
				return http.HandlerFunc(fn)
			}
			srv := httptest.NewServer(router(authenticationMiddleware, setContentTypeMiddleware))
			c.Cleanup(srv.Close)

			setup, err := structpb.NewStruct(map[string]any{
				"token":    token,
				"email":    email,
				"base-url": srv.URL,
			})
			c.Assert(err, qt.IsNil)

			e, err := cmp.CreateExecution(base.ComponentExecution{
				Component: cmp,
				Setup:     setup,
				Task:      task,
			})
			c.Assert(err, qt.IsNil)

			ir, ow, eh, job := mock.GenerateMockJob(c)
			ir.ReadDataMock.Set(func(ctx context.Context, input any) error {
				if ptr, ok := input.(*inType); ok {
					*ptr = tc.input
					return nil
				}
				return fmt.Errorf("unsupported input type: %T", input)
			})
			ow.WriteDataMock.Optional().Set(func(ctx context.Context, output any) error {
				c.Assert(output, qt.DeepEquals, tc.wantOutput)
				return nil
			})
			eh.ErrorMock.Optional().Set(func(ctx context.Context, err error) {
				c.Assert(err, qt.ErrorMatches, tc.wantErr)
			})
			err = e.Execute(context.Background(), []*base.Job{job})
			c.Assert(err, qt.IsNil)
		})
	}
}
