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

	"github.com/instill-ai/pipeline-backend/internal/resource"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/logger"
	"github.com/instill-ai/pipeline-backend/pkg/repository"

	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
	mgmtPB "github.com/instill-ai/protogen-go/vdp/mgmt/v1alpha"
	modelPB "github.com/instill-ai/protogen-go/vdp/model/v1alpha"
	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1alpha"
)

type TextToImageInput struct {
	Prompt   string
	Steps    int64
	CfgScale float32
	Seed     int64
	Samples  int64
}

type TextGenerationInput struct {
	Prompt        string
	OutputLen     int64
	BadWordsList  string
	StopWordsList string
	TopK          int64
	Seed          int64
}

type ImageInput struct {
	Content     []byte
	FileNames   []string
	FileLengths []uint64
}

// Service interface
type Service interface {
	GetMgmtPrivateServiceClient() mgmtPB.MgmtPrivateServiceClient

	CreatePipeline(pipeline *datamodel.Pipeline) (*datamodel.Pipeline, error)
	ListPipelines(ownerRscName string, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter) ([]datamodel.Pipeline, int64, string, error)
	GetPipelineByID(id string, ownerRscName string, isBasicView bool) (*datamodel.Pipeline, error)
	GetPipelineByUID(uid uuid.UUID, ownerRscName string, isBasicView bool) (*datamodel.Pipeline, error)
	UpdatePipeline(id string, ownerRscName string, updatedPipeline *datamodel.Pipeline) (*datamodel.Pipeline, error)
	DeletePipeline(id string, ownerRscName string) error
	UpdatePipelineState(id string, ownerRscName string, state datamodel.PipelineState) (*datamodel.Pipeline, error)
	UpdatePipelineID(id string, ownerRscName string, newID string) (*datamodel.Pipeline, error)
	TriggerPipeline(req *pipelinePB.TriggerPipelineRequest, pipeline *datamodel.Pipeline) (*pipelinePB.TriggerPipelineResponse, error)
	TriggerPipelineBinaryFileUpload(pipeline *datamodel.Pipeline, task modelPB.ModelInstance_Task, input interface{}) (*pipelinePB.TriggerPipelineBinaryFileUploadResponse, error)
	GetModelInstanceByName(modelInstanceName string) (*modelPB.ModelInstance, error)

	ListPipelinesAdmin(pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter) ([]datamodel.Pipeline, int64, string, error)
	GetPipelineByIDAdmin(id string, isBasicView bool) (*datamodel.Pipeline, error)
	GetPipelineByUIDAdmin(uid uuid.UUID, isBasicView bool) (*datamodel.Pipeline, error)
}

type service struct {
	repository                   repository.Repository
	mgmtPrivateServiceClient     mgmtPB.MgmtPrivateServiceClient
	connectorPublicServiceClient connectorPB.ConnectorPublicServiceClient
	modelPublicServiceClient     modelPB.ModelPublicServiceClient
	redisClient                  *redis.Client
}

// NewService initiates a service instance
func NewService(r repository.Repository, u mgmtPB.MgmtPrivateServiceClient, c connectorPB.ConnectorPublicServiceClient, m modelPB.ModelPublicServiceClient, rc *redis.Client) Service {
	return &service{
		repository:                   r,
		mgmtPrivateServiceClient:     u,
		connectorPublicServiceClient: c,
		modelPublicServiceClient:     m,
		redisClient:                  rc,
	}
}

