package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gofrs/uuid"
	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/internal/resource"
	"github.com/instill-ai/pipeline-backend/pkg/constant"
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

func (s *service) recipeNameToPermalink(ctx context.Context, recipeRscName *pipelinePB.Recipe) (*pipelinePB.Recipe, error) {

	recipePermalink := &pipelinePB.Recipe{
		Version:    recipeRscName.Version,
		Components: make([]*pipelinePB.Component, len(recipeRscName.Components)),
	}

	type result struct {
		idx       int
		component *pipelinePB.Component
		err       error
	}
	ch := make(chan result)
	var wg sync.WaitGroup
	wg.Add(len(recipeRscName.Components))

	for idx := range recipeRscName.Components {

		go func(i int, component *pipelinePB.Component) {
			defer wg.Done()
			componentPermalink := &pipelinePB.Component{
				Id:        component.Id,
				Metadata:  component.Metadata,
				Component: component.Component,
			}

			switch component.Component.(type) {
			case *pipelinePB.Component_ConnectorComponent:
				connectorName := component.GetConnectorComponent().ConnectorName
				if connectorName != "" {
					connectorNameSplits := strings.Split(connectorName, "/")
					if len(connectorNameSplits) == 4 && (connectorNameSplits[0] == "users" || connectorNameSplits[0] == "organizations") && connectorNameSplits[2] == "connectors" {
						permalink, err := s.connectorNameToPermalink(ctx, component.GetConnectorComponent().ConnectorName)
						if err != nil {
							ch <- result{
								idx:       i,
								component: nil,
								err:       ErrConnectorNotFound,
							}
							return
						}
						componentPermalink.GetConnectorComponent().ConnectorName = permalink
					} else {
						ch <- result{
							idx:       i,
							component: nil,
							err:       fmt.Errorf("%s %v", connectorName, ErrConnectorNameError),
						}
						return
					}
				}
				defPermalink, err := s.connectorDefinitionNameToPermalink(ctx, component.GetConnectorComponent().DefinitionName)
				if err != nil {
					ch <- result{
						idx:       i,
						component: nil,
						err:       err,
					}
					return
				}
				componentPermalink.GetConnectorComponent().DefinitionName = defPermalink
			case *pipelinePB.Component_OperatorComponent:
				defPermalink, err := s.operatorDefinitionNameToPermalink(ctx, component.GetOperatorComponent().DefinitionName)
				if err != nil {
					ch <- result{
						idx:       i,
						component: nil,
						err:       err,
					}
					return
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
						connectorName := nestedComponent.GetConnectorComponent().ConnectorName
						if connectorName != "" {
							connectorNameSplits := strings.Split(connectorName, "/")
							if len(connectorNameSplits) == 4 && (connectorNameSplits[0] == "users" || connectorNameSplits[0] == "organizations") && connectorNameSplits[2] == "connectors" {
								permalink, err := s.connectorNameToPermalink(ctx, nestedComponent.GetConnectorComponent().ConnectorName)
								if err != nil {
									ch <- result{
										idx:       i,
										component: nil,
										err:       ErrConnectorNotFound,
									}
									return
								}
								nestedComponentPermalink.GetConnectorComponent().ConnectorName = permalink
							} else {
								ch <- result{
									idx:       i,
									component: nil,
									err:       fmt.Errorf("%s %v", connectorName, ErrConnectorNameError),
								}
								return
							}

						}
						defPermalink, err := s.connectorDefinitionNameToPermalink(ctx, nestedComponent.GetConnectorComponent().DefinitionName)
						if err != nil {
							ch <- result{
								idx:       i,
								component: nil,
								err:       err,
							}
							return
						}
						nestedComponentPermalink.GetConnectorComponent().DefinitionName = defPermalink
						nestedComponentPermalinks = append(nestedComponentPermalinks, nestedComponentPermalink)
					case *pipelinePB.NestedComponent_OperatorComponent:
						defPermalink, err := s.operatorDefinitionNameToPermalink(ctx, nestedComponent.GetOperatorComponent().DefinitionName)
						if err != nil {
							ch <- result{
								idx:       i,
								component: nil,
								err:       err,
							}
							return
						}
						nestedComponentPermalink.GetOperatorComponent().DefinitionName = defPermalink
						nestedComponentPermalinks = append(nestedComponentPermalinks, nestedComponentPermalink)
					}
				}
				componentPermalink.GetIteratorComponent().Components = nestedComponentPermalinks

			}
			ch <- result{
				idx:       i,
				component: componentPermalink,
				err:       nil,
			}

		}(idx, recipeRscName.Components[idx])
	}

	for range recipeRscName.Components {
		r := <-ch
		if r.err != nil {
			return nil, r.err
		}
		recipePermalink.Components[r.idx] = r.component
	}

	return recipePermalink, nil
}

