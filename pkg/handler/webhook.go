package handler

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/instill-ai/pipeline-backend/pkg/service"

	pipelinepb "github.com/instill-ai/protogen-go/pipeline/pipeline/v1beta"
)

func (h *PublicHandler) DispatchPipelineWebhookEvent(ctx context.Context, req *pipelinepb.DispatchPipelineWebhookEventRequest) (resp *pipelinepb.DispatchPipelineWebhookEventResponse, err error) {

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Internal, "failed to get metadata")
	}

	headers := make(map[string][]string)
	for k, v := range md {
		headers[k] = v
	}

	output, err := h.service.DispatchPipelineWebhookEvent(ctx, service.DispatchPipelineWebhookEventParams{
		WebhookType: req.GetWebhookType(),
		Headers:     headers,
		Message:     req.GetMessage(),
	})
	if err != nil {
		return nil, err
	}

	return &pipelinepb.DispatchPipelineWebhookEventResponse{Response: output.Response}, nil

}