// GetMgmtPrivateServiceClient returns the management private service client
func (h *service) GetMgmtPrivateServiceClient() mgmtPB.MgmtPrivateServiceClient {
	return h.mgmtPrivateServiceClient
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

func (s *service) ListPipelines(ownerRscName string, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter) ([]datamodel.Pipeline, int64, string, error) {

	ownerPermalink, err := s.ownerRscNameToPermalink(ownerRscName)
	if err != nil {
		return nil, 0, "", status.Errorf(codes.InvalidArgument, err.Error())
	}

	dbPipelines, ps, pt, err := s.repository.ListPipelines(ownerPermalink, pageSize, pageToken, isBasicView, filter)
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

func (s *service) ListPipelinesAdmin(pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter) ([]datamodel.Pipeline, int64, string, error) {

	dbPipelines, ps, pt, err := s.repository.ListPipelinesAdmin(pageSize, pageToken, isBasicView, filter)
	if err != nil {
		return nil, 0, "", err
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

func (s *service) GetPipelineByIDAdmin(id string, isBasicView bool) (*datamodel.Pipeline, error) {

	dbPipeline, err := s.repository.GetPipelineByIDAdmin(id, isBasicView)
	if err != nil {
		return nil, err
	}

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

func (s *service) GetPipelineByUIDAdmin(uid uuid.UUID, isBasicView bool) (*datamodel.Pipeline, error) {

	dbPipeline, err := s.repository.GetPipelineByUIDAdmin(uid, isBasicView)
	if err != nil {
		return nil, err
	}

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
	var taskInputs []*modelPB.TaskInput
	for _, taskInput := range req.TaskInputs {
		taskInputs = append(taskInputs, &modelPB.TaskInput{
			Input: taskInput.GetInput(),
		})
		dataMappingIndices = append(dataMappingIndices, ulid.Make().String())
	}

	wg := sync.WaitGroup{}
	wg.Add(1)

	var modelInstOutputs []*pipelinePB.ModelInstanceOutput
	errors := make(chan error)
	go func() {
		defer wg.Done()

		for idx, modelInstance := range dbPipeline.Recipe.ModelInstances {

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			defer cancel()
			resp, err := s.modelPublicServiceClient.TriggerModelInstance(ctx, &modelPB.TriggerModelInstanceRequest{
				Name:       modelInstance,
				TaskInputs: taskInputs,
			})
			if err != nil {
				errors <- err
				logger.Error(fmt.Sprintf("[model-backend] Error %s at %dth model instance %s: %v", "TriggerModel", idx, modelInstance, err.Error()))
				return
			}

			taskOutputs := cvtModelTaskOutputToPipelineTaskOutput(resp.TaskOutputs)
			for idx, taskOutput := range taskOutputs {
				taskOutput.Index = dataMappingIndices[idx]
			}

			modelInstOutputs = append(modelInstOutputs, &pipelinePB.ModelInstanceOutput{
				ModelInstance: modelInstance,
				Task:          resp.Task,
				TaskOutputs:   taskOutputs,
			})

			// Increment trigger image numbers
			uid, err := resource.GetPermalinkUID(dbPipeline.Owner)
			if err != nil {
				errors <- err
				logger.Error(err.Error())
				return
			}
			if strings.HasPrefix(dbPipeline.Owner, "users/") {
				s.redisClient.IncrBy(context.Background(), fmt.Sprintf("user:%s:trigger.num", uid), int64(len(taskInputs)))
			} else if strings.HasPrefix(dbPipeline.Owner, "orgs/") {
				s.redisClient.IncrBy(context.Background(), fmt.Sprintf("org:%s:trigger.num", uid), int64(len(taskInputs)))
			}
		}
		errors <- nil
	}()

	switch {
	// If this is a SYNC trigger (i.e., HTTP, gRPC source and destination connectors), return right away
	case dbPipeline.Mode == datamodel.PipelineMode(pipelinePB.Pipeline_MODE_SYNC):
		go func() {
			wg.Wait()
			close(errors)
		}()
		for err := range errors {
			if err != nil {
				switch {
				case strings.Contains(err.Error(), "code = DeadlineExceeded"):
					return nil, status.Errorf(codes.DeadlineExceeded, "trigger model instance got timeout error")
				default:
					return nil, status.Errorf(codes.Internal, fmt.Sprintf("trigger model instance got error %v", err.Error()))
				}
			}
		}
		return &pipelinePB.TriggerPipelineResponse{
			DataMappingIndices:   dataMappingIndices,
			ModelInstanceOutputs: modelInstOutputs,
		}, nil
	// If this is a async trigger, write to the destination connector
	case dbPipeline.Mode == datamodel.PipelineMode(pipelinePB.Pipeline_MODE_ASYNC):
		go func() {
			go func() {
				wg.Wait()
				close(errors)
			}()
			for err := range errors {
				if err != nil {
					logger.Error(fmt.Sprintf("[model-backend] Error trigger model instance got error %v", err.Error()))
					return
				}
			}

			for idx, modelInstRecName := range dbPipeline.Recipe.ModelInstances {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_, err = s.connectorPublicServiceClient.WriteDestinationConnector(ctx, &connectorPB.WriteDestinationConnectorRequest{
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

func (s *service) triggerImageTask(dbPipeline *datamodel.Pipeline, task modelPB.ModelInstance_Task, input interface{}, dataMappingIndices []string) ([]*pipelinePB.ModelInstanceOutput, error) {
	imageInput := input.(*ImageInput)
	var modelInstOutputs []*pipelinePB.ModelInstanceOutput
	for idx, modelInstance := range dbPipeline.Recipe.ModelInstances {

		// TODO: async call model-backend
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		stream, err := s.modelPublicServiceClient.TriggerModelInstanceBinaryFileUpload(ctx)
		defer func() {
			_ = stream.CloseSend()
		}()

		if err != nil {
			return modelInstOutputs, status.Errorf(codes.Internal, "[model-backend] Error %s at %dth model instance %s: cannot init stream: %v", "TriggerModelBinaryFileUpload", idx, modelInstance, err.Error())
		}
		var triggerRequest modelPB.TriggerModelInstanceBinaryFileUploadRequest
		switch task {
		case modelPB.ModelInstance_TASK_CLASSIFICATION:
			triggerRequest = modelPB.TriggerModelInstanceBinaryFileUploadRequest{
				Name: modelInstance,
				TaskInput: &modelPB.TaskInputStream{
					Input: &modelPB.TaskInputStream_Classification{
						Classification: &modelPB.ClassificationInputStream{
							FileLengths: imageInput.FileLengths,
						},
					},
				},
			}
		case modelPB.ModelInstance_TASK_DETECTION:
			triggerRequest = modelPB.TriggerModelInstanceBinaryFileUploadRequest{
				Name: modelInstance,
				TaskInput: &modelPB.TaskInputStream{
					Input: &modelPB.TaskInputStream_Detection{
						Detection: &modelPB.DetectionInputStream{
							FileLengths: imageInput.FileLengths,
						},
					},
				},
			}
		case modelPB.ModelInstance_TASK_KEYPOINT:
			triggerRequest = modelPB.TriggerModelInstanceBinaryFileUploadRequest{
				Name: modelInstance,
				TaskInput: &modelPB.TaskInputStream{
					Input: &modelPB.TaskInputStream_Keypoint{
						Keypoint: &modelPB.KeypointInputStream{
							FileLengths: imageInput.FileLengths,
						},
					},
				},
			}
		case modelPB.ModelInstance_TASK_OCR:
			triggerRequest = modelPB.TriggerModelInstanceBinaryFileUploadRequest{
				Name: modelInstance,
				TaskInput: &modelPB.TaskInputStream{
					Input: &modelPB.TaskInputStream_Ocr{
						Ocr: &modelPB.OcrInputStream{
							FileLengths: imageInput.FileLengths,
						},
					},
				},
			}

		case modelPB.ModelInstance_TASK_INSTANCE_SEGMENTATION:
			triggerRequest = modelPB.TriggerModelInstanceBinaryFileUploadRequest{
				Name: modelInstance,
				TaskInput: &modelPB.TaskInputStream{
					Input: &modelPB.TaskInputStream_InstanceSegmentation{
						InstanceSegmentation: &modelPB.InstanceSegmentationInputStream{
							FileLengths: imageInput.FileLengths,
						},
					},
				},
			}
		case modelPB.ModelInstance_TASK_SEMANTIC_SEGMENTATION:
			triggerRequest = modelPB.TriggerModelInstanceBinaryFileUploadRequest{
				Name: modelInstance,
				TaskInput: &modelPB.TaskInputStream{
					Input: &modelPB.TaskInputStream_SemanticSegmentation{
						SemanticSegmentation: &modelPB.SemanticSegmentationInputStream{
							FileLengths: imageInput.FileLengths,
						},
					},
				},
			}
		}
		if err := stream.Send(&triggerRequest); err != nil {
			return modelInstOutputs, status.Errorf(codes.Internal, "[model-backend] Error %s at %dth model instance %s: cannot send data info to server: %v", "TriggerModelInstanceBinaryFileUploadRequest", idx, modelInstance, err.Error())
		}
		fb := bytes.Buffer{}
		fb.Write(imageInput.Content)
		buf := make([]byte, 64*1024)
		for {
			n, err := fb.Read(buf)
			if err == io.EOF {
				break
			} else if err != nil {
				return modelInstOutputs, err
			}

			var triggerRequest modelPB.TriggerModelInstanceBinaryFileUploadRequest
			switch task {
			case modelPB.ModelInstance_TASK_CLASSIFICATION:
				triggerRequest = modelPB.TriggerModelInstanceBinaryFileUploadRequest{
					TaskInput: &modelPB.TaskInputStream{
						Input: &modelPB.TaskInputStream_Classification{
							Classification: &modelPB.ClassificationInputStream{
								Content: buf[:n],
							},
						},
					},
				}
			case modelPB.ModelInstance_TASK_DETECTION:
				triggerRequest = modelPB.TriggerModelInstanceBinaryFileUploadRequest{
					TaskInput: &modelPB.TaskInputStream{
						Input: &modelPB.TaskInputStream_Detection{
							Detection: &modelPB.DetectionInputStream{
								Content: buf[:n],
							},
						},
					},
				}
			case modelPB.ModelInstance_TASK_KEYPOINT:
				triggerRequest = modelPB.TriggerModelInstanceBinaryFileUploadRequest{
					TaskInput: &modelPB.TaskInputStream{
						Input: &modelPB.TaskInputStream_Keypoint{
							Keypoint: &modelPB.KeypointInputStream{
								Content: buf[:n],
							},
						},
					},
				}
			case modelPB.ModelInstance_TASK_OCR:
				triggerRequest = modelPB.TriggerModelInstanceBinaryFileUploadRequest{
					TaskInput: &modelPB.TaskInputStream{
						Input: &modelPB.TaskInputStream_Ocr{
							Ocr: &modelPB.OcrInputStream{
								Content: buf[:n],
							},
						},
					},
				}

			case modelPB.ModelInstance_TASK_INSTANCE_SEGMENTATION:
				triggerRequest = modelPB.TriggerModelInstanceBinaryFileUploadRequest{
					TaskInput: &modelPB.TaskInputStream{
						Input: &modelPB.TaskInputStream_InstanceSegmentation{
							InstanceSegmentation: &modelPB.InstanceSegmentationInputStream{
								Content: buf[:n],
							},
						},
					},
				}
			case modelPB.ModelInstance_TASK_SEMANTIC_SEGMENTATION:
				triggerRequest = modelPB.TriggerModelInstanceBinaryFileUploadRequest{
					TaskInput: &modelPB.TaskInputStream{
						Input: &modelPB.TaskInputStream_SemanticSegmentation{
							SemanticSegmentation: &modelPB.SemanticSegmentationInputStream{
								Content: buf[:n],
							},
						},
					},
				}
			}
			if err := stream.Send(&triggerRequest); err != nil {
				return modelInstOutputs, status.Errorf(codes.Internal, "[model-backend] Error %s at %dth model instance %s: cannot send chunk to server: %v", "TriggerModelInstanceBinaryFileUploadRequest", idx, modelInstance, err.Error())
			}
		}

		resp, err := stream.CloseAndRecv()
		if err != nil {
			return modelInstOutputs, status.Errorf(codes.Internal, "[model-backend] Error %s at %dth model instance %s: cannot receive response: %v", "TriggerModelInstanceBinaryFileUploadRequest", idx, modelInstance, err.Error())
		}

		taskOutputs := cvtModelTaskOutputToPipelineTaskOutput(resp.TaskOutputs)
		for idx, taskOutput := range taskOutputs {
			taskOutput.Index = dataMappingIndices[idx]
		}

		modelInstOutputs = append(modelInstOutputs, &pipelinePB.ModelInstanceOutput{
			ModelInstance: modelInstance,
			Task:          resp.Task,
			TaskOutputs:   taskOutputs,
		})

		// Increment trigger image numbers
		uid, err := resource.GetPermalinkUID(dbPipeline.Owner)
		if err != nil {
			return modelInstOutputs, err
		}
		if strings.HasPrefix(dbPipeline.Owner, "users/") {
			s.redisClient.IncrBy(ctx, fmt.Sprintf("user:%s:trigger.num", uid), int64(len(imageInput.FileLengths)))
		} else if strings.HasPrefix(dbPipeline.Owner, "orgs/") {
			s.redisClient.IncrBy(ctx, fmt.Sprintf("org:%s:trigger.num", uid), int64(len(imageInput.FileLengths)))
		}
	}

	return modelInstOutputs, nil
}

func (s *service) triggerTextTask(dbPipeline *datamodel.Pipeline, task modelPB.ModelInstance_Task, input interface{}, dataMappingIndices []string) ([]*pipelinePB.ModelInstanceOutput, error) {

	var modelInstOutputs []*pipelinePB.ModelInstanceOutput
	for idx, modelInstance := range dbPipeline.Recipe.ModelInstances {

		// TODO: async call model-backend
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		stream, err := s.modelPublicServiceClient.TriggerModelInstanceBinaryFileUpload(ctx)
		defer func() {
			_ = stream.CloseSend()
		}()

		if err != nil {
			return modelInstOutputs, status.Errorf(codes.Internal, "[model-backend] Error %s at %dth model instance %s: cannot init stream: %v", "TriggerModelBinaryFileUpload", idx, modelInstance, err.Error())
		}

		var triggerRequest modelPB.TriggerModelInstanceBinaryFileUploadRequest
		switch task {
		case modelPB.ModelInstance_TASK_TEXT_TO_IMAGE:
			textToImageInput := input.(*TextToImageInput)
			triggerRequest = modelPB.TriggerModelInstanceBinaryFileUploadRequest{
				Name: modelInstance,
				TaskInput: &modelPB.TaskInputStream{
					Input: &modelPB.TaskInputStream_TextToImage{
						TextToImage: &modelPB.TextToImageInput{
							Prompt:   textToImageInput.Prompt,
							Steps:    &textToImageInput.Steps,
							CfgScale: &textToImageInput.CfgScale,
							Seed:     &textToImageInput.Seed,
							Samples:  &textToImageInput.Samples,
						},
					},
				},
			}
		case modelPB.ModelInstance_TASK_TEXT_GENERATION:
			textGenerationInput := input.(*TextGenerationInput)
			triggerRequest = modelPB.TriggerModelInstanceBinaryFileUploadRequest{
				Name: modelInstance,
				TaskInput: &modelPB.TaskInputStream{
					Input: &modelPB.TaskInputStream_TextGeneration{
						TextGeneration: &modelPB.TextGenerationInput{
							Prompt:        textGenerationInput.Prompt,
							OutputLen:     &textGenerationInput.OutputLen,
							BadWordsList:  &textGenerationInput.BadWordsList,
							StopWordsList: &textGenerationInput.StopWordsList,
							Topk:          &textGenerationInput.TopK,
							Seed:          &textGenerationInput.Seed,
						},
					},
				},
			}
		}

		if err := stream.Send(&triggerRequest); err != nil {
			return modelInstOutputs, status.Errorf(codes.Internal, "[model-backend] Error %s at %dth model instance %s: cannot send data info to server: %v", "TriggerModelInstanceBinaryFileUploadRequest", idx, modelInstance, err.Error())
		}

		resp, err := stream.CloseAndRecv()
		if err != nil {
			return modelInstOutputs, status.Errorf(codes.Internal, "[model-backend] Error %s at %dth model instance %s: cannot receive response: %v", "TriggerModelInstanceBinaryFileUploadRequest", idx, modelInstance, err.Error())
		}

		taskOutputs := cvtModelTaskOutputToPipelineTaskOutput(resp.TaskOutputs)
		for idx, taskOutput := range taskOutputs {
			taskOutput.Index = dataMappingIndices[idx]
		}

		modelInstOutputs = append(modelInstOutputs, &pipelinePB.ModelInstanceOutput{
			ModelInstance: modelInstance,
			Task:          resp.Task,
			TaskOutputs:   taskOutputs,
		})

		// Increment trigger image numbers
		uid, err := resource.GetPermalinkUID(dbPipeline.Owner)
		if err != nil {
			return modelInstOutputs, err
		}
		if strings.HasPrefix(dbPipeline.Owner, "users/") {
			s.redisClient.IncrBy(ctx, fmt.Sprintf("user:%s:trigger.num", uid), 1)
		} else if strings.HasPrefix(dbPipeline.Owner, "orgs/") {
			s.redisClient.IncrBy(ctx, fmt.Sprintf("org:%s:trigger.num", uid), 1)
		}
	}

	return modelInstOutputs, nil
}

func (s *service) TriggerPipelineBinaryFileUpload(dbPipeline *datamodel.Pipeline, task modelPB.ModelInstance_Task, input interface{}) (*pipelinePB.TriggerPipelineBinaryFileUploadResponse, error) {
	if dbPipeline.State != datamodel.PipelineState(pipelinePB.Pipeline_STATE_ACTIVE) {
		return nil, status.Error(codes.FailedPrecondition, fmt.Sprintf("The pipeline %s is not active", dbPipeline.ID))
	}

	ownerPermalink, err := s.ownerRscNameToPermalink(dbPipeline.Owner)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	dbPipeline.Owner = ownerPermalink

	batching := 1
	switch task {
	case modelPB.ModelInstance_TASK_CLASSIFICATION,
		modelPB.ModelInstance_TASK_DETECTION,
		modelPB.ModelInstance_TASK_KEYPOINT,
		modelPB.ModelInstance_TASK_OCR,
		modelPB.ModelInstance_TASK_INSTANCE_SEGMENTATION,
		modelPB.ModelInstance_TASK_SEMANTIC_SEGMENTATION:
		inp := input.(*ImageInput)
		batching = len(inp.FileNames)
	case modelPB.ModelInstance_TASK_TEXT_TO_IMAGE,
		modelPB.ModelInstance_TASK_TEXT_GENERATION:
		batching = 1
	}
	var dataMappingIndices []string
	for i := 0; i < batching; i++ {
		dataMappingIndices = append(dataMappingIndices, ulid.Make().String())
	}

	var modelInstOutputs []*pipelinePB.ModelInstanceOutput
	switch task {
	case modelPB.ModelInstance_TASK_CLASSIFICATION,
		modelPB.ModelInstance_TASK_DETECTION,
		modelPB.ModelInstance_TASK_KEYPOINT,
		modelPB.ModelInstance_TASK_OCR,
		modelPB.ModelInstance_TASK_INSTANCE_SEGMENTATION,
		modelPB.ModelInstance_TASK_SEMANTIC_SEGMENTATION:
		modelInstOutputs, err = s.triggerImageTask(dbPipeline, task, input, dataMappingIndices)
		if err != nil {
			return nil, status.Errorf(codes.Internal, err.Error())
		}
	case modelPB.ModelInstance_TASK_TEXT_TO_IMAGE,
		modelPB.ModelInstance_TASK_TEXT_GENERATION:
		modelInstOutputs, err = s.triggerTextTask(dbPipeline, task, input, dataMappingIndices)
		if err != nil {
			return nil, status.Errorf(codes.Internal, err.Error())
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
			_, err = s.connectorPublicServiceClient.WriteDestinationConnector(ctx, &connectorPB.WriteDestinationConnectorRequest{
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
		return &pipelinePB.TriggerPipelineBinaryFileUploadResponse{
			DataMappingIndices:   dataMappingIndices,
			ModelInstanceOutputs: nil,
		}, nil

	}

	return nil, status.Errorf(codes.Internal, "something went very wrong - unable to trigger the pipeline")

}

func (s *service) GetModelInstanceByName(modelInstanceName string) (*modelPB.ModelInstance, error) {
	modeInstanceResq, err := s.modelPublicServiceClient.GetModelInstance(context.Background(), &modelPB.GetModelInstanceRequest{
		Name: modelInstanceName,
	})
	if err != nil {
		return nil, err
	}
	return modeInstanceResq.Instance, nil
}
