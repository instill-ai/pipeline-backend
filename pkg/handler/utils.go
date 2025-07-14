package handler

import (
	"context"

	"github.com/instill-ai/pipeline-backend/pkg/constant"
	"github.com/instill-ai/x/resource"

	constantx "github.com/instill-ai/x/constant"
	errorsx "github.com/instill-ai/x/errors"
)

func authenticateUser(ctx context.Context, allowVisitor bool) error {
	if resource.GetRequestSingleHeader(ctx, constant.HeaderServiceKey) == "instill" {
		return nil
	}

	if resource.GetRequestSingleHeader(ctx, constantx.HeaderAuthTypeKey) == "user" {
		if resource.GetRequestSingleHeader(ctx, constantx.HeaderUserUIDKey) == "" {
			return errorsx.ErrUnauthenticated
		}
		return nil
	}

	if !allowVisitor {
		return errorsx.ErrUnauthenticated
	}

	if resource.GetRequestSingleHeader(ctx, constantx.HeaderVisitorUIDKey) == "" {
		return errorsx.ErrUnauthenticated
	}

	return nil
}
