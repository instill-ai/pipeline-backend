package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/instill-ai/pipeline-backend/internal/resource"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/logger"
	"github.com/instill-ai/pipeline-backend/pkg/utils"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1beta"
)

type View int32

const (
	VIEW_UNSPECIFIED View = 0
	VIEW_BASIC       View = 1
	VIEW_FULL        View = 2
	VIEW_RECIPE      View = 3
)

func (s *service) recipeNameToPermalink(recipeRscName *pipelinePB.Recipe) (*pipelinePB.Recipe, error) {

	recipePermalink := &pipelinePB.Recipe{Version: recipeRscName.Version}
	for _, component := range recipeRscName.Components {
		componentPermalink := &pipelinePB.Component{
			Id:            component.Id,
			Configuration: component.Configuration,
		}

		permalink := ""
		var err error
		if utils.IsConnectorWithNamespace(component.ResourceName) {
			permalink, err = s.connectorNameToPermalink(component.ResourceName)
			if err != nil {
				// Allow not created resource
				componentPermalink.ResourceName = ""
			} else {
				componentPermalink.ResourceName = permalink
			}
		}

		if utils.IsConnectorDefinition(component.DefinitionName) {
			permalink, err = s.connectorDefinitionNameToPermalink(component.DefinitionName)
			if err != nil {
				return nil, err
			}
			componentPermalink.DefinitionName = permalink
		} else if utils.IsOperatorDefinition(component.DefinitionName) {
			permalink, err = s.operatorDefinitionNameToPermalink(component.DefinitionName)
			if err != nil {
				return nil, err
			}
			componentPermalink.DefinitionName = permalink
		}

		recipePermalink.Components = append(recipePermalink.Components, componentPermalink)
	}
	return recipePermalink, nil
}

func (s *service) recipePermalinkToName(recipePermalink *pipelinePB.Recipe) (*pipelinePB.Recipe, error) {

	recipe := &pipelinePB.Recipe{Version: recipePermalink.Version}

	for _, componentPermalink := range recipePermalink.Components {
		component := &pipelinePB.Component{
			Id:            componentPermalink.Id,
			Configuration: componentPermalink.Configuration,
			Definition:    componentPermalink.Definition,
			Resource:      componentPermalink.Resource,
			Type:          componentPermalink.Type,
		}

		if utils.IsConnector(componentPermalink.ResourceName) {
			name, err := s.connectorPermalinkToName(componentPermalink.ResourceName)
			if err != nil {
				// Allow resource not created
				component.ResourceName = ""
			} else {
				component.ResourceName = name
			}
		}
		if utils.IsConnectorDefinition(componentPermalink.DefinitionName) {
			name, err := s.connectorDefinitionPermalinkToName(componentPermalink.DefinitionName)
			if err != nil {
				return nil, err
			}
			component.DefinitionName = name
		} else if utils.IsOperatorDefinition(componentPermalink.DefinitionName) {
			name, err := s.operatorDefinitionPermalinkToName(componentPermalink.DefinitionName)
			if err != nil {
				return nil, err
			}
			component.DefinitionName = name
		}

		recipe.Components = append(recipe.Components, component)
	}
	return recipe, nil
}

func (s *service) connectorNameToPermalink(name string) (string, error) {

	ownerPermalink, err := s.ConvertOwnerNameToPermalink(fmt.Sprintf("%s/%s", strings.Split(name, "/")[0], strings.Split(name, "/")[1]))
	if err != nil {
		return "", err
	}
	dbConnector, err := s.repository.GetNamespaceConnectorByID(context.Background(), ownerPermalink, strings.Split(name, "/")[3], true)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("connectors/%s", dbConnector.UID), nil
}

func (s *service) connectorPermalinkToName(permalink string) (string, error) {

	dbConnector, err := s.repository.GetConnectorByUIDAdmin(context.Background(), uuid.FromStringOrNil(strings.Split(permalink, "/")[1]), true)
	if err != nil {
		return "", err
	}
	owner, err := s.ConvertOwnerPermalinkToName(dbConnector.Owner)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/connectors/%s", owner, dbConnector.ID), nil

}

func (s *service) connectorDefinitionNameToPermalink(name string) (string, error) {

	def, err := s.connector.GetConnectorDefinitionByID(strings.Split(name, "/")[1])
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("connector-definitions/%s", def.Uid), nil
}

