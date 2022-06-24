package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/go-redis/redis/v9"
	"github.com/gofrs/uuid"
	"github.com/gogo/status"
	"go.einride.tech/aip/filtering"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/internal/resource"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/repository"

	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
	mgmtPB "github.com/instill-ai/protogen-go/vdp/mgmt/v1alpha"
	modelPB "github.com/instill-ai/protogen-go/vdp/model/v1alpha"
	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1alpha"
)

// Service interface
type Service interface {
	CreatePipeline(pipeline *datamodel.Pipeline) (*datamodel.Pipeline, error)
	ListPipeline(ownerRscName string, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter) ([]datamodel.Pipeline, int64, string, error)
	GetPipelineByID(id string, ownerRscName string, isBasicView bool) (*datamodel.Pipeline, error)
	GetPipelineByUID(uid uuid.UUID, ownerRscName string, isBasicView bool) (*datamodel.Pipeline, error)
	UpdatePipeline(id string, ownerRscName string, updatedPipeline *datamodel.Pipeline) (*datamodel.Pipeline, error)
	DeletePipeline(id string, ownerRscName string) error
	UpdatePipelineState(id string, ownerRscName string, state datamodel.PipelineState) (*datamodel.Pipeline, error)
	UpdatePipelineID(id string, ownerRscName string, newID string) (*datamodel.Pipeline, error)
	TriggerPipeline(req *pipelinePB.TriggerPipelineRequest, pipeline *datamodel.Pipeline) (*pipelinePB.TriggerPipelineResponse, error)
	TriggerPipelineBinaryFileUpload(fileBuf bytes.Buffer, fileLengths []uint64, pipeline *datamodel.Pipeline) (*pipelinePB.TriggerPipelineBinaryFileUploadResponse, error)
	ValidatePipeline(pipeline *datamodel.Pipeline) error
}

type service struct {
	repository             repository.Repository
	userServiceClient      mgmtPB.UserServiceClient
	connectorServiceClient connectorPB.ConnectorServiceClient
	modelServiceClient     modelPB.ModelServiceClient
	redisClient            *redis.Client
}

// NewService initiates a service instance
func NewService(r repository.Repository, mu mgmtPB.UserServiceClient, c connectorPB.ConnectorServiceClient, m modelPB.ModelServiceClient, rc *redis.Client) Service {
	return &service{
		repository:             r,
		userServiceClient:      mu,
		connectorServiceClient: c,
		modelServiceClient:     m,
		redisClient:            rc,
	}
}

func (s *service) CreatePipeline(dbPipeline *datamodel.Pipeline) (*datamodel.Pipeline, error) {

	mode, err := s.getModeByConnRscName(dbPipeline.Recipe.Source, dbPipeline.Recipe.Destination)
	if err != nil {
		return nil, err
	}

	dbPipeline.Mode = mode

	ownerRscName := dbPipeline.Owner
	ownerPermalink, err := s.ownerRscNameToPermalink(ownerRscName)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	dbPipeline.Owner = ownerPermalink

	if err := s.recipeNameToPermalink(dbPipeline.Recipe); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	if dbPipeline.Mode == datamodel.PipelineMode(pipelinePB.Pipeline_MODE_SYNC) {
		dbPipeline.State = datamodel.PipelineState(pipelinePB.Pipeline_STATE_ACTIVE)
	} else {
		// TODO: Dispatch job to Temporal for periodical connection state check
		dbPipeline.State = datamodel.PipelineState(pipelinePB.Pipeline_STATE_INACTIVE)
	}

	if err := s.repository.CreatePipeline(dbPipeline); err != nil {
		return nil, err
	}

	dbCreatedPipeline, err := s.repository.GetPipelineByID(dbPipeline.ID, ownerPermalink, false)
	if err != nil {
		return nil, err
	}

	if err := s.recipePermalinkToName(dbCreatedPipeline.Recipe); err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	dbCreatedPipeline.Owner = ownerRscName

	return dbCreatedPipeline, nil
}

