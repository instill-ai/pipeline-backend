package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/instill-ai/pipeline-backend/pkg/handler"
	pb "github.com/instill-ai/protogen-go/vdp/pipeline/v1beta"
	"net/http"
	"net/http/httptest"
)

type fn func(*runtime.ServeMux, pb.PipelinePublicServiceClient, http.ResponseWriter, *http.Request, map[string]string)

// AppendCustomHeaderMiddleware appends custom headers
func AppendCustomHeaderMiddleware(mux *runtime.ServeMux, client pb.PipelinePublicServiceClient, next fn) runtime.HandlerFunc {

	return runtime.HandlerFunc(func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
		next(mux, client, w, r, pathParams)
	})
}

// SessionMetadata holds the session ID and source instance ID.
type SessionMetadata struct {
	SessionUUID string `json:"session_uuid"`
	// Source instance identifier is used for network routing scenarios
	// for example could be included as header in the SSE request to make sure
	// it is getting routed to the initiating server e.g. running a pod
	SourceInstanceID string `json:"source_instance_id"`
}

// generateSecureSessionID generated a cryptographic secure session token to be used
// to the url of the sse handler.
func generateSecureSessionID() string {
	generatedUUID := uuid.New().String()
	hash := sha256.Sum256([]byte(generatedUUID))
	return fmt.Sprintf("%x", hash)
}

// sseResponseStreamingMiddleware intercepts requests with X-Use-SSE header present
// and gives back immediately a session token. It continues calling the grpc-gateway
// endpoint and streams data to the SSE handler using the session ID token.
func SSEStreamResponseMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Instill-Use-SSE") == "true" {

			sessionUUID := generateSecureSessionID()
			dataChan := make(chan []byte, 100) //TODO tillknuesting: Make the buffer configurable
			handler.DataChanMap.Store(sessionUUID, dataChan)

			defer close(dataChan)

			sessionData := SessionMetadata{
				SessionUUID:      sessionUUID,
				SourceInstanceID: "test-server-1", // TODO tillknuesting: Make configurable
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

			// Create a new request with the new context
			newReq := r.Clone(context.Background())

			sw := &captureResponseWriter{
				ResponseWriter: httptest.NewRecorder(),
				DataChan:       dataChan,
			}

			next.ServeHTTP(sw, newReq)
			return
		}

		next.ServeHTTP(w, r)
	})
}

type captureResponseWriter struct {
	http.ResponseWriter
	DataChan chan []byte
}

func (mw *captureResponseWriter) Write(b []byte) (int, error) {
	if len(b) > 1 {
		mw.DataChan <- b
	}
	return mw.ResponseWriter.Write(b)
}

// Unwrap is used by the ResponseController in the grpc-gateway runtime to flush
// if method is not present there would be a server error.
func (mw *captureResponseWriter) Unwrap() http.ResponseWriter {
	return mw.ResponseWriter
}
