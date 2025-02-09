package service

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"slices"
	"strings"
	"time"

	"github.com/gabriel-vasile/mimetype"
	"github.com/gofrs/uuid"
	"github.com/gogo/status"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"golang.org/x/image/draw"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/pkg/acl"
	"github.com/instill-ai/pipeline-backend/pkg/constant"
	"github.com/instill-ai/pipeline-backend/pkg/data/path"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/logger"
	"github.com/instill-ai/pipeline-backend/pkg/recipe"
	"github.com/instill-ai/pipeline-backend/pkg/repository"
	"github.com/instill-ai/pipeline-backend/pkg/resource"

	componentbase "github.com/instill-ai/pipeline-backend/pkg/component/base"
	componentstore "github.com/instill-ai/pipeline-backend/pkg/component/store"
	errdomain "github.com/instill-ai/pipeline-backend/pkg/errors"
	mgmtpb "github.com/instill-ai/protogen-go/core/mgmt/v1beta"
	pb "github.com/instill-ai/protogen-go/pipeline/pipeline/v1beta"
)

type Converter interface {
	ConvertPipelineToDB(ctx context.Context, ns resource.Namespace, pbPipeline *pb.Pipeline) (*datamodel.Pipeline, error)
	ConvertPipelineToPB(ctx context.Context, dbPipeline *datamodel.Pipeline, view pb.Pipeline_View, checkPermission bool, useDynamicDef bool) (*pb.Pipeline, error)
	ConvertPipelinesToPB(ctx context.Context, dbPipelines []*datamodel.Pipeline, view pb.Pipeline_View, checkPermission bool) ([]*pb.Pipeline, error)

	ConvertPipelineReleaseToDB(ctx context.Context, pipelineUID uuid.UUID, pbPipelineRelease *pb.PipelineRelease) (*datamodel.PipelineRelease, error)
	ConvertPipelineReleaseToPB(ctx context.Context, dbPipeline *datamodel.Pipeline, dbPipelineRelease *datamodel.PipelineRelease, view pb.Pipeline_View) (*pb.PipelineRelease, error)
	ConvertPipelineReleasesToPB(ctx context.Context, dbPipeline *datamodel.Pipeline, dbPipelineRelease []*datamodel.PipelineRelease, view pb.Pipeline_View) ([]*pb.PipelineRelease, error)

	ConvertSecretToDB(ctx context.Context, ns resource.Namespace, pbSecret *pb.Secret) (*datamodel.Secret, error)
	ConvertSecretToPB(ctx context.Context, dbSecret *datamodel.Secret) (*pb.Secret, error)
	ConvertSecretsToPB(ctx context.Context, dbSecrets []*datamodel.Secret) ([]*pb.Secret, error)

	IncludeDetailInRecipe(ctx context.Context, ownerPermalink string, recipe *datamodel.Recipe, useDynamicDef bool) error
	GeneratePipelineDataSpec(variables map[string]*datamodel.Variable, outputs map[string]*datamodel.Output, compsOrigin datamodel.ComponentMap) (*pb.DataSpecification, error)
}

type converter struct {
	mgmtPrivateServiceClient mgmtpb.MgmtPrivateServiceClient
	redisClient              *redis.Client
	component                *componentstore.Store
	aclClient                acl.ACLClientInterface
	repository               repository.Repository
	instillCoreHost          string
}

type ConverterConfig struct {
	MgmtClient      mgmtpb.MgmtPrivateServiceClient
	RedisClient     *redis.Client
	ACLClient       acl.ACLClientInterface
	Repository      repository.Repository
	InstillCoreHost string
	ComponentStore  *componentstore.Store
}

// NewService initiates a service instance
func NewConverter(cfg ConverterConfig) Converter {
	return &converter{
		mgmtPrivateServiceClient: cfg.MgmtClient,
		redisClient:              cfg.RedisClient,
		component:                cfg.ComponentStore,
		aclClient:                cfg.ACLClient,
		repository:               cfg.Repository,
		instillCoreHost:          cfg.InstillCoreHost,
	}
}

