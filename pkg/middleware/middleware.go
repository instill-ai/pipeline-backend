package middleware

import (
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1alpha"
)

type fn func(*runtime.ServeMux, pipelinePB.PipelinePublicServiceClient, http.ResponseWriter, *http.Request, map[string]string)

// AppendCustomHeaderMiddleware appends custom headers
func AppendCustomHeaderMiddleware(mux *runtime.ServeMux, client pipelinePB.PipelinePublicServiceClient, next fn) runtime.HandlerFunc {

	return runtime.HandlerFunc(func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
		next(mux, client, w, r, pathParams)
	})
}
