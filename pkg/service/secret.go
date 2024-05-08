package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/gofrs/uuid"
	"go.einride.tech/aip/filtering"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/constant"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/resource"
	"github.com/instill-ai/x/errmsg"

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
		return nil, ErrNotFound
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

func (s *service) checkSecretFields(ctx context.Context, uid uuid.UUID, connection *structpb.Struct, prefix string) error {

	for k, v := range connection.GetFields() {
		key := prefix + k
		if ok, err := s.connector.IsSecretField(uid, key); err == nil && ok {
			if v.GetStringValue() != "" {
				if !strings.HasPrefix(v.GetStringValue(), "${") || !strings.HasSuffix(v.GetStringValue(), "}") {
					return errCanNotUsePlaintextSecret
				}
			}
		}
		if v.GetStructValue() != nil {
			err := s.checkSecretFields(ctx, uid, v.GetStructValue(), fmt.Sprintf("%s.", key))
			if err != nil {
				return err
			}
		}
	}
	return nil
}
func (s *service) checkSecret(ctx context.Context, recipe *datamodel.Recipe) error {

	for _, comp := range recipe.Components {
		if comp.IsConnectorComponent() {
			defUID := uuid.FromStringOrNil(strings.Split(comp.ConnectorComponent.DefinitionName, "/")[1])
			connection := comp.ConnectorComponent.Connection
			err := s.checkSecretFields(ctx, defUID, connection, "")
			if err != nil {
				return err
			}
		}
		if comp.IsIteratorComponent() {
			for _, nestedComp := range comp.IteratorComponent.Components {
				if comp.IsConnectorComponent() {
					defUID := uuid.FromStringOrNil(strings.Split(nestedComp.ConnectorComponent.DefinitionName, "/")[1])
					connection := nestedComp.ConnectorComponent.Connection
					err := s.checkSecretFields(ctx, defUID, connection, "")
					if err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}
