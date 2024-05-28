package middleware

import (
	"net/http"
	"net/http/httptest"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/instill-ai/protogen-go/vdp/pipeline/v1beta"
)

type fn func(*runtime.ServeMux, pb.PipelinePublicServiceClient, http.ResponseWriter, *http.Request, map[string]string)

// AppendCustomHeaderMiddleware appends custom headers
func AppendCustomHeaderMiddleware(mux *runtime.ServeMux, client pb.PipelinePublicServiceClient, next fn) runtime.HandlerFunc {

	return runtime.HandlerFunc(func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
		next(mux, client, w, r, pathParams)
	})
}

// sseResponseStreamingMiddleware intercepts requests with X-Use-SSE header present
// and gives back immediately a session token. It continues calling the grpc-gateway
// endpoint and streams data to the SSE handler using the session ID token.
func sseResponseStreamingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Use-SSE") == "true" {

			sessionUUID := generateSecureSessionID()
			dataChan := make(chan []byte, 10) //TODO tillknuesting: Make the buffer configurable
			DataChanMap.Store(sessionUUID, dataChan)

			sessionData := SessionMetadata{
				SessionUUID:      sessionUUID,
				SourceInstanceID: "test-server-1", // TODO tillknuesting: get with from env
			}

			// Marshal session metadata into JSON
			responseData, err := json.Marshal(sessionData)
			if err != nil {
				http.Error(w, "Failed to generate session", http.StatusInternalServerError)
				return
			}

			// Get the underlying connection using http.Hijacker
			hijacker, ok := w.(http.Hijacker)
			if !ok {
				http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
				return
			}

			conn, bufw, err := hijacker.Hijack()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			// Set the response headers
			bufw.WriteString("HTTP/1.1 200 OK\r\n")
			bufw.WriteString("Content-Type: application/json\r\n")
			bufw.WriteString("Connection: close\r\n\r\n")

			// Write the initial response
			bufw.WriteString(string(responseData) + "\n\n")
			bufw.Flush()
			conn.Close()

			sw := &captureResponseWriter{
				ResponseWriter: httptest.NewRecorder(),
				DataChan:       dataChan,
			}

			next.ServeHTTP(sw, r)
			return
		}

		next.ServeHTTP(w, r)
	})
}