func (s *service) connectorDefinitionPermalinkToName(permalink string) (string, error) {
	def, err := s.connector.GetConnectorDefinitionByUID(uuid.FromStringOrNil(strings.Split(permalink, "/")[1]))
	if err != nil {
		return "", err
	}
	return def.Name, nil
}

func (s *service) operatorDefinitionNameToPermalink(name string) (string, error) {
	id, err := resource.GetRscNameID(name)
	if err != nil {
		return "", err
	}
	def, err := s.operator.GetOperatorDefinitionByID(id)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("operator-definitions/%s", def.Uid), nil
}

func (s *service) operatorDefinitionPermalinkToName(permalink string) (string, error) {
	uid, err := resource.GetRscPermalinkUID(permalink)
	if err != nil {
		return "", err
	}
	def, err := s.operator.GetOperatorDefinitionByUID(uid)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("operator-definitions/%s", def.Id), nil
}

func ConvertResourceUIDToControllerResourcePermalink(resourceUID uuid.UUID, resourceType string) string {
	resourcePermalink := fmt.Sprintf("resources/%s/types/%s", resourceUID.String(), resourceType)

	return resourcePermalink
}

func (s *service) includeDetailInRecipe(recipe *pipelinePB.Recipe) error {

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for idx := range recipe.Components {

		if utils.IsConnector(recipe.Components[idx].ResourceName) {
			conn, err := s.repository.GetConnectorByUIDAdmin(context.Background(), uuid.FromStringOrNil(strings.Split(recipe.Components[idx].ResourceName, "/")[1]), false)
			if err != nil {
				// Allow resource not created
				recipe.Components[idx].Resource = nil
			} else {
				pbConnector, err := s.convertDatamodelToProto(ctx, conn, VIEW_FULL, true)
				if err != nil {
					// Allow resource not created
					recipe.Components[idx].Resource = nil
				} else {
					recipe.Components[idx].Resource = pbConnector
				}
			}

		}
		if utils.IsConnectorDefinition(recipe.Components[idx].DefinitionName) {
			uid, err := resource.GetRscPermalinkUID(recipe.Components[idx].DefinitionName)
			if err != nil {
				return err
			}
			def, err := s.connector.GetConnectorDefinitionByUID(uid)
			if err != nil {
				return err
			}

			detail := &structpb.Struct{}
			// Note: need to deal with camelCase or under_score for grpc in future
			json, marshalErr := protojson.MarshalOptions{UseProtoNames: true}.Marshal(def)
			if marshalErr != nil {
				return marshalErr
			}
			unmarshalErr := detail.UnmarshalJSON(json)
			if unmarshalErr != nil {
				return unmarshalErr
			}

			recipe.Components[idx].Definition = &pipelinePB.Component_ConnectorDefinition{ConnectorDefinition: def}
		}
		if utils.IsOperatorDefinition(recipe.Components[idx].DefinitionName) {
			uid, err := resource.GetRscPermalinkUID(recipe.Components[idx].DefinitionName)
			if err != nil {
				return err
			}
			def, err := s.operator.GetOperatorDefinitionByUID(uid)
			if err != nil {
				return err
			}

			detail := &structpb.Struct{}
			// Note: need to deal with camelCase or under_score for grpc in future
			json, marshalErr := protojson.MarshalOptions{UseProtoNames: true}.Marshal(def)
			if marshalErr != nil {
				return marshalErr
			}
			unmarshalErr := detail.UnmarshalJSON(json)
			if unmarshalErr != nil {
				return unmarshalErr
			}

			recipe.Components[idx].Definition = &pipelinePB.Component_OperatorDefinition{OperatorDefinition: def}
		}

	}
	return nil
}

