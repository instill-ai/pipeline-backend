package service

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/gofrs/uuid"

	"github.com/instill-ai/pipeline-backend/pkg/constant"
	"github.com/instill-ai/pipeline-backend/pkg/resource"
	mgmtpb "github.com/instill-ai/protogen-go/core/mgmt/v1beta"

	errdomain "github.com/instill-ai/pipeline-backend/pkg/errors"
)

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

var seededRand *rand.Rand = rand.New(
	rand.NewSource(time.Now().UnixNano()))

func randomStrWithCharset(length int, charset string) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func generateShareCode() string {
	return randomStrWithCharset(32, charset)
}

func (s *service) checkNamespacePermission(ctx context.Context, ns resource.Namespace) error {
	// TODO: optimize ACL model
	if ns.NsType == "organizations" {
		granted, err := s.aclClient.CheckPermission(ctx, "organization", ns.NsUID, "member")
		if err != nil {
			return err
		}
		if !granted {
			return errdomain.ErrUnauthorized
		}
	} else {
		if ns.NsUID != uuid.FromStringOrNil(resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey)) {
			return errdomain.ErrUnauthorized
		}
	}
	return nil
}

func (s *service) GetCtxUserNamespace(ctx context.Context) (resource.Namespace, error) {

	uid := uuid.FromStringOrNil(resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey))
	resp, err := s.mgmtPrivateServiceClient.CheckNamespaceByUIDAdmin(ctx, &mgmtpb.CheckNamespaceByUIDAdminRequest{
		Uid: uid.String(),
	})
	if err != nil || resp.Type != mgmtpb.CheckNamespaceByUIDAdminResponse_NAMESPACE_USER {
		return resource.Namespace{}, fmt.Errorf("namespace error")
	}
	return resource.Namespace{
		NsType: resource.NamespaceType("users"),
		NsID:   resp.Id,
		NsUID:  uid,
	}, nil
}
func (s *service) GetRscNamespace(ctx context.Context, namespaceID string) (resource.Namespace, error) {

	resp, err := s.mgmtPrivateServiceClient.CheckNamespaceAdmin(ctx, &mgmtpb.CheckNamespaceAdminRequest{
		Id: namespaceID,
	})
	if err != nil {
		return resource.Namespace{}, err
	}
	if resp.Type == mgmtpb.CheckNamespaceAdminResponse_NAMESPACE_USER {
		return resource.Namespace{
			NsType: resource.User,
			NsID:   namespaceID,
			NsUID:  uuid.FromStringOrNil(resp.Uid),
		}, nil
	} else if resp.Type == mgmtpb.CheckNamespaceAdminResponse_NAMESPACE_ORGANIZATION {
		return resource.Namespace{
			NsType: resource.Organization,
			NsID:   namespaceID,
			NsUID:  uuid.FromStringOrNil(resp.Uid),
		}, nil
	}
	return resource.Namespace{}, fmt.Errorf("namespace error")
}
