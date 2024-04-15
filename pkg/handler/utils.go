package handler

import (
	"context"

	"github.com/instill-ai/pipeline-backend/internal/resource"
	"github.com/instill-ai/pipeline-backend/pkg/constant"
	"github.com/instill-ai/pipeline-backend/pkg/service"
	// pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1beta"
)

func authenticateUser(ctx context.Context, allowVisitor bool) error {
	if resource.GetRequestSingleHeader(ctx, constant.HeaderAuthTypeKey) == "user" {
		if resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey) == "" {
			return service.ErrUnauthenticated
		}
		return nil
	} else {
		if !allowVisitor {
			return service.ErrUnauthenticated
		}
		if resource.GetRequestSingleHeader(ctx, constant.HeaderVisitorUIDKey) == "" {
			return service.ErrUnauthenticated
		}
		return nil
	}
}
