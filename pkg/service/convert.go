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
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/instill-ai/pipeline-backend/internal/resource"
	"github.com/instill-ai/pipeline-backend/pkg/constant"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/logger"

	mgmtPB "github.com/instill-ai/protogen-go/core/mgmt/v1beta"
	pb "github.com/instill-ai/protogen-go/vdp/pipeline/v1beta"
)

// In the API, we expose the human-readable ID to the user. But in the database, we store it with UUID as the permanent identifier.
// The `convertResourceNameToPermalink` function converts all resources that use ID to UUID.
func (s *service) convertResourceNameToPermalink(ctx context.Context, rsc any) error {

	switch rsc := rsc.(type) {
	case *pb.Recipe:
		for idx := range rsc.Components {
			if err := s.convertResourceNameToPermalink(ctx, rsc.Components[idx]); err != nil {
				return err
			}
		}
	case *pb.Component:
		return s.convertResourceNameToPermalink(ctx, rsc.Component)
	case *pb.NestedComponent:
		return s.convertResourceNameToPermalink(ctx, rsc.Component)
	case *pb.Component_IteratorComponent:
		for idx := range rsc.IteratorComponent.Components {
			if err := s.convertResourceNameToPermalink(ctx, rsc.IteratorComponent.Components[idx]); err != nil {
				return err
			}
		}

	case *pb.Component_ConnectorComponent:
		id, err := resource.GetRscNameID(rsc.ConnectorComponent.DefinitionName)
		if err != nil {
			return err
		}
		def, err := s.connector.GetConnectorDefinitionByID(id, nil)
		if err != nil {
			return err
		}
		rsc.ConnectorComponent.DefinitionName = fmt.Sprintf("connector-definitions/%s", def.Uid)
		return nil

	case *pb.NestedComponent_ConnectorComponent:
		id, err := resource.GetRscNameID(rsc.ConnectorComponent.DefinitionName)
		if err != nil {
			return err
		}
		def, err := s.connector.GetConnectorDefinitionByID(id, nil)
		if err != nil {
			return err
		}
		rsc.ConnectorComponent.DefinitionName = fmt.Sprintf("connector-definitions/%s", def.Uid)
		return nil

	case *pb.Component_OperatorComponent:
		id, err := resource.GetRscNameID(rsc.OperatorComponent.DefinitionName)
		if err != nil {
			return err
		}
		def, err := s.operator.GetOperatorDefinitionByID(id, nil)
		if err != nil {
			return err
		}
		rsc.OperatorComponent.DefinitionName = fmt.Sprintf("operator-definitions/%s", def.Uid)
		return nil

	case *pb.NestedComponent_OperatorComponent:
		id, err := resource.GetRscNameID(rsc.OperatorComponent.DefinitionName)
		if err != nil {
			return err
		}
		def, err := s.operator.GetOperatorDefinitionByID(id, nil)
		if err != nil {
			return err
		}
		rsc.OperatorComponent.DefinitionName = fmt.Sprintf("operator-definitions/%s", def.Uid)
		return nil
	}
	return nil
}