// PBToDBPipeline converts protobuf data model to db data model
func (s *service) PBToDBPipeline(ctx context.Context, pbPipeline *pipelinePB.Pipeline) (*datamodel.Pipeline, error) {
	logger, _ := logger.GetZapLogger(ctx)

	var owner string
	var err error

	switch pbPipeline.Owner.(type) {
	case *pipelinePB.Pipeline_User:
		owner, err = s.ConvertOwnerNameToPermalink(pbPipeline.GetUser())
		if err != nil {
			return nil, err
		}
	case *pipelinePB.Pipeline_Organization:
		owner, err = s.ConvertOwnerNameToPermalink(pbPipeline.GetOrganization())
		if err != nil {
			return nil, err
		}
	}

	recipe := &datamodel.Recipe{}
	if pbPipeline.GetRecipe() != nil {
		recipePermalink, err := s.recipeNameToPermalink(pbPipeline.Recipe)
		if err != nil {
			return nil, err
		}

		b, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(recipePermalink)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(b, &recipe); err != nil {
			return nil, err
		}

	}

	dbPermission := &datamodel.Permission{}
	if pbPipeline.GetPermission() != nil {

		if err != nil {
			return nil, err
		}

		b, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(pbPipeline.GetPermission())
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(b, &dbPermission); err != nil {
			return nil, err
		}

	}

	return &datamodel.Pipeline{
		Owner: owner,
		ID:    pbPipeline.GetId(),

		BaseDynamic: datamodel.BaseDynamic{
			UID: func() uuid.UUID {
				if pbPipeline.GetUid() == "" {
					return uuid.UUID{}
				}
				id, err := uuid.FromString(pbPipeline.GetUid())
				if err != nil {
					logger.Error(err.Error())
				}
				return id
			}(),

			CreateTime: func() time.Time {
				if pbPipeline.GetCreateTime() != nil {
					return pbPipeline.GetCreateTime().AsTime()
				}
				return time.Time{}
			}(),

			UpdateTime: func() time.Time {
				if pbPipeline.GetUpdateTime() != nil {
					return pbPipeline.GetUpdateTime().AsTime()
				}
				return time.Time{}
			}(),
		},

		Description: sql.NullString{
			String: pbPipeline.GetDescription(),
			Valid:  true,
		},

		Recipe:     recipe,
		Permission: dbPermission,
		Metadata: func() []byte {
			if pbPipeline.GetMetadata() != nil {
				b, err := pbPipeline.GetMetadata().MarshalJSON()
				if err != nil {
					logger.Error(err.Error())
				}
				return b
			}
			return []byte{}
		}(),
	}, nil
}

