package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/otel"

	"github.com/instill-ai/pipeline-backend/pkg/service"

	errdomain "github.com/instill-ai/pipeline-backend/pkg/errors"
	healthcheckpb "github.com/instill-ai/protogen-go/common/healthcheck/v1beta"
	pipelinepb "github.com/instill-ai/protogen-go/vdp/pipeline/v1beta"
)

// TODO: in the public_handler, we should convert all id to uuid when calling service

var tracer = otel.Tracer("pipeline-backend.public-handler.tracer")

// PublicHandler handles public API
type PublicHandler struct {
	pipelinepb.UnimplementedPipelinePublicServiceServer
	service service.Service
}

type Streamer interface {
	Context() context.Context
}

type TriggerPipelineRequestInterface interface {
	GetNamespaceId() string
	GetPipelineId() string
}
type TriggerPipelineReleaseRequestInterface interface {
	GetNamespaceId() string
	GetPipelineId() string
	GetReleaseId() string
}

// NewPublicHandler initiates a handler instance
func NewPublicHandler(ctx context.Context, s service.Service) pipelinepb.PipelinePublicServiceServer {
	return &PublicHandler{
		service: s,
	}
}

// GetService returns the service
func (h *PublicHandler) GetService() service.Service {
	return h.service
}

// SetService sets the service
func (h *PublicHandler) SetService(s service.Service) {
	h.service = s
}

func (h *PublicHandler) Liveness(ctx context.Context, req *pipelinepb.LivenessRequest) (*pipelinepb.LivenessResponse, error) {
	return &pipelinepb.LivenessResponse{
		HealthCheckResponse: &healthcheckpb.HealthCheckResponse{
			Status: healthcheckpb.HealthCheckResponse_SERVING_STATUS_SERVING,
		},
	}, nil
}

func (h *PublicHandler) Readiness(ctx context.Context, req *pipelinepb.ReadinessRequest) (*pipelinepb.ReadinessResponse, error) {
	return &pipelinepb.ReadinessResponse{
		HealthCheckResponse: &healthcheckpb.HealthCheckResponse{
			Status: healthcheckpb.HealthCheckResponse_SERVING_STATUS_SERVING,
		},
	}, nil
}

// PrivateHandler handles private API
type PrivateHandler struct {
	pipelinepb.UnimplementedPipelinePrivateServiceServer
	service service.Service
}

// NewPrivateHandler initiates a handler instance
func NewPrivateHandler(ctx context.Context, s service.Service) pipelinepb.PipelinePrivateServiceServer {
	return &PrivateHandler{
		service: s,
	}
}

// GetService returns the service
func (h *PrivateHandler) GetService() service.Service {
	return h.service
}

// SetService sets the service
func (h *PrivateHandler) SetService(s service.Service) {
	h.service = s
}

func (h *PublicHandler) CheckName(ctx context.Context, req *pipelinepb.CheckNameRequest) (resp *pipelinepb.CheckNameResponse, err error) {
	name := req.GetName()

	ns, err := h.service.GetRscNamespace(ctx, name)
	if err != nil {
		return nil, err
	}
	if err := authenticateUser(ctx, false); err != nil {
		return nil, err
	}
	rscType := strings.Split(name, "/")[2]

	if rscType == "pipelines" {
		_, err := h.service.GetNamespacePipelineByID(ctx, ns, name, pipelinepb.Pipeline_VIEW_BASIC)
		if err != nil && errors.Is(err, errdomain.ErrNotFound) {
			return &pipelinepb.CheckNameResponse{
				Availability: pipelinepb.CheckNameResponse_NAME_AVAILABLE,
			}, nil
		}
	} else {
		return &pipelinepb.CheckNameResponse{
			Availability: pipelinepb.CheckNameResponse_NAME_UNAVAILABLE,
		}, nil
	}
	return &pipelinepb.CheckNameResponse{
		Availability: pipelinepb.CheckNameResponse_NAME_UNAVAILABLE,
	}, nil
}

// DataChanMap is used to store data channels by session UUID.
var DataChanMap sync.Map //TODO tillknuesting: Cleanup mechanism when a chan is closed or not used

func HandleSSEStreamResponse(w http.ResponseWriter, r *http.Request) {
	// Get the session UUID from the request URL
	sessionUUID := r.URL.Path[len("/sse/"):]

	// Get the data channel for the session UUID
	dataChanValue, ok := DataChanMap.Load(sessionUUID)
	if !ok {
		http.Error(w, "Invalid session UUID", http.StatusBadRequest)
		return
	}
	dataChan, ok := dataChanValue.(chan []byte)
	if !ok {
		http.Error(w, "Invalid data channel", http.StatusInternalServerError)
		return
	}

	// Set the response headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	w.Header().Set("Access-Control-Allow-Origin", "*")

	var lastTimestamp int64 = 0
	var eventIDCounter int64 = 0

	// Send the data chunks as SSE events
	for data := range dataChan {
		timestamp := time.Now().UnixNano()
		if timestamp == lastTimestamp {
			eventIDCounter++
		} else {
			eventIDCounter = 0
		}
		lastTimestamp = timestamp

		fmt.Fprintf(w, "event: output\n")
		fmt.Fprintf(w, "id: %d:%d\n", timestamp, eventIDCounter)
		fmt.Fprintf(w, "data: %s\n\n", data)

		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
	}

	// Send the "done" event
	currentTimestamp := time.Now().UnixNano()
	if currentTimestamp == lastTimestamp {
		eventIDCounter++
	} else {
		eventIDCounter = 0
	}

	fmt.Fprintf(w, "event: done\n")
	fmt.Fprintf(w, "id: %d:%d\n", currentTimestamp, eventIDCounter)
	fmt.Fprintf(w, "data: {}\n\n")
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	// Remove the data channel from the map when the SSE connection is closed
	DataChanMap.Delete(sessionUUID)
}