// In the API, we expose the human-readable ID to the user. But in the database, we store it with UUID as the permanent identifier.
// The `convertResourceNameToPermalink` function converts all resources that use UUID to ID.
func (s *service) convertResourcePermalinkToName(ctx context.Context, rsc any) error {

	switch rsc := rsc.(type) {
	case *pb.Recipe:
		for idx := range rsc.Components {
			if err := s.convertResourcePermalinkToName(ctx, rsc.Components[idx]); err != nil {
				return err
			}
		}
	case *pb.Component:
		return s.convertResourcePermalinkToName(ctx, rsc.Component)
	case *pb.NestedComponent:
		return s.convertResourcePermalinkToName(ctx, rsc.Component)
	case *pb.Component_IteratorComponent:
		for idx := range rsc.IteratorComponent.Components {
			if err := s.convertResourcePermalinkToName(ctx, rsc.IteratorComponent.Components[idx]); err != nil {
				return err
			}
		}

	case *pb.Component_ConnectorComponent:
		uid, err := resource.GetRscPermalinkUID(rsc.ConnectorComponent.DefinitionName)
		if err != nil {
			return err
		}
		def, err := s.connector.GetConnectorDefinitionByUID(uid, nil)
		if err != nil {
			return err
		}
		rsc.ConnectorComponent.DefinitionName = def.Name

	case *pb.NestedComponent_ConnectorComponent:
		uid, err := resource.GetRscPermalinkUID(rsc.ConnectorComponent.DefinitionName)
		if err != nil {
			return err
		}
		def, err := s.connector.GetConnectorDefinitionByUID(uid, nil)
		if err != nil {
			return err
		}
		rsc.ConnectorComponent.DefinitionName = def.Name

	case *pb.Component_OperatorComponent:
		uid, err := resource.GetRscPermalinkUID(rsc.OperatorComponent.DefinitionName)
		if err != nil {
			return err
		}
		def, err := s.operator.GetOperatorDefinitionByUID(uid, nil)
		if err != nil {
			return err
		}
		rsc.OperatorComponent.DefinitionName = def.Name

	case *pb.NestedComponent_OperatorComponent:
		uid, err := resource.GetRscPermalinkUID(rsc.OperatorComponent.DefinitionName)
		if err != nil {
			return err
		}
		def, err := s.operator.GetOperatorDefinitionByUID(uid, nil)
		if err != nil {
			return err
		}
		rsc.OperatorComponent.DefinitionName = def.Name
	}
	return nil
}

func (s *service) includeOperatorComponentDetail(ctx context.Context, comp *pb.OperatorComponent) error {
	uid, err := resource.GetRscPermalinkUID(comp.DefinitionName)
	if err != nil {
		return err
	}
	def, err := s.operator.GetOperatorDefinitionByUID(uid, comp)
	if err != nil {
		return err
	}

	comp.Definition = def
	return nil
}

func (s *service) includeConnectorComponentDetail(ctx context.Context, comp *pb.ConnectorComponent) error {
	uid, err := resource.GetRscPermalinkUID(comp.DefinitionName)
	if err != nil {
		return err
	}
	def, err := s.connector.GetConnectorDefinitionByUID(uid, comp)
	if err != nil {
		return err
	}

	comp.Definition = def
	return nil
}

