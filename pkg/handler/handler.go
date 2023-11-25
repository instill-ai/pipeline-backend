package handler

import (
	"context"

	"go.opentelemetry.io/otel"

	"github.com/instill-ai/pipeline-backend/pkg/datamodel"

	"github.com/instill-ai/pipeline-backend/pkg/service"

	healthcheckPB "github.com/instill-ai/protogen-go/common/healthcheck/v1alpha"
	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1alpha"
)

// TODO: in the public_handler, we should convert all id to uuid when calling service

var tracer = otel.Tracer("pipeline-backend.public-handler.tracer")

// PublicHandler handles public API
type PublicHandler struct {
	pipelinePB.UnimplementedPipelinePublicServiceServer
	service service.Service
}

type Streamer interface {
	Context() context.Context
}

type TriggerPipelineRequestInterface interface {
	GetName() string
}

// NewPublicHandler initiates a handler instance
func NewPublicHandler(ctx context.Context, s service.Service) pipelinePB.PipelinePublicServiceServer {
	datamodel.InitJSONSchema(ctx)
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

func (h *PublicHandler) Liveness(ctx context.Context, req *pipelinePB.LivenessRequest) (*pipelinePB.LivenessResponse, error) {
	return &pipelinePB.LivenessResponse{
		HealthCheckResponse: &healthcheckPB.HealthCheckResponse{
			Status: healthcheckPB.HealthCheckResponse_SERVING_STATUS_SERVING,
		},
	}, nil
}

func (h *PublicHandler) Readiness(ctx context.Context, req *pipelinePB.ReadinessRequest) (*pipelinePB.ReadinessResponse, error) {
	return &pipelinePB.ReadinessResponse{
		HealthCheckResponse: &healthcheckPB.HealthCheckResponse{
			Status: healthcheckPB.HealthCheckResponse_SERVING_STATUS_SERVING,
		},
	}, nil
}

// PrivateHandler handles private API
type PrivateHandler struct {
	pipelinePB.UnimplementedPipelinePrivateServiceServer
	service service.Service
}

// NewPrivateHandler initiates a handler instance
func NewPrivateHandler(ctx context.Context, s service.Service) pipelinePB.PipelinePrivateServiceServer {
	datamodel.InitJSONSchema(ctx)
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
