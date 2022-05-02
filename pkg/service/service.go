package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"regexp"
	"time"

	"github.com/gofrs/uuid"
	"github.com/gogo/status"
	"google.golang.org/grpc/codes"

	"github.com/instill-ai/pipeline-backend/internal/constant"
	"github.com/instill-ai/pipeline-backend/internal/util"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/repository"

	modelPB "github.com/instill-ai/protogen-go/model/v1alpha"
	pipelinePB "github.com/instill-ai/protogen-go/pipeline/v1alpha"
)

// Service interface
type Service interface {
	CreatePipeline(pipeline *datamodel.Pipeline) (*datamodel.Pipeline, error)
	ListPipeline(ownerID uuid.UUID, view pipelinePB.PipelineView, pageSize int, pageToken string) ([]datamodel.Pipeline, string, error)
	GetPipelineByName(name string, ownerID uuid.UUID) (*datamodel.Pipeline, error)
	UpdatePipeline(id uuid.UUID, ownerID uuid.UUID, updatedPipeline *datamodel.Pipeline) (*datamodel.Pipeline, error)
	DeletePipeline(id uuid.UUID, ownerID uuid.UUID) error
	TriggerPipeline(req *pipelinePB.TriggerPipelineRequest, pipeline *datamodel.Pipeline) (*modelPB.TriggerModelResponse, error)
	TriggerPipelineBinaryFileUpload(fileBuf bytes.Buffer, fileLengths []uint64, pipeline *datamodel.Pipeline) (*modelPB.TriggerModelBinaryFileUploadResponse, error)
	ValidatePipeline(pipeline *datamodel.Pipeline) error
	ValidateModel(selectedModel []*datamodel.Model) error
}

type service struct {
	repository         repository.Repository
	modelServiceClient modelPB.ModelServiceClient
}

// NewService initiates a service instance
func NewService(r repository.Repository, m modelPB.ModelServiceClient) Service {
	return &service{
		repository:         r,
		modelServiceClient: m,
	}
}

func (s *service) CreatePipeline(pipeline *datamodel.Pipeline) (*datamodel.Pipeline, error) {

	// Validatation: Required field
	if pipeline.Name == "" {
		return nil, status.Error(codes.FailedPrecondition, "The required field name is not specified")
	}

	// Validatation: Required field
	if pipeline.Recipe == nil {
		return nil, status.Error(codes.FailedPrecondition, "The required field recipe is not specified")
	}

	// Validatation: name naming rule
	if match, _ := regexp.MatchString("^[A-Za-z0-9][a-zA-Z0-9_.-]*$", pipeline.Name); !match {
		return nil, status.Error(codes.FailedPrecondition, "The name of pipeline is invalid")
	}

	// Validation: name length
	if len(pipeline.Name) > 100 {
		return nil, status.Error(codes.FailedPrecondition, "The name of pipeline has more than 100 characters")
	}

	// Validation: Model availability
	if pipeline.Recipe != nil && pipeline.Recipe.Models != nil && len(pipeline.Recipe.Models) > 0 {
		err := s.ValidateModel(pipeline.Recipe.Models)
		if err != nil {
			return nil, err
		}
	}

	// Determine pipeline mode
	if util.Contains(constant.ConnectionTypeDirectness, pipeline.Recipe.Source.Name) &&
		util.Contains(constant.ConnectionTypeDirectness, pipeline.Recipe.Destination.Name) {
		if pipeline.Recipe.Source.Name == pipeline.Recipe.Destination.Name {
			pipeline.Mode = datamodel.PipelineMode(pipelinePB.Pipeline_MODE_SYNC)
		} else {
			return nil, status.Error(codes.FailedPrecondition, "Source and destination connector must be the same for directness connection type")
		}
	} else {
		pipeline.Mode = datamodel.PipelineMode(pipelinePB.Pipeline_MODE_ASYNC)
	}

	// TODO: Check connectors and model status to assign pipeline status

	if err := s.repository.CreatePipeline(pipeline); err != nil {
		return nil, err
	}

	dbPipeline, err := s.GetPipelineByName(pipeline.Name, pipeline.OwnerID)
	if err != nil {
		return nil, err
	}

	return dbPipeline, nil
}

func (s *service) ListPipeline(ownerID uuid.UUID, view pipelinePB.PipelineView, pageSize int, pageToken string) ([]datamodel.Pipeline, string, error) {
	return s.repository.ListPipeline(ownerID, view, pageSize, pageToken)
}

func (s *service) GetPipelineByName(name string, ownerID uuid.UUID) (*datamodel.Pipeline, error) {
	dbPipeline, err := s.repository.GetPipelineByName(name, ownerID)
	if err != nil {
		return nil, err
	}

	// TODO: Use owner_id to query owner_name in mgmt-backend
	dbPipeline.FullName = fmt.Sprintf("local-user/%s", dbPipeline.Name)

	return dbPipeline, nil
}

