package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gofrs/uuid"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/instill-ai/pipeline-backend/pkg/logger"

	customotel "github.com/instill-ai/pipeline-backend/pkg/logger/otel"
	pb "github.com/instill-ai/protogen-go/vdp/pipeline/v1beta"
)

// GetIntegration returns the details of an integration.
func (h *PublicHandler) GetIntegration(ctx context.Context, req *pb.GetIntegrationRequest) (*pb.GetIntegrationResponse, error) {
	eventName := "GetIntegration"
	ctx, span := tracer.Start(ctx, eventName, trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger, _ := logger.GetZapLogger(ctx)
	logUUID, _ := uuid.NewV4()

	view := req.GetView()
	if view == pb.View_VIEW_UNSPECIFIED {
		view = pb.View_VIEW_BASIC
	}

	integration, err := h.service.GetIntegration(ctx, req.GetIntegrationId(), view)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	logger.Info(string(customotel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
	)))
	return &pb.GetIntegrationResponse{Integration: integration}, nil
}

// ListIntegrations returns a paginated list of available integrations.
func (h *PublicHandler) ListIntegrations(ctx context.Context, req *pb.ListIntegrationsRequest) (*pb.ListIntegrationsResponse, error) {
	eventName := "ListIntegrations"
	ctx, span := tracer.Start(ctx, eventName, trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger, _ := logger.GetZapLogger(ctx)
	logUUID, _ := uuid.NewV4()

	resp, err := h.service.ListIntegrations(ctx, req)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	logger.Info(string(customotel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
	)))
	return resp, nil
}

// CreateNamespaceConnection creates a connection under the ownership of
// a namespace.
func (h *PublicHandler) CreateNamespaceConnection(ctx context.Context, req *pb.CreateNamespaceConnectionRequest) (*pb.CreateNamespaceConnectionResponse, error) {
	eventName := "CreateNamespaceConnection"
	ctx, span := tracer.Start(ctx, eventName, trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger, _ := logger.GetZapLogger(ctx)
	logUUID, _ := uuid.NewV4()

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

	logger.Info(string(customotel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
	)))
	return &pb.CreateNamespaceConnectionResponse{Connection: conn}, nil
}

// GetNamespaceConnection fetches the details of a namespace connection.
func (h *PublicHandler) GetNamespaceConnection(ctx context.Context, req *pb.GetNamespaceConnectionRequest) (*pb.GetNamespaceConnectionResponse, error) {
	eventName := "GetNamespaceConnection"
	ctx, span := tracer.Start(ctx, eventName, trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger, _ := logger.GetZapLogger(ctx)
	logUUID, _ := uuid.NewV4()

	if err := authenticateUser(ctx, false); err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	conn, err := h.service.GetNamespaceConnection(ctx, req)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	logger.Info(string(customotel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
	)))
	return &pb.GetNamespaceConnectionResponse{Connection: conn}, nil
}

// ListNamespaceConnections returns a paginated list of connections created by
// a namespace.
func (h *PublicHandler) ListNamespaceConnections(ctx context.Context, req *pb.ListNamespaceConnectionsRequest) (*pb.ListNamespaceConnectionsResponse, error) {
	eventName := "ListNamespaceConnections"
	ctx, span := tracer.Start(ctx, eventName, trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger, _ := logger.GetZapLogger(ctx)
	logUUID, _ := uuid.NewV4()

	if err := authenticateUser(ctx, false); err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	resp, err := h.service.ListNamespaceConnections(ctx, req)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	logger.Info(string(customotel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
	)))
	return resp, nil
}
