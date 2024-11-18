package handler

import (
	"context"

	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/instill-ai/pipeline-backend/pkg/service"

	pb "github.com/instill-ai/protogen-go/vdp/pipeline/v1beta"
)

func (h *PublicHandler) DispatchPipelineWebhookEvent(ctx context.Context, req *pb.DispatchPipelineWebhookEventRequest) (resp *pb.DispatchPipelineWebhookEventResponse, err error) {

	eventName := "DispatchPipelineWebhookEvent"
	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

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
		span.SetStatus(1, err.Error())
		return nil, err
	}

	return &pb.DispatchPipelineWebhookEventResponse{Response: output.Response}, nil

}
