package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/internal/resource"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/logger"
	"github.com/instill-ai/pipeline-backend/pkg/utils"
	"go.einride.tech/aip/filtering"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	mgmtPB "github.com/instill-ai/protogen-go/core/mgmt/v1beta"
	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1beta"
)

type View int32

const (
	ViewUnspecified View = 0
	ViewBasic       View = 1
	ViewFull        View = 2
	ViewRecipe      View = 3
)

func parseView(i int32) View {
	if i == 0 {
		return ViewBasic
	}

	return View(i)
}

func (s *service) recipeNameToPermalink(recipeRscName *pipelinePB.Recipe) (*pipelinePB.Recipe, error) {

	recipePermalink := &pipelinePB.Recipe{Version: recipeRscName.Version}
	for _, component := range recipeRscName.Components {
		componentPermalink := &pipelinePB.Component{
			Id:        component.Id,
			Metadata:  component.Metadata,
			Component: component.Component,
		}

		switch component.Component.(type) {
		case *pipelinePB.Component_ConnectorComponent:
			permalink, err := s.connectorNameToPermalink(component.GetConnectorComponent().ConnectorName)
			if err != nil {
				// Allow not created resource
				componentPermalink.GetConnectorComponent().ConnectorName = ""
			} else {
				componentPermalink.GetConnectorComponent().ConnectorName = permalink
			}
			defPermalink, err := s.connectorDefinitionNameToPermalink(component.GetConnectorComponent().DefinitionName)
			if err != nil {
				return nil, err
			}
			componentPermalink.GetConnectorComponent().DefinitionName = defPermalink
		case *pipelinePB.Component_OperatorComponent:
			defPermalink, err := s.operatorDefinitionNameToPermalink(component.GetOperatorComponent().DefinitionName)
			if err != nil {
				return nil, err
			}
			componentPermalink.GetOperatorComponent().DefinitionName = defPermalink
		case *pipelinePB.Component_IteratorComponent:
			nestedComponentPermalinks := []*pipelinePB.NestedComponent{}
			for _, nestedComponent := range componentPermalink.GetIteratorComponent().Components {

				nestedComponentPermalink := &pipelinePB.NestedComponent{
					Id:        nestedComponent.Id,
					Metadata:  nestedComponent.Metadata,
					Component: nestedComponent.Component,
				}

				switch nestedComponent.Component.(type) {
				case *pipelinePB.NestedComponent_ConnectorComponent:
					permalink, err := s.connectorNameToPermalink(nestedComponent.GetConnectorComponent().ConnectorName)
					if err != nil {
						// Allow not created resource
						nestedComponentPermalink.GetConnectorComponent().ConnectorName = ""
					} else {
						nestedComponentPermalink.GetConnectorComponent().ConnectorName = permalink
					}
					defPermalink, err := s.connectorDefinitionNameToPermalink(nestedComponent.GetConnectorComponent().DefinitionName)
					if err != nil {
						return nil, err
					}
					nestedComponentPermalink.GetConnectorComponent().DefinitionName = defPermalink
					nestedComponentPermalinks = append(nestedComponentPermalinks, nestedComponentPermalink)
				case *pipelinePB.NestedComponent_OperatorComponent:
					defPermalink, err := s.operatorDefinitionNameToPermalink(nestedComponent.GetOperatorComponent().DefinitionName)
					if err != nil {
						return nil, err
					}
					nestedComponentPermalink.GetOperatorComponent().DefinitionName = defPermalink
					nestedComponentPermalinks = append(nestedComponentPermalinks, nestedComponentPermalink)
				}
			}
			componentPermalink.GetIteratorComponent().Components = nestedComponentPermalinks

		}

		recipePermalink.Components = append(recipePermalink.Components, componentPermalink)
	}
	return recipePermalink, nil
}

