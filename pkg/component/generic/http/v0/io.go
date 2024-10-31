package http

import "github.com/instill-ai/pipeline-backend/pkg/data/format"

type httpInput struct {
	EndpointURL string              `instill:"endpoint-url"`
	Header      map[string][]string `instill:"header"`
	Body        format.Value        `instill:"body"`
}

type httpOutput struct {
	Header     map[string][]string `instill:"header"`
	Body       format.Value        `instill:"body"`
	StatusCode int                 `instill:"status-code"`
}