func (s *service) ListPipeline(ownerRscName string, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter) ([]datamodel.Pipeline, int64, string, error) {

	ownerPermalink, err := s.ownerRscNameToPermalink(ownerRscName)
	if err != nil {
		return nil, 0, "", status.Errorf(codes.InvalidArgument, err.Error())
	}

	dbPipelines, ps, pt, err := s.repository.ListPipeline(ownerPermalink, pageSize, pageToken, isBasicView, filter)
	if err != nil {
		return nil, 0, "", err
	}

	for _, dbPipeline := range dbPipelines {
		dbPipeline.Owner = ownerRscName
	}

	if !isBasicView {
		for _, dbPipeline := range dbPipelines {
			if err := s.recipePermalinkToName(dbPipeline.Recipe); err != nil {
				return nil, 0, "", status.Errorf(codes.Internal, err.Error())
			}
		}
	}

	return dbPipelines, ps, pt, nil
}

func (s *service) GetPipelineByID(id string, ownerRscName string, isBasicView bool) (*datamodel.Pipeline, error) {

	ownerPermalink, err := s.ownerRscNameToPermalink(ownerRscName)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	dbPipeline, err := s.repository.GetPipelineByID(id, ownerPermalink, isBasicView)
	if err != nil {
		return nil, err
	}

	dbPipeline.Owner = ownerRscName

	if !isBasicView {
		if err := s.recipePermalinkToName(dbPipeline.Recipe); err != nil {
			return nil, status.Errorf(codes.Internal, err.Error())
		}
	}

	return dbPipeline, nil
}

func (s *service) GetPipelineByUID(uid uuid.UUID, ownerRscName string, isBasicView bool) (*datamodel.Pipeline, error) {

	ownerPermalink, err := s.ownerRscNameToPermalink(ownerRscName)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	dbPipeline, err := s.repository.GetPipelineByUID(uid, ownerPermalink, isBasicView)
	if err != nil {
		return nil, err
	}

	dbPipeline.Owner = ownerRscName

	if !isBasicView {
		if err := s.recipePermalinkToName(dbPipeline.Recipe); err != nil {
			return nil, status.Errorf(codes.Internal, err.Error())
		}
	}

	return dbPipeline, nil
}

func (s *service) UpdatePipeline(id string, ownerRscName string, toUpdPipeline *datamodel.Pipeline) (*datamodel.Pipeline, error) {

	ownerPermalink, err := s.ownerRscNameToPermalink(ownerRscName)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	toUpdPipeline.Owner = ownerPermalink

	// Validation: Pipeline existence
	if existingPipeline, _ := s.repository.GetPipelineByID(id, ownerPermalink, true); existingPipeline == nil {
		return nil, status.Errorf(codes.NotFound, "Pipeline id %s is not found", id)
	}

	if err := s.repository.UpdatePipeline(id, ownerPermalink, toUpdPipeline); err != nil {
		return nil, err
	}

	dbPipeline, err := s.repository.GetPipelineByID(toUpdPipeline.ID, ownerPermalink, false)
	if err != nil {
		return nil, err
	}

	if err := s.recipePermalinkToName(dbPipeline.Recipe); err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	dbPipeline.Owner = ownerRscName

	return dbPipeline, nil
}

func (s *service) DeletePipeline(id string, ownerRscName string) error {
	ownerPermalink, err := s.ownerRscNameToPermalink(ownerRscName)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, err.Error())
	}
	return s.repository.DeletePipeline(id, ownerPermalink)
}

