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

	mode, err := s.checkMode(dbPipeline.Recipe)
	if err != nil {
		return nil, err
	}

	dbPipeline.Mode = mode

	if dbPipeline.Mode == datamodel.PipelineMode(pipelinePB.Pipeline_MODE_SYNC) {
		dbPipeline.State = datamodel.PipelineState(pipelinePB.Pipeline_STATE_ACTIVE)
	} else {
		// TODO: Dispatch job to Temporal for periodical connection state check
		dbPipeline.State, err = s.checkState(dbPipeline.Recipe)
		if err != nil {
			return nil, err
		}
	}

	ownerRscName := dbPipeline.Owner
	ownerPermalink, err := s.ownerRscNameToPermalink(ownerRscName)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	dbPipeline.Owner = ownerPermalink

	recipeRscName := dbPipeline.Recipe
	recipePermalink, err := s.recipeNameToPermalink(dbPipeline.Recipe)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	dbPipeline.Recipe = recipePermalink

	if err := s.repository.CreatePipeline(dbPipeline); err != nil {
		return nil, err
	}

	dbCreatedPipeline, err := s.repository.GetPipelineByID(dbPipeline.ID, ownerPermalink, false)
	if err != nil {
		return nil, err
	}

	dbCreatedPipeline.Owner = ownerRscName
	dbCreatedPipeline.Recipe = recipeRscName

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
		for idx, dbPipeline := range dbPipelines {
			recipeRscName, err := s.recipePermalinkToName(dbPipeline.Recipe)
			if err != nil {
				return nil, 0, "", status.Errorf(codes.Internal, err.Error())
			}
			dbPipelines[idx].Recipe = recipeRscName
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
		recipeRscName, err := s.recipePermalinkToName(dbPipeline.Recipe)
		if err != nil {
			return nil, status.Errorf(codes.Internal, err.Error())
		}
		dbPipeline.Recipe = recipeRscName
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
		recipeRscName, err := s.recipePermalinkToName(dbPipeline.Recipe)
		if err != nil {
			return nil, status.Errorf(codes.Internal, err.Error())
		}
		dbPipeline.Recipe = recipeRscName
	}

	return dbPipeline, nil
}

func (s *service) UpdatePipeline(id string, ownerRscName string, toUpdPipeline *datamodel.Pipeline) (*datamodel.Pipeline, error) {

	if toUpdPipeline.Recipe != nil {
		mode, err := s.checkMode(toUpdPipeline.Recipe)
		if err != nil {
			return nil, err
		}

		toUpdPipeline.Mode = mode

		if toUpdPipeline.Mode == datamodel.PipelineMode(pipelinePB.Pipeline_MODE_SYNC) {
			toUpdPipeline.State = datamodel.PipelineState(pipelinePB.Pipeline_STATE_ACTIVE)
		} else {
			toUpdPipeline.State, err = s.checkState(toUpdPipeline.Recipe)
			if err != nil {
				return nil, err
			}
		}

		recipePermalink, err := s.recipeNameToPermalink(toUpdPipeline.Recipe)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, err.Error())
		}

		toUpdPipeline.Recipe = recipePermalink
	}

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
	dbPipeline.Owner = ownerRscName

	recipeRscName, err := s.recipePermalinkToName(dbPipeline.Recipe)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	dbPipeline.Recipe = recipeRscName

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

	recipeRscName, err := s.recipePermalinkToName(dbPipeline.Recipe)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	mode, err := s.checkMode(recipeRscName)
	if err != nil {
		return nil, err
	}

	if mode == datamodel.PipelineMode(pipelinePB.Pipeline_MODE_SYNC) && state == datamodel.PipelineState(pipelinePB.Pipeline_STATE_INACTIVE) {
		return nil, status.Errorf(codes.InvalidArgument, "Pipeline id %s is in the sync mode, which is always active", dbPipeline.ID)
	}

	if state == datamodel.PipelineState(pipelinePB.Pipeline_STATE_ACTIVE) {
		state, err = s.checkState(recipeRscName)
		if err != nil {
			return nil, err
		}
	}

	if err := s.repository.UpdatePipelineState(id, ownerPermalink, state); err != nil {
		return nil, err
	}

	dbPipeline, err = s.repository.GetPipelineByID(id, ownerPermalink, false)
	if err != nil {
		return nil, err
	}

	dbPipeline.Owner = ownerRscName
	dbPipeline.Recipe = recipeRscName

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

	recipeRscName, err := s.recipePermalinkToName(dbPipeline.Recipe)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	dbPipeline.Owner = ownerRscName
	dbPipeline.Recipe = recipeRscName

	return dbPipeline, nil
}

