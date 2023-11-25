package handler

import (
	"context"

	"github.com/instill-ai/pipeline-backend/internal/resource"

	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1alpha"
)

func (h *PublicHandler) GetOperation(ctx context.Context, req *pipelinePB.GetOperationRequest) (*pipelinePB.GetOperationResponse, error) {

	operationId, err := resource.GetOperationID(req.Name)
	if err != nil {
		return &pipelinePB.GetOperationResponse{}, err
	}
	operation, err := h.service.GetOperation(ctx, operationId)
	if err != nil {
		return &pipelinePB.GetOperationResponse{}, err
	}

	return &pipelinePB.GetOperationResponse{
		Operation: operation,
	}, nil
}
