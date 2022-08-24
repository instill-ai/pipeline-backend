package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis/v9"
	"github.com/gofrs/uuid"
	"github.com/gogo/status"
	"github.com/oklog/ulid/v2"
	"go.einride.tech/aip/filtering"
	"google.golang.org/grpc/codes"

	"github.com/instill-ai/pipeline-backend/internal/logger"
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
	TriggerPipelineBinaryFileUpload(fileBuf bytes.Buffer, fileNames []string, fileLengths []uint64, pipeline *datamodel.Pipeline) (*pipelinePB.TriggerPipelineBinaryFileUploadResponse, error)
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
		return nil, status.Errorf(codes.InvalidArgument, "Pipeline %s is in the SYNC mode, which is always active", dbPipeline.ID)
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

	logger, _ := logger.GetZapLogger()

	if dbPipeline.State != datamodel.PipelineState(pipelinePB.Pipeline_STATE_ACTIVE) {
		return nil, status.Error(codes.FailedPrecondition, fmt.Sprintf("The pipeline %s is not active", dbPipeline.ID))
	}

	ownerPermalink, err := s.ownerRscNameToPermalink(dbPipeline.Owner)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	dbPipeline.Owner = ownerPermalink

	var dataMappingIndices []string
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
		dataMappingIndices = append(dataMappingIndices, ulid.Make().String())
	}

	wg := sync.WaitGroup{}
	wg.Add(1)

	var modelInstOutputs []*pipelinePB.ModelInstanceOutput
	go func() {
		for idx, modelInstance := range dbPipeline.Recipe.ModelInstances {

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			resp, err := s.modelServiceClient.TriggerModelInstance(ctx, &modelPB.TriggerModelInstanceRequest{
				Name:   modelInstance,
				Inputs: inputs,
			})
			if err != nil {
				logger.Error(fmt.Sprintf("[model-backend] Error %s at %dth model instance %s: %v", "TriggerModel", idx, modelInstance, err.Error()))
			}

			batchOutputs := cvtModelBatchOutputToPipelineBatchOutput(resp.BatchOutputs)
			for idx, batchOutput := range batchOutputs {
				batchOutput.Index = dataMappingIndices[idx]
			}

			modelInstOutputs = append(modelInstOutputs, &pipelinePB.ModelInstanceOutput{
				ModelInstance: modelInstance,
				Task:          resp.Task,
				BatchOutputs:  batchOutputs,
			})

			// Increment trigger image numbers
			uid, err := resource.GetPermalinkUID(dbPipeline.Owner)
			if err != nil {
				logger.Error(err.Error())
			}
			if strings.HasPrefix(dbPipeline.Owner, "users/") {
				s.redisClient.IncrBy(context.Background(), fmt.Sprintf("user:%s:trigger.image.num", uid), int64(len(inputs)))
			} else if strings.HasPrefix(dbPipeline.Owner, "orgs/") {
				s.redisClient.IncrBy(context.Background(), fmt.Sprintf("org:%s:trigger.image.num", uid), int64(len(inputs)))
			}
		}
		wg.Done()
	}()

	switch {
	// If this is a SYNC trigger (i.e., HTTP, gRPC source and destination connectors), return right away
	case dbPipeline.Mode == datamodel.PipelineMode(pipelinePB.Pipeline_MODE_SYNC):
		wg.Wait()
		return &pipelinePB.TriggerPipelineResponse{
			DataMappingIndices:   dataMappingIndices,
			ModelInstanceOutputs: modelInstOutputs,
		}, nil
	// If this is a async trigger, write to the destination connector
	case dbPipeline.Mode == datamodel.PipelineMode(pipelinePB.Pipeline_MODE_ASYNC):
		go func() {
			wg.Wait()
			for idx, modelInstRecName := range dbPipeline.Recipe.ModelInstances {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_, err = s.connectorServiceClient.WriteDestinationConnector(ctx, &connectorPB.WriteDestinationConnectorRequest{
					Name:                 dbPipeline.Recipe.Destination,
					SyncMode:             connectorPB.SupportedSyncModes_SUPPORTED_SYNC_MODES_FULL_REFRESH,
					DestinationSyncMode:  connectorPB.SupportedDestinationSyncModes_SUPPORTED_DESTINATION_SYNC_MODES_APPEND,
					Pipeline:             fmt.Sprintf("pipelines/%s", dbPipeline.ID),
					DataMappingIndices:   dataMappingIndices,
					ModelInstanceOutputs: modelInstOutputs,
					Recipe: func() *pipelinePB.Recipe {
						if dbPipeline.Recipe != nil {
							b, err := json.Marshal(dbPipeline.Recipe)
							if err != nil {
								logger.Error(err.Error())
							}
							pbRecipe := pipelinePB.Recipe{}
							err = json.Unmarshal(b, &pbRecipe)
							if err != nil {
								logger.Error(err.Error())
							}
							return &pbRecipe
						}
						return nil
					}(),
				})
				if err != nil {
					logger.Error(fmt.Sprintf("[connector-backend] Error %s at %dth model instance %s: %v", "WriteDestinationConnector", idx, modelInstRecName, err.Error()))
				}
			}
		}()
		return &pipelinePB.TriggerPipelineResponse{
			DataMappingIndices:   dataMappingIndices,
			ModelInstanceOutputs: nil,
		}, nil
	}

	return nil, status.Errorf(codes.Internal, "something went very wrong - unable to trigger the pipeline")

}