func (s *service) recipePermalinkToName(ctx context.Context, recipePermalink *pipelinePB.Recipe) (*pipelinePB.Recipe, error) {

	recipe := &pipelinePB.Recipe{
		Version:    recipePermalink.Version,
		Components: make([]*pipelinePB.Component, len(recipePermalink.Components)),
	}

	type result struct {
		idx       int
		component *pipelinePB.Component
		err       error
	}
	ch := make(chan result)
	var wg sync.WaitGroup
	wg.Add(len(recipePermalink.Components))

	for idx := range recipePermalink.Components {

		go func(i int, componentPermalink *pipelinePB.Component) {
			defer wg.Done()
			component := &pipelinePB.Component{
				Id:        componentPermalink.Id,
				Metadata:  componentPermalink.Metadata,
				Component: componentPermalink.Component,
			}

			switch component.Component.(type) {
			case *pipelinePB.Component_ConnectorComponent:
				if componentPermalink.GetConnectorComponent().ConnectorName != "" {
					name, err := s.connectorPermalinkToName(ctx, componentPermalink.GetConnectorComponent().ConnectorName)
					if err != nil {
						// Allow the connector to not exist instead of returning an error.
						component.GetConnectorComponent().ConnectorName = ""
					} else {
						component.GetConnectorComponent().ConnectorName = name
					}
				}
				defName, err := s.connectorDefinitionPermalinkToName(ctx, componentPermalink.GetConnectorComponent().DefinitionName)
				if err != nil {
					ch <- result{
						idx:       i,
						component: nil,
						err:       err,
					}
					return
				}
				component.GetConnectorComponent().DefinitionName = defName
			case *pipelinePB.Component_OperatorComponent:
				defName, err := s.operatorDefinitionPermalinkToName(ctx, componentPermalink.GetOperatorComponent().DefinitionName)
				if err != nil {
					ch <- result{
						idx:       i,
						component: nil,
						err:       err,
					}
					return
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
						if nestedComponentPermalink.GetConnectorComponent().ConnectorName != "" {
							name, err := s.connectorPermalinkToName(ctx, nestedComponentPermalink.GetConnectorComponent().ConnectorName)
							if err != nil {
								// Allow the connector to not exist instead of returning an error.
								nestedComponent.GetConnectorComponent().ConnectorName = ""
							} else {
								nestedComponent.GetConnectorComponent().ConnectorName = name
							}
						}
						defName, err := s.connectorDefinitionPermalinkToName(ctx, nestedComponentPermalink.GetConnectorComponent().DefinitionName)
						if err != nil {
							ch <- result{
								idx:       i,
								component: nil,
								err:       err,
							}
							return
						}
						nestedComponent.GetConnectorComponent().DefinitionName = defName
						nestedComponents = append(nestedComponents, nestedComponent)
					case *pipelinePB.NestedComponent_OperatorComponent:
						defName, err := s.operatorDefinitionPermalinkToName(ctx, nestedComponentPermalink.GetOperatorComponent().DefinitionName)
						if err != nil {
							ch <- result{
								idx:       i,
								component: nil,
								err:       err,
							}
							return
						}
						nestedComponent.GetOperatorComponent().DefinitionName = defName
						nestedComponents = append(nestedComponents, nestedComponent)
					}
				}
				component.GetIteratorComponent().Components = nestedComponents
			}
			ch <- result{
				idx:       i,
				component: component,
				err:       nil,
			}

		}(idx, recipePermalink.Components[idx])
	}

	for range recipe.Components {
		r := <-ch
		if r.err != nil {
			return nil, r.err
		}
		recipe.Components[r.idx] = r.component
	}

	return recipe, nil
}