// DBToPBPipeline converts db data model to protobuf data model
func (s *service) DBToPBPipeline(ctx context.Context, dbPipeline *datamodel.Pipeline, view View) (*pipelinePB.Pipeline, error) {

	logger, _ := logger.GetZapLogger(ctx)

	owner, err := s.ConvertOwnerPermalinkToName(dbPipeline.Owner)
	if err != nil {
		return nil, err
	}

	var pbRecipe *pipelinePB.Recipe

	var startComp *pipelinePB.Component
	var endComp *pipelinePB.Component

	if dbPipeline.Recipe != nil && view != VIEW_BASIC {
		pbRecipe = &pipelinePB.Recipe{}

		b, err := json.Marshal(dbPipeline.Recipe)
		if err != nil {
			return nil, err
		}

		err = protojson.Unmarshal(b, pbRecipe)
		if err != nil {
			return nil, err
		}

	}

	if view == VIEW_RECIPE || view == VIEW_FULL {
		for i := range pbRecipe.Components {
			if strings.HasPrefix(pbRecipe.Components[i].DefinitionName, "connector-definitions") {
				con, err := s.connector.GetConnectorDefinitionByUID(uuid.FromStringOrNil(strings.Split(pbRecipe.Components[i].DefinitionName, "/")[1]))
				if err != nil {
					return nil, err
				}
				switch con.Type {
				case pipelinePB.ConnectorType_CONNECTOR_TYPE_AI:
					pbRecipe.Components[i].Type = pipelinePB.ComponentType_COMPONENT_TYPE_CONNECTOR_AI
				case pipelinePB.ConnectorType_CONNECTOR_TYPE_BLOCKCHAIN:
					pbRecipe.Components[i].Type = pipelinePB.ComponentType_COMPONENT_TYPE_CONNECTOR_BLOCKCHAIN
				case pipelinePB.ConnectorType_CONNECTOR_TYPE_DATA:
					pbRecipe.Components[i].Type = pipelinePB.ComponentType_COMPONENT_TYPE_CONNECTOR_DATA
				}

			}
			if strings.HasPrefix(pbRecipe.Components[i].DefinitionName, "operator-definitions") {
				pbRecipe.Components[i].Type = pipelinePB.ComponentType_COMPONENT_TYPE_OPERATOR
			}
		}
	}
	if view == VIEW_FULL {
		if err := s.includeDetailInRecipe(pbRecipe); err != nil {
			return nil, err
		}
		for i := range pbRecipe.Components {
			if pbRecipe.Components[i].DefinitionName == "operator-definitions/2ac8be70-0f7a-4b61-a33d-098b8acfa6f3" {
				startComp = pbRecipe.Components[i]
			}
			if pbRecipe.Components[i].DefinitionName == "operator-definitions/4f39c8bc-8617-495d-80de-80d0f5397516" {
				endComp = pbRecipe.Components[i]
			}
		}
	}

	if pbRecipe != nil {
		pbRecipe, err = s.recipePermalinkToName(pbRecipe)
		if err != nil {
			return nil, err
		}
	}

	pbPermission := &pipelinePB.Permission{}

	b, err := json.Marshal(dbPipeline.Permission)
	if err != nil {
		return nil, err
	}

	err = protojson.Unmarshal(b, pbPermission)
	if err != nil {
		return nil, err
	}
	if pbPermission != nil && pbPermission.ShareCode != nil {
		pbPermission.ShareCode.Code = dbPipeline.ShareCode
	}

	pbPipeline := pipelinePB.Pipeline{
		Name:       fmt.Sprintf("%s/pipelines/%s", owner, dbPipeline.ID),
		Uid:        dbPipeline.BaseDynamic.UID.String(),
		Id:         dbPipeline.ID,
		CreateTime: timestamppb.New(dbPipeline.CreateTime),
		UpdateTime: timestamppb.New(dbPipeline.UpdateTime),
		DeleteTime: func() *timestamppb.Timestamp {
			if dbPipeline.DeleteTime.Time.IsZero() {
				return nil
			} else {
				return timestamppb.New(dbPipeline.DeleteTime.Time)
			}
		}(),
		Description: &dbPipeline.Description.String,
		Recipe:      pbRecipe,
		Permission:  pbPermission,
	}

	if view != VIEW_BASIC {
		if dbPipeline.Metadata != nil {
			str := structpb.Struct{}
			err := str.UnmarshalJSON(dbPipeline.Metadata)
			if err != nil {
				logger.Error(err.Error())
			}
			pbPipeline.Metadata = &str
		}
	}

	if pbRecipe != nil && view == VIEW_FULL && startComp != nil && endComp != nil {
		spec, err := s.GenerateOpenApiSpec(startComp, endComp, pbRecipe.Components)
		if err == nil {
			pbPipeline.OpenapiSchema = spec
		}
	}

	if strings.HasPrefix(dbPipeline.Owner, "users/") {
		pbPipeline.Owner = &pipelinePB.Pipeline_User{User: owner}
	} else if strings.HasPrefix(dbPipeline.Owner, "organizations/") {
		pbPipeline.Owner = &pipelinePB.Pipeline_Organization{Organization: owner}
	}

	return &pbPipeline, nil
}

// DBToPBPipeline converts db data model to protobuf data model
func (s *service) DBToPBPipelines(ctx context.Context, dbPipelines []*datamodel.Pipeline, view View) ([]*pipelinePB.Pipeline, error) {
	var err error
	pbPipelines := make([]*pipelinePB.Pipeline, len(dbPipelines))
	for idx := range dbPipelines {
		pbPipelines[idx], err = s.DBToPBPipeline(
			ctx,
			dbPipelines[idx],
			view,
		)
		if err != nil {
			return nil, err
		}

	}
	return pbPipelines, nil
}

