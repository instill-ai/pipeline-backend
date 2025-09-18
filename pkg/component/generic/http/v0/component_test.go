package http

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
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

// createLocalTestServer creates a local HTTP server that mimics httpbin.org functionality
func createLocalTestServer() *httptest.Server {
	mux := http.NewServeMux()

	// GET /json - Returns fixed JSON document
	mux.HandleFunc("/json", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		response := map[string]any{
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
		}
		_ = json.NewEncoder(w).Encode(response)
	})

	// GET /headers - Returns request headers
	mux.HandleFunc("/headers", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		response := map[string]any{
			"headers": r.Header,
		}
		_ = json.NewEncoder(w).Encode(response)
	})

	// GET /robots.txt - Returns plain text
	mux.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("User-agent: *\nDisallow: /deny\n"))
	})

	// GET /bytes/10 - Returns 10 deterministic bytes for testing
	mux.HandleFunc("/bytes/10", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		// Return deterministic bytes for consistent testing
		_, _ = w.Write([]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A})
	})

	// POST /post - Echo request data
	mux.HandleFunc("/post", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")

		var requestBody any
		if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
			bodyBytes, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(bodyBytes, &requestBody)
		}

		response := map[string]any{
			"json":    requestBody,
			"headers": r.Header,
		}
		_ = json.NewEncoder(w).Encode(response)
	})

	// PATCH /patch - Echo request data
	mux.HandleFunc("/patch", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")

		bodyBytes, _ := io.ReadAll(r.Body)

		response := map[string]any{
			"data":    string(bodyBytes),
			"headers": r.Header,
		}
		_ = json.NewEncoder(w).Encode(response)
	})

	// DELETE /delete - Simple response
	mux.HandleFunc("/delete", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		response := map[string]any{
			"url": r.URL.String(),
		}
		_ = json.NewEncoder(w).Encode(response)
	})

	// PUT /put - Echo request data
	mux.HandleFunc("/put", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")

		var requestBody any
		if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
			bodyBytes, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(bodyBytes, &requestBody)
		}

		response := map[string]any{
			"json":    requestBody,
			"headers": r.Header,
		}
		_ = json.NewEncoder(w).Encode(response)
	})

	// GET /status/500 - Return 500 error
	mux.HandleFunc("/status/500", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(500)
	})

	// Basic auth endpoints
	mux.HandleFunc("/basic-auth/foo/bar", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		username, password, ok := r.BasicAuth()
		if !ok || username != "foo" || password != "bar" {
			w.WriteHeader(401)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		response := map[string]any{
			"authenticated": true,
			"user":          username,
		}
		_ = json.NewEncoder(w).Encode(response)
	})

	// Bearer token endpoint
	mux.HandleFunc("/bearer", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			w.WriteHeader(401)
			return
		}

		token := strings.TrimPrefix(auth, "Bearer ")
		w.Header().Set("Content-Type", "application/json")
		response := map[string]any{
			"authenticated": true,
			"token":         token,
		}
		_ = json.NewEncoder(w).Encode(response)
	})

	return httptest.NewServer(mux)
}

func TestComponent(t *testing.T) {
	c := qt.New(t)
	c.Parallel()

	// Set test environment to bypass URL validation
	os.Setenv("GO_TESTING", "true")
	defer os.Unsetenv("GO_TESTING")

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

	// These tests use a local test server to eliminate external dependencies
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
				EndpointURL: "PLACEHOLDER_URL/json",
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
				EndpointURL: "PLACEHOLDER_URL/headers",
				Header:      http.Header{"Instill-User-Uid": {"unodostres"}},
			},
			setup: noAuthType,
			expect: func(c *qt.C, out httpOutput) {
				c.Check(out.StatusCode, qt.Equals, http.StatusOK)

				// Safely check if Body is a Map before type assertion
				bodyMap, ok := out.Body.(data.Map)
				if !ok {
					c.Fatalf("expected response body to be data.Map, got %T. Body content: %v", out.Body, out.Body)
				}

				headers, ok := bodyMap["headers"]
				c.Assert(ok, qt.IsTrue, qt.Commentf("response has no headers field"))

				got, err := headers.ToJSONValue()
				c.Assert(err, qt.IsNil)

				headerMap, ok := got.(map[string]any)
				c.Assert(ok, qt.IsTrue, qt.Commentf("headers should be a map"))

				c.Check(headerMap["Instill-User-Uid"], qt.DeepEquals, []any{"unodostres"})
			},
		},
		{
			name: "GET text response",
			task: "TASK_GET",
			input: httpInput{
				// Returns a fixed text document.
				EndpointURL: "PLACEHOLDER_URL/robots.txt",
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
				EndpointURL: "PLACEHOLDER_URL/bytes/10",
			},
			setup: noAuthType,
			expect: func(c *qt.C, out httpOutput) {
				c.Check(out.StatusCode, qt.Equals, http.StatusOK)
				c.Check(out.Header.Get("Content-Type"), qt.Equals, "application/octet-stream")

				// Check that the body is a binary file
				file, ok := out.Body.(format.File)
				c.Assert(ok, qt.IsTrue, qt.Commentf("expected binary file response"))
				c.Check(file.ContentType().String(), qt.Equals, "application/octet-stream")

				// Verify we got the expected deterministic bytes
				bytes, err := file.Binary()
				c.Assert(err, qt.IsNil)
				expected := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A}
				c.Check(bytes.ByteArray(), qt.DeepEquals, expected)
			},
		},
		{
			name: "POST JSON request/response",
			task: "TASK_POST",
			input: httpInput{
				// Returns the request body in the response.
				EndpointURL: "PLACEHOLDER_URL/post",
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
				EndpointURL: "PLACEHOLDER_URL/patch",
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
				EndpointURL: "PLACEHOLDER_URL/delete",
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
				EndpointURL: "PLACEHOLDER_URL/put",
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
				EndpointURL: "PLACEHOLDER_URL/status/500",
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
				EndpointURL: "PLACEHOLDER_URL/basic-auth/foo/bar",
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
				EndpointURL: "PLACEHOLDER_URL/basic-auth/foo/bar",
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
				EndpointURL: "PLACEHOLDER_URL/bearer",
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
				EndpointURL: "PLACEHOLDER_URL/bearer",
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

			// Create local test server for this specific test
			server := createLocalTestServer()
			defer server.Close()

			// Replace placeholder URL with actual server URL
			actualInput := tc.input
			actualInput.EndpointURL = strings.Replace(actualInput.EndpointURL, "PLACEHOLDER_URL", server.URL, 1)

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
					*input = actualInput
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
	auth := map[string]any{}
	switch atype {
	case basicAuthType:
		auth["username"] = username
		auth["password"] = password
	case bearerTokenType:
		auth["token"] = token
	case apiKeyType:
		auth["auth-location"] = string(query)
		auth["key"] = authKey
		auth["value"] = authValue
	}

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