func (s *service) connectorNameToPermalink(ctx context.Context, name string) (string, error) {

	ownerPermalink, err := s.convertOwnerNameToPermalink(ctx, fmt.Sprintf("%s/%s", strings.Split(name, "/")[0], strings.Split(name, "/")[1]))
	if err != nil {
		return "", err
	}
	dbConnector, err := s.repository.GetNamespaceConnectorByID(ctx, ownerPermalink, strings.Split(name, "/")[3], true)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("connectors/%s", dbConnector.UID), nil
}

func (s *service) connectorPermalinkToName(ctx context.Context, permalink string) (string, error) {

	dbConnector, err := s.repository.GetConnectorByUID(ctx, uuid.FromStringOrNil(strings.Split(permalink, "/")[1]), true)
	if err != nil {
		return "", err
	}
	owner, err := s.convertOwnerPermalinkToName(ctx, dbConnector.Owner)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/connectors/%s", owner, dbConnector.ID), nil

}

func (s *service) connectorDefinitionNameToPermalink(ctx context.Context, name string) (string, error) {

	def, err := s.connector.GetConnectorDefinitionByID(strings.Split(name, "/")[1], nil, nil)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("connector-definitions/%s", def.Uid), nil
}

func (s *service) connectorDefinitionPermalinkToName(ctx context.Context, permalink string) (string, error) {
	def, err := s.connector.GetConnectorDefinitionByUID(uuid.FromStringOrNil(strings.Split(permalink, "/")[1]), nil, nil)
	if err != nil {
		return "", err
	}
	return def.Name, nil
}

