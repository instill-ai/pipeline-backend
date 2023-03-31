package middleware

import (
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	"github.com/instill-ai/pipeline-backend/pkg/constant"
)

// AppendCustomHeaderMiddleware appends custom headers
func AppendCustomHeaderMiddleware(next runtime.HandlerFunc) runtime.HandlerFunc {
	return runtime.HandlerFunc(func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
		r.Header.Add(constant.HeaderOwnerIDKey, constant.DefaultOwnerID)
		next(w, r, pathParams)
	})
}
