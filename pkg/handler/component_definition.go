package handler

import (
	"context"

	pipelinepb "github.com/instill-ai/protogen-go/pipeline/v1beta"
)

// ListComponentDefinitions returns a paginated list of component definitions.
func (h *PublicHandler) ListComponentDefinitions(ctx context.Context, req *pipelinepb.ListComponentDefinitionsRequest) (*pipelinepb.ListComponentDefinitionsResponse, error) {
	resp, err := h.service.ListComponentDefinitions(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