func (s *service) operatorDefinitionNameToPermalink(ctx context.Context, name string) (string, error) {
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

func (s *service) operatorDefinitionPermalinkToName(ctx context.Context, permalink string) (string, error) {
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

func (s *service) includeOperatorComponentDetail(ctx context.Context, comp *pipelinePB.OperatorComponent) error {
	uid, err := resource.GetRscPermalinkUID(comp.DefinitionName)
	if err != nil {
		return err
	}
	def, err := s.operator.GetOperatorDefinitionByUID(uid, comp)
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

	comp.Definition = def
	return nil
}

func (s *service) includeConnectorComponentDetail(ctx context.Context, comp *pipelinePB.ConnectorComponent) error {
	if comp.ConnectorName != "" {
		conn, err := s.repository.GetConnectorByUIDAdmin(
			ctx,
			uuid.FromStringOrNil(strings.Split(comp.ConnectorName, "/")[1]),
			false,
		)
		if err != nil {
			// Allow the connector to not exist instead of returning an error.
			comp.Connector = nil
		} else {
			ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			pbConnector, err := s.convertDatamodelToProto(ctx, conn, ViewFull, true)
			if err != nil {
				// Allow the connector to not exist instead of returning an error.
				comp.Connector = nil
			} else {
				comp.Connector = pbConnector
				str := structpb.Struct{}
				_ = str.UnmarshalJSON(conn.Configuration)
				// TODO: optimize this
				str.Fields["instill_user_uid"] = structpb.NewStringValue(resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey))
				str.Fields["instill_model_backend"] = structpb.NewStringValue(fmt.Sprintf("%s:%d", config.Config.ModelBackend.Host, config.Config.ModelBackend.PublicPort))
				str.Fields["instill_mgmt_backend"] = structpb.NewStringValue(fmt.Sprintf("%s:%d", config.Config.MgmtBackend.Host, config.Config.MgmtBackend.PublicPort))

				d, err := s.connector.GetConnectorDefinitionByID(pbConnector.ConnectorDefinition.Id, &str, comp)
				if err != nil {
					return err
				}
				comp.Definition = d
			}
		}
	}
	if comp.Definition == nil {
		uid, err := resource.GetRscPermalinkUID(comp.DefinitionName)
		if err != nil {
			return err
		}
		def, err := s.connector.GetConnectorDefinitionByUID(uid, nil, comp)
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

		comp.Definition = def
	}
	return nil
}

func (s *service) includeIteratorComponentDetail(ctx context.Context, comp *pipelinePB.IteratorComponent) error {

	var wg sync.WaitGroup
	wg.Add(len(comp.Components))
	ch := make(chan error)

	for nestIdx := range comp.Components {
		go func(c *pipelinePB.NestedComponent) {
			defer wg.Done()
			switch c.Component.(type) {
			case *pipelinePB.NestedComponent_ConnectorComponent:
				err := s.includeConnectorComponentDetail(ctx, c.GetConnectorComponent())
				ch <- err
			case *pipelinePB.NestedComponent_OperatorComponent:
				err := s.includeOperatorComponentDetail(ctx, c.GetOperatorComponent())
				ch <- err
			default:
				ch <- nil
			}
		}(comp.Components[nestIdx])

	}

	for range comp.Components {
		err := <-ch
		if err != nil {
			return err
		}
	}

	dataOutput := &structpb.Struct{Fields: make(map[string]*structpb.Value)}
	dataOutput.Fields["type"] = structpb.NewStringValue("object")
	dataOutput.Fields["properties"] = structpb.NewStructValue(&structpb.Struct{Fields: make(map[string]*structpb.Value)})

	for k, v := range comp.OutputElements {
		path := v
		if strings.HasPrefix(path, "${") && strings.HasSuffix(path, "}") && strings.Count(path, "${") == 1 {
			// Remove "${" and "}"
			path = path[2:]
			path = path[:len(path)-1]
			path = strings.ReplaceAll(path, " ", "")

			// Find upstream component
			compID := strings.Split(path, ".")[0]
			path = path[len(compID):]
			upstreamCompIdx := -1
			for compIdx := range comp.Components {
				if comp.Components[compIdx].Id == compID {
					upstreamCompIdx = compIdx
				}
			}
			if upstreamCompIdx != -1 {
				nestedComp := comp.Components[upstreamCompIdx]

				var walk *structpb.Value
				switch nestedComp.Component.(type) {
				case *pipelinePB.NestedComponent_ConnectorComponent:
					task := nestedComp.GetConnectorComponent().GetTask()
					if task == "" {
						// Skip schema generation if the task is not set.
						continue
					}

					splits := strings.Split(path, ".")

					if splits[1] == "output" {
						walk = structpb.NewStructValue(nestedComp.GetConnectorComponent().GetDefinition().Spec.DataSpecifications[task].Output)
					} else if splits[1] == "input" {
						walk = structpb.NewStructValue(nestedComp.GetConnectorComponent().GetDefinition().Spec.DataSpecifications[task].Input)
					} else {
						// Skip schema generation if the configuration is not valid.
						continue
					}
					path = path[len(splits[1])+1:]
				case *pipelinePB.NestedComponent_OperatorComponent:
					task := nestedComp.GetOperatorComponent().GetTask()
					if task == "" {
						// Skip schema generation if the task is not set.
						continue
					}
					splits := strings.Split(path, ".")

					if splits[1] == "output" {
						walk = structpb.NewStructValue(nestedComp.GetOperatorComponent().GetDefinition().Spec.DataSpecifications[task].Output)
					} else if splits[1] == "input" {
						walk = structpb.NewStructValue(nestedComp.GetOperatorComponent().GetDefinition().Spec.DataSpecifications[task].Input)
					} else {
						// Skip schema generation if the configuration is not valid.
						continue
					}
					path = path[len(splits[1])+1:]
				}

				success := true

				// Traverse the schema of upstream component
				for {
					if len(path) == 0 {
						break
					}

					splits := strings.Split(path, ".")
					curr := splits[1]

					if strings.Contains(curr, "[") && strings.Contains(curr, "]") {
						target := strings.Split(curr, "[")[0]
						if _, ok := walk.GetStructValue().Fields["properties"]; ok {
							if _, ok := walk.GetStructValue().Fields["properties"].GetStructValue().Fields[target]; ok {
								walk = walk.GetStructValue().Fields["properties"].GetStructValue().Fields[target].GetStructValue().Fields["items"]
							} else {
								success = false
								break
							}
						} else {
							success = false
							break
						}
					} else {
						target := curr

						if _, ok := walk.GetStructValue().Fields["properties"]; ok {
							if _, ok := walk.GetStructValue().Fields["properties"].GetStructValue().Fields[target]; ok {
								walk = walk.GetStructValue().Fields["properties"].GetStructValue().Fields[target]
							} else {
								success = false
								break
							}
						} else {
							success = false
							break
						}

					}

					path = path[len(curr)+1:]
				}
				if success {
					s := &structpb.Struct{Fields: map[string]*structpb.Value{}}
					s.Fields["type"] = structpb.NewStringValue("array")
					if f := walk.GetStructValue().Fields["instillFormat"].GetStringValue(); f != "" {
						// Limitation: console can not support more then three levels of array.
						if strings.Count(f, "array:") < 2 {
							s.Fields["instillFormat"] = structpb.NewStringValue("array:" + f)
						}
					}
					s.Fields["items"] = structpb.NewStructValue(walk.GetStructValue())
					dataOutput.Fields["properties"].GetStructValue().Fields[k] = structpb.NewStructValue(s)
				}

			}
		}
	}

	comp.DataSpecification = &pipelinePB.DataSpecification{
		Output: dataOutput,
	}

	return nil
}

func (s *service) includeDetailInRecipe(ctx context.Context, recipe *pipelinePB.Recipe) error {

	var wg sync.WaitGroup
	wg.Add(len(recipe.Components))
	ch := make(chan error)
	for idx := range recipe.Components {
		go func(comp *pipelinePB.Component) {
			defer wg.Done()
			switch comp.Component.(type) {
			case *pipelinePB.Component_ConnectorComponent:
				err := s.includeConnectorComponentDetail(ctx, comp.GetConnectorComponent())
				ch <- err
			case *pipelinePB.Component_OperatorComponent:
				err := s.includeOperatorComponentDetail(ctx, comp.GetOperatorComponent())
				ch <- err
			case *pipelinePB.Component_IteratorComponent:
				err := s.includeIteratorComponentDetail(ctx, comp.GetIteratorComponent())
				ch <- err
			default:
				ch <- nil
			}
		}(recipe.Components[idx])
	}
	for range recipe.Components {
		err := <-ch
		if err != nil {
			return err
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
		recipePermalink, err := s.recipeNameToPermalink(ctx, pbPipeline.Recipe)
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

// ConnectorTypeToComponentType ...
var ConnectorTypeToComponentType = map[pipelinePB.ConnectorType]pipelinePB.ComponentType{
	pipelinePB.ConnectorType_CONNECTOR_TYPE_AI:          pipelinePB.ComponentType_COMPONENT_TYPE_CONNECTOR_AI,
	pipelinePB.ConnectorType_CONNECTOR_TYPE_APPLICATION: pipelinePB.ComponentType_COMPONENT_TYPE_CONNECTOR_APPLICATION,
	pipelinePB.ConnectorType_CONNECTOR_TYPE_DATA:        pipelinePB.ComponentType_COMPONENT_TYPE_CONNECTOR_DATA,
}

// DBToPBPipeline converts db data model to protobuf data model
func (s *service) DBToPBPipeline(ctx context.Context, dbPipeline *datamodel.Pipeline, view View, checkPermission bool) (*pipelinePB.Pipeline, error) {

	logger, _ := logger.GetZapLogger(ctx)

	ownerName, err := s.convertOwnerPermalinkToName(ctx, dbPipeline.Owner)
	if err != nil {
		return nil, err
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
		if err := s.includeDetailInRecipe(ctx, pbRecipe); err != nil {
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
		pbRecipe, err = s.recipePermalinkToName(ctx, pbRecipe)
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
	}

	var wg sync.WaitGroup
	wg.Add(5)
	go func() {
		defer wg.Done()
		var owner *mgmtPB.Owner
		if view != ViewBasic {
			owner, err = s.fetchOwnerByPermalink(ctx, dbPipeline.Owner)
			if err != nil {
				return
			}
			pbPipeline.Owner = owner
		}
	}()
	pbPipeline.Permission = &pipelinePB.Permission{}
	go func() {
		defer wg.Done()
		if !checkPermission {
			return
		}

		canEdit, err := s.aclClient.CheckPermission(ctx, "pipeline", dbPipeline.UID, "writer")
		if err != nil {
			return
		}
		pbPipeline.Permission.CanEdit = canEdit
	}()
	go func() {
		defer wg.Done()
		if !checkPermission {
			return
		}

		canTrigger, err := s.aclClient.CheckPermission(ctx, "pipeline", dbPipeline.UID, "executor")
		if err != nil {
			return
		}
		pbPipeline.Permission.CanTrigger = canTrigger
	}()

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

	go func() {
		defer wg.Done()
		if pbRecipe != nil && view == ViewFull && startComp != nil && endComp != nil {
			spec, err := s.GeneratePipelineDataSpec(startComp, endComp, pbRecipe.Components)
			if err != nil {
				return
			}
			pbPipeline.DataSpecification = spec
		}
	}()

	go func() {
		defer wg.Done()
		releases := []*datamodel.PipelineRelease{}
		pageToken := ""
		for {
			var page []*datamodel.PipelineRelease
			page, _, pageToken, err = s.repository.ListNamespacePipelineReleases(ctx, dbPipeline.Owner, dbPipeline.UID, 100, pageToken, false, filtering.Filter{}, false)
			if err != nil {
				return
			}
			releases = append(releases, page...)
			if pageToken == "" {
				break
			}
		}

		pbReleases, err := s.DBToPBPipelineReleases(ctx, releases, ViewFull)
		if err != nil {
			return
		}
		pbPipeline.Releases = pbReleases
	}()

	wg.Wait()

	pbPipeline.Visibility = pipelinePB.Pipeline_VISIBILITY_PRIVATE
	if u, ok := pbPipeline.Sharing.Users["*/*"]; ok {
		if u.Enabled {
			pbPipeline.Visibility = pipelinePB.Pipeline_VISIBILITY_PUBLIC
		}
	}
	return &pbPipeline, nil
}

// DBToPBPipeline converts db data model to protobuf data model
func (s *service) DBToPBPipelines(ctx context.Context, dbPipelines []*datamodel.Pipeline, view View, checkPermission bool) ([]*pipelinePB.Pipeline, error) {

	type result struct {
		idx      int
		pipeline *pipelinePB.Pipeline
		err      error
	}

	var wg sync.WaitGroup
	wg.Add(len(dbPipelines))
	ch := make(chan result)

	for idx := range dbPipelines {
		go func(i int) {
			defer wg.Done()
			pbPipeline, err := s.DBToPBPipeline(
				ctx,
				dbPipelines[i],
				view,
				checkPermission,
			)
			ch <- result{
				idx:      i,
				pipeline: pbPipeline,
				err:      err,
			}
		}(idx)
	}

	pbPipelines := make([]*pipelinePB.Pipeline, len(dbPipelines))
	for range dbPipelines {
		r := <-ch
		if r.err != nil {
			return nil, r.err
		}
		pbPipelines[r.idx] = r.pipeline
	}
	return pbPipelines, nil
}

// PBToDBPipelineRelease converts protobuf data model to db data model
func (s *service) PBToDBPipelineRelease(ctx context.Context, pipelineUID uuid.UUID, pbPipelineRelease *pipelinePB.PipelineRelease) (*datamodel.PipelineRelease, error) {
	logger, _ := logger.GetZapLogger(ctx)

	recipe := &datamodel.Recipe{}
	if pbPipelineRelease.GetRecipe() != nil {
		recipePermalink, err := s.recipeNameToPermalink(ctx, pbPipelineRelease.Recipe)
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
func (s *service) DBToPBPipelineRelease(ctx context.Context, dbPipelineRelease *datamodel.PipelineRelease, view View) (*pipelinePB.PipelineRelease, error) {

	logger, _ := logger.GetZapLogger(ctx)

	dbPipeline, err := s.repository.GetPipelineByUIDAdmin(ctx, dbPipelineRelease.PipelineUID, true)
	if err != nil {
		return nil, err
	}
	owner, err := s.convertOwnerPermalinkToName(ctx, dbPipeline.Owner)
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
		if err := s.includeDetailInRecipe(ctx, pbRecipe); err != nil {
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
		pbRecipe, err = s.recipePermalinkToName(ctx, pbRecipe)
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
		spec, err := s.GeneratePipelineDataSpec(startComp, endComp, pbRecipe.Components)
		if err == nil {
			pbPipelineRelease.DataSpecification = spec
		}
	}

	return &pbPipelineRelease, nil
}

// DBToPBPipelineRelease converts db data model to protobuf data model
func (s *service) DBToPBPipelineReleases(ctx context.Context, dbPipelineRelease []*datamodel.PipelineRelease, view View) ([]*pipelinePB.PipelineRelease, error) {

	type result struct {
		idx     int
		release *pipelinePB.PipelineRelease
		err     error
	}

	var wg sync.WaitGroup
	wg.Add(len(dbPipelineRelease))
	ch := make(chan result)

	for idx := range dbPipelineRelease {
		go func(i int) {
			defer wg.Done()
			pbRelease, err := s.DBToPBPipelineRelease(
				ctx,
				dbPipelineRelease[i],
				view,
			)
			ch <- result{
				idx:     i,
				release: pbRelease,
				err:     err,
			}
		}(idx)
	}

	pbPipelineReleases := make([]*pipelinePB.PipelineRelease, len(dbPipelineRelease))
	for range dbPipelineRelease {
		r := <-ch
		if r.err != nil {
			return nil, r.err
		}
		pbPipelineReleases[r.idx] = r.release
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

	ownerName, err := s.convertOwnerPermalinkToName(ctx, dbConnector.Owner)
	if err != nil {
		return nil, err
	}
	var owner *mgmtPB.Owner
	if view != ViewBasic {
		owner, err = s.fetchOwnerByPermalink(ctx, dbConnector.Owner)
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

// TODO: refactor these codes
func (s *service) GeneratePipelineDataSpec(startCompOrigin *pipelinePB.Component, endCompOrigin *pipelinePB.Component, compsOrigin []*pipelinePB.Component) (*pipelinePB.DataSpecification, error) {
	success := true
	pipelineDataSpec := &pipelinePB.DataSpecification{}

	dataInput := &structpb.Struct{Fields: make(map[string]*structpb.Value)}
	dataInput.Fields["type"] = structpb.NewStringValue("object")
	dataInput.Fields["properties"] = structpb.NewStructValue(&structpb.Struct{Fields: make(map[string]*structpb.Value)})

	startComp := proto.Clone(startCompOrigin).(*pipelinePB.Component)
	for k, v := range startComp.GetStartComponent().GetFields() {
		b, _ := protojson.Marshal(v)
		p := &structpb.Struct{}
		_ = protojson.Unmarshal(b, p)
		dataInput.Fields["properties"].GetStructValue().Fields[k] = structpb.NewStructValue(p)
	}

	// output
	dataOutput := &structpb.Struct{Fields: make(map[string]*structpb.Value)}
	dataOutput.Fields["type"] = structpb.NewStringValue("object")
	dataOutput.Fields["properties"] = structpb.NewStructValue(&structpb.Struct{Fields: make(map[string]*structpb.Value)})

	endComp := proto.Clone(endCompOrigin).(*pipelinePB.Component)

	for k, v := range endComp.GetEndComponent().Fields {
		var m *structpb.Value

		var err error

		str := v.Value
		if strings.HasPrefix(str, "${") && strings.HasSuffix(str, "}") && strings.Count(str, "${") == 1 {
			// TODO
			str = str[2:]
			str = str[:len(str)-1]
			str = strings.ReplaceAll(str, " ", "")

			compID := strings.Split(str, ".")[0]
			str = str[len(strings.Split(str, ".")[0]):]
			upstreamCompIdx := -1
			for compIdx := range compsOrigin {
				if compsOrigin[compIdx].Id == compID {
					upstreamCompIdx = compIdx
				}
			}

			if upstreamCompIdx != -1 {
				comp := proto.Clone(compsOrigin[upstreamCompIdx]).(*pipelinePB.Component)

				var walk *structpb.Value
				switch comp.Component.(type) {
				case *pipelinePB.Component_IteratorComponent:

					splits := strings.Split(str, ".")

					if splits[1] == "output" {
						walk = structpb.NewStructValue(comp.GetIteratorComponent().DataSpecification.Output)
					} else {
						return nil, fmt.Errorf("generate OpenAPI spec error")
					}
					str = str[len(splits[1])+1:]
				case *pipelinePB.Component_ConnectorComponent:
					task := comp.GetConnectorComponent().GetTask()
					if task == "" {
						return nil, fmt.Errorf("generate OpenAPI spec error")
					}

					splits := strings.Split(str, ".")

					if splits[1] == "output" {
						walk = structpb.NewStructValue(comp.GetConnectorComponent().GetDefinition().Spec.DataSpecifications[task].Output)
					} else if splits[1] == "input" {
						walk = structpb.NewStructValue(comp.GetConnectorComponent().GetDefinition().Spec.DataSpecifications[task].Input)
					} else {
						return nil, fmt.Errorf("generate OpenAPI spec error")
					}
					str = str[len(splits[1])+1:]
				case *pipelinePB.Component_StartComponent:
					walk = structpb.NewStructValue(dataInput)
				case *pipelinePB.Component_OperatorComponent:
					task := comp.GetOperatorComponent().GetTask()
					if task == "" {
						return nil, fmt.Errorf("generate OpenAPI spec error")
					}

					splits := strings.Split(str, ".")

					if splits[1] == "output" {
						walk = structpb.NewStructValue(comp.GetOperatorComponent().GetDefinition().Spec.DataSpecifications[task].Output)
					} else if splits[1] == "input" {
						walk = structpb.NewStructValue(comp.GetOperatorComponent().GetDefinition().Spec.DataSpecifications[task].Input)
					} else {
						return nil, fmt.Errorf("generate OpenAPI spec error")
					}
					str = str[len(splits[1])+1:]
				}

				for {
					if len(str) == 0 {
						break
					}

					splits := strings.Split(str, ".")
					curr := splits[1]

					if strings.Contains(curr, "[") && strings.Contains(curr, "]") {
						target := strings.Split(curr, "[")[0]
						if _, ok := walk.GetStructValue().Fields["properties"]; ok {
							if _, ok := walk.GetStructValue().Fields["properties"].GetStructValue().Fields[target]; !ok {
								break
							}
						} else {
							break
						}
						walk = walk.GetStructValue().Fields["properties"].GetStructValue().Fields[target].GetStructValue().Fields["items"]
					} else {
						target := curr

						if _, ok := walk.GetStructValue().Fields["properties"]; ok {
							if _, ok := walk.GetStructValue().Fields["properties"].GetStructValue().Fields[target]; !ok {
								break
							}
						} else {
							break
						}

						walk = walk.GetStructValue().Fields["properties"].GetStructValue().Fields[target]

					}

					str = str[len(curr)+1:]
				}
				m = structpb.NewStructValue(walk.GetStructValue())

			} else {
				return nil, fmt.Errorf("generate data spec error")
			}

			if m.GetStructValue() != nil && m.GetStructValue().Fields != nil {
				m.GetStructValue().Fields["title"] = structpb.NewStringValue(v.Title)
			}
			if m.GetStructValue() != nil && m.GetStructValue().Fields != nil {
				m.GetStructValue().Fields["description"] = structpb.NewStringValue(v.Description)
			}
			if m.GetStructValue() != nil && m.GetStructValue().Fields != nil {
				m.GetStructValue().Fields["instillUIOrder"] = structpb.NewNumberValue(float64(v.InstillUiOrder))
			}

		} else {
			m, err = structpb.NewValue(map[string]interface{}{
				"title":          v.Title,
				"description":    v.Description,
				"instillUIOrder": v.InstillUiOrder,
				"type":           "string",
				"instillFormat":  "string",
			})
		}

		if err != nil {
			success = false
		} else {
			dataOutput.Fields["properties"].GetStructValue().Fields[k] = m
		}

	}

	if success {
		pipelineDataSpec.Input = dataInput
		pipelineDataSpec.Output = dataOutput
		return pipelineDataSpec, nil
	}
	return nil, fmt.Errorf("generate data spec error")

}
