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
	"github.com/redis/go-redis/v9"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/instill-ai/component"
	"github.com/instill-ai/pipeline-backend/pkg/acl"
	"github.com/instill-ai/pipeline-backend/pkg/constant"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/logger"
	"github.com/instill-ai/pipeline-backend/pkg/recipe"
	"github.com/instill-ai/pipeline-backend/pkg/resource"

	mgmtPB "github.com/instill-ai/protogen-go/core/mgmt/v1beta"
	pb "github.com/instill-ai/protogen-go/vdp/pipeline/v1beta"
)

type Converter interface {
	ConvertPipelineToDB(ctx context.Context, ns resource.Namespace, pbPipeline *pb.Pipeline) (*datamodel.Pipeline, error)
	ConvertPipelineToPB(ctx context.Context, dbPipeline *datamodel.Pipeline, view pb.Pipeline_View, checkPermission bool) (*pb.Pipeline, error)
	ConvertPipelinesToPB(ctx context.Context, dbPipelines []*datamodel.Pipeline, view pb.Pipeline_View, checkPermission bool) ([]*pb.Pipeline, error)

	ConvertPipelineReleaseToDB(ctx context.Context, pipelineUID uuid.UUID, pbPipelineRelease *pb.PipelineRelease) (*datamodel.PipelineRelease, error)
	ConvertPipelineReleaseToPB(ctx context.Context, dbPipeline *datamodel.Pipeline, dbPipelineRelease *datamodel.PipelineRelease, view pb.Pipeline_View) (*pb.PipelineRelease, error)
	ConvertPipelineReleasesToPB(ctx context.Context, dbPipeline *datamodel.Pipeline, dbPipelineRelease []*datamodel.PipelineRelease, view pb.Pipeline_View) ([]*pb.PipelineRelease, error)

	ConvertSecretToDB(ctx context.Context, ns resource.Namespace, pbSecret *pb.Secret) (*datamodel.Secret, error)
	ConvertSecretToPB(ctx context.Context, dbSecret *datamodel.Secret) (*pb.Secret, error)
	ConvertSecretsToPB(ctx context.Context, dbSecrets []*datamodel.Secret) ([]*pb.Secret, error)

	ConvertOwnerPermalinkToName(ctx context.Context, permalink string) (string, error)
	ConvertOwnerNameToPermalink(ctx context.Context, name string) (string, error)
}

type converter struct {
	mgmtPrivateServiceClient mgmtPB.MgmtPrivateServiceClient
	redisClient              *redis.Client
	component                *component.Store
	aclClient                *acl.ACLClient
}

// NewService initiates a service instance
func NewConverter(
	m mgmtPB.MgmtPrivateServiceClient,
	rc *redis.Client,
	acl *acl.ACLClient,
) Converter {
	logger, _ := logger.GetZapLogger(context.Background())

	return &converter{
		mgmtPrivateServiceClient: m,
		redisClient:              rc,
		component:                component.Init(logger, nil, nil),
		aclClient:                acl,
	}
}

