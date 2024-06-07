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
	"github.com/instill-ai/x/errmsg"

	componentbase "github.com/instill-ai/component/base"
	errdomain "github.com/instill-ai/pipeline-backend/pkg/errors"
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

func (s *service) ListNamespaceSecrets(ctx context.Context, ns resource.Namespace, pageSize int32, pageToken string, filter filtering.Filter) ([]*pb.Secret, int32, string, error) {

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

func (s *service) GetNamespaceSecretByID(ctx context.Context, ns resource.Namespace, id string) (*pb.Secret, error) {

	if err := s.checkNamespacePermission(ctx, ns); err != nil {
		return nil, err
	}

	ownerPermalink := ns.Permalink()

	dbSecret, err := s.repository.GetNamespaceSecretByID(ctx, ownerPermalink, id)
	if err != nil {
		return nil, errdomain.ErrNotFound
	}
	return s.converter.ConvertSecretToPB(ctx, dbSecret)
}

func (s *service) UpdateNamespaceSecretByID(ctx context.Context, ns resource.Namespace, id string, updatedSecret *pb.Secret) (*pb.Secret, error) {

	if err := s.checkNamespacePermission(ctx, ns); err != nil {
		return nil, err
	}

	ownerPermalink := ns.Permalink()

	dbSecret, err := s.converter.ConvertSecretToDB(ctx, ns, updatedSecret)
	if err != nil {
		return nil, errdomain.ErrNotFound
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

func (s *service) checkSecretFields(ctx context.Context, uid uuid.UUID, setup map[string]any, prefix string) error {

	for k, v := range setup {
		key := prefix + k
		if ok, err := s.component.IsSecretField(uid, key); err == nil && ok {
			if s, ok := v.(string); ok {
				if !strings.HasPrefix(s, "${") || !strings.HasSuffix(s, "}") {
					return errCanNotUsePlaintextSecret
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
func (s *service) checkSecret(ctx context.Context, recipe *datamodel.Recipe) error {

	for _, comp := range recipe.Component {
		switch c := comp.(type) {
		case *componentbase.ComponentConfig:
			defUID := uuid.FromStringOrNil(c.Type)
			setup := c.Setup
			err := s.checkSecretFields(ctx, defUID, setup, "")
			if err != nil {
				return err
			}
		case *datamodel.IteratorComponent:
			for _, nestedComp := range c.Component {
				switch nestedC := nestedComp.(type) {
				case *componentbase.ComponentConfig:
					defUID := uuid.FromStringOrNil(nestedC.Type)
					setup := nestedC.Setup
					err := s.checkSecretFields(ctx, defUID, setup, "")
					if err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}