// PBToDBPipelineRelease converts protobuf data model to db data model
func (s *service) PBToDBPipelineRelease(ctx context.Context, pipelineUid uuid.UUID, pbPipelineRelease *pipelinePB.PipelineRelease) (*datamodel.PipelineRelease, error) {
	logger, _ := logger.GetZapLogger(ctx)

	recipe := &datamodel.Recipe{}
	if pbPipelineRelease.GetRecipe() != nil {
		recipePermalink, err := s.recipeNameToPermalink(pbPipelineRelease.Recipe)
		if err != nil {
			return nil, err
		}

		b, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(recipePermalink)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(b, &recipe); err != nil {
			return nil, err
		}

	}
	return &datamodel.PipelineRelease{
		ID: pbPipelineRelease.GetId(),

		BaseDynamic: datamodel.BaseDynamic{
			UID: func() uuid.UUID {
				if pbPipelineRelease.GetUid() == "" {
					return uuid.UUID{}
				}
				id, err := uuid.FromString(pbPipelineRelease.GetUid())
				if err != nil {
					logger.Error(err.Error())
				}
				return id
			}(),

			CreateTime: func() time.Time {
				if pbPipelineRelease.GetCreateTime() != nil {
					return pbPipelineRelease.GetCreateTime().AsTime()
				}
				return time.Time{}
			}(),

			UpdateTime: func() time.Time {
				if pbPipelineRelease.GetUpdateTime() != nil {
					return pbPipelineRelease.GetUpdateTime().AsTime()
				}
				return time.Time{}
			}(),
		},

		Description: sql.NullString{
			String: pbPipelineRelease.GetDescription(),
			Valid:  true,
		},

		Recipe:      recipe,
		PipelineUID: pipelineUid,

		Metadata: func() []byte {
			if pbPipelineRelease.GetMetadata() != nil {
				b, err := pbPipelineRelease.GetMetadata().MarshalJSON()
				if err != nil {
					logger.Error(err.Error())
				}
				return b
			}
			return []byte{}
		}(),
	}, nil
}

// DBToPBPipelineRelease converts db data model to protobuf data model
func (s *service) DBToPBPipelineRelease(ctx context.Context, dbPipelineRelease *datamodel.PipelineRelease, view View, latestUUID uuid.UUID, defaultUUID uuid.UUID) (*pipelinePB.PipelineRelease, error) {

	logger, _ := logger.GetZapLogger(ctx)

	dbPipeline, err := s.repository.GetPipelineByUIDAdmin(ctx, dbPipelineRelease.PipelineUID, true)
	if err != nil {
		return nil, err
	}
	owner, err := s.ConvertOwnerPermalinkToName(dbPipeline.Owner)
	if err != nil {
		return nil, err
	}
	var pbRecipe *pipelinePB.Recipe
	if dbPipelineRelease.Recipe != nil && view != VIEW_BASIC {
		pbRecipe = &pipelinePB.Recipe{}

		b, err := json.Marshal(dbPipelineRelease.Recipe)
		if err != nil {
			return nil, err
		}

		err = protojson.Unmarshal(b, pbRecipe)
		if err != nil {
			return nil, err
		}
	}

	if view == VIEW_RECIPE || view == VIEW_FULL {
		for i := range pbRecipe.Components {
			if strings.HasPrefix(pbRecipe.Components[i].DefinitionName, "connector-definitions") {
				con, err := s.connector.GetConnectorDefinitionByUID(uuid.FromStringOrNil(strings.Split(pbRecipe.Components[i].DefinitionName, "/")[1]))
				if err != nil {
					return nil, err
				}
				switch con.Type {
				case pipelinePB.ConnectorType_CONNECTOR_TYPE_AI:
					pbRecipe.Components[i].Type = pipelinePB.ComponentType_COMPONENT_TYPE_CONNECTOR_AI
				case pipelinePB.ConnectorType_CONNECTOR_TYPE_BLOCKCHAIN:
					pbRecipe.Components[i].Type = pipelinePB.ComponentType_COMPONENT_TYPE_CONNECTOR_BLOCKCHAIN
				case pipelinePB.ConnectorType_CONNECTOR_TYPE_DATA:
					pbRecipe.Components[i].Type = pipelinePB.ComponentType_COMPONENT_TYPE_CONNECTOR_DATA
				}
			}
			if strings.HasPrefix(pbRecipe.Components[i].DefinitionName, "operator-definitions") {
				pbRecipe.Components[i].Type = pipelinePB.ComponentType_COMPONENT_TYPE_OPERATOR
			}
		}
	}

	var startComp *pipelinePB.Component
	var endComp *pipelinePB.Component

	if view == VIEW_FULL {
		if err := s.includeDetailInRecipe(pbRecipe); err != nil {
			return nil, err
		}
		for i := range pbRecipe.Components {
			if pbRecipe.Components[i].DefinitionName == "operator-definitions/2ac8be70-0f7a-4b61-a33d-098b8acfa6f3" {
				startComp = pbRecipe.Components[i]
			}
			if pbRecipe.Components[i].DefinitionName == "operator-definitions/4f39c8bc-8617-495d-80de-80d0f5397516" {
				endComp = pbRecipe.Components[i]
			}
		}
	}

	if pbRecipe != nil {
		pbRecipe, err = s.recipePermalinkToName(pbRecipe)
		if err != nil {
			return nil, err
		}
	}
	pbPipelineRelease := pipelinePB.PipelineRelease{
		Name:       fmt.Sprintf("%s/pipelines/%s/releases/%s", owner, dbPipeline.ID, dbPipelineRelease.ID),
		Uid:        dbPipelineRelease.BaseDynamic.UID.String(),
		Id:         dbPipelineRelease.ID,
		CreateTime: timestamppb.New(dbPipelineRelease.CreateTime),
		UpdateTime: timestamppb.New(dbPipelineRelease.UpdateTime),
		DeleteTime: func() *timestamppb.Timestamp {
			if dbPipelineRelease.DeleteTime.Time.IsZero() {
				return nil
			} else {
				return timestamppb.New(dbPipelineRelease.DeleteTime.Time)
			}
		}(),
		Description: &dbPipelineRelease.Description.String,
		Recipe:      pbRecipe,
	}

	if view != VIEW_BASIC {
		if dbPipelineRelease.Metadata != nil {
			str := structpb.Struct{}
			err := str.UnmarshalJSON(dbPipelineRelease.Metadata)
			if err != nil {
				logger.Error(err.Error())
			}
			pbPipelineRelease.Metadata = &str
		}
	}

	if pbRecipe != nil && view == VIEW_FULL && startComp != nil && endComp != nil {
		spec, err := s.GenerateOpenApiSpec(startComp, endComp, pbRecipe.Components)
		if err == nil {
			pbPipelineRelease.OpenapiSchema = spec
		}
	}
	if pbPipelineRelease.Uid == latestUUID.String() {
		pbPipelineRelease.Alias = "latest"
	}
	if pbPipelineRelease.Uid == defaultUUID.String() {
		pbPipelineRelease.Alias = "default"
	}

	return &pbPipelineRelease, nil
}

