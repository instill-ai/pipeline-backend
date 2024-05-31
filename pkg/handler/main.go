package handler

import (
	"context"
	"errors"
	"strings"

	"go.opentelemetry.io/otel"

	"github.com/instill-ai/pipeline-backend/pkg/service"

	healthcheckPB "github.com/instill-ai/protogen-go/common/healthcheck/v1beta"
	pb "github.com/instill-ai/protogen-go/vdp/pipeline/v1beta"
)

// TODO: in the public_handler, we should convert all id to uuid when calling service

var tracer = otel.Tracer("pipeline-backend.public-handler.tracer")

// PublicHandler handles public API
type PublicHandler struct {
	pb.UnimplementedPipelinePublicServiceServer
	service service.Service
}

type Streamer interface {
	Context() context.Context
}

type TriggerPipelineRequestInterface interface {
	GetName() string
}

// NewPublicHandler initiates a handler instance
func NewPublicHandler(ctx context.Context, s service.Service) pb.PipelinePublicServiceServer {
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

func (h *PublicHandler) Liveness(ctx context.Context, req *pb.LivenessRequest) (*pb.LivenessResponse, error) {
	return &pb.LivenessResponse{
		HealthCheckResponse: &healthcheckPB.HealthCheckResponse{
			Status: healthcheckPB.HealthCheckResponse_SERVING_STATUS_SERVING,
		},
	}, nil
}

func (h *PublicHandler) Readiness(ctx context.Context, req *pb.ReadinessRequest) (*pb.ReadinessResponse, error) {
	return &pb.ReadinessResponse{
		HealthCheckResponse: &healthcheckPB.HealthCheckResponse{
			Status: healthcheckPB.HealthCheckResponse_SERVING_STATUS_SERVING,
		},
	}, nil
}

// PrivateHandler handles private API
type PrivateHandler struct {
	pb.UnimplementedPipelinePrivateServiceServer
	service service.Service
}

// NewPrivateHandler initiates a handler instance
func NewPrivateHandler(ctx context.Context, s service.Service) pb.PipelinePrivateServiceServer {
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

func (h *PublicHandler) CheckName(ctx context.Context, req *pb.CheckNameRequest) (resp *pb.CheckNameResponse, err error) {
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
		_, err := h.service.GetNamespacePipelineByID(ctx, ns, id, pb.Pipeline_VIEW_BASIC)
		if err != nil && errors.Is(err, service.ErrNotFound) {
			return &pb.CheckNameResponse{
				Availability: pb.CheckNameResponse_NAME_AVAILABLE,
			}, nil
		}
	} else {
		return &pb.CheckNameResponse{
			Availability: pb.CheckNameResponse_NAME_UNAVAILABLE,
		}, nil
	}
	return &pb.CheckNameResponse{
		Availability: pb.CheckNameResponse_NAME_UNAVAILABLE,
	}, nil
}