func (s *service) UpdatePipelineState(id string, ownerRscName string, state datamodel.PipelineState) (*datamodel.Pipeline, error) {

	if state == datamodel.PipelineState(pipelinePB.Pipeline_STATE_UNSPECIFIED) {
		return nil, status.Errorf(codes.InvalidArgument, "State update with unspecified is not allowed")
	}

	ownerPermalink, err := s.ownerRscNameToPermalink(ownerRscName)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	dbPipeline, err := s.repository.GetPipelineByID(id, ownerPermalink, false)
	if err != nil {
		return nil, err
	}

	if err := s.recipePermalinkToName(dbPipeline.Recipe); err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	mode, err := s.getModeByConnRscName(dbPipeline.Recipe.Source, dbPipeline.Recipe.Destination)
	if err != nil {
		return nil, err
	}

	if mode == datamodel.PipelineMode(pipelinePB.Pipeline_MODE_SYNC) && state == datamodel.PipelineState(pipelinePB.Pipeline_STATE_INACTIVE) {
		return nil, status.Errorf(codes.InvalidArgument, "Pipeline id %s is in the sync mode, which is always active", dbPipeline.ID)
	}

	if err := s.repository.UpdatePipelineState(id, ownerPermalink, state); err != nil {
		return nil, err
	}

	dbPipeline, err = s.repository.GetPipelineByID(id, ownerPermalink, false)
	if err != nil {
		return nil, err
	}

	if err := s.recipePermalinkToName(dbPipeline.Recipe); err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	dbPipeline.Owner = ownerRscName

	return dbPipeline, nil
}

func (s *service) UpdatePipelineID(id string, ownerRscName string, newID string) (*datamodel.Pipeline, error) {

	ownerPermalink, err := s.ownerRscNameToPermalink(ownerRscName)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	// Validation: Pipeline existence
	if existingPipeline, _ := s.repository.GetPipelineByID(id, ownerPermalink, true); existingPipeline == nil {
		return nil, status.Errorf(codes.NotFound, "Pipeline id %s is not found", id)
	}

	if err := s.repository.UpdatePipelineID(id, ownerPermalink, newID); err != nil {
		return nil, err
	}

	dbPipeline, err := s.repository.GetPipelineByID(newID, ownerPermalink, false)
	if err != nil {
		return nil, err
	}

	if err := s.recipePermalinkToName(dbPipeline.Recipe); err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	dbPipeline.Owner = ownerRscName

	return dbPipeline, nil
}

func (s *service) ValidatePipeline(pipeline *datamodel.Pipeline) error {

	// Validation: Pipeline is in inactive state
	if pipeline.State == datamodel.PipelineState(pipelinePB.Pipeline_STATE_INACTIVE) {
		return status.Error(codes.FailedPrecondition, "This pipeline is inactivated")
	}

	// Validation: Pipeline is in error state
	if pipeline.State == datamodel.PipelineState(pipelinePB.Pipeline_STATE_ERROR) {
		return status.Error(codes.FailedPrecondition, "This pipeline has errors")
	}

	return nil
}

func (s *service) TriggerPipeline(req *pipelinePB.TriggerPipelineRequest, dbPipeline *datamodel.Pipeline) (*pipelinePB.TriggerPipelineResponse, error) {

	ownerPermalink, err := s.ownerRscNameToPermalink(dbPipeline.Owner)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	dbPipeline.Owner = ownerPermalink

	var inputs []*modelPB.Input
	for _, input := range req.Inputs {
		if len(input.GetImageUrl()) > 0 {
			inputs = append(inputs, &modelPB.Input{
				Type: &modelPB.Input_ImageUrl{
					ImageUrl: input.GetImageUrl(),
				},
			})
		} else if len(input.GetImageBase64()) > 0 {
			inputs = append(inputs, &modelPB.Input{
				Type: &modelPB.Input_ImageBase64{
					ImageBase64: input.GetImageBase64(),
				},
			})
		}
	}

	var outputs []*structpb.Struct
	for idx, modelInst := range dbPipeline.Recipe.ModelInstances {

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// TODO: async call model-backend
		resp, err := s.modelServiceClient.TriggerModelInstance(ctx, &modelPB.TriggerModelInstanceRequest{
			Name:   modelInst,
			Inputs: inputs,
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "[model-backend] Error %s at %dth model instance %s: %v", "TriggerModel", idx, modelInst, err.Error())
		}

		outputs = append(outputs, resp.Output)

		// Increment trigger image numbers
		uid, err := resource.GetPermalinkUID(dbPipeline.Owner)
		if err != nil {
			return nil, err
		}
		if strings.HasPrefix(dbPipeline.Owner, "users/") {
			s.redisClient.IncrBy(context.Background(), fmt.Sprintf("user:%s:trigger.image.num", uid), int64(len(inputs)))
		} else if strings.HasPrefix(dbPipeline.Owner, "orgs/") {
			s.redisClient.IncrBy(context.Background(), fmt.Sprintf("org:%s:trigger.image.num", uid), int64(len(inputs)))
		}
	}

	switch {
	// If this is a sync trigger (i.e., HTTP, gRPC source and destination connectors), return right away
	case dbPipeline.Mode == datamodel.PipelineMode(pipelinePB.Pipeline_MODE_SYNC):
		return &pipelinePB.TriggerPipelineResponse{
			Output: outputs,
		}, nil
	// If this is a async trigger, write to the destination connector
	case dbPipeline.Mode == datamodel.PipelineMode(pipelinePB.Pipeline_MODE_ASYNC):
		return nil, nil
	// The default case should never been reached
	default:
		return nil, nil
	}

}

