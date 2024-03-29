package handler

import (
	"context"
	"errors"
	"strings"

	"go.opentelemetry.io/otel"

	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/service"

	healthcheckPB "github.com/instill-ai/protogen-go/common/healthcheck/v1beta"
	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1beta"
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

func (h *PublicHandler) CheckName(ctx context.Context, req *pipelinePB.CheckNameRequest) (resp *pipelinePB.CheckNameResponse, err error) {
	name := req.GetName()

	ns, id, err := h.service.GetRscNamespaceAndNameID(ctx, name)
	if err != nil {
		return nil, err
	}
	if err := authenticateUser(ctx, false); err != nil {
		return nil, err
	}
	rscType := strings.Split(name, "/")[2]

	if rscType == "pipelines" {
		_, err := h.service.GetNamespacePipelineByID(ctx, ns, id, service.ViewBasic)
		if err != nil && errors.Is(err, service.ErrNotFound) {
			return &pipelinePB.CheckNameResponse{
				Availability: pipelinePB.CheckNameResponse_NAME_AVAILABLE,
			}, nil
		}
	} else if rscType == "connectors" {
		_, err := h.service.GetNamespaceConnectorByID(ctx, ns, id, service.ViewBasic, true)
		if err != nil && errors.Is(err, service.ErrNotFound) {
			return &pipelinePB.CheckNameResponse{
				Availability: pipelinePB.CheckNameResponse_NAME_AVAILABLE,
			}, nil
		}
	} else {
		return &pipelinePB.CheckNameResponse{
			Availability: pipelinePB.CheckNameResponse_NAME_UNAVAILABLE,
		}, nil
	}
	return &pipelinePB.CheckNameResponse{
		Availability: pipelinePB.CheckNameResponse_NAME_UNAVAILABLE,
	}, nil
}