func (s *service) TriggerPipeline(req *pipelinePB.TriggerPipelineRequest, dbPipeline *datamodel.Pipeline) (*pipelinePB.TriggerPipelineResponse, error) {

	if dbPipeline.State != datamodel.PipelineState(pipelinePB.Pipeline_STATE_ACTIVE) {
		return nil, status.Error(codes.FailedPrecondition, fmt.Sprintf("The pipeline %s is not active", dbPipeline.ID))
	}

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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var outputs []*structpb.Struct
	for idx, modelInstRscName := range dbPipeline.Recipe.ModelInstances {

		// TODO: async call model-backend
		resp, err := s.modelServiceClient.TriggerModelInstance(ctx, &modelPB.TriggerModelInstanceRequest{
			Name:   modelInstRscName,
			Inputs: inputs,
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "[model-backend] Error %s at %dth model instance %s: %v", "TriggerModel", idx, modelInstRscName, err.Error())
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

		for idx, modelInstRecName := range dbPipeline.Recipe.ModelInstances {

			modelInstResp, err := s.modelServiceClient.GetModelInstance(ctx, &modelPB.GetModelInstanceRequest{
				Name: modelInstRecName,
			})
			if err != nil {
				return nil, status.Errorf(codes.Internal, "[model-backend] Error %s at %dth model instance %s: %v", "GetModelInstance", idx, modelInstRecName, err.Error())
			}

			_, err = s.connectorServiceClient.WriteDestinationConnector(ctx, &connectorPB.WriteDestinationConnectorRequest{
				Name: dbPipeline.Recipe.Destination,
				Task: modelInstResp.Instance.GetTask(),
				Data: outputs[idx],
			})
			if err != nil {
				return nil, status.Errorf(codes.Internal, "[connector-backend] Error %s at %dth model instance %s: %v", "WriteDestinationConnector", idx, modelInstRecName, err.Error())
			}
		}
		return &pipelinePB.TriggerPipelineResponse{}, nil
	default:
		return &pipelinePB.TriggerPipelineResponse{}, nil
	}

}

func (s *service) TriggerPipelineBinaryFileUpload(fileBuf bytes.Buffer, fileLengths []uint64, dbPipeline *datamodel.Pipeline) (*pipelinePB.TriggerPipelineBinaryFileUploadResponse, error) {

	if dbPipeline.State != datamodel.PipelineState(pipelinePB.Pipeline_STATE_ACTIVE) {
		return nil, status.Error(codes.FailedPrecondition, fmt.Sprintf("The pipeline %s is not active", dbPipeline.ID))
	}

	ownerPermalink, err := s.ownerRscNameToPermalink(dbPipeline.Owner)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	dbPipeline.Owner = ownerPermalink

	var outputs []*structpb.Struct
	for idx, modelInst := range dbPipeline.Recipe.ModelInstances {

		// TODO: async call model-backend
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		stream, err := s.modelServiceClient.TriggerModelInstanceBinaryFileUpload(ctx)
		defer func() {
			_ = stream.CloseSend()
		}()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "[model-backend] Error %s at %dth model instance %s: cannot init stream: %v", "TriggerModelBinaryFileUpload", idx, modelInst, err.Error())
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
