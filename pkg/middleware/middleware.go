package middleware

import (
	"context"
	"crypto/sha256"
	"fmt"
	"github.com/google/uuid"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/instill-ai/pipeline-backend/pkg/handler"
	"net/http"
	"net/http/httptest"

	pb "github.com/instill-ai/protogen-go/vdp/pipeline/v1beta"
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

type contextKey string

const SessionUUIDKey = contextKey("sessionUUID")

// SSEStreamResponseMiddleware intercepts requests with X-Use-SSE header present
// and gives back immediately a session token. It continues calling the grpc-gateway
// endpoint and streams data to the SSE handler using the session ID token.
func SSEStreamResponseMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Use-SSE") == "true" {
			fmt.Println("\"SSE Middleware:SSE request detected")
			sessionUUID := generateSecureSessionID()
			dataChan := make(chan []byte, 1000)
			handler.DataChanMap.Store(sessionUUID, dataChan)

			sessionData := SessionMetadata{
				SessionUUID:      sessionUUID,
				SourceInstanceID: "test-server-1",
			}

			fmt.Println("SessionDataID", sessionData.SessionUUID)

			// Return the session ID to the caller
			w.Header().Set("X-Session-ID", sessionData.SessionUUID)
			w.Header().Set("X-Session-InstanceID", sessionData.SourceInstanceID)
			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "application/json")
			jsonString := fmt.Sprintf(`{"SessionUUID": "%s", "SourceInstanceID": "%s"}`, sessionData.SessionUUID, sessionData.SourceInstanceID)
			w.Write([]byte(jsonString))

			ctx := context.WithValue(r.Context(), SessionUUIDKey, sessionUUID)

			// Create a new request with a new context
			newReq := r.Clone(ctx)

			// Create a new response writer that captures the response
			sw := &captureResponseWriter{
				ResponseWriter: httptest.NewRecorder(),
				DataChan:       dataChan,
			}

			// Serve the new request with the new response writer
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
