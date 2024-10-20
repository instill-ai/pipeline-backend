package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/gofrs/uuid"
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
		PipelineId:      &run.Pipeline.ID,
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

func parseMetadataToStructArray(metadataMap map[string][]byte, key string) ([]*structpb.Struct, error) {
	md, ok := metadataMap[key]
	if !ok {
		return nil, fmt.Errorf("key doesn't exist")
	}

	structArr := make([]*structpb.Struct, 0)
	if err := json.Unmarshal(md, &structArr); err != nil {
		return nil, err
	}

	return structArr, nil
}

func parseRecipeMetadata(ctx context.Context, metadataMap map[string][]byte, converter Converter, key string) (*structpb.Struct, *pipelinepb.DataSpecification, error) {
	md, ok := metadataMap[key]
	if !ok {
		return nil, nil, fmt.Errorf("key doesn't exist")
	}

	r := make(map[string]any)
	err := json.Unmarshal(md, &r)
	if err != nil {
		return nil, nil, err
	}

	pbStruct, err := structpb.NewStruct(r)
	if err != nil {
		return nil, nil, err
	}

	dbRecipe := &datamodel.Recipe{}
	if err = json.Unmarshal(md, dbRecipe); err != nil {
		return pbStruct, nil, err
	}

	if err = converter.IncludeDetailInRecipe(ctx, "", dbRecipe, false); err != nil {
		return pbStruct, nil, err
	}

	// Some recipes cannot generate a DataSpecification, so we can ignore the error.
	dataSpec, _ := converter.GeneratePipelineDataSpec(dbRecipe.Variable, dbRecipe.Output, dbRecipe.Component)
	return pbStruct, dataSpec, nil
}