// DBToPBPipelineRelease converts db data model to protobuf data model
func (s *service) DBToPBPipelineReleases(ctx context.Context, dbPipelineRelease []*datamodel.PipelineRelease, view View, latestUUID uuid.UUID, defaultUUID uuid.UUID) ([]*pipelinePB.PipelineRelease, error) {
	var err error
	pbPipelineReleases := make([]*pipelinePB.PipelineRelease, len(dbPipelineRelease))
	for idx := range dbPipelineRelease {
		pbPipelineReleases[idx], err = s.DBToPBPipelineRelease(
			ctx,
			dbPipelineRelease[idx],
			view,
			latestUUID,
			defaultUUID,
		)
		if err != nil {
			return nil, err
		}

	}
	return pbPipelineReleases, nil
}

// convertProtoToDatamodel converts protobuf data model to db data model
func (s *service) convertProtoToDatamodel(
	ctx context.Context,
	pbConnector *pipelinePB.Connector,
) (*datamodel.Connector, error) {

	logger, _ := logger.GetZapLogger(ctx)

	var uid uuid.UUID
	var id string
	var state datamodel.ConnectorState
	var tombstone bool
	var description sql.NullString
	var configuration *structpb.Struct
	var createTime time.Time
	var updateTime time.Time
	var err error

	id = pbConnector.GetId()
	state = datamodel.ConnectorState(pbConnector.GetState())
	tombstone = pbConnector.GetTombstone()
	configuration = pbConnector.GetConfiguration()
	createTime = pbConnector.GetCreateTime().AsTime()
	updateTime = pbConnector.GetUpdateTime().AsTime()

	connectorDefinition, err := s.connector.GetConnectorDefinitionByID(strings.Split(pbConnector.ConnectorDefinitionName, "/")[1])
	if err != nil {
		return nil, err
	}

	uid = uuid.FromStringOrNil(pbConnector.GetUid())
	if err != nil {
		return nil, err
	}

	description = sql.NullString{
		String: pbConnector.GetDescription(),
		Valid:  true,
	}

	var owner string

	switch pbConnector.Owner.(type) {
	case *pipelinePB.Connector_User:
		owner, err = s.ConvertOwnerNameToPermalink(pbConnector.GetUser())
		if err != nil {
			return nil, err
		}
	case *pipelinePB.Connector_Organization:
		owner, err = s.ConvertOwnerNameToPermalink(pbConnector.GetOrganization())
		if err != nil {
			return nil, err
		}
	}

	return &datamodel.Connector{
		Owner:                  owner,
		ID:                     id,
		ConnectorType:          datamodel.ConnectorType(connectorDefinition.Type),
		Description:            description,
		State:                  state,
		Tombstone:              tombstone,
		ConnectorDefinitionUID: uuid.FromStringOrNil(connectorDefinition.Uid),
		Visibility:             datamodel.ConnectorVisibility(pbConnector.Visibility),

		Configuration: func() []byte {
			if configuration != nil {
				b, err := configuration.MarshalJSON()
				if err != nil {
					logger.Error(err.Error())
				}
				return b
			}
			return []byte{}
		}(),

		BaseDynamic: datamodel.BaseDynamic{
			UID:        uid,
			CreateTime: createTime,
			UpdateTime: updateTime,
		},
	}, nil
}

