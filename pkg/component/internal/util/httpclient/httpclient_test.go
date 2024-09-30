package httpclient

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/x/errmsg"
)

func TestClient_SendReqAndUnmarshal(t *testing.T) {
	c := qt.New(t)

	const testName = "Pokédex"
	const path = "/137"
	data := struct{ Name string }{Name: "Porygon"}

	c.Run("ok - with default headers", func(c *qt.C) {
		h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c.Check(r.URL.Path, qt.Equals, path)

			c.Check(r.Header.Get("Content-Type"), qt.Equals, MIMETypeJSON)
			c.Check(r.Header.Get("Accept"), qt.Equals, MIMETypeJSON)

			c.Assert(r.Body, qt.IsNotNil)
			defer r.Body.Close()

			body, err := io.ReadAll(r.Body)
			c.Assert(err, qt.IsNil)
			c.Check(body, qt.JSONEquals, data)

			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintln(w, `{"added": 1}`)
		})

		srv := httptest.NewServer(h)
		c.Cleanup(srv.Close)

		client := New(testName, srv.URL)

		var got okBody
		resp, err := client.R().
			SetBody(data).
			SetResult(&got).
			Post(path)

		c.Assert(err, qt.IsNil)
		c.Check(resp.IsError(), qt.IsFalse)
		c.Check(got.Added, qt.Equals, 1)
	})

	c.Run("nok - client error", func(c *qt.C) {
		zCore, zLogs := observer.New(zap.InfoLevel)
		host := "https://uninitialized.server.zz"

		client := New(testName, host, WithLogger(zap.New(zCore)))

		_, err := client.R().Post(path)
		c.Check(err, qt.ErrorMatches, ".*no such host*")

		logs := zLogs.All()
		c.Assert(logs, qt.HasLen, 1)

		entry := logs[0].ContextMap()
		c.Check(err, qt.ErrorMatches, fmt.Sprintf(".*%s", entry["error"]))
		c.Check(entry["url"], qt.Equals, host+path)
	})

	testcases := []struct {
		name           string
		gotStatus      int
		gotBody        string
		gotContentType string
		wantIssue      string
		wantLogFields  []string
	}{
		{
			name:           "nok - 401 (unexpected response body)",
			gotStatus:      http.StatusUnauthorized,
			gotContentType: "plain/text",
			gotBody:        `Incorrect API key`,
			wantIssue:      fmt.Sprintf("%s responded with a 401 status code. Incorrect API key", testName),
			wantLogFields:  []string{"url", "body", "status"},
		},
		{
			name:          "nok - 401 (no response body)",
			gotStatus:     http.StatusUnauthorized,
			wantIssue:     fmt.Sprintf("%s responded with a 401 status code. Please refer to %s's API reference for more information.", testName, testName),
			wantLogFields: []string{"url", "body", "status"},
		},
		{
			name:           "nok - 401",
			gotStatus:      http.StatusUnauthorized,
			gotContentType: "application/json",
			gotBody:        `{ "message": "Incorrect API key provided." }`,
			wantIssue:      fmt.Sprintf("%s responded with a 401 status code. Incorrect API key provided.", testName),
			wantLogFields:  []string{"url", "body", "status"},
		},
		{
			name:           "nok - JSON error",
			gotStatus:      http.StatusOK,
			gotContentType: "application/json",
			gotBody:        `{ `,
			wantLogFields:  []string{"url", "body", "error"},
		},
	}

	for _, tc := range testcases {
		c.Run(tc.name, func(c *qt.C) {
			h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", tc.gotContentType)
				w.WriteHeader(tc.gotStatus)
				fmt.Fprintln(w, tc.gotBody)
			})

			srv := httptest.NewServer(h)
			c.Cleanup(srv.Close)

			var errResp errBody
			zCore, zLogs := observer.New(zap.InfoLevel)
			client := New(testName, srv.URL,
				WithLogger(zap.New(zCore)),
				WithEndUserError(errResp),
			)

			_, err := client.R().SetResult(new(okBody)).Post(path)
			c.Check(err, qt.IsNotNil)
			c.Check(errmsg.Message(err), qt.Equals, tc.wantIssue)

			// Error log contains desired keys.
			for _, k := range tc.wantLogFields {
				logs := zLogs.FilterFieldKey(k)
				c.Check(logs.Len(), qt.Equals, 1, qt.Commentf("missing field in log: %s", k))
			}

			// All logs contain the "name" key. Sometimes (e.g. on
			// unmarshalling error) we'll have > 1 log so the assertion above
			// is too particular.
			logs := zLogs.FilterFieldKey("name")
			c.Assert(logs.Len(), qt.Not(qt.Equals), 0)
			c.Check(logs.All()[0].ContextMap()["name"], qt.Equals, testName)
		})
	}
}

type okBody struct {
	Added int `json:"added"`
}

// errBody is the error paylaod of the test API.
type errBody struct {
	Msg string `json:"message"`
}

// Message is a way to access the error message from the error payload.
func (e errBody) Message() string {
	return e.Msg
}