func (c *converter) compressProfileImage(profileImage string) (string, error) {

	// Due to the local env, we don't set the `InstillCoreHost` config, the avatar path is not working.
	// As a workaround, if the profileAvatar is not a base64 string, we ignore the avatar.
	if !strings.HasPrefix(profileImage, "data:") {
		return "", nil
	}

	profileImageStr := strings.Split(profileImage, ",")
	b, err := base64.StdEncoding.DecodeString(profileImageStr[len(profileImageStr)-1])
	if err != nil {
		return "", err
	}
	if len(b) > 200*1024 {
		mimeType := strings.Split(mimetype.Detect(b).String(), ";")[0]

		var src image.Image
		switch mimeType {
		case "image/png":
			src, _ = png.Decode(bytes.NewReader(b))
		case "image/jpeg":
			src, _ = jpeg.Decode(bytes.NewReader(b))
		default:
			return "", status.Errorf(codes.InvalidArgument, "only support profile image in jpeg and png formats")
		}

		// Set the expected size that you want:
		dst := image.NewRGBA(image.Rect(0, 0, 256, 256*src.Bounds().Max.Y/src.Bounds().Max.X))

		// Resize:
		draw.NearestNeighbor.Scale(dst, dst.Rect, src, src.Bounds(), draw.Over, nil)

		var buf bytes.Buffer
		encoder := png.Encoder{CompressionLevel: png.BestCompression}
		err = encoder.Encode(bufio.NewWriter(&buf), dst)
		if err != nil {
			return "", status.Errorf(codes.InvalidArgument, "profile image error")
		}
		profileImage = fmt.Sprintf("data:%s;base64,%s", "image/png", base64.StdEncoding.EncodeToString(buf.Bytes()))
	}
	return profileImage, nil
}

// processSetup resolves the setup references (a connection if defined as a
// string or secrets in its fields if defined as a map).
func (c *converter) processSetup(ctx context.Context, ownerPermalink string, setup any) (map[string]any, error) {
	switch v := setup.(type) {
	case nil:
		return nil, nil
	case map[string]any:
		return c.processSetupMap(ctx, ownerPermalink, v), nil
	case string:
		fetchConn := func(ctx context.Context, id string) (*datamodel.Connection, error) {
			nsUID, err := resource.GetRscPermalinkUID(ownerPermalink)
			if err != nil {
				return nil, fmt.Errorf("extracting owner UID: %w", err)
			}

			return c.repository.GetNamespaceConnectionByID(ctx, nsUID, id)
		}

		setup, err := recipe.FetchReferencedSetup(ctx, v, fetchConn)
		if err != nil {
			if !errors.Is(err, recipe.ErrInvalidConnectionReference) {
				return nil, fmt.Errorf("resolving connection reference: %w", err)
			}

			// A string setup should reference an existing connection but
			// unfinished pipeline recipes are allowed, so we ignore the errors
			// here.
			return map[string]any{}, nil
		}

		return setup, nil
	default:
		return nil, fmt.Errorf(
			"%w: setup field can only have string or map[string]any types",
			errdomain.ErrInvalidArgument,
		)
	}
}

// processSetupMap processes the setup when deifned as a map[string]any, i.e.,
// when it doesn't reference a connection.
func (c *converter) processSetupMap(ctx context.Context, ownerPermalink string, setup map[string]any) map[string]any {
	rendered := map[string]any{}
	for k, v := range setup {
		switch v := v.(type) {
		case map[string]any:
			rendered[k] = c.processSetupMap(ctx, ownerPermalink, v)
		case string:
			if !(strings.HasPrefix(v, "${"+constant.SegSecret+".") && strings.HasSuffix(v, "}")) {
				rendered[k] = v
				continue
			}

			// Remove the prefix and suffix
			secretKey := v[9 : len(v)-1]

			if secretKey == "INSTILL_SECRET" {
				rendered[k] = v
				continue
			}

			// Since we allow unfinished pipeline recipes, the secret
			// reference target might not exist. We ignore the error here.
			s, err := c.repository.GetNamespaceSecretByID(ctx, ownerPermalink, secretKey)
			if err == nil {
				rendered[k] = *s.Value
			} else {
				rendered[k] = v
			}
		default:
			rendered[k] = v
		}
	}

	return rendered
}

