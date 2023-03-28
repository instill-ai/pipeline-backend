package middleware

import (
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
)

// AppendCustomHeaderMiddleware appends custom headers
func AppendCustomHeaderMiddleware(next runtime.HandlerFunc) runtime.HandlerFunc {
	return runtime.HandlerFunc(func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
		r.Header.Add("owner", "users/local-user")
		next(w, r, pathParams)
	})
}
