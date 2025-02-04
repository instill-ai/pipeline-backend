package handler

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"go.opentelemetry.io/otel"

	"github.com/instill-ai/pipeline-backend/pkg/service"

	errdomain "github.com/instill-ai/pipeline-backend/pkg/errors"
	healthcheckpb "github.com/instill-ai/protogen-go/common/healthcheck/v1beta"
	pipelinepb "github.com/instill-ai/protogen-go/pipeline/pipeline/v1beta"
)

// TODO: in the public_handler, we should convert all id to uuid when calling service

var tracer = otel.Tracer("pipeline-backend.public-handler.tracer")

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

	ns, err := h.service.GetNamespaceByID(ctx, name)
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