func (s *service) TriggerPipelineBinaryFileUpload(fileBuf bytes.Buffer, fileNames []string, fileLengths []uint64, dbPipeline *datamodel.Pipeline) (*pipelinePB.TriggerPipelineBinaryFileUploadResponse, error) {

	if dbPipeline.State != datamodel.PipelineState(pipelinePB.Pipeline_STATE_ACTIVE) {
		return nil, status.Error(codes.FailedPrecondition, fmt.Sprintf("The pipeline %s is not active", dbPipeline.ID))
	}

	ownerPermalink, err := s.ownerRscNameToPermalink(dbPipeline.Owner)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	dbPipeline.Owner = ownerPermalink

	var dataMappingIndices []string
	for i := 0; i < len(fileNames); i++ {
		dataMappingIndices = append(dataMappingIndices, ulid.Make().String())
	}

	var modelInstOutputs []*pipelinePB.ModelInstanceOutput
	for idx, modelInstance := range dbPipeline.Recipe.ModelInstances {

		// TODO: async call model-backend
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		stream, err := s.modelServiceClient.TriggerModelInstanceBinaryFileUpload(ctx)
		defer func() {
			_ = stream.CloseSend()
		}()

		if err != nil {
			return nil, status.Errorf(codes.Internal, "[model-backend] Error %s at %dth model instance %s: cannot init stream: %v", "TriggerModelBinaryFileUpload", idx, modelInstance, err.Error())
		}

		if err := stream.Send(&modelPB.TriggerModelInstanceBinaryFileUploadRequest{
			Name:        modelInstance,
			FileLengths: fileLengths,
		}); err != nil {
			return nil, status.Errorf(codes.Internal, "[model-backend] Error %s at %dth model instance %s: cannot send data info to server: %v", "TriggerModelInstanceBinaryFileUploadRequest", idx, modelInstance, err.Error())
		}

		fb := bytes.Buffer{}
		fb.Write(fileBuf.Bytes())
		buf := make([]byte, 64*1024)
		for {
			n, err := fb.Read(buf)
			if err == io.EOF {
				break
			} else if err != nil {
				return nil, err
			}

			if err := stream.Send(&modelPB.TriggerModelInstanceBinaryFileUploadRequest{
				Content: buf[:n],
			}); err != nil {
				return nil, status.Errorf(codes.Internal, "[model-backend] Error %s at %dth model instance %s: cannot send chunk to server: %v", "TriggerModelInstanceBinaryFileUploadRequest", idx, modelInstance, err.Error())
			}
		}

		resp, err := stream.CloseAndRecv()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "[model-backend] Error %s at %dth model instance %s: cannot receive response: %v", "TriggerModelInstanceBinaryFileUploadRequest", idx, modelInstance, err.Error())
		}

		batchOutputs := cvtModelBatchOutputToPipelineBatchOutput(resp.BatchOutputs)
		for idx, batchOutput := range batchOutputs {
			batchOutput.Index = dataMappingIndices[idx]
		}

		modelInstOutputs = append(modelInstOutputs, &pipelinePB.ModelInstanceOutput{
			ModelInstance: modelInstance,
			Task:          resp.Task,
			BatchOutputs:  batchOutputs,
		})

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

	switch {
	// Check if this is a SYNC trigger (i.e., HTTP, gRPC source and destination connectors)
	case dbPipeline.Mode == datamodel.PipelineMode(pipelinePB.Pipeline_MODE_SYNC):
		return &pipelinePB.TriggerPipelineBinaryFileUploadResponse{
			DataMappingIndices:   dataMappingIndices,
			ModelInstanceOutputs: modelInstOutputs,
		}, nil
	case dbPipeline.Mode == datamodel.PipelineMode(pipelinePB.Pipeline_MODE_ASYNC):
		for idx, modelInstRecName := range dbPipeline.Recipe.ModelInstances {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_, err = s.connectorServiceClient.WriteDestinationConnector(ctx, &connectorPB.WriteDestinationConnectorRequest{
				Name:                 dbPipeline.Recipe.Destination,
				SyncMode:             connectorPB.SupportedSyncModes_SUPPORTED_SYNC_MODES_FULL_REFRESH,
				DestinationSyncMode:  connectorPB.SupportedDestinationSyncModes_SUPPORTED_DESTINATION_SYNC_MODES_APPEND,
				Pipeline:             fmt.Sprintf("pipelines/%s", dbPipeline.ID),
				DataMappingIndices:   dataMappingIndices,
				ModelInstanceOutputs: modelInstOutputs,
				Recipe: func() *pipelinePB.Recipe {
					logger, _ := logger.GetZapLogger()

					if dbPipeline.Recipe != nil {
						b, err := json.Marshal(dbPipeline.Recipe)
						if err != nil {
							logger.Error(err.Error())
						}
						pbRecipe := pipelinePB.Recipe{}
						err = json.Unmarshal(b, &pbRecipe)
						if err != nil {
							logger.Error(err.Error())
						}
						return &pbRecipe
					}
					return nil
				}(),
			})
			if err != nil {
				return nil, status.Errorf(codes.Internal, "[connector-backend] Error %s at %dth model instance %s: %v", "WriteDestinationConnector", idx, modelInstRecName, err.Error())
			}
		}
		return &pipelinePB.TriggerPipelineBinaryFileUploadResponse{}, nil

	}

	return nil, status.Errorf(codes.Internal, "something went very wrong - unable to trigger the pipeline")

}