func (c *converter) includeComponentDetail(ctx context.Context, ownerPermalink string, comp *datamodel.Component, useDynamicDef bool) error {
	l, _ := logger.GetZapLogger(ctx)
	l = l.With(
		zap.String("owner", ownerPermalink),
		zap.String("compType", comp.Type),
	)

	if !useDynamicDef || comp.Input == nil {
		def, err := c.component.GetDefinitionByID(comp.Type, nil, nil)
		if err != nil {
			l.Error("Couldn't include component details.", zap.Error(err))
			comp.Definition = nil
			return nil
		}

		comp.Definition = &datamodel.Definition{ComponentDefinition: def}
		return nil
	}

	vars, err := recipe.GenerateSystemVariables(ctx, recipe.SystemVariables{})
	if err != nil {
		return err
	}

	setup, err := c.processSetup(ctx, ownerPermalink, comp.Setup)
	if err != nil {
		return fmt.Errorf("processing setup: %w", err)
	}

	def, err := c.component.GetDefinitionByID(comp.Type, vars, &componentbase.ComponentConfig{
		Task:  comp.Task,
		Input: comp.Input.(map[string]any),
		Setup: setup,
	})
	if err != nil {
		l.Error("Couldn't include component details.", zap.Error(err))
		comp.Definition = nil
		return nil
	}

	comp.Definition = &datamodel.Definition{ComponentDefinition: def}
	return nil
}

