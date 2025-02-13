package handler

import (
	"context"

	"go.opentelemetry.io/otel/trace"

	pb "github.com/instill-ai/protogen-go/pipeline/pipeline/v1beta"
)

// ListComponentDefinitions returns a paginated list of component definitions.
func (h *PublicHandler) ListComponentDefinitions(ctx context.Context, req *pb.ListComponentDefinitionsRequest) (*pb.ListComponentDefinitionsResponse, error) {
	ctx, span := tracer.Start(ctx, "ListComponentDefinitions", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	resp, err := h.service.ListComponentDefinitions(ctx, req)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	h.log.Info("ListComponentDefinitions")
	return resp, nil
}