func (s *service) TriggerPipelineBinaryFileUpload(fileBuf bytes.Buffer, fileLengths []uint64, dbPipeline *datamodel.Pipeline) (*pipelinePB.TriggerPipelineBinaryFileUploadResponse, error) {

	ownerPermalink, err := s.ownerRscNameToPermalink(dbPipeline.Owner)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	dbPipeline.Owner = ownerPermalink

	var outputs []*structpb.Struct
	for idx, modelInst := range dbPipeline.Recipe.ModelInstances {

		// TODO: async call model-backend
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		stream, err := s.modelServiceClient.TriggerModelInstanceBinaryFileUpload(ctx)
		defer func() {
			_ = stream.CloseSend()
		}()
		if err != nil {
			return nil, fmt.Errorf("[model-backend] Error %s at %dth model instance %s: cannot init stream: %v", "TriggerModelBinaryFileUpload", idx, modelInst, err.Error())
		}

		err = stream.Send(&modelPB.TriggerModelInstanceBinaryFileUploadRequest{
			Name:        modelInst,
			FileLengths: fileLengths,
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "[model-backend] Error %s at %dth model instance %s: cannot send data info to server: %v", "TriggerModelInstanceBinaryFileUploadRequest", idx, modelInst, err.Error())
		}

		const chunkSize = 64 * 1024
		buf := make([]byte, chunkSize)

		fb := bytes.Buffer{}
		fb.Write(fileBuf.Bytes())
		for {
			n, err := fb.Read(buf)
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, err
			}

			err = stream.Send(&modelPB.TriggerModelInstanceBinaryFileUploadRequest{Bytes: buf[:n]})
			if err != nil {
				return nil, status.Errorf(codes.Internal, "[model-backend] Error %s at %dth model instance %s: cannot send chunk to server: %v", "TriggerModelInstanceBinaryFileUploadRequest", idx, modelInst, err.Error())
			}
		}

		resp, err := stream.CloseAndRecv()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "[model-backend] Error %s at %dth model instance %s: cannot receive response: %v", "TriggerModelInstanceBinaryFileUploadRequest", idx, modelInst, err.Error())
		}

		outputs = append(outputs, resp.Output)

		// Increment trigger image numbers
		uid, err := resource.GetPermalinkUID(dbPipeline.Owner)
		if err != nil {
			return nil, err
		}
		if strings.HasPrefix(dbPipeline.Owner, "users/") {
			s.redisClient.IncrBy(ctx, fmt.Sprintf("user:%s:trigger.image.num", uid), int64(len(fileLengths)))
		} else if strings.HasPrefix(dbPipeline.Owner, "orgs/") {
			s.redisClient.IncrBy(ctx, fmt.Sprintf("org:%s:trigger.image.num", uid), int64(len(fileLengths)))
		}
	}

	// Check if this is a direct trigger (i.e., HTTP, gRPC source and destination connectors)
	switch {
	case dbPipeline.Mode == datamodel.PipelineMode(pipelinePB.Pipeline_MODE_SYNC):
		return &pipelinePB.TriggerPipelineBinaryFileUploadResponse{
			Output: outputs,
		}, nil
	case dbPipeline.Mode == datamodel.PipelineMode(pipelinePB.Pipeline_MODE_ASYNC):
		return nil, nil
	}

	return nil, nil
}
