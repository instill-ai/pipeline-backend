package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/gofrs/uuid"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/instill-ai/pipeline-backend/pkg/constant"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/resource"

	errdomain "github.com/instill-ai/pipeline-backend/pkg/errors"
	runpb "github.com/instill-ai/protogen-go/common/run/v1alpha"
	mgmtpb "github.com/instill-ai/protogen-go/core/mgmt/v1beta"
	pipelinepb "github.com/instill-ai/protogen-go/vdp/pipeline/v1beta"
)

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

var seededRand *rand.Rand = rand.New(
	rand.NewSource(time.Now().UnixNano()))

func randomStrWithCharset(length int, charset string) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func generateShareCode() string {
	return randomStrWithCharset(32, charset)
}

func (s *service) checkNamespacePermission(ctx context.Context, ns resource.Namespace) error {
	// TODO: optimize ACL model
	if ns.NsType == "organizations" {
		granted, err := s.aclClient.CheckPermission(ctx, "organization", ns.NsUID, "member")
		if err != nil {
			return err
		}
		if !granted {
			return errdomain.ErrUnauthorized
		}
	} else {
		if ns.NsUID != uuid.FromStringOrNil(resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey)) {
			// TODO: remove this debug print
			fmt.Println("nsuid", ns.NsUID, constant.HeaderUserUIDKey)
			return errdomain.ErrUnauthorized
		}
	}
	return nil
}

func (s *service) GetCtxUserNamespace(ctx context.Context) (resource.Namespace, error) {

	uid := uuid.FromStringOrNil(resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey))
	resp, err := s.mgmtPrivateServiceClient.CheckNamespaceByUIDAdmin(ctx, &mgmtpb.CheckNamespaceByUIDAdminRequest{
		Uid: uid.String(),
	})
	if err != nil || resp.Type != mgmtpb.CheckNamespaceByUIDAdminResponse_NAMESPACE_USER {
		return resource.Namespace{}, fmt.Errorf("namespace error")
	}
	return resource.Namespace{
		NsType: resource.NamespaceType("users"),
		NsID:   resp.Id,
		NsUID:  uid,
	}, nil
}
func (s *service) GetRscNamespace(ctx context.Context, namespaceID string) (resource.Namespace, error) {

	resp, err := s.mgmtPrivateServiceClient.CheckNamespaceAdmin(ctx, &mgmtpb.CheckNamespaceAdminRequest{
		Id: namespaceID,
	})
	if err != nil {
		return resource.Namespace{}, err
	}
	if resp.Type == mgmtpb.CheckNamespaceAdminResponse_NAMESPACE_USER {
		return resource.Namespace{
			NsType: resource.User,
			NsID:   namespaceID,
			NsUID:  uuid.FromStringOrNil(resp.Uid),
		}, nil
	} else if resp.Type == mgmtpb.CheckNamespaceAdminResponse_NAMESPACE_ORGANIZATION {
		return resource.Namespace{
			NsType: resource.Organization,
			NsID:   namespaceID,
			NsUID:  uuid.FromStringOrNil(resp.Uid),
		}, nil
	}
	return resource.Namespace{}, fmt.Errorf("namespace error")
}

// Helper methods
func (s *service) convertPipelineRunToPB(run datamodel.PipelineRun) (*pipelinepb.PipelineRun, error) {
	result := &pipelinepb.PipelineRun{
		PipelineUid:     run.PipelineUID.String(),
		PipelineRunUid:  run.PipelineTriggerUID.String(),
		PipelineVersion: run.PipelineVersion,
		Status:          runpb.RunStatus(run.Status),
		Source:          runpb.RunSource(run.Source),
		StartTime:       timestamppb.New(run.StartedTime),
		Error:           run.Error.Ptr(),
	}

	if run.TotalDuration.Valid {
		totalDuration := int32(run.TotalDuration.Int64)
		result.TotalDuration = &totalDuration
	}
	if run.CompletedTime.Valid {
		result.CompleteTime = timestamppb.New(run.CompletedTime.Time)
	}

	return result, nil
}

func (s *service) convertComponentRunToPB(run datamodel.ComponentRun) (*pipelinepb.ComponentRun, error) {
	result := &pipelinepb.ComponentRun{
		PipelineRunUid: run.PipelineTriggerUID.String(),
		ComponentId:    run.ComponentID,
		Status:         runpb.RunStatus(run.Status),
		StartTime:      timestamppb.New(run.StartedTime),
		Error:          run.Error.Ptr(),
	}

	if run.TotalDuration.Valid {
		totalDuration := int32(run.TotalDuration.Int64)
		result.TotalDuration = &totalDuration
	}
	if run.CompletedTime.Valid {
		result.CompleteTime = timestamppb.New(run.CompletedTime.Time)
	}

	for _, fileReference := range run.Inputs {
		result.InputsReference = append(result.InputsReference, &pipelinepb.FileReference{
			Name: fileReference.Name,
			Type: fileReference.Type,
			Size: fileReference.Size,
			Url:  fileReference.URL,
		})
	}
	for _, fileReference := range run.Outputs {
		result.OutputsReference = append(result.OutputsReference, &pipelinepb.FileReference{
			Name: fileReference.Name,
			Type: fileReference.Type,
			Size: fileReference.Size,
			Url:  fileReference.URL,
		})
	}
	return result, nil
}

// CanViewPrivateData - only with credit owner ns could users see their input/output data
func CanViewPrivateData(namespace, requesterUID string) bool {
	return namespace == requesterUID
}

func parseMetadataToStructArray(metadataMap map[string][]byte, log *zap.Logger, key string, metadataType string, logFields ...zap.Field) []*structpb.Struct {
	md, ok := metadataMap[key]
	if !ok {
		log.Error(fmt.Sprintf("failed to load %s metadata", metadataType), logFields...)
		return nil
	}

	structArr := make([]*structpb.Struct, 0)
	if err := json.Unmarshal(md, &structArr); err != nil {
		log.Error(fmt.Sprintf("failed to parse %s metadata", metadataType), logFields...)
		return nil
	}

	return structArr
}

func parseRecipeMetadata(metadataMap map[string][]byte, log *zap.Logger, converter Converter, key string, metadataType string, logFields ...zap.Field) (*structpb.Struct, *pipelinepb.DataSpecification) {
	md, ok := metadataMap[key]
	if !ok {
		log.Error(fmt.Sprintf("failed to load %s metadata", metadataType), logFields...)
		return nil, nil
	}

	r := make(map[string]any)
	err := json.Unmarshal(md, &r)
	if err != nil {
		log.Error(fmt.Sprintf("failed to unmarshal %s metadata to map", metadataType), logFields...)
		return nil, nil
	}

	pbStruct, err := structpb.NewStruct(r)
	if err != nil {
		log.Error(fmt.Sprintf("failed to convert %s metadata to struct", metadataType), logFields...)
		return nil, nil
	}

	dbRecipe := &datamodel.Recipe{}
	if err = json.Unmarshal(md, dbRecipe); err != nil {
		log.Error(fmt.Sprintf("failed to unmarshal %s metadata to datamodel", metadataType), logFields...)
		return nil, nil
	}
	
	if err = converter.IncludeDetailInRecipe(context.Background(), "", dbRecipe, false); err != nil {
		log.Error("IncludeDetailInRecipe failed", logFields...)
		return nil, nil
	}

	// Some recipes cannot generate a DataSpecification, so we can ignore the error.
	dataSpec, _ := converter.GeneratePipelineDataSpec(dbRecipe.Variable, dbRecipe.Output, dbRecipe.Component)
	return pbStruct, dataSpec
}
