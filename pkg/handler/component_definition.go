package handler

import (
	"context"
	"fmt"
	"time"

	"go.einride.tech/aip/filtering"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/protobuf/proto"

	"github.com/instill-ai/pipeline-backend/internal/resource"
	"github.com/instill-ai/pipeline-backend/pkg/logger"

	"github.com/instill-ai/pipeline-backend/pkg/repository"
	"github.com/instill-ai/x/paginate"
	"github.com/instill-ai/x/sterr"

	pb "github.com/instill-ai/protogen-go/vdp/pipeline/v1beta"
)

func (h *PublicHandler) ListConnectorDefinitions(ctx context.Context, req *pb.ListConnectorDefinitionsRequest) (resp *pb.ListConnectorDefinitionsResponse, err error) {
	ctx, span := tracer.Start(ctx, "ListConnectorDefinitions",
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger, _ := logger.GetZapLogger(ctx)

	resp = &pb.ListConnectorDefinitionsResponse{}
	pageSize := int64(req.GetPageSize())
	pageToken := req.GetPageToken()

	var connType pb.ConnectorType
	declarations, err := filtering.NewDeclarations([]filtering.DeclarationOption{
		filtering.DeclareStandardFunctions(),
		filtering.DeclareEnumIdent("connector_type", connType.Type()),
	}...)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}
	filter, err := filtering.ParseFilter(req, declarations)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}
	defs, totalSize, nextPageToken, err := h.service.ListConnectorDefinitions(ctx, int32(pageSize), pageToken, parseView(int32(*req.GetView().Enum())), filter)

	if err != nil {
		return nil, err
	}

	resp.ConnectorDefinitions = defs
	resp.NextPageToken = nextPageToken
	resp.TotalSize = int32(totalSize)

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

func (h *PublicHandler) ListOperatorDefinitions(ctx context.Context, req *pb.ListOperatorDefinitionsRequest) (resp *pb.ListOperatorDefinitionsResponse, err error) {
	ctx, span := tracer.Start(ctx, "ListOperatorDefinitions",
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger, _ := logger.GetZapLogger(ctx)

	resp = &pb.ListOperatorDefinitionsResponse{}
	pageSize := req.GetPageSize()
	pageToken := req.GetPageToken()
	isBasicView := (req.GetView() == pb.ComponentDefinition_VIEW_BASIC) || (req.GetView() == pb.ComponentDefinition_VIEW_UNSPECIFIED)

	prevLastUID := ""

	if pageToken != "" {
		_, prevLastUID, err = paginate.DecodeToken(pageToken)
		if err != nil {
			st, err := sterr.CreateErrorBadRequest(
				fmt.Sprintf("[db] list operator error: %s", err.Error()),
				[]*errdetails.BadRequest_FieldViolation{
					{
						Field:       "page_token",
						Description: fmt.Sprintf("Invalid page token: %s", err.Error()),
					},
				},
			)
			if err != nil {
				logger.Error(err.Error())
			}
			return nil, st.Err()
		}
	}

	if pageSize == 0 {
		pageSize = repository.DefaultPageSize
	} else if pageSize > repository.MaxPageSize {
		pageSize = repository.MaxPageSize
	}

	defs := h.service.ListOperatorDefinitions(ctx)

	startIdx := 0
	lastUID := ""
	for idx, def := range defs {
		if def.Uid == prevLastUID {
			startIdx = idx + 1
			break
		}
	}

	page := []*pb.OperatorDefinition{}
	for i := 0; i < int(pageSize) && startIdx+i < len(defs); i++ {
		def := proto.Clone(defs[startIdx+i]).(*pb.OperatorDefinition)
		page = append(page, def)
		lastUID = def.Uid
	}

	nextPageToken := ""

	if startIdx+len(page) < len(defs) {
		nextPageToken = paginate.EncodeToken(time.Time{}, lastUID)
	}
	for _, def := range page {
		def.Name = fmt.Sprintf("operator-definitions/%s", def.Id)
		if isBasicView {
			def.Spec = nil
		}
		resp.OperatorDefinitions = append(
			resp.OperatorDefinitions,
			def)
	}
	resp.NextPageToken = nextPageToken
	resp.TotalSize = int32(len(defs))

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
