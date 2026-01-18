package handler

import (
	"context"
	"fmt"

	"github.com/instill-ai/pipeline-backend/pkg/service"

	healthcheckpb "github.com/instill-ai/protogen-go/common/healthcheck/v1beta"
	pipelinepb "github.com/instill-ai/protogen-go/pipeline/v1beta"
)

// TODO: in the public_handler, we should convert all id to uuid when calling service

// PublicHandler handles public API
type PublicHandler struct {
	pipelinepb.UnimplementedPipelinePublicServiceServer
	service service.Service

	ready bool
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
func NewPublicHandler(s service.Service) *PublicHandler {
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

func (h *PublicHandler) SetReadiness(r bool) {
	h.ready = r
}

func (h *PublicHandler) Liveness(ctx context.Context, req *pipelinepb.LivenessRequest) (*pipelinepb.LivenessResponse, error) {
	return &pipelinepb.LivenessResponse{
		HealthCheckResponse: &healthcheckpb.HealthCheckResponse{
			Status: healthcheckpb.HealthCheckResponse_SERVING_STATUS_SERVING,
		},
	}, nil
}

func (h *PublicHandler) Readiness(ctx context.Context, req *pipelinepb.ReadinessRequest) (*pipelinepb.ReadinessResponse, error) {
	if h.ready {
		return &pipelinepb.ReadinessResponse{
			HealthCheckResponse: &healthcheckpb.HealthCheckResponse{
				Status: healthcheckpb.HealthCheckResponse_SERVING_STATUS_SERVING,
			},
		}, nil
	} else {
		return nil, fmt.Errorf("service not ready")
	}
}

// PrivateHandler handles private API
type PrivateHandler struct {
	pipelinepb.UnimplementedPipelinePrivateServiceServer
	service service.Service
}

// NewPrivateHandler initiates a handler instance
func NewPrivateHandler(s service.Service) *PrivateHandler {
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
