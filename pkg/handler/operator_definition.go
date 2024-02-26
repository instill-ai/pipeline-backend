package handler

import (
	"context"
	"fmt"

	"time"

	"go.opentelemetry.io/otel/trace"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/protobuf/proto"

	"github.com/instill-ai/pipeline-backend/internal/resource"
	"github.com/instill-ai/pipeline-backend/pkg/logger"

	"github.com/instill-ai/pipeline-backend/pkg/repository"
	"github.com/instill-ai/x/paginate"
	"github.com/instill-ai/x/sterr"

	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1beta"
)

func (h *PublicHandler) ListOperatorDefinitions(ctx context.Context, req *pipelinePB.ListOperatorDefinitionsRequest) (resp *pipelinePB.ListOperatorDefinitionsResponse, err error) {
	ctx, span := tracer.Start(ctx, "ListOperatorDefinitions",
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger, _ := logger.GetZapLogger(ctx)

	resp = &pipelinePB.ListOperatorDefinitionsResponse{}
	pageSize := req.GetPageSize()
	pageToken := req.GetPageToken()
	isBasicView := (req.GetView() == pipelinePB.OperatorDefinition_VIEW_BASIC) || (req.GetView() == pipelinePB.OperatorDefinition_VIEW_UNSPECIFIED)

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

	page := []*pipelinePB.OperatorDefinition{}
	for i := 0; i < int(pageSize) && startIdx+i < len(defs); i++ {
		def := proto.Clone(defs[startIdx+i]).(*pipelinePB.OperatorDefinition)
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

func (h *PublicHandler) GetOperatorDefinition(ctx context.Context, req *pipelinePB.GetOperatorDefinitionRequest) (resp *pipelinePB.GetOperatorDefinitionResponse, err error) {
	ctx, span := tracer.Start(ctx, "GetOperatorDefinition",
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger, _ := logger.GetZapLogger(ctx)

	resp = &pipelinePB.GetOperatorDefinitionResponse{}

	var connID string

	if connID, err = resource.GetRscNameID(req.GetName()); err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}
	isBasicView := (req.GetView() == pipelinePB.OperatorDefinition_VIEW_BASIC) || (req.GetView() == pipelinePB.OperatorDefinition_VIEW_UNSPECIFIED)

	dbDef, err := h.service.GetOperatorDefinitionByID(ctx, connID)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}
	resp.OperatorDefinition = proto.Clone(dbDef).(*pipelinePB.OperatorDefinition)
	if isBasicView {
		resp.OperatorDefinition.Spec = nil
	}
	resp.OperatorDefinition.Name = fmt.Sprintf("operator-definitions/%s", resp.OperatorDefinition.GetId())

	logger.Info("GetOperatorDefinition")
	return resp, nil
}
