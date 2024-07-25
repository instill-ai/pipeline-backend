package handler

import (
	"context"

	"github.com/instill-ai/pipeline-backend/pkg/constant"
	"github.com/instill-ai/pipeline-backend/pkg/resource"
	"github.com/instill-ai/pipeline-backend/pkg/service"
)

func authenticateUser(ctx context.Context, allowVisitor bool) error {
	if resource.GetRequestSingleHeader(ctx, constant.HeaderAuthTypeKey) == "user" {
		if resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey) == "" {
			return service.ErrUnauthenticated
		}
		return nil
	}

	if !allowVisitor {
		return service.ErrUnauthenticated
	}

	if resource.GetRequestSingleHeader(ctx, constant.HeaderVisitorUIDKey) == "" {
		return service.ErrUnauthenticated
	}

	return nil
}
