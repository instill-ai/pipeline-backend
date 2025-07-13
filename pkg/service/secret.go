package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/gofrs/uuid"
	"go.einride.tech/aip/filtering"

	"github.com/instill-ai/pipeline-backend/pkg/constant"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/resource"
	pipelinepb "github.com/instill-ai/protogen-go/pipeline/pipeline/v1beta"

	errorsx "github.com/instill-ai/x/errors"
)

func (s *service) CreateNamespaceSecret(ctx context.Context, ns resource.Namespace, pbSecret *pipelinepb.Secret) (*pipelinepb.Secret, error) {

	if err := s.checkNamespacePermission(ctx, ns); err != nil {
		return nil, err
	}

	if pbSecret.GetId() == constant.GlobalSecretKey {
		return nil, errorsx.AddMessage(
			fmt.Errorf("%w: reserved secret ID", errorsx.ErrInvalidArgument),
			"The secret ID is reserved",
		)
	}

	dbSecret, err := s.converter.ConvertSecretToDB(ctx, ns, pbSecret)
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

	return s.converter.ConvertSecretToPB(ctx, dbCreatedSecret)

}

func (s *service) ListNamespaceSecrets(ctx context.Context, ns resource.Namespace, pageSize int32, pageToken string, filter filtering.Filter) ([]*pipelinepb.Secret, int32, string, error) {

	if err := s.checkNamespacePermission(ctx, ns); err != nil {
		return nil, 0, "", err
	}

	dbSecrets, ps, pt, err := s.repository.ListNamespaceSecrets(ctx, ns.Permalink(), int64(pageSize), pageToken, filter)
	if err != nil {
		return nil, 0, "", err
	}

	pbSecrets, err := s.converter.ConvertSecretsToPB(ctx, dbSecrets)
	return pbSecrets, int32(ps), pt, err
}

func (s *service) GetNamespaceSecretByID(ctx context.Context, ns resource.Namespace, id string) (*pipelinepb.Secret, error) {

	if err := s.checkNamespacePermission(ctx, ns); err != nil {
		return nil, err
	}

	ownerPermalink := ns.Permalink()

	dbSecret, err := s.repository.GetNamespaceSecretByID(ctx, ownerPermalink, id)
	if err != nil {
		return nil, errorsx.ErrNotFound
	}
	return s.converter.ConvertSecretToPB(ctx, dbSecret)
}

func (s *service) UpdateNamespaceSecretByID(ctx context.Context, ns resource.Namespace, id string, updatedSecret *pipelinepb.Secret) (*pipelinepb.Secret, error) {

	if err := s.checkNamespacePermission(ctx, ns); err != nil {
		return nil, err
	}

	ownerPermalink := ns.Permalink()

	dbSecret, err := s.converter.ConvertSecretToDB(ctx, ns, updatedSecret)
	if err != nil {
		return nil, errorsx.ErrNotFound
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

func (s *service) checkSecretFields(ctx context.Context, uid uuid.UUID, setup any, prefix string) error {
	var setupMap map[string]any

	switch s := setup.(type) {
	case map[string]any:
		setupMap = s
	case string, nil:
		// Setup should either not be present or contain a reference to a
		// connection.
		return nil
	default:
		return fmt.Errorf("invalid type for setup field")
	}

	for k, v := range setupMap {
		key := prefix + k
		if ok, err := s.component.IsSecretField(uid, key); err == nil && ok {
			if s, ok := v.(string); ok {
				if !strings.HasPrefix(s, "${") || !strings.HasSuffix(s, "}") {
					return errorsx.ErrCanNotUsePlaintextSecret
				}
			}
		}
		if str, ok := v.(map[string]any); ok {
			err := s.checkSecretFields(ctx, uid, str, fmt.Sprintf("%s.", key))
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *service) checkSecret(ctx context.Context, components datamodel.ComponentMap) error {

	for _, comp := range components {
		switch comp.Type {
		default:

			c, err := s.component.GetDefinitionByID(comp.Type, nil, nil)
			if err == nil {
				err := s.checkSecretFields(ctx, uuid.FromStringOrNil(c.Uid), comp.Setup, "")
				if err != nil {
					return err
				}
			}

		case datamodel.Iterator:
			err := s.checkSecret(ctx, comp.Component)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