func (s *service) recipePermalinkToName(recipePermalink *pipelinePB.Recipe) (*pipelinePB.Recipe, error) {

	recipe := &pipelinePB.Recipe{Version: recipePermalink.Version}

	for _, componentPermalink := range recipePermalink.Components {
		component := &pipelinePB.Component{
			Id:        componentPermalink.Id,
			Metadata:  componentPermalink.Metadata,
			Component: componentPermalink.Component,
		}

		switch component.Component.(type) {
		case *pipelinePB.Component_ConnectorComponent:
			name, err := s.connectorPermalinkToName(componentPermalink.GetConnectorComponent().ConnectorName)
			if err != nil {
				// Allow resource not created
				component.GetConnectorComponent().ConnectorName = ""
			} else {
				component.GetConnectorComponent().ConnectorName = name
			}
			defName, err := s.connectorDefinitionPermalinkToName(componentPermalink.GetConnectorComponent().DefinitionName)
			if err != nil {
				return nil, err
			}
			component.GetConnectorComponent().DefinitionName = defName
		case *pipelinePB.Component_OperatorComponent:
			defName, err := s.operatorDefinitionPermalinkToName(componentPermalink.GetOperatorComponent().DefinitionName)
			if err != nil {
				return nil, err
			}
			component.GetOperatorComponent().DefinitionName = defName
		case *pipelinePB.Component_IteratorComponent:
			nestedComponents := []*pipelinePB.NestedComponent{}
			for _, nestedComponentPermalink := range componentPermalink.GetIteratorComponent().Components {
				nestedComponent := &pipelinePB.NestedComponent{
					Id:        nestedComponentPermalink.Id,
					Metadata:  nestedComponentPermalink.Metadata,
					Component: nestedComponentPermalink.Component,
				}
				switch nestedComponentPermalink.Component.(type) {
				case *pipelinePB.NestedComponent_ConnectorComponent:
					name, err := s.connectorPermalinkToName(nestedComponentPermalink.GetConnectorComponent().ConnectorName)
					if err != nil {
						// Allow resource not created
						nestedComponent.GetConnectorComponent().ConnectorName = ""
					} else {
						nestedComponent.GetConnectorComponent().ConnectorName = name
					}
					defName, err := s.connectorDefinitionPermalinkToName(nestedComponentPermalink.GetConnectorComponent().DefinitionName)
					if err != nil {
						return nil, err
					}
					nestedComponent.GetConnectorComponent().DefinitionName = defName
					nestedComponents = append(nestedComponents, nestedComponent)
				case *pipelinePB.NestedComponent_OperatorComponent:
					defName, err := s.operatorDefinitionPermalinkToName(nestedComponentPermalink.GetOperatorComponent().DefinitionName)
					if err != nil {
						return nil, err
					}
					nestedComponent.GetOperatorComponent().DefinitionName = defName
					nestedComponents = append(nestedComponents, nestedComponent)
				}
			}
			component.GetIteratorComponent().Components = nestedComponents
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

	def, err := s.connector.GetConnectorDefinitionByID(strings.Split(name, "/")[1], nil, nil)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("connector-definitions/%s", def.Uid), nil
}

func (s *service) connectorDefinitionPermalinkToName(permalink string) (string, error) {
	def, err := s.connector.GetConnectorDefinitionByUID(uuid.FromStringOrNil(strings.Split(permalink, "/")[1]), nil, nil)
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
	def, err := s.operator.GetOperatorDefinitionByID(id, nil)
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
	def, err := s.operator.GetOperatorDefinitionByUID(uid, nil)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("operator-definitions/%s", def.Id), nil
}

func (s *service) includeDetailInRecipe(recipe *pipelinePB.Recipe, userUID uuid.UUID) error {

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for idx := range recipe.Components {

		switch recipe.Components[idx].Component.(type) {
		case *pipelinePB.Component_ConnectorComponent:
			conn, err := s.repository.GetConnectorByUIDAdmin(
				context.Background(),
				uuid.FromStringOrNil(strings.Split(recipe.Components[idx].GetConnectorComponent().ConnectorName, "/")[1]),
				false,
			)
			if err != nil {
				// Allow resource not created
				recipe.Components[idx].GetConnectorComponent().Connector = nil
			} else {
				pbConnector, err := s.convertDatamodelToProto(ctx, conn, ViewFull, true)
				if err != nil {
					// Allow resource not created
					recipe.Components[idx].GetConnectorComponent().Connector = nil
				} else {
					recipe.Components[idx].GetConnectorComponent().Connector = pbConnector
					str := structpb.Struct{}
					_ = str.UnmarshalJSON(conn.Configuration)
					// TODO: optimize this
					str.Fields["instill_user_uid"] = structpb.NewStringValue(userUID.String())
					str.Fields["instill_model_backend"] = structpb.NewStringValue(fmt.Sprintf("%s:%d", config.Config.ModelBackend.Host, config.Config.ModelBackend.PublicPort))
					str.Fields["instill_mgmt_backend"] = structpb.NewStringValue(fmt.Sprintf("%s:%d", config.Config.MgmtBackend.Host, config.Config.MgmtBackend.PublicPort))

					conf := &structpb.Struct{Fields: map[string]*structpb.Value{}}
					conf.Fields["input"] = structpb.NewStructValue(recipe.Components[idx].GetConnectorComponent().Input)
					d, err := s.connector.GetConnectorDefinitionByID(pbConnector.ConnectorDefinition.Id, &str, conf)
					if err != nil {
						return err
					}
					recipe.Components[idx].GetConnectorComponent().Definition = d
				}
			}
			if recipe.Components[idx].GetConnectorComponent().Definition == nil {
				uid, err := resource.GetRscPermalinkUID(recipe.Components[idx].GetConnectorComponent().DefinitionName)
				if err != nil {
					return err
				}
				conf := &structpb.Struct{Fields: map[string]*structpb.Value{}}
				conf.Fields["input"] = structpb.NewStructValue(recipe.Components[idx].GetConnectorComponent().Input)
				def, err := s.connector.GetConnectorDefinitionByUID(uid, nil, conf)
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

				recipe.Components[idx].GetConnectorComponent().Definition = def
			}

		case *pipelinePB.Component_OperatorComponent:
			uid, err := resource.GetRscPermalinkUID(recipe.Components[idx].GetOperatorComponent().DefinitionName)
			if err != nil {
				return err
			}
			conf := &structpb.Struct{Fields: map[string]*structpb.Value{}}
			conf.Fields["input"] = structpb.NewStructValue(recipe.Components[idx].GetOperatorComponent().Input)
			def, err := s.operator.GetOperatorDefinitionByUID(uid, conf)
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

			recipe.Components[idx].GetOperatorComponent().Definition = def
		}

	}
	return nil
}

// PBToDBPipeline converts protobuf data model to db data model
func (s *service) PBToDBPipeline(ctx context.Context, ns resource.Namespace, pbPipeline *pipelinePB.Pipeline) (*datamodel.Pipeline, error) {
	logger, _ := logger.GetZapLogger(ctx)

	var err error

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

	dbSharing := &datamodel.Sharing{}
	if pbPipeline.GetSharing() != nil {

		if err != nil {
			return nil, err
		}

		b, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(pbPipeline.GetSharing())
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(b, &dbSharing); err != nil {
			return nil, err
		}

	}

	return &datamodel.Pipeline{
		Owner: ns.Permalink(),
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
		Readme:  pbPipeline.Readme,
		Recipe:  recipe,
		Sharing: dbSharing,
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

var connectorTypeToComponentType = map[pipelinePB.ConnectorType]pipelinePB.ComponentType{
	pipelinePB.ConnectorType_CONNECTOR_TYPE_AI:          pipelinePB.ComponentType_COMPONENT_TYPE_CONNECTOR_AI,
	pipelinePB.ConnectorType_CONNECTOR_TYPE_APPLICATION: pipelinePB.ComponentType_COMPONENT_TYPE_CONNECTOR_APPLICATION,
	pipelinePB.ConnectorType_CONNECTOR_TYPE_DATA:        pipelinePB.ComponentType_COMPONENT_TYPE_CONNECTOR_DATA,
}

// DBToPBPipeline converts db data model to protobuf data model
func (s *service) DBToPBPipeline(ctx context.Context, dbPipeline *datamodel.Pipeline, authUser *AuthUser, view View) (*pipelinePB.Pipeline, error) {

	logger, _ := logger.GetZapLogger(ctx)

	ownerName, err := s.ConvertOwnerPermalinkToName(dbPipeline.Owner)
	if err != nil {
		return nil, err
	}
	var owner *mgmtPB.Owner
	if view != ViewBasic {
		owner, err = s.FetchOwnerWithPermalink(dbPipeline.Owner)
		if err != nil {
			return nil, err
		}
	}

	var pbRecipe *pipelinePB.Recipe

	var startComp *pipelinePB.Component
	var endComp *pipelinePB.Component

	if dbPipeline.Recipe != nil && view != ViewBasic {
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

	if view == ViewFull {
		if err := s.includeDetailInRecipe(pbRecipe, authUser.UID); err != nil {
			return nil, err
		}
		for i := range pbRecipe.Components {
			switch pbRecipe.Components[i].Component.(type) {
			case *pipelinePB.Component_StartComponent:
				startComp = pbRecipe.Components[i]
			case *pipelinePB.Component_EndComponent:
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

	pbSharing := &pipelinePB.Sharing{}

	b, err := json.Marshal(dbPipeline.Sharing)
	if err != nil {
		return nil, err
	}

	err = protojson.Unmarshal(b, pbSharing)
	if err != nil {
		return nil, err
	}
	if pbSharing != nil && pbSharing.ShareCode != nil {
		pbSharing.ShareCode.Code = dbPipeline.ShareCode
	}

	pbPipeline := pipelinePB.Pipeline{
		Name:       fmt.Sprintf("%s/pipelines/%s", ownerName, dbPipeline.ID),
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
		Readme:      dbPipeline.Readme,
		Recipe:      pbRecipe,
		Sharing:     pbSharing,
		OwnerName:   ownerName,
		Owner:       owner,
	}
	if authUser != nil {
		canEdit, err := s.aclClient.CheckPermission("pipeline", dbPipeline.UID, authUser.GetACLType(), authUser.UID, "", "writer")
		if err != nil {
			return nil, err
		}
		canTrigger, err := s.aclClient.CheckPermission("pipeline", dbPipeline.UID, authUser.GetACLType(), authUser.UID, "", "executor")
		if err != nil {
			return nil, err
		}
		pbPipeline.Permission = &pipelinePB.Permission{
			CanEdit:    canEdit,
			CanTrigger: canTrigger,
		}
	} else {
		pbPipeline.Permission = &pipelinePB.Permission{
			CanEdit:    true,
			CanTrigger: true,
		}
	}

	if view != ViewBasic {
		if dbPipeline.Metadata != nil {
			str := structpb.Struct{}
			err := str.UnmarshalJSON(dbPipeline.Metadata)
			if err != nil {
				logger.Error(err.Error())
			}
			pbPipeline.Metadata = &str
		}
	}

	if pbRecipe != nil && view == ViewFull && startComp != nil && endComp != nil {
		spec, err := s.GenerateOpenAPISpec(startComp, endComp, pbRecipe.Components)
		if err == nil {
			pbPipeline.OpenapiSchema = spec
		}
	}
	releases := []*datamodel.PipelineRelease{}
	pageToken := ""
	for {
		var page []*datamodel.PipelineRelease
		page, _, pageToken, err = s.repository.ListNamespacePipelineReleases(ctx, dbPipeline.Owner, dbPipeline.UID, 100, pageToken, false, filtering.Filter{}, false)
		if err != nil {
			return nil, err
		}
		releases = append(releases, page...)
		if pageToken == "" {
			break
		}
	}

	latestReleaseUID := uuid.Nil
	defaultReleaseUID := uuid.Nil
	latestRelease, err := s.repository.GetLatestNamespacePipelineRelease(ctx, dbPipeline.Owner, dbPipeline.UID, true)
	if err == nil {
		latestReleaseUID = latestRelease.UID
	}
	defaultRelease, err := s.repository.GetNamespacePipelineByID(ctx, dbPipeline.Owner, pbPipeline.Id, true)
	if err == nil {
		defaultReleaseUID = defaultRelease.UID
	}

	pbReleases, err := s.DBToPBPipelineReleases(ctx, releases, ViewFull, latestReleaseUID, defaultReleaseUID)
	if err != nil {
		return nil, err
	}
	pbPipeline.Releases = pbReleases
	pbPipeline.Visibility = pipelinePB.Pipeline_VISIBILITY_PRIVATE
	if u, ok := pbPipeline.Sharing.Users["*/*"]; ok {
		if u.Enabled {
			pbPipeline.Visibility = pipelinePB.Pipeline_VISIBILITY_PUBLIC
		}
	}

	return &pbPipeline, nil
}

// DBToPBPipeline converts db data model to protobuf data model
func (s *service) DBToPBPipelines(ctx context.Context, dbPipelines []*datamodel.Pipeline, authUser *AuthUser, view View) ([]*pipelinePB.Pipeline, error) {
	var err error
	pbPipelines := make([]*pipelinePB.Pipeline, len(dbPipelines))
	for idx := range dbPipelines {
		pbPipelines[idx], err = s.DBToPBPipeline(
			ctx,
			dbPipelines[idx],
			authUser,
			view,
		)
		if err != nil {
			return nil, err
		}

	}
	return pbPipelines, nil
}

// PBToDBPipelineRelease converts protobuf data model to db data model
func (s *service) PBToDBPipelineRelease(ctx context.Context, pipelineUID uuid.UUID, pbPipelineRelease *pipelinePB.PipelineRelease) (*datamodel.PipelineRelease, error) {
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
		Readme:      pbPipelineRelease.Readme,
		Recipe:      recipe,
		PipelineUID: pipelineUID,

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
	if dbPipelineRelease.Recipe != nil && view != ViewBasic {
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

	var startComp *pipelinePB.Component
	var endComp *pipelinePB.Component

	if view == ViewFull {
		if err := s.includeDetailInRecipe(pbRecipe, uuid.FromStringOrNil(strings.Split(dbPipeline.Owner, "/")[1])); err != nil {
			return nil, err
		}
		for i := range pbRecipe.Components {
			switch pbRecipe.Components[i].Component.(type) {
			case *pipelinePB.Component_StartComponent:
				startComp = pbRecipe.Components[i]
			case *pipelinePB.Component_EndComponent:
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
		Readme:      dbPipelineRelease.Readme,
		Recipe:      pbRecipe,
	}

	if view != ViewBasic {
		if dbPipelineRelease.Metadata != nil {
			str := structpb.Struct{}
			err := str.UnmarshalJSON(dbPipelineRelease.Metadata)
			if err != nil {
				logger.Error(err.Error())
			}
			pbPipelineRelease.Metadata = &str
		}
	}

	if pbRecipe != nil && view == ViewFull && startComp != nil && endComp != nil {
		spec, err := s.GenerateOpenAPISpec(startComp, endComp, pbRecipe.Components)
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
	ns resource.Namespace,
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

	connectorDefinition, err := s.connector.GetConnectorDefinitionByID(strings.Split(pbConnector.ConnectorDefinitionName, "/")[1], nil, nil)
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

	return &datamodel.Connector{
		Owner:                  ns.Permalink(),
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

	ownerName, err := s.ConvertOwnerPermalinkToName(dbConnector.Owner)
	if err != nil {
		return nil, err
	}
	var owner *mgmtPB.Owner
	if view != ViewBasic {
		owner, err = s.FetchOwnerWithPermalink(dbConnector.Owner)
		if err != nil {
			return nil, err
		}
	}

	dbConnDef, err := s.connector.GetConnectorDefinitionByUID(dbConnector.ConnectorDefinitionUID, nil, nil)
	if err != nil {
		return nil, err
	}

	pbConnector := &pipelinePB.Connector{
		Uid:                     dbConnector.UID.String(),
		Name:                    fmt.Sprintf("%s/connectors/%s", ownerName, dbConnector.ID),
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
					logger.Error(err.Error())
				}
				return &str
			}
			return nil
		}(),
		OwnerName: ownerName,
		Owner:     owner,
	}

	if view != ViewBasic {
		if view == ViewFull {

			str := proto.Clone(pbConnector.Configuration).(*structpb.Struct)
			// TODO: optimize this
			if str.Fields != nil {
				str.Fields["instill_user_uid"] = structpb.NewStringValue(uuid.FromStringOrNil(strings.Split(dbConnector.Owner, "/")[1]).String())
				str.Fields["instill_model_backend"] = structpb.NewStringValue(fmt.Sprintf("%s:%d", config.Config.ModelBackend.Host, config.Config.ModelBackend.PublicPort))
				str.Fields["instill_mgmt_backend"] = structpb.NewStringValue(fmt.Sprintf("%s:%d", config.Config.MgmtBackend.Host, config.Config.MgmtBackend.PublicPort))
			}

			dbConnDef, err := s.connector.GetConnectorDefinitionByUID(dbConnector.ConnectorDefinitionUID, str, nil)
			if err != nil {
				return nil, err
			}
			pbConnector.ConnectorDefinition = dbConnDef
		}
		if credentialMask {
			utils.MaskCredentialFields(s.connector, dbConnDef.Id, pbConnector.Configuration)
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
