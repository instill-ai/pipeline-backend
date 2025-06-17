package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	qt "github.com/frankban/quicktest"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/mock"
	"github.com/instill-ai/pipeline-backend/pkg/data"
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
)

const (
	username  = "foo"
	password  = "bar"
	token     = "123"
	authKey   = "api-key"
	authValue = "321"
)

var testAuth = map[authType]map[string]any{

	noAuthType: {},
	basicAuthType: {
		"username": username,
		"password": password,
	},
	bearerTokenType: {
		"token": token,
	},
	apiKeyType: {
		"auth-location": string(query),
		"key":           authKey,
		"value":         authValue,
	},
}

var passThroughHeaders = []string{"Instill-User-Uid", "Content-Type"}

func TestComponent(t *testing.T) {
	c := qt.New(t)

	// Setup test HTTP server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return the received headers
		for k, h := range r.Header {
			if !slices.Contains(passThroughHeaders, k) {
				continue
			}

			w.Header().Set(k, strings.Join(h, ","))
		}

		switch r.URL.Path {
		case "/json":
			w.Header().Set("Content-Type", "application/json")
			err := json.NewEncoder(w).Encode(map[string]string{"message": "hello"})
			c.Assert(err, qt.IsNil)
		case "/text":
			w.Header().Set("Content-Type", "text/plain")
			_, err := w.Write([]byte("hello"))
			c.Assert(err, qt.IsNil)
		case "/file":
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Header().Set("Content-Disposition", `attachment; filename="test.bin"`)
			_, err := w.Write([]byte("hello"))
			c.Assert(err, qt.IsNil)
		case "/error":
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		case "/auth":
			user, pass, ok := r.BasicAuth()
			if !ok || user != username || pass != password {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			err := json.NewEncoder(w).Encode(map[string]string{"message": "authenticated"})
			c.Assert(err, qt.IsNil)
		case "/bearer":
			authHeader := r.Header.Get("Authorization")
			if authHeader != "Bearer "+token {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			err := json.NewEncoder(w).Encode(map[string]string{"message": "authenticated"})
			c.Assert(err, qt.IsNil)
		}
	}))
	defer ts.Close()

	testCases := []struct {
		name     string
		task     string
		input    httpInput
		setup    authType
		expected httpOutput
	}{
		{
			name: "GET JSON response",
			task: "TASK_GET",
			input: httpInput{
				EndpointURL: ts.URL + "/json",
			},
			setup: noAuthType,
			expected: httpOutput{
				Body:       data.Map{"message": data.NewString("hello")},
				Header:     http.Header{"Content-Type": {"application/json"}},
				StatusCode: 200,
			},
		},
		{
			name: "GET JSON response, pass headers",
			task: "TASK_GET",
			input: httpInput{
				EndpointURL: ts.URL + "/json",
				Header:      http.Header{"Instill-User-Uid": {"unodostres"}},
			},
			setup: noAuthType,
			expected: httpOutput{
				Body: data.Map{"message": data.NewString("hello")},
				Header: http.Header{
					"Content-Type":     {"application/json"},
					"Instill-User-Uid": {"unodostres"},
				},
				StatusCode: 200,
			},
		},
		{
			name: "GET text response",
			task: "TASK_GET",
			input: httpInput{
				EndpointURL: ts.URL + "/text",
			},
			setup: noAuthType,
			expected: httpOutput{
				Body:       data.NewString("hello"),
				Header:     http.Header{"Content-Type": {"text/plain"}},
				StatusCode: 200,
			},
		},
		{
			name: "GET binary response",
			task: "TASK_GET",
			input: httpInput{
				EndpointURL: ts.URL + "/file",
			},
			setup: noAuthType,
			expected: httpOutput{
				Body: func() format.Value {
					v, _ := data.NewFileFromBytes([]byte("hello"), "application/octet-stream", "test.bin")
					return v
				}(),
				Header:     http.Header{"Content-Type": {"application/octet-stream"}},
				StatusCode: 200,
			},
		},
		{
			name: "POST JSON request/response",
			task: "TASK_POST",
			input: httpInput{
				EndpointURL: ts.URL + "/json",
				Header:      map[string][]string{"Content-Type": {"application/json"}},
				Body:        data.Map{"message": data.NewString("hello")},
			},
			setup: noAuthType,
			expected: httpOutput{
				Body:       data.Map{"message": data.NewString("hello")},
				Header:     http.Header{"Content-Type": {"application/json"}},
				StatusCode: 200,
			},
		},
		{
			name: "PATCH text request/response",
			task: "TASK_PATCH",
			input: httpInput{
				EndpointURL: ts.URL + "/text",
				Header:      http.Header{"Content-Type": {"text/plain"}},
				Body:        data.NewString("hello"),
			},
			setup: noAuthType,
			expected: httpOutput{
				Body:       data.NewString("hello"),
				Header:     http.Header{"Content-Type": {"text/plain"}},
				StatusCode: 200,
			},
		},
		{
			name: "DELETE request",
			task: "TASK_DELETE",
			input: httpInput{
				EndpointURL: ts.URL + "/json",
			},
			setup: noAuthType,
			expected: httpOutput{
				Body:       data.Map{"message": data.NewString("hello")},
				Header:     http.Header{"Content-Type": {"application/json"}},
				StatusCode: 200,
			},
		},
		{
			name: "PUT request",
			task: "TASK_PUT",
			input: httpInput{
				EndpointURL: ts.URL + "/json",
				Body:        data.Map{"message": data.NewString("hello")},
			},
			setup: noAuthType,
			expected: httpOutput{
				Body:       data.Map{"message": data.NewString("hello")},
				Header:     http.Header{"Content-Type": {"application/json"}},
				StatusCode: 200,
			},
		},
		{
			name: "GET error response",
			task: "TASK_GET",
			input: httpInput{
				EndpointURL: ts.URL + "/error",
			},
			setup: noAuthType,
			expected: httpOutput{
				Body:       data.NewString("Internal Server Error\n"),
				Header:     http.Header{"Content-Type": {"text/plain; charset=utf-8"}},
				StatusCode: 500,
			},
		},
		{
			name: "GET with basic auth",
			task: "TASK_GET",
			input: httpInput{
				EndpointURL: ts.URL + "/auth",
			},
			setup: basicAuthType,
			expected: httpOutput{
				Body:       data.Map{"message": data.NewString("authenticated")},
				Header:     http.Header{"Content-Type": {"application/json"}},
				StatusCode: 200,
			},
		},
		{
			name: "GET with invalid basic auth",
			task: "TASK_GET",
			input: httpInput{
				EndpointURL: ts.URL + "/auth",
			},
			setup: noAuthType,
			expected: httpOutput{
				Body:       data.NewString("Unauthorized\n"),
				Header:     http.Header{"Content-Type": {"text/plain; charset=utf-8"}},
				StatusCode: 401,
			},
		},
		{
			name: "GET with bearer token",
			task: "TASK_GET",
			input: httpInput{
				EndpointURL: ts.URL + "/bearer",
			},
			setup: bearerTokenType,
			expected: httpOutput{
				Body:       data.Map{"message": data.NewString("authenticated")},
				Header:     http.Header{"Content-Type": {"application/json"}},
				StatusCode: 200,
			},
		},
		{
			name: "GET with invalid bearer token",
			task: "TASK_GET",
			input: httpInput{
				EndpointURL: ts.URL + "/bearer",
			},
			setup: noAuthType,
			expected: httpOutput{
				Body:       data.NewString("Unauthorized\n"),
				Header:     http.Header{"Content-Type": {"text/plain; charset=utf-8"}},
				StatusCode: 401,
			},
		},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *qt.C) {
			component := Init(base.Component{})
			c.Assert(component, qt.IsNotNil)

			execution, err := component.CreateExecution(base.ComponentExecution{
				Component: component,
				Task:      tc.task,
				Setup:     cfg(tc.setup),
			})
			c.Assert(err, qt.IsNil)

			ir, ow, _, job := mock.GenerateMockJob(c)
			ir.ReadDataMock.Set(func(ctx context.Context, input any) error {
				switch input := input.(type) {
				case *httpInput:
					*input = tc.input
				}
				return nil
			})

			var capturedOutput httpOutput
			ow.WriteDataMock.Set(func(ctx context.Context, output any) error {
				capturedOutput = output.(httpOutput)
				return nil
			})

			err = execution.Execute(context.Background(), []*base.Job{job})
			c.Assert(err, qt.IsNil)
			c.Assert(capturedOutput.Body.Equal(tc.expected.Body), qt.IsTrue)
			c.Assert(capturedOutput.StatusCode, qt.Equals, tc.expected.StatusCode)

			for k, h := range tc.expected.Header {
				c.Assert(capturedOutput.Header[k], qt.DeepEquals, h)
			}
		})
	}
}

func cfg(atype authType) *structpb.Struct {
	auth := testAuth[atype]
	auth["auth-type"] = string(atype)
	setup, _ := structpb.NewStruct(map[string]any{
		"authentication": auth,
	})

	return setup
}
