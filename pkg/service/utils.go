package service

import (
	"context"

	mgmtPB "github.com/instill-ai/protogen-go/vdp/mgmt/v1alpha"
	"google.golang.org/grpc/metadata"
)

func GenOwnerPermalink(owner *mgmtPB.User) string {
	return "users/" + owner.GetUid()
}

func InjectOwnerToContext(ctx context.Context, owner *mgmtPB.User) context.Context {
	ctx = metadata.AppendToOutgoingContext(ctx, "Jwt-Sub", owner.GetUid())
	return ctx
}
