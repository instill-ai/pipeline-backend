package handler

import (
	"context"

	"github.com/instill-ai/pipeline-backend/pkg/logger"
	"go.opentelemetry.io/otel/trace"

	pb "github.com/instill-ai/protogen-go/vdp/pipeline/v1beta"
)

// GetIntegration returns the details of an integration.
func (h *PublicHandler) GetIntegration(ctx context.Context, req *pb.GetIntegrationRequest) (*pb.GetIntegrationResponse, error) {
	ctx, span := tracer.Start(ctx, "GetIntegration", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()
	logger, _ := logger.GetZapLogger(ctx)

	view := req.GetView()
	if view == pb.View_VIEW_UNSPECIFIED {
		view = pb.View_VIEW_BASIC
	}

	integration, err := h.service.GetIntegration(ctx, req.GetIntegrationId(), view)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	logger.Info("GetIntegration")
	return &pb.GetIntegrationResponse{Integration: integration}, nil
}
