package http

import (
	"context"
	"net/http"
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

func TestComponent(t *testing.T) {
	c := qt.New(t)
	c.Parallel()

	// respEquals returns a checker for equality between the received response
	// and the expected one.
	respEquals := func(want httpOutput) func(*qt.C, httpOutput) {
		return func(c *qt.C, got httpOutput) {
			c.Check(got.StatusCode, qt.Equals, want.StatusCode)

			jGot, err := got.Body.ToJSONValue()
			c.Assert(err, qt.IsNil)

			jWant, err := want.Body.ToJSONValue()
			c.Assert(err, qt.IsNil)

			c.Check(jGot, qt.DeepEquals, jWant)

			for k, h := range want.Header {
				c.Check(got.Header[k], qt.DeepEquals, h)
			}
		}
	}

	// These tests use public test endpoints instead of httptest.NewServer() in
	// order to avoid the private address restriction.
	testCases := []struct {
		name   string
		task   string
		input  httpInput
		setup  authType
		expect func(c *qt.C, got httpOutput)
	}{
		{
			name: "GET JSON response",
			task: "TASK_GET",
			input: httpInput{
				// Returns a fixed JSON document.
				EndpointURL: "https://httpbin.org/json",
			},
			setup: noAuthType,
			expect: respEquals(httpOutput{
				Body: body(c, map[string]any{
					"slideshow": map[string]any{
						"author": "Yours Truly",
						"date":   "date of publication",
						"slides": []any{
							map[string]any{
								"title": "Wake up to WonderWidgets!",
								"type":  "all",
							},
							map[string]any{
								"items": []any{
									"Why <em>WonderWidgets</em> are great",
									"Who <em>buys</em> WonderWidgets",
								},
								"title": "Overview",
								"type":  "all",
							},
						},
						"title": "Sample Slide Show",
					},
				}),
				Header:     http.Header{"Content-Type": {"application/json"}},
				StatusCode: http.StatusOK,
			}),
		},
		{
			name: "GET JSON response, pass headers",
			task: "TASK_GET",
			input: httpInput{
				// Returns the headers in the request.
				EndpointURL: "https://httpbin.org/headers",
				Header:      http.Header{"Instill-User-Uid": {"unodostres"}},
			},
			setup: noAuthType,
			expect: func(c *qt.C, out httpOutput) {
				c.Check(out.StatusCode, qt.Equals, http.StatusOK)

				h, ok := out.Body.(data.Map)["headers"]
				c.Assert(ok, qt.IsTrue, qt.Commentf("response has no headers field"))

				got, err := h.ToJSONValue()
				c.Assert(err, qt.IsNil)

				c.Check(got.(map[string]any)["Instill-User-Uid"], qt.Equals, "unodostres")
			},
		},
		{
			name: "GET text response",
			task: "TASK_GET",
			input: httpInput{
				// Returns a fixed text document.
				EndpointURL: "https://httpbin.org/robots.txt",
			},
			setup: noAuthType,
			expect: respEquals(httpOutput{
				Body:       data.NewString("User-agent: *\nDisallow: /deny\n"),
				Header:     http.Header{"Content-Type": {"text/plain"}},
				StatusCode: http.StatusOK,
			}),
		},
		{
			name: "GET binary response",
			task: "TASK_GET",
			input: httpInput{
				// Returns 10 random bytes.
				EndpointURL: "https://httpbin.org/bytes/10",
			},
			setup: noAuthType,
			expect: func(c *qt.C, out httpOutput) {
				c.Check(out.StatusCode, qt.Equals, http.StatusOK)
				c.Check(out.Header.Get("Content-Type"), qt.Equals, "application/octet-stream")

				// Check that the body is a binary file
				file, ok := out.Body.(format.File)
				c.Assert(ok, qt.IsTrue, qt.Commentf("expected binary file response"))
				c.Check(file.ContentType().String(), qt.Equals, "application/octet-stream")

				// Verify we got 10 bytes as expected
				bytes, err := file.Binary()
				c.Assert(err, qt.IsNil)
				c.Check(len(bytes.ByteArray()), qt.Equals, 10)
			},
		},
		{
			name: "POST JSON request/response",
			task: "TASK_POST",
			input: httpInput{
				// Returns the request body in the response.
				EndpointURL: "https://httpbin.org/post",
				Header:      map[string][]string{"Content-Type": {"application/json"}},
				Body:        data.Map{"message": data.NewString("hello")},
			},
			setup: noAuthType,
			expect: func(c *qt.C, out httpOutput) {
				c.Check(out.StatusCode, qt.Equals, http.StatusOK)
				c.Check(out.Header.Get("Content-Type"), qt.Equals, "application/json")

				body, ok := out.Body.(data.Map)
				c.Assert(ok, qt.IsTrue, qt.Commentf("expected JSON object response"))

				// Check that the request data was echoed back correctly
				c.Check(body["json"].(data.Map)["message"].String(), qt.Equals, "hello")
			},
		},
		{
			name: "PATCH text request/response",
			task: "TASK_PATCH",
			input: httpInput{
				// Returns the request body in the response.
				EndpointURL: "https://httpbin.org/patch",
				Header:      http.Header{"Content-Type": {"text/plain"}},
				Body:        data.NewString("hello"),
			},
			setup: noAuthType,
			expect: func(c *qt.C, out httpOutput) {
				c.Check(out.StatusCode, qt.Equals, http.StatusOK)
				c.Check(out.Header.Get("Content-Type"), qt.Equals, "application/json")

				body, ok := out.Body.(data.Map)
				c.Assert(ok, qt.IsTrue, qt.Commentf("expected JSON object response"))

				c.Check(body["data"].String(), qt.Equals, "hello")
			},
		},
		{
			name: "DELETE request",
			task: "TASK_DELETE",
			input: httpInput{
				// Returns the request body in the response.
				EndpointURL: "https://httpbin.org/delete",
			},
			setup: noAuthType,
			expect: func(c *qt.C, out httpOutput) {
				c.Check(out.StatusCode, qt.Equals, http.StatusOK)
				c.Check(out.Header.Get("Content-Type"), qt.Equals, "application/json")
			},
		},
		{
			name: "PUT request",
			task: "TASK_PUT",
			input: httpInput{
				// Returns the request body in the response.
				EndpointURL: "https://httpbin.org/put",
				Body:        data.Map{"message": data.NewString("hello")},
			},
			setup: noAuthType,
			expect: func(c *qt.C, out httpOutput) {
				c.Check(out.StatusCode, qt.Equals, http.StatusOK)
				c.Check(out.Header.Get("Content-Type"), qt.Equals, "application/json")

				body, ok := out.Body.(data.Map)
				c.Assert(ok, qt.IsTrue, qt.Commentf("expected JSON object response"))

				// Check that the request data was echoed back correctly
				c.Check(body["json"].(data.Map)["message"].String(), qt.Equals, "hello")
			},
		},
		{
			name: "GET error response",
			task: "TASK_GET",
			input: httpInput{
				// Returns the provided status code.
				EndpointURL: "https://httpbin.org/status/500",
			},
			setup: noAuthType,
			expect: respEquals(httpOutput{
				Body:       data.NewString(""),
				Header:     http.Header{"Content-Type": {"text/html; charset=utf-8"}},
				StatusCode: 500,
			}),
		},
		{
			name: "GET with basic auth",
			task: "TASK_GET",
			input: httpInput{
				// Requires basic auth with foo/bar.
				EndpointURL: "https://httpbin.org/basic-auth/foo/bar",
			},
			setup: basicAuthType,
			expect: respEquals(httpOutput{
				Body: body(c, map[string]any{
					"authenticated": true,
					"user":          "foo",
				}),
				Header:     http.Header{"Content-Type": {"application/json"}},
				StatusCode: http.StatusOK,
			}),
		},
		{
			name: "GET with invalid basic auth",
			task: "TASK_GET",
			input: httpInput{
				// Requires basic auth with foo/bar.
				EndpointURL: "https://httpbin.org/basic-auth/foo/bar",
			},
			setup: noAuthType,
			expect: respEquals(httpOutput{
				Body:       data.NewString(""),
				StatusCode: 401,
			}),
		},
		{
			name: "GET with bearer token",
			task: "TASK_GET",
			input: httpInput{
				// Requires bearer token.
				EndpointURL: "https://httpbin.org/bearer",
			},
			setup: bearerTokenType,
			expect: respEquals(httpOutput{
				Body: body(c, map[string]any{
					"authenticated": true,
					"token":         "123",
				}),
				Header:     http.Header{"Content-Type": {"application/json"}},
				StatusCode: http.StatusOK,
			}),
		},
		{
			name: "GET with invalid bearer token",
			task: "TASK_GET",
			input: httpInput{
				// Requires bearer token.
				EndpointURL: "https://httpbin.org/bearer",
			},
			setup: noAuthType,
			expect: respEquals(httpOutput{
				Body:       data.NewString(""),
				StatusCode: 401,
			}),
		},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *qt.C) {
			c.Parallel()

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

			tc.expect(c, capturedOutput)
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

func body(c *qt.C, in map[string]any) format.Value {
	v, err := data.NewJSONValue(in)
	if err != nil {
		c.Fatal(err)
	}
	return v
}