func (s *service) UpdatePipeline(id uuid.UUID, ownerID uuid.UUID, updatedPipeline *datamodel.Pipeline) (*datamodel.Pipeline, error) {

	// Validation: Pipeline existence
	if existingPipeline, _ := s.repository.GetPipeline(id, ownerID); existingPipeline == nil {
		return nil, status.Errorf(codes.NotFound, "Pipeline id \"%s\" is not found", id.String())
	}

	// Validatation: name naming rule
	if match, _ := regexp.MatchString("^[A-Za-z0-9][a-zA-Z0-9_.-]*$", updatedPipeline.Name); !match {
		return nil, status.Error(codes.FailedPrecondition, "The name of pipeline is invalid")
	}

	// Validation: name length
	if len(updatedPipeline.Name) > 100 {
		return nil, status.Error(codes.FailedPrecondition, "The name of pipeline has more than 100 characters")
	}

	// Validation: Model instance
	if updatedPipeline.Recipe != nil && updatedPipeline.Recipe.Models != nil && len(updatedPipeline.Recipe.Models) > 0 {
		err := s.ValidateModel(updatedPipeline.Recipe.Models)
		if err != nil {
			return &datamodel.Pipeline{}, err
		}
	}

	if err := s.repository.UpdatePipeline(id, ownerID, updatedPipeline); err != nil {
		return nil, err
	}

	dbPipeline, err := s.GetPipelineByName(updatedPipeline.Name, ownerID)
	if err != nil {
		return nil, err
	}

	return dbPipeline, nil
}

func (s *service) DeletePipeline(id uuid.UUID, ownerID uuid.UUID) error {
	return s.repository.DeletePipeline(id, ownerID)
}

func (s *service) ValidatePipeline(pipeline *datamodel.Pipeline) error {

	// Validation: Pipeline is in inactivated status
	if pipeline.Status == datamodel.PipelineStatus(pipelinePB.Pipeline_STATUS_INACTIVATED) {
		return status.Error(codes.FailedPrecondition, "This pipeline is inactivated")
	}

	// Validation: Pipeline is in error status
	if pipeline.Status == datamodel.PipelineStatus(pipelinePB.Pipeline_STATUS_ERROR) {
		return status.Error(codes.FailedPrecondition, "This pipeline has errors")
	}

	return nil
}

func (s *service) TriggerPipeline(req *pipelinePB.TriggerPipelineRequest, pipeline *datamodel.Pipeline) (*modelPB.TriggerModelResponse, error) {

	// Check if this is a direct trigger (i.e., HTTP, gRPC source and destination connectors)
	if pipeline.Mode == datamodel.PipelineMode(pipelinePB.Pipeline_MODE_SYNC) {

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		model := pipeline.Recipe.Models[0]

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

		resp, err := s.modelServiceClient.TriggerModel(ctx, &modelPB.TriggerModelRequest{
			ModelName:    model.Name,
			InstanceName: model.InstanceName,
			Inputs:       inputs,
		})

		if err != nil {
			return nil, status.Errorf(codes.Internal, "Error model-backend %s: %v", "TriggerModel", err.Error())
		}

		return resp, nil
	}

	return nil, nil

}

func (s *service) TriggerPipelineBinaryFileUpload(fileBuf bytes.Buffer, fileLengths []uint64, pipeline *datamodel.Pipeline) (*modelPB.TriggerModelBinaryFileUploadResponse, error) {

	// Check if this is a direct trigger (i.e., HTTP, gRPC source and destination connectors)
	if pipeline.Mode == datamodel.PipelineMode(pipelinePB.Pipeline_MODE_SYNC) {

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		stream, err := s.modelServiceClient.TriggerModelBinaryFileUpload(ctx)
		defer func() {
			_ = stream.CloseSend()
		}()
		if err != nil {
			return nil, fmt.Errorf("Error model-backend %s: %v", "TriggerModelBinaryFileUpload", err.Error())
		}

		err = stream.Send(&modelPB.TriggerModelBinaryFileUploadRequest{
			ModelName:    pipeline.Recipe.Models[0].Name,
			InstanceName: pipeline.Recipe.Models[0].InstanceName,
			FileLengths:  fileLengths,
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "cannot send data info to server: %s", err.Error())
		}

		const chunkSize = 64 * 1024
		buf := make([]byte, chunkSize)

		for {
			n, err := fileBuf.Read(buf)
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, err
			}

			err = stream.Send(&modelPB.TriggerModelBinaryFileUploadRequest{Bytes: buf[:n]})
			if err != nil {
				return nil, status.Errorf(codes.Internal, "cannot send chunk to server: %s", err)
			}
		}
		res, err := stream.CloseAndRecv()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "cannot receive response: %s", err.Error())
		}

		return res, nil
	}

	return nil, nil

}

func (s *service) ValidateModel(selectedModels []*datamodel.Model) error {

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	supportModelResp, err := s.modelServiceClient.ListModel(ctx, &modelPB.ListModelRequest{})
	if err != nil {
		return fmt.Errorf("Error model-backend %s: %v", "ListModel", err.Error())
	}

	hasInvalidModel := false
	invalidErrorString := ""
	for _, selectedModel := range selectedModels {
		matchModel := false
		for _, supportModel := range supportModelResp.Models {
			if selectedModel.Name == supportModel.Name {
				for _, supportInstance := range supportModel.Instances {
					if selectedModel.InstanceName == supportInstance.Name {
						if supportInstance.Status == modelPB.ModelInstance_STATUS_ONLINE {
							matchModel = true
							break
						}
					}
				}
			}
		}
		if !matchModel {
			hasInvalidModel = true
			invalidErrorString += fmt.Sprintf("The model name %s with its instance %s you specified is not applicable\n", selectedModel.Name, selectedModel.InstanceName)
		}
	}

	if hasInvalidModel {
		return status.Error(codes.InvalidArgument, invalidErrorString)
	}

	return nil

}