// convertDatamodelToProto converts db data model to protobuf data model
func (s *service) convertDatamodelToProto(
	ctx context.Context,
	dbConnector *datamodel.Connector,
	view View,
	credentialMask bool,
) (*pipelinePB.Connector, error) {

	logger, _ := logger.GetZapLogger(ctx)

	owner, err := s.ConvertOwnerPermalinkToName(dbConnector.Owner)
	if err != nil {
		return nil, err
	}
	dbConnDef, err := s.connector.GetConnectorDefinitionByUID(dbConnector.ConnectorDefinitionUID)
	if err != nil {
		return nil, err
	}

	pbConnector := &pipelinePB.Connector{
		Uid:                     dbConnector.UID.String(),
		Name:                    fmt.Sprintf("%s/connectors/%s", owner, dbConnector.ID),
		Id:                      dbConnector.ID,
		ConnectorDefinitionName: dbConnDef.GetName(),
		Type:                    pipelinePB.ConnectorType(dbConnector.ConnectorType),
		Description:             &dbConnector.Description.String,
		State:                   pipelinePB.Connector_State(dbConnector.State),
		Tombstone:               dbConnector.Tombstone,
		CreateTime:              timestamppb.New(dbConnector.CreateTime),
		UpdateTime:              timestamppb.New(dbConnector.UpdateTime),
		DeleteTime: func() *timestamppb.Timestamp {
			if dbConnector.DeleteTime.Time.IsZero() {
				return nil
			} else {
				return timestamppb.New(dbConnector.DeleteTime.Time)
			}
		}(),
		Visibility: pipelinePB.Connector_Visibility(dbConnector.Visibility),

		Configuration: func() *structpb.Struct {
			if dbConnector.Configuration != nil {
				str := structpb.Struct{}
				err := str.UnmarshalJSON(dbConnector.Configuration)
				if err != nil {
					logger.Fatal(err.Error())
				}
				return &str
			}
			return nil
		}(),
	}

	if strings.HasPrefix(owner, "users/") {
		pbConnector.Owner = &pipelinePB.Connector_User{User: owner}
	} else if strings.HasPrefix(owner, "organizations/") {
		pbConnector.Owner = &pipelinePB.Connector_Organization{Organization: owner}
	}
	if view != VIEW_BASIC {
		if credentialMask {
			utils.MaskCredentialFields(s.connector, dbConnDef.Id, pbConnector.Configuration)
		}
		if view == VIEW_FULL {
			pbConnector.ConnectorDefinition = dbConnDef
		}
	}

	return pbConnector, nil

}

func (s *service) convertDatamodelArrayToProtoArray(
	ctx context.Context,
	dbConnectors []*datamodel.Connector,
	view View,
	credentialMask bool,
) ([]*pipelinePB.Connector, error) {

	var err error
	pbConnectors := make([]*pipelinePB.Connector, len(dbConnectors))
	for idx := range dbConnectors {
		pbConnectors[idx], err = s.convertDatamodelToProto(
			ctx,
			dbConnectors[idx],
			view,
			credentialMask,
		)
		if err != nil {
			return nil, err
		}

	}

	return pbConnectors, nil

}
