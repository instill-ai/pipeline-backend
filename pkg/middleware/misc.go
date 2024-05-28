package middleware

import (
	"context"
	"crypto/sha256"
	"fmt"
	"github.com/google/uuid"
	"io"
	"net/http"
	"net/textproto"
	"strconv"
	"strings"
	"sync"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"go.opentelemetry.io/otel"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/instill-ai/pipeline-backend/pkg/logger"
)

// HTTPResponseModifier is a callback function for gRPC-Gateway runtime.WithForwardResponseOption
func HTTPResponseModifier(ctx context.Context, w http.ResponseWriter, p proto.Message) error {
	md, ok := runtime.ServerMetadataFromContext(ctx)
	if !ok {
		return nil
	}

	// set http status code
	if vals := md.HeaderMD.Get("x-http-code"); len(vals) > 0 {
		code, err := strconv.Atoi(vals[0])
		if err != nil {
			return err
		}
		// delete the headers to not expose any grpc-metadata in http response
		delete(md.HeaderMD, "x-http-code")
		delete(w.Header(), "Grpc-Metadata-X-Http-Code")
		w.WriteHeader(code)
	}

	return nil
}

// ErrorHandler is a callback function for gRPC-Gateway runtime.WithErrorHandler
func ErrorHandler(ctx context.Context, mux *runtime.ServeMux, marshaler runtime.Marshaler, w http.ResponseWriter, r *http.Request, err error) {

	ctx, span := otel.Tracer("ErrorTracer").Start(ctx,
		"ErrorHandler",
	)
	defer span.End()

	logger, _ := logger.GetZapLogger(ctx)

	// return Internal when Marshal failed
	const fallback = `{"code": 13, "message": "failed to marshal error message"}`

	s := status.Convert(err)
	pb := s.Proto()

	w.Header().Del("Trailer")
	w.Header().Del("Transfer-Encoding")

	contentType := marshaler.ContentType(pb)
	if contentType == "application/json" {
		w.Header().Set("Content-Type", "application/problem+json")
	} else {
		w.Header().Set("Content-Type", contentType)
	}

	if s.Code() == codes.Unauthenticated {
		w.Header().Set("WWW-Authenticate", s.Message())
	}

	buf, err := marshaler.Marshal(pb)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to marshal error message %q: %v", s, err))
		w.WriteHeader(http.StatusInternalServerError)
		if _, err := io.WriteString(w, fallback); err != nil {
			logger.Error(fmt.Sprintf("Failed to write response: %v", err))
		}
		return
	}

	md, ok := runtime.ServerMetadataFromContext(ctx)
	if !ok {
		logger.Error("Failed to extract ServerMetadata from context")
	}

	for k, vs := range md.HeaderMD {
		if h, ok := func(key string) (string, bool) {
			return fmt.Sprintf("%s%s", runtime.MetadataHeaderPrefix, key), true
		}(k); ok {
			for _, v := range vs {
				w.Header().Add(h, v)
			}
		}
	}

	// RFC 7230 https://tools.ietf.org/html/rfc7230#section-4.1.2
	// Unless the request includes a TE header field indicating "trailers"
	// is acceptable, as described in Section 4.3, a server SHOULD NOT
	// generate trailer fields that it believes are necessary for the user
	// agent to receive.
	doForwardTrailers := strings.Contains(strings.ToLower(r.Header.Get("TE")), "trailers")

	if doForwardTrailers {
		for k := range md.TrailerMD {
			tKey := textproto.CanonicalMIMEHeaderKey(fmt.Sprintf("%s%s", runtime.MetadataTrailerPrefix, k))
			w.Header().Add("Trailer", tKey)
		}
		w.Header().Set("Transfer-Encoding", "chunked")
	}

	var httpStatus int
	switch {
	case s.Code() == codes.FailedPrecondition:
		if len(s.Details()) > 0 {
			switch v := s.Details()[0].(type) {
			case *errdetails.PreconditionFailure:
				switch v.Violations[0].Type {
				case "UPDATE", "DELETE", "STATE", "RENAME", "TRIGGER":
					httpStatus = http.StatusUnprocessableEntity
				}
			}
		} else {
			httpStatus = http.StatusBadRequest
		}
	default:
		httpStatus = runtime.HTTPStatusFromCode(s.Code())
	}

	w.WriteHeader(httpStatus)
	if _, err := w.Write(buf); err != nil {
		logger.Error(fmt.Sprintf("Failed to write response: %v", err))
	}

	if doForwardTrailers {
		for k, vs := range md.TrailerMD {
			tKey := fmt.Sprintf("%s%s", runtime.MetadataTrailerPrefix, k)
			for _, v := range vs {
				w.Header().Add(tKey, v)
			}
		}
	}

}

// CustomMatcher is a callback function for gRPC-Gateway runtime.WithIncomingHeaderMatcher
func CustomMatcher(key string) (string, bool) {
	if strings.HasPrefix(strings.ToLower(key), "jwt-") {
		return key, true
	}
	if strings.HasPrefix(strings.ToLower(key), "instill-") {
		return key, true
	}

	switch key {
	case "request-id":
		return key, true
	case "X-B3-Traceid", "X-B3-Spanid", "X-B3-Sampled":
		return key, true
	default:
		return runtime.DefaultHeaderMatcher(key)
	}
}

// generateSecureSessionID generated a cryptographic secure session token to be used
// to the url of the sse handler.
func generateSecureSessionID() string {
	generatedUUID := uuid.New().String()
	hash := sha256.Sum256([]byte(generatedUUID))
	return fmt.Sprintf("%x", hash)
}

type captureResponseWriter struct {
	http.ResponseWriter
	DataChan chan []byte
}

func (mw *captureResponseWriter) Write(b []byte) (int, error) {
	if len(b) > 1 { // TODO tillknuesting: Verify why there are []bytes with len <2
		mw.DataChan <- b
	}
	return mw.ResponseWriter.Write(b)
}

// Unwrap is used by the ResponseController in the grpc-gateway runtime to flush
// if method is not present there would be a server error.
func (mw *captureResponseWriter) Unwrap() http.ResponseWriter {
	return mw.ResponseWriter
}

// TODO: Refactor to not use global
var DataChanMap sync.Map // Map to store data channels by session UUID.

// SessionMetadata holds the session ID and source instance ID.
type SessionMetadata struct {
	SessionUUID string `json:"session_uuid"`
	// Source instance identifier is used for network routing scenarios
	// for example could be included as header in the SSE request to make sure
	// it is getting routed to the initiating server e.g. running a pod
	SourceInstanceID string `json:"source_instance_id"`
}
