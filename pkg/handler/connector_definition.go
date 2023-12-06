package handler

import (
	"context"

	"go.einride.tech/aip/filtering"
	"go.opentelemetry.io/otel/trace"

	"github.com/instill-ai/pipeline-backend/internal/resource"
	"github.com/instill-ai/pipeline-backend/pkg/logger"

	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1beta"
)

func (h *PrivateHandler) LookUpConnectorDefinitionAdmin(ctx context.Context, req *pipelinePB.LookUpConnectorDefinitionAdminRequest) (resp *pipelinePB.LookUpConnectorDefinitionAdminResponse, err error) {

	resp = &pipelinePB.LookUpConnectorDefinitionAdminResponse{}

	connUID, err := resource.GetRscPermalinkUID(req.GetPermalink())
	if err != nil {
		return resp, err
	}

	// TODO add a service wrapper
	def, err := h.service.GetConnectorDefinitionByUIDAdmin(ctx, connUID, parseView(int32(*req.GetView().Enum())))
	if err != nil {
		return resp, err
	}
	resp.ConnectorDefinition = def

	return resp, nil
}

func (h *PublicHandler) ListConnectorDefinitions(ctx context.Context, req *pipelinePB.ListConnectorDefinitionsRequest) (resp *pipelinePB.ListConnectorDefinitionsResponse, err error) {
	ctx, span := tracer.Start(ctx, "ListConnectorDefinitions",
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger, _ := logger.GetZapLogger(ctx)

	resp = &pipelinePB.ListConnectorDefinitionsResponse{}
	pageSize := int64(req.GetPageSize())
	pageToken := req.GetPageToken()

	var connType pipelinePB.ConnectorType
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

func (h *PublicHandler) GetConnectorDefinition(ctx context.Context, req *pipelinePB.GetConnectorDefinitionRequest) (resp *pipelinePB.GetConnectorDefinitionResponse, err error) {
	ctx, span := tracer.Start(ctx, "GetConnectorDefinition",
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger, _ := logger.GetZapLogger(ctx)

	resp = &pipelinePB.GetConnectorDefinitionResponse{}

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
