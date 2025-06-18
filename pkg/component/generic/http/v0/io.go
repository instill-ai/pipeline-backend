package http

import (
	"net/http"

	"github.com/instill-ai/pipeline-backend/pkg/data/format"
)

type httpInput struct {
	EndpointURL string       `instill:"endpoint-url"`
	Header      http.Header  `instill:"header"`
	Body        format.Value `instill:"body"`
}

type httpOutput struct {
	Header     http.Header  `instill:"header"`
	Body       format.Value `instill:"body"`
	StatusCode int          `instill:"status-code"`
}