func (c *converter) includeIteratorComponentDetail(ctx context.Context, ownerPermalink string, comp *datamodel.Component, useDynamicDef bool) error {

	for _, itComp := range comp.Component {
		if itComp.Type != datamodel.Iterator {
			err := c.includeComponentDetail(ctx, ownerPermalink, itComp, useDynamicDef)
			if err != nil {
				return err
			}
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
				if nestedComp.Type != datamodel.Iterator {
					task = nestedComp.Task
					if _, ok := nestedComp.Definition.Spec.DataSpecifications[task]; ok {
						input = nestedComp.Definition.Spec.DataSpecifications[task].Input
						output = nestedComp.Definition.Spec.DataSpecifications[task].Output
					}
				}
				if task == "" {
					// Skip schema generation if the task is not set.
					continue
				}
				splits := strings.Split(path, ".")

				if splits[1] == constant.SegOutput {
					walk = structpb.NewStructValue(output)
				} else if splits[1] == constant.SegInput {
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
					if walk.GetStructValue() != nil && walk.GetStructValue().Fields["instillFormat"] != nil {
						if f := walk.GetStructValue().Fields["instillFormat"].GetStringValue(); f != "" {
							// Limitation: console can not support more then three levels of array.
							if strings.Count(f, "array:") < 2 {
								s.Fields["instillFormat"] = structpb.NewStringValue("array:" + f)
							}
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

func (c *converter) IncludeDetailInRecipe(ctx context.Context, ownerPermalink string, recipe *datamodel.Recipe, useDynamicDef bool) error {

	if recipe == nil {
		return nil
	}
	for _, comp := range recipe.Component {
		var err error
		if comp.Type != datamodel.Iterator {
			err = c.includeComponentDetail(ctx, ownerPermalink, comp, useDynamicDef)
		} else {
			err = c.includeIteratorComponentDetail(ctx, ownerPermalink, comp, useDynamicDef)
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

	profileImage, err := c.compressProfileImage(pbPipeline.GetProfileImage())
	if err != nil {
		return nil, err
	}

	dbSharing := &datamodel.Sharing{}
	if pbPipeline.GetSharing() != nil {
		b, err := protojson.Marshal(pbPipeline.GetSharing())
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(b, &dbSharing); err != nil {
			return nil, err
		}

	}

	return &datamodel.Pipeline{
		Owner:         ns.Permalink(),
		ID:            pbPipeline.GetId(),
		NamespaceID:   ns.NsID,
		NamespaceType: string(ns.NsType),
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
		Readme:     pbPipeline.Readme,
		RecipeYAML: pbPipeline.RawRecipe,
		Sharing:    dbSharing,
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
		SourceURL: sql.NullString{
			String: pbPipeline.GetSourceUrl(),
			Valid:  true,
		},
		DocumentationURL: sql.NullString{
			String: pbPipeline.GetDocumentationUrl(),
			Valid:  true,
		},
		License: sql.NullString{
			String: pbPipeline.GetLicense(),
			Valid:  true,
		},
		ProfileImage: sql.NullString{
			String: profileImage,
			Valid:  len(profileImage) > 0,
		},
	}, nil
}

// ConvertPipelineToPB converts db data model to protobuf data model
func (c *converter) ConvertPipelineToPB(ctx context.Context, dbPipelineOrigin *datamodel.Pipeline, view pb.Pipeline_View, checkPermission bool, useDynamicDef bool) (*pb.Pipeline, error) {

	logger, _ := logger.GetZapLogger(ctx)

	// Clone the pipeline to avoid share memory write
	dbPipelineByte, err := json.Marshal(dbPipelineOrigin)
	if err != nil {
		return nil, err
	}
	dbPipeline := &datamodel.Pipeline{}

	err = json.Unmarshal(dbPipelineByte, dbPipeline)
	if err != nil {
		return nil, err
	}

	ownerName := fmt.Sprintf("%s/%s", dbPipeline.NamespaceType, dbPipeline.NamespaceID)

	ctxUserUID := resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey)

	if view == pb.Pipeline_VIEW_FULL {
		if err := c.IncludeDetailInRecipe(ctx, dbPipeline.Owner, dbPipeline.Recipe, useDynamicDef); err != nil {
			return nil, err
		}
	}

	profileImage := fmt.Sprintf("%s/v1beta/%s/pipelines/%s/image", c.instillCoreHost, ownerName, dbPipeline.ID)

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

	tags := make([]string, len(dbPipeline.Tags))
	for i, tag := range dbPipeline.TagNames() {
		if slices.Contains(preserveTags, tag) {
			tags[i] = cases.Title(language.English).String(tag)
		} else {
			tags[i] = tag
		}
	}

	var pbRecipe *structpb.Struct
	webhooks := map[string]*pb.Endpoints_WebhookEndpoint{}
	if dbPipeline.Recipe != nil {
		b, err = json.Marshal(dbPipeline.Recipe)
		if err != nil {
			return nil, err
		}
		if dbPipeline.Recipe.On != nil {
			for w := range dbPipeline.Recipe.On {
				webhooks[w] = &pb.Endpoints_WebhookEndpoint{
					Url: fmt.Sprintf(
						"%s/v1beta/namespaces/%s/pipelines/%s/events?event=%s&code=%s",
						config.Config.Server.InstillCoreHost,
						dbPipeline.NamespaceID,
						dbPipeline.ID,
						w,
						dbPipeline.ShareCode,
					),
				}
			}
		}

		pbRecipe = &structpb.Struct{}
		err = protojson.Unmarshal(b, pbRecipe)
		if err != nil {
			return nil, err
		}
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
		RawRecipe:   dbPipeline.RecipeYAML,
		Sharing:     pbSharing,
		OwnerName:   ownerName,
		Tags:        tags,
		Stats: &pb.Pipeline_Stats{
			NumberOfRuns:   int32(dbPipeline.NumberOfRuns),
			NumberOfClones: int32(dbPipeline.NumberOfClones),
			LastRunTime:    timestamppb.New(dbPipeline.LastRunTime),
		},
		SourceUrl:        &dbPipeline.SourceURL.String,
		DocumentationUrl: &dbPipeline.DocumentationURL.String,
		License:          &dbPipeline.License.String,
		ProfileImage:     &profileImage,
		Endpoints: &pb.Endpoints{
			Webhooks: webhooks,
		},
	}

	var owner *mgmtpb.Owner
	owner, err = c.fetchOwnerByPermalink(ctx, dbPipeline.Owner)
	if err != nil {
		return nil, err
	}
	pbPipeline.Owner = owner

	pbPipeline.Permission = &pb.Permission{}
	if checkPermission {
		if dbPipeline.OwnerUID().String() == ctxUserUID {
			pbPipeline.Permission.CanEdit = true
			pbPipeline.Permission.CanRelease = true
			pbPipeline.Permission.CanTrigger = true
		} else {
			canEdit, err := c.aclClient.CheckPermission(ctx, "pipeline", dbPipeline.UID, "writer")
			if err != nil {
				return nil, err
			}
			pbPipeline.Permission.CanEdit = canEdit
			pbPipeline.Permission.CanRelease = canEdit

			canTrigger, err := c.aclClient.CheckPermission(ctx, "pipeline", dbPipeline.UID, "executor")
			if err != nil {
				return nil, err
			}
			pbPipeline.Permission.CanTrigger = canTrigger
		}

	}

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

	if pbRecipe != nil && view == pb.Pipeline_VIEW_FULL {
		spec, err := c.GeneratePipelineDataSpec(dbPipeline.Recipe.Variable, dbPipeline.Recipe.Output, dbPipeline.Recipe.Component)
		if err == nil {
			pbPipeline.DataSpecification = spec
		}
	}

	pbReleases, err := c.ConvertPipelineReleasesToPB(ctx, dbPipelineOrigin, dbPipeline.Releases, view)
	if err != nil {
		return nil, err
	}
	pbPipeline.Releases = pbReleases

	pbPipeline.Visibility = pb.Pipeline_VISIBILITY_PRIVATE
	if dbPipeline.IsPublic() {
		pbPipeline.Visibility = pb.Pipeline_VISIBILITY_PUBLIC
	}
	return &pbPipeline, nil
}

// ConvertPipelinesToPB converts db data model to protobuf data model
func (c *converter) ConvertPipelinesToPB(ctx context.Context, dbPipelines []*datamodel.Pipeline, view pb.Pipeline_View, checkPermission bool) ([]*pb.Pipeline, error) {
	pbPipelines := make([]*pb.Pipeline, len(dbPipelines))

	for idx := range dbPipelines {
		pbPipeline, err := c.ConvertPipelineToPB(
			ctx,
			dbPipelines[idx],
			view,
			checkPermission,
			false, // to reduce loading, we don't use dynamic definition when convert the list
		)
		if err != nil {
			return nil, err
		}
		pbPipelines[idx] = pbPipeline
	}

	return pbPipelines, nil
}

// ConvertPipelineReleaseToDB converts protobuf data model to db data model
func (c *converter) ConvertPipelineReleaseToDB(ctx context.Context, pipelineUID uuid.UUID, pbPipelineRelease *pb.PipelineRelease) (*datamodel.PipelineRelease, error) {
	logger, _ := logger.GetZapLogger(ctx)

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
		RecipeYAML:  pbPipelineRelease.RawRecipe,
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

	owner := fmt.Sprintf("%s/%s", dbPipeline.NamespaceType, dbPipeline.NamespaceID)

	if view == pb.Pipeline_VIEW_FULL {
		if err := c.IncludeDetailInRecipe(ctx, dbPipeline.Owner, dbPipelineRelease.Recipe, false); err != nil {
			return nil, err
		}
	}

	var pbRecipe *structpb.Struct
	webhooks := map[string]*pb.Endpoints_WebhookEndpoint{}
	if dbPipelineRelease.Recipe != nil {
		b, err := json.Marshal(dbPipelineRelease.Recipe)
		if err != nil {
			return nil, err
		}
		if dbPipelineRelease.Recipe.On != nil {
			for w := range dbPipelineRelease.Recipe.On {
				webhooks[w] = &pb.Endpoints_WebhookEndpoint{
					Url: fmt.Sprintf(
						"%s/v1beta/namespaces/%s/pipelines/%s/releases/%s/events?event=%s&code=%s",
						config.Config.Server.InstillCoreHost,
						dbPipeline.NamespaceID,
						dbPipeline.ID,
						dbPipelineRelease.ID,
						w,
						dbPipeline.ShareCode,
					),
				}
			}
		}

		pbRecipe = &structpb.Struct{}
		err = protojson.Unmarshal(b, pbRecipe)
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
		RawRecipe:   dbPipelineRelease.RecipeYAML,
		Endpoints: &pb.Endpoints{
			Webhooks: webhooks,
		},
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
		spec, err := c.GeneratePipelineDataSpec(dbPipelineRelease.Recipe.Variable, dbPipelineRelease.Recipe.Output, dbPipelineRelease.Recipe.Component)
		if err == nil {
			pbPipelineRelease.DataSpecification = spec
		}
	}

	return &pbPipelineRelease, nil
}

// ConvertPipelineReleaseToPB converts db data model to protobuf data model
func (c *converter) ConvertPipelineReleasesToPB(ctx context.Context, dbPipeline *datamodel.Pipeline, dbPipelineRelease []*datamodel.PipelineRelease, view pb.Pipeline_View) ([]*pb.PipelineRelease, error) {
	pbPipelineReleases := make([]*pb.PipelineRelease, len(dbPipelineRelease))
	for idx := range dbPipelineRelease {
		pbRelease, err := c.ConvertPipelineReleaseToPB(
			ctx,
			dbPipeline,
			dbPipelineRelease[idx],
			view,
		)
		if err != nil {
			return nil, err
		}
		pbPipelineReleases[idx] = pbRelease
	}

	return pbPipelineReleases, nil
}

var supportedFormats = []string{
	"boolean", "array:boolean",
	"boolean", "array:boolean",
	"string", "array:string",
	"integer", "array:integer",
	"number", "array:number",
	"image", "array:image",
	"audio", "array:audio",
	"video", "array:video",
	"document", "array:document",
	"file", "array:file",
}

// For fields without valid "format", we will fall back to using JSON format.
func checkFormat(format string) string {

	// We used */* to present document in the past.
	if format == "*/*" {
		return "document"
	}
	if format == "array:*/*" {
		return "array:document"
	}

	// Remove subtype, for example, image/jpeg -> image
	format, _, _ = strings.Cut(format, "/")
	if slices.Contains(supportedFormats, format) {
		return format
	}

	return "json"
}

// TODO: refactor these codes
func (c *converter) GeneratePipelineDataSpec(variables map[string]*datamodel.Variable, outputs map[string]*datamodel.Output, compsOrigin datamodel.ComponentMap) (pipelineDataSpec *pb.DataSpecification, err error) {

	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("panic: %+v", r)
			err = fmt.Errorf("generate pipeline data spec error")
		}
	}()

	success := true
	pipelineDataSpec = &pb.DataSpecification{}

	dataInput := &structpb.Struct{Fields: make(map[string]*structpb.Value)}
	dataInput.Fields["type"] = structpb.NewStringValue("object")
	dataInput.Fields["properties"] = structpb.NewStructValue(&structpb.Struct{Fields: make(map[string]*structpb.Value)})

	for k, v := range variables {
		b, _ := json.Marshal(v)
		p := &structpb.Struct{}
		_ = protojson.Unmarshal(b, p)
		if _, ok := p.Fields["instillFormat"]; ok {
			p.Fields["instillFormat"] = structpb.NewStringValue(checkFormat(p.Fields["instillFormat"].GetStringValue()))
		}
		dataInput.Fields["properties"].GetStructValue().Fields[k] = structpb.NewStructValue(p)
	}

	// output
	dataOutput := &structpb.Struct{Fields: make(map[string]*structpb.Value)}
	dataOutput.Fields["type"] = structpb.NewStringValue("object")
	dataOutput.Fields["properties"] = structpb.NewStructValue(&structpb.Struct{Fields: make(map[string]*structpb.Value)})

	for k, v := range outputs {
		var m *structpb.Value

		var err error

		str := v.Value
		if strings.HasPrefix(str, "${") && strings.HasSuffix(str, "}") && strings.Count(str, "${") == 1 {
			str = str[2:]
			str = str[:len(str)-1]
			str = strings.ReplaceAll(str, " ", "")
			p, err := path.NewPath(str)
			if err != nil {
				return nil, err
			}
			seg, remainingPath, err := p.TrimFirst()
			if err != nil {
				return nil, err
			}
			compID := seg.Key
			upstreamCompID := ""
			for id := range compsOrigin {
				if id == compID {
					upstreamCompID = id
				}
			}

			if upstreamCompID != "" || compID == constant.SegVariable {
				var walk *structpb.Value
				if compID == constant.SegVariable {
					walk = structpb.NewStructValue(dataInput)
				} else {

					seg, remainingPath, err = remainingPath.TrimFirst()
					if err != nil {
						return nil, err
					}

					comp := compsOrigin[upstreamCompID]

					switch comp.Type {
					case datamodel.Iterator:

						if seg.Key == constant.SegOutput {
							walk = structpb.NewStructValue(comp.DataSpecification.Output)
						} else {
							return nil, fmt.Errorf("generate pipeline data spec error")
						}
					default:
						input := &structpb.Struct{}
						output := &structpb.Struct{}

						task := comp.Task
						if comp.Definition != nil {
							if _, ok := comp.Definition.Spec.DataSpecifications[task]; ok {
								input = comp.Definition.Spec.DataSpecifications[task].Input
								output = comp.Definition.Spec.DataSpecifications[task].Output
							}
						}

						if task == "" {
							return nil, fmt.Errorf("generate pipeline data spec error")
						}

						if seg.Key == constant.SegOutput {
							walk = structpb.NewStructValue(output)
						} else if seg.Key == constant.SegInput {
							walk = structpb.NewStructValue(input)
						} else {
							return nil, fmt.Errorf("generate pipeline data spec error")
						}
					}
				}

				for {
					if remainingPath == nil || remainingPath.IsEmpty() {
						break
					}

					seg, remainingPath, err = remainingPath.TrimFirst()
					if err != nil {
						return nil, err
					}

					if seg.SegmentType == path.KeySegment {
						curr := seg.Key
						if _, ok := walk.GetStructValue().Fields["properties"]; ok {
							if _, ok := walk.GetStructValue().Fields["properties"].GetStructValue().Fields[curr]; !ok {
								break
							}
						} else {
							break
						}

						walk = walk.GetStructValue().Fields["properties"].GetStructValue().Fields[curr]
					} else if seg.SegmentType == path.IndexSegment {
						// insert instillFormat to items
						arrayFormat, ok := walk.GetStructValue().Fields["instillFormat"]
						walk = walk.GetStructValue().Fields["items"]
						if !ok {
							continue
						}
						// It will be like `array:image/*``
						bef, _, ok := strings.Cut(arrayFormat.GetStringValue(), "/")
						if !ok {
							continue
						}
						_, instillFormat, ok := strings.Cut(bef, ":")
						if !ok {
							continue
						}
						if walk.GetStructValue() != nil {
							walk.GetStructValue().Fields["instillFormat"] = structpb.NewStringValue(instillFormat)
						}
					} else {
						walk, _ = structpb.NewValue(map[string]interface{}{
							"title":          v.Title,
							"description":    v.Description,
							"instillFormat":  "json",
							"instillUIOrder": v.InstillUIOrder,
						})
					}

				}
				if walk.GetStructValue() != nil && walk.GetStructValue().Fields != nil {
					instillFormat := walk.GetStructValue().Fields["instillFormat"].GetStringValue()
					m, err = structpb.NewValue(map[string]interface{}{
						"title":          v.Title,
						"description":    v.Description,
						"type":           walk.GetStructValue().Fields["type"].GetStringValue(),
						"instillFormat":  checkFormat(instillFormat),
						"instillUIOrder": v.InstillUIOrder,
					})
					if err != nil {
						return nil, err
					}
				}

			} else {
				return nil, fmt.Errorf("generate data spec error")
			}

		} else {
			m, err = structpb.NewValue(map[string]interface{}{
				"title":          v.Title,
				"description":    v.Description,
				"type":           "string",
				"instillFormat":  "string",
				"instillUIOrder": v.InstillUIOrder,
			})
		}

		if m == nil || err != nil {
			success = false
		} else {
			if _, ok := m.GetStructValue().Fields["instillFormat"]; ok {
				m.GetStructValue().Fields["instillFormat"] = structpb.NewStringValue(m.GetStructValue().Fields["instillFormat"].GetStringValue())
			}
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
		Owner:         ns.Permalink(),
		NamespaceID:   ns.NsID,
		NamespaceType: string(ns.NsType),
		ID:            pbSecret.GetId(),
		Value:         pbSecret.Value,
		Description:   pbSecret.Description,
	}, nil
}

func (c *converter) ConvertSecretToPB(ctx context.Context, dbSecret *datamodel.Secret) (*pb.Secret, error) {

	ownerName := fmt.Sprintf("%s/%s", dbSecret.NamespaceType, dbSecret.NamespaceID)

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

func (c *converter) fetchOwnerByPermalink(ctx context.Context, permalink string) (*mgmtpb.Owner, error) {

	key := fmt.Sprintf("owner_profile:%s", permalink)
	if b, err := c.redisClient.Get(ctx, key).Bytes(); err == nil {
		owner := &mgmtpb.Owner{}
		if protojson.Unmarshal(b, owner) == nil {
			return owner, nil
		}
	}
	uid := strings.Split(permalink, "/")[1]

	resp, err := c.mgmtPrivateServiceClient.CheckNamespaceByUIDAdmin(ctx, &mgmtpb.CheckNamespaceByUIDAdminRequest{Uid: uid})
	if err != nil {
		return nil, fmt.Errorf("LookUpNamespaceAdmin error")
	}
	switch o := resp.Owner.(type) {
	case *mgmtpb.CheckNamespaceByUIDAdminResponse_User:
		owner := &mgmtpb.Owner{Owner: &mgmtpb.Owner_User{User: o.User}}
		if b, err := protojson.Marshal(owner); err == nil {
			c.redisClient.Set(ctx, key, b, 5*time.Minute)
		}
		return owner, nil
	case *mgmtpb.CheckNamespaceByUIDAdminResponse_Organization:
		owner := &mgmtpb.Owner{Owner: &mgmtpb.Owner_Organization{Organization: o.Organization}}
		if b, err := protojson.Marshal(owner); err == nil {
			c.redisClient.Set(ctx, key, b, 5*time.Minute)
		}
		return owner, nil
	}

	return nil, fmt.Errorf("fetchOwnerByPermalink error")
}
