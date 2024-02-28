package handler

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/trace"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/protobuf/proto"

	pb "github.com/instill-ai/protogen-go/vdp/pipeline/v1beta"
	"github.com/instill-ai/x/paginate"
	"github.com/instill-ai/x/sterr"

	"github.com/instill-ai/pipeline-backend/internal/resource"
	"github.com/instill-ai/pipeline-backend/pkg/logger"
	"github.com/instill-ai/pipeline-backend/pkg/repository"
)

func (h *PublicHandler) ListConnectorDefinitions(ctx context.Context, req *pb.ListConnectorDefinitionsRequest) (*pb.ListConnectorDefinitionsResponse, error) {
	ctx, span := tracer.Start(ctx, "ListConnectorDefinitions", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger, _ := logger.GetZapLogger(ctx)

	resp, err := h.service.ListConnectorDefinitions(ctx, req)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	logger.Info("ListConnectorDefinitions")

	return resp, nil
}

func (h *PublicHandler) GetConnectorDefinition(ctx context.Context, req *pb.GetConnectorDefinitionRequest) (resp *pb.GetConnectorDefinitionResponse, err error) {
	ctx, span := tracer.Start(ctx, "GetConnectorDefinition",
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger, _ := logger.GetZapLogger(ctx)

	resp = &pb.GetConnectorDefinitionResponse{}

	var connID string

	if connID, err = resource.GetRscNameID(req.GetName()); err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	dbDef, err := h.service.GetConnectorDefinitionByID(ctx, connID, parseView(int32(*req.GetView().Enum())))
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}
	resp.ConnectorDefinition = dbDef

	logger.Info("GetConnectorDefinition")
	return resp, nil

}

func (h *PublicHandler) ListOperatorDefinitions(ctx context.Context, req *pb.ListOperatorDefinitionsRequest) (*pb.ListOperatorDefinitionsResponse, error) {
	ctx, span := tracer.Start(ctx, "ListOperatorDefinitions", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger, _ := logger.GetZapLogger(ctx)

	resp, err := h.service.ListOperatorDefinitions(ctx, req)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	logger.Info("ListOperatorDefinitions")
	return resp, nil
}

func (h *PublicHandler) GetOperatorDefinition(ctx context.Context, req *pb.GetOperatorDefinitionRequest) (resp *pb.GetOperatorDefinitionResponse, err error) {
	ctx, span := tracer.Start(ctx, "GetOperatorDefinition",
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger, _ := logger.GetZapLogger(ctx)

	resp = &pb.GetOperatorDefinitionResponse{}

	var connID string

	if connID, err = resource.GetRscNameID(req.GetName()); err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}
	isBasicView := (req.GetView() == pb.ComponentDefinition_VIEW_BASIC) || (req.GetView() == pb.ComponentDefinition_VIEW_UNSPECIFIED)

	dbDef, err := h.service.GetOperatorDefinitionByID(ctx, connID)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}
	resp.OperatorDefinition = proto.Clone(dbDef).(*pb.OperatorDefinition)
	if isBasicView {
		resp.OperatorDefinition.Spec = nil
	}
	resp.OperatorDefinition.Name = fmt.Sprintf("operator-definitions/%s", resp.OperatorDefinition.GetId())

	logger.Info("GetOperatorDefinition")
	return resp, nil
}
