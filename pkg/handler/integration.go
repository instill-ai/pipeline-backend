package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/instill-ai/pipeline-backend/pkg/logger"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

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

// ListIntegrations returns a paginated list of available integrations.
func (h *PublicHandler) ListIntegrations(ctx context.Context, req *pb.ListIntegrationsRequest) (*pb.ListIntegrationsResponse, error) {
	ctx, span := tracer.Start(ctx, "ListIntegrations", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger, _ := logger.GetZapLogger(ctx)

	resp, err := h.service.ListIntegrations(ctx, req)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	logger.Info("ListIntegrations")
	return resp, nil
}

// CreateNamespaceConnection creates a connection under the ownership of
// a namespace.
func (h *PublicHandler) CreateNamespaceConnection(ctx context.Context, req *pb.CreateNamespaceConnectionRequest) (*pb.CreateNamespaceConnectionResponse, error) {
	ctx, span := tracer.Start(ctx, "CreateNamespaceConnection", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger, _ := logger.GetZapLogger(ctx)

	if err := authenticateUser(ctx, false); err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	conn, err := h.service.CreateNamespaceConnection(ctx, req.GetConnection())
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	// Manually set the custom header to have a StatusCreated http response for
	// REST endpoint.
	err = grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusCreated)))
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	logger.Info("CreateNamespaceConnection")
	return &pb.CreateNamespaceConnectionResponse{Connection: conn}, nil
}