func (s *service) includeIteratorComponentDetail(ctx context.Context, comp *pb.IteratorComponent) error {

	for nestIdx := range comp.Components {
		var err error
		switch comp.Components[nestIdx].Component.(type) {
		case *pb.NestedComponent_ConnectorComponent:
			err = s.includeConnectorComponentDetail(ctx, comp.Components[nestIdx].GetConnectorComponent())
		case *pb.NestedComponent_OperatorComponent:
			err = s.includeOperatorComponentDetail(ctx, comp.Components[nestIdx].GetOperatorComponent())
		}
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
				task := ""
				input := &structpb.Struct{}
				output := &structpb.Struct{}
				switch nestedComp.Component.(type) {
				case *pb.NestedComponent_ConnectorComponent:
					task = nestedComp.GetConnectorComponent().GetTask()
					input = nestedComp.GetConnectorComponent().GetDefinition().Spec.DataSpecifications[task].Input
					output = nestedComp.GetConnectorComponent().GetDefinition().Spec.DataSpecifications[task].Output

				case *pb.NestedComponent_OperatorComponent:
					task = nestedComp.GetOperatorComponent().GetTask()
					input = nestedComp.GetOperatorComponent().GetDefinition().Spec.DataSpecifications[task].Input
					output = nestedComp.GetOperatorComponent().GetDefinition().Spec.DataSpecifications[task].Output
				}
				if task == "" {
					// Skip schema generation if the task is not set.
					continue
				}
				splits := strings.Split(path, ".")

				if splits[1] == "output" {
					walk = structpb.NewStructValue(output)
				} else if splits[1] == "input" {
					walk = structpb.NewStructValue(input)
				} else {
					// Skip schema generation if the configuration is not valid.
					continue
				}
				path = path[len(splits[1])+1:]

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

	comp.DataSpecification = &pb.DataSpecification{
		Output: dataOutput,
	}

	return nil
}

func (s *service) includeDetailInRecipe(ctx context.Context, recipe *pb.Recipe) error {

	for idx := range recipe.Components {
		var err error
		switch recipe.Components[idx].Component.(type) {
		case *pb.Component_ConnectorComponent:
			err = s.includeConnectorComponentDetail(ctx, recipe.Components[idx].GetConnectorComponent())
		case *pb.Component_OperatorComponent:
			err = s.includeOperatorComponentDetail(ctx, recipe.Components[idx].GetOperatorComponent())
		case *pb.Component_IteratorComponent:
			err = s.includeIteratorComponentDetail(ctx, recipe.Components[idx].GetIteratorComponent())
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// convertPipelineToDB converts protobuf data model to db data model
func (s *service) convertPipelineToDB(ctx context.Context, ns resource.Namespace, pbPipeline *pb.Pipeline) (*datamodel.Pipeline, error) {
	logger, _ := logger.GetZapLogger(ctx)

	recipe := &datamodel.Recipe{}
	if pbPipeline.GetRecipe() != nil {
		err := s.convertResourceNameToPermalink(ctx, pbPipeline.Recipe)
		if err != nil {
			return nil, err
		}

		b, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(pbPipeline.Recipe)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(b, &recipe); err != nil {
			return nil, err
		}

	}

	dbSharing := &datamodel.Sharing{}
	if pbPipeline.GetSharing() != nil {
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
var ConnectorTypeToComponentType = map[pb.ConnectorType]pb.ComponentType{
	pb.ConnectorType_CONNECTOR_TYPE_AI:          pb.ComponentType_COMPONENT_TYPE_CONNECTOR_AI,
	pb.ConnectorType_CONNECTOR_TYPE_APPLICATION: pb.ComponentType_COMPONENT_TYPE_CONNECTOR_APPLICATION,
	pb.ConnectorType_CONNECTOR_TYPE_DATA:        pb.ComponentType_COMPONENT_TYPE_CONNECTOR_DATA,
}

// convertPipelineToPB converts db data model to protobuf data model
func (s *service) convertPipelineToPB(ctx context.Context, dbPipeline *datamodel.Pipeline, view pb.Pipeline_View, checkPermission bool) (*pb.Pipeline, error) {

	logger, _ := logger.GetZapLogger(ctx)

	ownerName, err := s.convertOwnerPermalinkToName(ctx, dbPipeline.Owner)
	if err != nil {
		return nil, err
	}

	ctxUserUID := resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey)
	var pbRecipe *pb.Recipe

	if dbPipeline.Recipe != nil && view > pb.Pipeline_VIEW_BASIC {
		pbRecipe = &pb.Recipe{}

		b, err := json.Marshal(dbPipeline.Recipe)
		if err != nil {
			return nil, err
		}

		err = protojson.Unmarshal(b, pbRecipe)
		if err != nil {
			return nil, err
		}

	}

	if view == pb.Pipeline_VIEW_FULL {
		if err := s.includeDetailInRecipe(ctx, pbRecipe); err != nil {
			return nil, err
		}
	}

	if pbRecipe != nil {
		err = s.convertResourcePermalinkToName(ctx, pbRecipe)
		if err != nil {
			return nil, err
		}
	}

	pbSharing := &pb.Sharing{}

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

	pbPipeline := pb.Pipeline{
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
		if view > pb.Pipeline_VIEW_BASIC {
			owner, err = s.fetchOwnerByPermalink(ctx, dbPipeline.Owner)
			if err != nil {
				return
			}
			pbPipeline.Owner = owner
		}
	}()
	pbPipeline.Permission = &pb.Permission{}
	go func() {
		defer wg.Done()
		if !checkPermission {
			return
		}
		if strings.Split(dbPipeline.Owner, "/")[1] == ctxUserUID {
			pbPipeline.Permission.CanEdit = true
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
		if strings.Split(dbPipeline.Owner, "/")[1] == ctxUserUID {
			pbPipeline.Permission.CanTrigger = true
			return
		}

		canTrigger, err := s.aclClient.CheckPermission(ctx, "pipeline", dbPipeline.UID, "executor")
		if err != nil {
			return
		}
		pbPipeline.Permission.CanTrigger = canTrigger
	}()

	if view > pb.Pipeline_VIEW_BASIC {
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
		if pbRecipe != nil && view == pb.Pipeline_VIEW_FULL && pbRecipe.Trigger.GetTriggerByRequest() != nil {
			spec, err := s.generatePipelineDataSpec(pbRecipe.Trigger.GetTriggerByRequest(), pbRecipe.Components)
			if err != nil {
				return
			}
			pbPipeline.DataSpecification = spec
		}
	}()

	go func() {
		defer wg.Done()
		pbReleases, err := s.convertPipelineReleasesToPB(ctx, dbPipeline, dbPipeline.Releases, view)
		if err != nil {
			return
		}
		pbPipeline.Releases = pbReleases
	}()

	wg.Wait()

	pbPipeline.Visibility = pb.Pipeline_VISIBILITY_PRIVATE
	if u, ok := pbPipeline.Sharing.Users["*/*"]; ok {
		if u.Enabled {
			pbPipeline.Visibility = pb.Pipeline_VISIBILITY_PUBLIC
		}
	}
	return &pbPipeline, nil
}

// convertPipelineToPB converts db data model to protobuf data model
func (s *service) convertPipelinesToPB(ctx context.Context, dbPipelines []*datamodel.Pipeline, view pb.Pipeline_View, checkPermission bool) ([]*pb.Pipeline, error) {

	type result struct {
		idx      int
		pipeline *pb.Pipeline
		err      error
	}

	var wg sync.WaitGroup
	wg.Add(len(dbPipelines))
	ch := make(chan result)

	for idx := range dbPipelines {
		go func(i int) {
			defer wg.Done()
			pbPipeline, err := s.convertPipelineToPB(
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

	pbPipelines := make([]*pb.Pipeline, len(dbPipelines))
	for range dbPipelines {
		r := <-ch
		if r.err != nil {
			return nil, r.err
		}
		pbPipelines[r.idx] = r.pipeline
	}
	return pbPipelines, nil
}

// convertPipelineReleaseToDB converts protobuf data model to db data model
func (s *service) convertPipelineReleaseToDB(ctx context.Context, pipelineUID uuid.UUID, pbPipelineRelease *pb.PipelineRelease) (*datamodel.PipelineRelease, error) {
	logger, _ := logger.GetZapLogger(ctx)

	recipe := &datamodel.Recipe{}
	if pbPipelineRelease.GetRecipe() != nil {
		err := s.convertResourceNameToPermalink(ctx, pbPipelineRelease.Recipe)
		if err != nil {
			return nil, err
		}

		b, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(pbPipelineRelease.Recipe)
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

// convertPipelineReleaseToPB converts db data model to protobuf data model
func (s *service) convertPipelineReleaseToPB(ctx context.Context, dbPipeline *datamodel.Pipeline, dbPipelineRelease *datamodel.PipelineRelease, view pb.Pipeline_View) (*pb.PipelineRelease, error) {

	logger, _ := logger.GetZapLogger(ctx)

	owner, err := s.convertOwnerPermalinkToName(ctx, dbPipeline.Owner)
	if err != nil {
		return nil, err
	}
	var pbRecipe *pb.Recipe
	if dbPipelineRelease.Recipe != nil && view > pb.Pipeline_VIEW_BASIC {
		pbRecipe = &pb.Recipe{}

		b, err := json.Marshal(dbPipelineRelease.Recipe)
		if err != nil {
			return nil, err
		}

		err = protojson.Unmarshal(b, pbRecipe)
		if err != nil {
			return nil, err
		}
	}

	var triggerByRequest *pb.TriggerByRequest

	if view == pb.Pipeline_VIEW_FULL {
		if err := s.includeDetailInRecipe(ctx, pbRecipe); err != nil {
			return nil, err
		}
	}

	if pbRecipe != nil {
		err = s.convertResourcePermalinkToName(ctx, pbRecipe)
		if err != nil {
			return nil, err
		}
	}
	pbPipelineRelease := pb.PipelineRelease{
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

	if view > pb.Pipeline_VIEW_BASIC {
		if dbPipelineRelease.Metadata != nil {
			str := structpb.Struct{}
			err := str.UnmarshalJSON(dbPipelineRelease.Metadata)
			if err != nil {
				logger.Error(err.Error())
			}
			pbPipelineRelease.Metadata = &str
		}
	}

	if pbRecipe != nil && view == pb.Pipeline_VIEW_FULL && triggerByRequest != nil {
		spec, err := s.generatePipelineDataSpec(triggerByRequest, pbRecipe.Components)
		if err == nil {
			pbPipelineRelease.DataSpecification = spec
		}
	}

	return &pbPipelineRelease, nil
}

// convertPipelineReleaseToPB converts db data model to protobuf data model
func (s *service) convertPipelineReleasesToPB(ctx context.Context, dbPipeline *datamodel.Pipeline, dbPipelineRelease []*datamodel.PipelineRelease, view pb.Pipeline_View) ([]*pb.PipelineRelease, error) {

	type result struct {
		idx     int
		release *pb.PipelineRelease
		err     error
	}

	var wg sync.WaitGroup
	wg.Add(len(dbPipelineRelease))
	ch := make(chan result)

	for idx := range dbPipelineRelease {
		go func(i int) {
			defer wg.Done()
			pbRelease, err := s.convertPipelineReleaseToPB(
				ctx,
				dbPipeline,
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

	pbPipelineReleases := make([]*pb.PipelineRelease, len(dbPipelineRelease))
	for range dbPipelineRelease {
		r := <-ch
		if r.err != nil {
			return nil, r.err
		}
		pbPipelineReleases[r.idx] = r.release
	}
	return pbPipelineReleases, nil
}

// TODO: refactor these codes
func (s *service) generatePipelineDataSpec(triggerByRequestOrigin *pb.TriggerByRequest, compsOrigin []*pb.Component) (*pb.DataSpecification, error) {
	success := true
	pipelineDataSpec := &pb.DataSpecification{}

	dataInput := &structpb.Struct{Fields: make(map[string]*structpb.Value)}
	dataInput.Fields["type"] = structpb.NewStringValue("object")
	dataInput.Fields["properties"] = structpb.NewStructValue(&structpb.Struct{Fields: make(map[string]*structpb.Value)})

	triggerByRequest := proto.Clone(triggerByRequestOrigin).(*pb.TriggerByRequest)
	for k, v := range triggerByRequest.GetRequestFields() {
		b, _ := protojson.Marshal(v)
		p := &structpb.Struct{}
		_ = protojson.Unmarshal(b, p)
		dataInput.Fields["properties"].GetStructValue().Fields[k] = structpb.NewStructValue(p)
	}

	// output
	dataOutput := &structpb.Struct{Fields: make(map[string]*structpb.Value)}
	dataOutput.Fields["type"] = structpb.NewStringValue("object")
	dataOutput.Fields["properties"] = structpb.NewStructValue(&structpb.Struct{Fields: make(map[string]*structpb.Value)})

	for k, v := range triggerByRequest.GetResponseFields() {
		var m *structpb.Value

		var err error

		v = proto.Clone(v).(*pb.TriggerByRequest_ResponseField)

		str := v.Value
		if strings.HasPrefix(str, "${") && strings.HasSuffix(str, "}") && strings.Count(str, "${") == 1 {
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

			if upstreamCompIdx != -1 || compID == "request" {
				var walk *structpb.Value
				if compID == "request" {
					walk = structpb.NewStructValue(dataInput)
				} else {
					comp := proto.Clone(compsOrigin[upstreamCompIdx]).(*pb.Component)

					switch comp.Component.(type) {
					case *pb.Component_IteratorComponent:
						splits := strings.Split(str, ".")
						if splits[1] == "output" {
							walk = structpb.NewStructValue(comp.GetIteratorComponent().DataSpecification.Output)
						} else {
							return nil, fmt.Errorf("generate pipeline data spec error")
						}
						str = str[len(splits[1])+1:]
					case *pb.Component_ConnectorComponent, *pb.Component_OperatorComponent:
						task := ""
						input := &structpb.Struct{}
						output := &structpb.Struct{}
						switch comp.Component.(type) {
						case *pb.Component_ConnectorComponent:
							task = comp.GetConnectorComponent().GetTask()
							if _, ok := comp.GetConnectorComponent().GetDefinition().Spec.DataSpecifications[task]; ok {
								input = comp.GetConnectorComponent().GetDefinition().Spec.DataSpecifications[task].Input
								output = comp.GetConnectorComponent().GetDefinition().Spec.DataSpecifications[task].Output
							}
						case *pb.Component_OperatorComponent:
							task = comp.GetOperatorComponent().GetTask()
							if _, ok := comp.GetOperatorComponent().GetDefinition().Spec.DataSpecifications[task]; ok {
								input = comp.GetOperatorComponent().GetDefinition().Spec.DataSpecifications[task].Input
								output = comp.GetOperatorComponent().GetDefinition().Spec.DataSpecifications[task].Output
							}
						}

						if task == "" {
							return nil, fmt.Errorf("generate pipeline data spec error")
						}

						splits := strings.Split(str, ".")

						if splits[1] == "output" {
							walk = structpb.NewStructValue(output)
						} else if splits[1] == "input" {
							walk = structpb.NewStructValue(input)
						} else {
							return nil, fmt.Errorf("generate pipeline data spec error")
						}
						str = str[len(splits[1])+1:]
					}
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
	return nil, fmt.Errorf("generate pipeline data spec error")

}

func (s *service) convertSecretToDB(ctx context.Context, ns resource.Namespace, pbSecret *pb.Secret) (*datamodel.Secret, error) {

	logger, _ := logger.GetZapLogger(ctx)

	return &datamodel.Secret{
		BaseDynamicHardDelete: datamodel.BaseDynamicHardDelete{
			UID: func() uuid.UUID {
				if pbSecret.GetUid() == "" {
					return uuid.UUID{}
				}
				id, err := uuid.FromString(pbSecret.GetUid())
				if err != nil {
					logger.Error(err.Error())
				}
				return id
			}(),

			CreateTime: func() time.Time {
				if pbSecret.GetCreateTime() != nil {
					return pbSecret.GetCreateTime().AsTime()
				}
				return time.Time{}
			}(),

			UpdateTime: func() time.Time {
				if pbSecret.GetUpdateTime() != nil {
					return pbSecret.GetUpdateTime().AsTime()
				}
				return time.Time{}
			}(),
		},
		Owner:       ns.Permalink(),
		ID:          pbSecret.GetId(),
		Value:       pbSecret.Value,
		Description: pbSecret.Description,
	}, nil
}

func (s *service) convertSecretToPB(ctx context.Context, dbSecret *datamodel.Secret) (*pb.Secret, error) {

	ownerName, err := s.convertOwnerPermalinkToName(ctx, dbSecret.Owner)
	if err != nil {
		return nil, err
	}

	return &pb.Secret{
		Name:        fmt.Sprintf("%s/secrets/%s", ownerName, dbSecret.ID),
		Uid:         dbSecret.BaseDynamicHardDelete.UID.String(),
		Id:          dbSecret.ID,
		CreateTime:  timestamppb.New(dbSecret.CreateTime),
		UpdateTime:  timestamppb.New(dbSecret.UpdateTime),
		Description: dbSecret.Description,
	}, nil

}

func (s *service) convertSecretsToPB(ctx context.Context, dbSecrets []*datamodel.Secret) ([]*pb.Secret, error) {

	var err error
	pbSecrets := make([]*pb.Secret, len(dbSecrets))
	for idx := range dbSecrets {
		pbSecrets[idx], err = s.convertSecretToPB(ctx, dbSecrets[idx])
		if err != nil {
			return nil, err
		}
	}
	return pbSecrets, nil
}