// In the API, we expose the human-readable ID to the user. But in the database, we store it with UUID as the permanent identifier.
// The `convertResourceNameToPermalink` function converts all resources that use ID to UUID.
func (c *converter) convertResourceNameToPermalink(ctx context.Context, rsc any) error {

	switch rsc := rsc.(type) {
	case *pb.Recipe:
		for id := range rsc.Component {
			if err := c.convertResourceNameToPermalink(ctx, rsc.Component[id]); err != nil {
				return err
			}
		}
	case *pb.Component:
		return c.convertResourceNameToPermalink(ctx, rsc.Component)
	case *pb.NestedComponent:
		return c.convertResourceNameToPermalink(ctx, rsc.Component)
	case *pb.Component_IteratorComponent:
		for id := range rsc.IteratorComponent.Component {
			if err := c.convertResourceNameToPermalink(ctx, rsc.IteratorComponent.Component[id]); err != nil {
				return err
			}
		}

	case *pb.Component_ConnectorComponent:
		id, err := resource.GetRscNameID(rsc.ConnectorComponent.DefinitionName)
		if err != nil {
			return err
		}

		def, err := c.component.GetConnectorDefinitionByID(id, nil, nil)
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
		def, err := c.component.GetConnectorDefinitionByID(id, nil, nil)
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
		def, err := c.component.GetOperatorDefinitionByID(id, nil, nil)
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
		def, err := c.component.GetOperatorDefinitionByID(id, nil, nil)
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
func (c *converter) convertResourcePermalinkToName(ctx context.Context, rsc any) error {

	switch rsc := rsc.(type) {
	case *pb.Recipe:
		for id := range rsc.Component {
			if err := c.convertResourcePermalinkToName(ctx, rsc.Component[id]); err != nil {
				return err
			}
		}
	case *pb.Component:
		return c.convertResourcePermalinkToName(ctx, rsc.Component)
	case *pb.NestedComponent:
		return c.convertResourcePermalinkToName(ctx, rsc.Component)
	case *pb.Component_IteratorComponent:
		for id := range rsc.IteratorComponent.Component {
			if err := c.convertResourcePermalinkToName(ctx, rsc.IteratorComponent.Component[id]); err != nil {
				return err
			}
		}

	case *pb.Component_ConnectorComponent:
		uid, err := resource.GetRscPermalinkUID(rsc.ConnectorComponent.DefinitionName)
		if err != nil {
			return err
		}
		def, err := c.component.GetConnectorDefinitionByUID(uid, nil, nil)
		if err != nil {
			return err
		}
		rsc.ConnectorComponent.DefinitionName = def.Name

	case *pb.NestedComponent_ConnectorComponent:
		uid, err := resource.GetRscPermalinkUID(rsc.ConnectorComponent.DefinitionName)
		if err != nil {
			return err
		}
		def, err := c.component.GetConnectorDefinitionByUID(uid, nil, nil)
		if err != nil {
			return err
		}
		rsc.ConnectorComponent.DefinitionName = def.Name

	case *pb.Component_OperatorComponent:
		uid, err := resource.GetRscPermalinkUID(rsc.OperatorComponent.DefinitionName)
		if err != nil {
			return err
		}
		def, err := c.component.GetOperatorDefinitionByUID(uid, nil, nil)
		if err != nil {
			return err
		}
		rsc.OperatorComponent.DefinitionName = def.Name

	case *pb.NestedComponent_OperatorComponent:
		uid, err := resource.GetRscPermalinkUID(rsc.OperatorComponent.DefinitionName)
		if err != nil {
			return err
		}
		def, err := c.component.GetOperatorDefinitionByUID(uid, nil, nil)
		if err != nil {
			return err
		}
		rsc.OperatorComponent.DefinitionName = def.Name
	}
	return nil
}

func (c *converter) includeOperatorComponentDetail(ctx context.Context, comp *pb.OperatorComponent) error {
	uid, err := resource.GetRscPermalinkUID(comp.DefinitionName)
	if err != nil {
		return err
	}
	vars, err := recipe.GenerateSystemVariables(ctx, recipe.SystemVariables{})
	if err != nil {
		return err
	}
	def, err := c.component.GetOperatorDefinitionByUID(uid, vars, comp)
	if err != nil {
		return err
	}

	comp.Definition = def
	return nil
}

func (c *converter) includeConnectorComponentDetail(ctx context.Context, comp *pb.ConnectorComponent) error {
	uid, err := resource.GetRscPermalinkUID(comp.DefinitionName)
	if err != nil {
		return err
	}
	vars, err := recipe.GenerateSystemVariables(ctx, recipe.SystemVariables{})
	if err != nil {
		return err
	}
	def, err := c.component.GetConnectorDefinitionByUID(uid, vars, comp)
	if err != nil {
		return err
	}

	comp.Definition = def
	return nil
}

func (c *converter) includeIteratorComponentDetail(ctx context.Context, comp *pb.IteratorComponent) error {

	for nestID := range comp.Component {
		var err error
		switch comp.Component[nestID].Component.(type) {
		case *pb.NestedComponent_ConnectorComponent:
			err = c.includeConnectorComponentDetail(ctx, comp.Component[nestID].GetConnectorComponent())
		case *pb.NestedComponent_OperatorComponent:
			err = c.includeOperatorComponentDetail(ctx, comp.Component[nestID].GetOperatorComponent())
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
			upstreamCompID := ""
			for id := range comp.Component {
				if id == compID {
					upstreamCompID = id
				}
			}
			if upstreamCompID != "" {
				nestedComp := comp.Component[upstreamCompID]

				var walk *structpb.Value
				task := ""
				input := &structpb.Struct{}
				output := &structpb.Struct{}
				switch nestedComp.Component.(type) {
				case *pb.NestedComponent_ConnectorComponent:
					task = nestedComp.GetConnectorComponent().GetTask()
					if _, ok := nestedComp.GetConnectorComponent().GetDefinition().Spec.DataSpecifications[task]; ok {
						input = nestedComp.GetConnectorComponent().GetDefinition().Spec.DataSpecifications[task].Input
						output = nestedComp.GetConnectorComponent().GetDefinition().Spec.DataSpecifications[task].Output
					}

				case *pb.NestedComponent_OperatorComponent:
					task = nestedComp.GetOperatorComponent().GetTask()
					if _, ok := nestedComp.GetOperatorComponent().GetDefinition().Spec.DataSpecifications[task]; ok {
						input = nestedComp.GetOperatorComponent().GetDefinition().Spec.DataSpecifications[task].Input
						output = nestedComp.GetOperatorComponent().GetDefinition().Spec.DataSpecifications[task].Output
					}
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

func (c *converter) includeDetailInRecipe(ctx context.Context, recipe *pb.Recipe) error {

	for id := range recipe.Component {
		var err error
		switch recipe.Component[id].Component.(type) {
		case *pb.Component_ConnectorComponent:
			err = c.includeConnectorComponentDetail(ctx, recipe.Component[id].GetConnectorComponent())
		case *pb.Component_OperatorComponent:
			err = c.includeOperatorComponentDetail(ctx, recipe.Component[id].GetOperatorComponent())
		case *pb.Component_IteratorComponent:
			err = c.includeIteratorComponentDetail(ctx, recipe.Component[id].GetIteratorComponent())
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// ConvertPipelineToDB converts protobuf data model to db data model
func (c *converter) ConvertPipelineToDB(ctx context.Context, ns resource.Namespace, pbPipeline *pb.Pipeline) (*datamodel.Pipeline, error) {
	logger, _ := logger.GetZapLogger(ctx)

	recipe := &datamodel.Recipe{}
	if pbPipeline.GetRecipe() != nil {
		err := c.convertResourceNameToPermalink(ctx, pbPipeline.Recipe)
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

// ConvertPipelineToPB converts db data model to protobuf data model
func (c *converter) ConvertPipelineToPB(ctx context.Context, dbPipeline *datamodel.Pipeline, view pb.Pipeline_View, checkPermission bool) (*pb.Pipeline, error) {

	logger, _ := logger.GetZapLogger(ctx)

	ownerName, err := c.ConvertOwnerPermalinkToName(ctx, dbPipeline.Owner)
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
		if err := c.includeDetailInRecipe(ctx, pbRecipe); err != nil {
			return nil, err
		}
	}

	if pbRecipe != nil {
		err = c.convertResourcePermalinkToName(ctx, pbRecipe)
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

	tags := []string{}
	for _, t := range dbPipeline.Tags {
		tags = append(tags, t.TagName)
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
		Tags:        tags,
	}

	var wg sync.WaitGroup
	wg.Add(5)
	go func() {
		defer wg.Done()
		var owner *mgmtPB.Owner
		if view > pb.Pipeline_VIEW_BASIC {
			owner, err = c.fetchOwnerByPermalink(ctx, dbPipeline.Owner)
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

		canEdit, err := c.aclClient.CheckPermission(ctx, "pipeline", dbPipeline.UID, "writer")
		if err != nil {
			return
		}
		pbPipeline.Permission.CanEdit = canEdit
		pbPipeline.Permission.CanRelease = canEdit
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

		canTrigger, err := c.aclClient.CheckPermission(ctx, "pipeline", dbPipeline.UID, "executor")
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
		if pbRecipe != nil && view == pb.Pipeline_VIEW_FULL && pbRecipe.Variable != nil {
			spec, err := c.generatePipelineDataSpec(pbRecipe.Variable, pbRecipe.Output, pbRecipe.Component)
			if err != nil {
				return
			}
			pbPipeline.DataSpecification = spec
		}
	}()

	go func() {
		defer wg.Done()
		pbReleases, err := c.ConvertPipelineReleasesToPB(ctx, dbPipeline, dbPipeline.Releases, view)
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

// ConvertPipelinesToPB converts db data model to protobuf data model
func (c *converter) ConvertPipelinesToPB(ctx context.Context, dbPipelines []*datamodel.Pipeline, view pb.Pipeline_View, checkPermission bool) ([]*pb.Pipeline, error) {

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
			pbPipeline, err := c.ConvertPipelineToPB(
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

// ConvertPipelineReleaseToDB converts protobuf data model to db data model
func (c *converter) ConvertPipelineReleaseToDB(ctx context.Context, pipelineUID uuid.UUID, pbPipelineRelease *pb.PipelineRelease) (*datamodel.PipelineRelease, error) {
	logger, _ := logger.GetZapLogger(ctx)

	recipe := &datamodel.Recipe{}
	if pbPipelineRelease.GetRecipe() != nil {
		err := c.convertResourceNameToPermalink(ctx, pbPipelineRelease.Recipe)
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

// ConvertPipelineReleaseToPB converts db data model to protobuf data model
func (c *converter) ConvertPipelineReleaseToPB(ctx context.Context, dbPipeline *datamodel.Pipeline, dbPipelineRelease *datamodel.PipelineRelease, view pb.Pipeline_View) (*pb.PipelineRelease, error) {

	logger, _ := logger.GetZapLogger(ctx)

	owner, err := c.ConvertOwnerPermalinkToName(ctx, dbPipeline.Owner)
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

	if view == pb.Pipeline_VIEW_FULL {
		if err := c.includeDetailInRecipe(ctx, pbRecipe); err != nil {
			return nil, err
		}
	}

	if pbRecipe != nil {
		err = c.convertResourcePermalinkToName(ctx, pbRecipe)
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

	if pbRecipe != nil && view == pb.Pipeline_VIEW_FULL {
		spec, err := c.generatePipelineDataSpec(pbRecipe.Variable, pbRecipe.Output, pbRecipe.Component)
		if err == nil {
			pbPipelineRelease.DataSpecification = spec
		}
	}

	return &pbPipelineRelease, nil
}

// ConvertPipelineReleaseToPB converts db data model to protobuf data model
func (c *converter) ConvertPipelineReleasesToPB(ctx context.Context, dbPipeline *datamodel.Pipeline, dbPipelineRelease []*datamodel.PipelineRelease, view pb.Pipeline_View) ([]*pb.PipelineRelease, error) {

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
			pbRelease, err := c.ConvertPipelineReleaseToPB(
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
func (c *converter) generatePipelineDataSpec(variables map[string]*pb.Variable, outputs map[string]*pb.Output, compsOrigin map[string]*pb.Component) (*pb.DataSpecification, error) {
	success := true
	pipelineDataSpec := &pb.DataSpecification{}

	dataInput := &structpb.Struct{Fields: make(map[string]*structpb.Value)}
	dataInput.Fields["type"] = structpb.NewStringValue("object")
	dataInput.Fields["properties"] = structpb.NewStructValue(&structpb.Struct{Fields: make(map[string]*structpb.Value)})

	for k, v := range variables {
		b, _ := protojson.Marshal(v)
		p := &structpb.Struct{}
		_ = protojson.Unmarshal(b, p)
		dataInput.Fields["properties"].GetStructValue().Fields[k] = structpb.NewStructValue(p)
	}

	// output
	dataOutput := &structpb.Struct{Fields: make(map[string]*structpb.Value)}
	dataOutput.Fields["type"] = structpb.NewStringValue("object")
	dataOutput.Fields["properties"] = structpb.NewStructValue(&structpb.Struct{Fields: make(map[string]*structpb.Value)})

	for k, v := range outputs {
		var m *structpb.Value

		var err error

		v = proto.Clone(v).(*pb.Output)

		str := v.Value
		if strings.HasPrefix(str, "${") && strings.HasSuffix(str, "}") && strings.Count(str, "${") == 1 {
			str = str[2:]
			str = str[:len(str)-1]
			str = strings.ReplaceAll(str, " ", "")

			compID := strings.Split(str, ".")[0]
			str = str[len(strings.Split(str, ".")[0]):]
			upstreamCompID := ""
			for id := range compsOrigin {
				if id == compID {
					upstreamCompID = id
				}
			}

			if upstreamCompID != "" || compID == recipe.SegVariable {
				var walk *structpb.Value
				if compID == recipe.SegVariable {
					walk = structpb.NewStructValue(dataInput)
				} else {
					comp := proto.Clone(compsOrigin[upstreamCompID]).(*pb.Component)

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

func (c *converter) ConvertSecretToDB(ctx context.Context, ns resource.Namespace, pbSecret *pb.Secret) (*datamodel.Secret, error) {

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

func (c *converter) ConvertSecretToPB(ctx context.Context, dbSecret *datamodel.Secret) (*pb.Secret, error) {

	ownerName, err := c.ConvertOwnerPermalinkToName(ctx, dbSecret.Owner)
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

func (c *converter) ConvertSecretsToPB(ctx context.Context, dbSecrets []*datamodel.Secret) ([]*pb.Secret, error) {

	var err error
	pbSecrets := make([]*pb.Secret, len(dbSecrets))
	for idx := range dbSecrets {
		pbSecrets[idx], err = c.ConvertSecretToPB(ctx, dbSecrets[idx])
		if err != nil {
			return nil, err
		}
	}
	return pbSecrets, nil
}

// Note: Currently, we don't allow changing the owner ID. We are safe to use a cache with a longer TTL for this function.
func (c *converter) ConvertOwnerPermalinkToName(ctx context.Context, permalink string) (string, error) {

	splits := strings.Split(permalink, "/")
	nsType := splits[0]
	uid := splits[1]
	key := fmt.Sprintf("user:%s:uid_to_id", uid)
	if id, err := c.redisClient.Get(ctx, key).Result(); err != redis.Nil {
		return fmt.Sprintf("%s/%s", nsType, id), nil
	}

	if nsType == "users" {
		userResp, err := c.mgmtPrivateServiceClient.LookUpUserAdmin(ctx, &mgmtPB.LookUpUserAdminRequest{Permalink: permalink})
		if err != nil {
			return "", fmt.Errorf("ConvertNamespaceToOwnerPath error")
		}
		c.redisClient.Set(ctx, key, userResp.User.Id, 24*time.Hour)
		return fmt.Sprintf("users/%s", userResp.User.Id), nil
	} else {
		orgResp, err := c.mgmtPrivateServiceClient.LookUpOrganizationAdmin(ctx, &mgmtPB.LookUpOrganizationAdminRequest{Permalink: permalink})
		if err != nil {
			return "", fmt.Errorf("ConvertNamespaceToOwnerPath error")
		}
		c.redisClient.Set(ctx, key, orgResp.Organization.Id, 24*time.Hour)
		return fmt.Sprintf("organizations/%s", orgResp.Organization.Id), nil
	}
}

func (c *converter) fetchOwnerByPermalink(ctx context.Context, permalink string) (*mgmtPB.Owner, error) {

	key := fmt.Sprintf("owner_profile:%s", permalink)
	if b, err := c.redisClient.Get(ctx, key).Bytes(); err == nil {
		owner := &mgmtPB.Owner{}
		if protojson.Unmarshal(b, owner) == nil {
			return owner, nil
		}
	}

	if strings.HasPrefix(permalink, "users") {
		resp, err := c.mgmtPrivateServiceClient.LookUpUserAdmin(ctx, &mgmtPB.LookUpUserAdminRequest{Permalink: permalink})
		if err != nil {
			return nil, fmt.Errorf("fetchOwnerByPermalink error")
		}
		owner := &mgmtPB.Owner{Owner: &mgmtPB.Owner_User{User: resp.User}}
		if b, err := protojson.Marshal(owner); err == nil {
			c.redisClient.Set(ctx, key, b, 5*time.Minute)
		}
		return owner, nil
	} else {
		resp, err := c.mgmtPrivateServiceClient.LookUpOrganizationAdmin(ctx, &mgmtPB.LookUpOrganizationAdminRequest{Permalink: permalink})
		if err != nil {
			return nil, fmt.Errorf("fetchOwnerByPermalink error")
		}
		owner := &mgmtPB.Owner{Owner: &mgmtPB.Owner_Organization{Organization: resp.Organization}}
		if b, err := protojson.Marshal(owner); err == nil {
			c.redisClient.Set(ctx, key, b, 5*time.Minute)
		}
		return owner, nil

	}
}

// Note: Currently, we don't allow changing the owner ID. We are safe to use a cache with a longer TTL for this function.
func (c *converter) ConvertOwnerNameToPermalink(ctx context.Context, name string) (string, error) {

	splits := strings.Split(name, "/")
	nsType := splits[0]
	id := splits[1]
	key := fmt.Sprintf("user:%s:id_to_uid", id)
	if uid, err := c.redisClient.Get(ctx, key).Result(); err != redis.Nil {
		return fmt.Sprintf("%s/%s", nsType, uid), nil
	}

	if nsType == "users" {
		userResp, err := c.mgmtPrivateServiceClient.GetUserAdmin(ctx, &mgmtPB.GetUserAdminRequest{Name: name})
		if err != nil {
			return "", fmt.Errorf("convertOwnerNameToPermalink error %w", err)
		}
		c.redisClient.Set(ctx, key, *userResp.User.Uid, 24*time.Hour)
		return fmt.Sprintf("users/%s", *userResp.User.Uid), nil
	} else {
		orgResp, err := c.mgmtPrivateServiceClient.GetOrganizationAdmin(ctx, &mgmtPB.GetOrganizationAdminRequest{Name: name})
		if err != nil {
			return "", fmt.Errorf("convertOwnerNameToPermalink error %w", err)
		}
		c.redisClient.Set(ctx, key, orgResp.Organization.Uid, 24*time.Hour)
		return fmt.Sprintf("organizations/%s", orgResp.Organization.Uid), nil
	}
}
