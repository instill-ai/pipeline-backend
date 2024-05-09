package service

import (
	"context"
	"fmt"

	"go.einride.tech/aip/filtering"

	"github.com/instill-ai/pipeline-backend/internal/resource"
	"github.com/instill-ai/pipeline-backend/pkg/constant"
	errdomain "github.com/instill-ai/pipeline-backend/pkg/errors"
	"github.com/instill-ai/x/errmsg"

	pb "github.com/instill-ai/protogen-go/vdp/pipeline/v1beta"
)

func (s *service) CreateNamespaceSecret(ctx context.Context, ns resource.Namespace, pbSecret *pb.Secret) (*pb.Secret, error) {

	if err := s.checkNamespacePermission(ctx, ns); err != nil {
		return nil, err
	}

	if pbSecret.GetId() == constant.GlobalSecretKey {
		return nil, errmsg.AddMessage(
			fmt.Errorf("%w: reserved secret ID", errdomain.ErrInvalidArgument),
			"The secret ID is reserved",
		)
	}

	dbSecret, err := s.convertSecretToDB(ctx, ns, pbSecret)
	if err != nil {
		return nil, err
	}

	if err := s.repository.CreateNamespaceSecret(ctx, ns.Permalink(), dbSecret); err != nil {
		return nil, err
	}

	dbCreatedSecret, err := s.repository.GetNamespaceSecretByID(ctx, ns.Permalink(), dbSecret.ID)
	if err != nil {
		return nil, err
	}

	return s.convertSecretToPB(ctx, dbCreatedSecret)

}

func (s *service) ListNamespaceSecrets(ctx context.Context, ns resource.Namespace, pageSize int32, pageToken string, filter filtering.Filter) ([]*pb.Secret, int32, string, error) {

	if err := s.checkNamespacePermission(ctx, ns); err != nil {
		return nil, 0, "", err
	}

	dbSecrets, ps, pt, err := s.repository.ListNamespaceSecrets(ctx, ns.Permalink(), int64(pageSize), pageToken, filter)
	if err != nil {
		return nil, 0, "", err
	}

	pbSecrets, err := s.convertSecretsToPB(ctx, dbSecrets)
	return pbSecrets, int32(ps), pt, err
}

func (s *service) GetNamespaceSecretByID(ctx context.Context, ns resource.Namespace, id string) (*pb.Secret, error) {

	if err := s.checkNamespacePermission(ctx, ns); err != nil {
		return nil, err
	}

	ownerPermalink := ns.Permalink()

	dbSecret, err := s.repository.GetNamespaceSecretByID(ctx, ownerPermalink, id)
	if err != nil {
		return nil, ErrNotFound
	}
	return s.convertSecretToPB(ctx, dbSecret)
}

func (s *service) UpdateNamespaceSecretByID(ctx context.Context, ns resource.Namespace, id string, updatedSecret *pb.Secret) (*pb.Secret, error) {

	if err := s.checkNamespacePermission(ctx, ns); err != nil {
		return nil, err
	}

	ownerPermalink := ns.Permalink()

	dbSecret, err := s.convertSecretToDB(ctx, ns, updatedSecret)
	if err != nil {
		return nil, ErrNotFound
	}

	if _, err = s.repository.GetNamespaceSecretByID(ctx, ownerPermalink, id); err != nil {
		return nil, err
	}

	if err := s.repository.UpdateNamespaceSecretByID(ctx, ns.Permalink(), id, dbSecret); err != nil {
		return nil, err
	}

	return s.GetNamespaceSecretByID(ctx, ns, id)
}

func (s *service) DeleteNamespaceSecretByID(ctx context.Context, ns resource.Namespace, id string) error {
	if err := s.checkNamespacePermission(ctx, ns); err != nil {
		return err
	}
	ownerPermalink := ns.Permalink()

	return s.repository.DeleteNamespaceSecretByID(ctx, ownerPermalink, id)
}
