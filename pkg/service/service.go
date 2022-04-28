package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"
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
	ListPipeline(ownerID uuid.UUID, view pipelinePB.PipelineView, pageSize int, pageCursor string) ([]datamodel.Pipeline, string, error)
	GetPipeline(ownerID uuid.UUID, name string) (*datamodel.Pipeline, error)
	UpdatePipeline(ownerID uuid.UUID, name string, updatedPipeline *datamodel.Pipeline) (*datamodel.Pipeline, error)
	DeletePipeline(ownerID uuid.UUID, name string) error
	TriggerPipeline(ownerID uuid.UUID, req *pipelinePB.TriggerPipelineRequest, pipeline *datamodel.Pipeline) (*modelPB.TriggerModelResponse, error)
	TriggerPipelineByUpload(ownerID uuid.UUID, image bytes.Buffer, pipeline *datamodel.Pipeline) (*modelPB.TriggerModelBinaryFileUploadResponse, error)
	ValidateTriggerPipeline(ownerID uuid.UUID, name string, pipeline *datamodel.Pipeline) error
	ValidateModel(ownerID uuid.UUID, selectedModel []*datamodel.Model) error
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

	// Validatation: Naming rule
	if match, _ := regexp.MatchString("^[A-Za-z0-9][a-zA-Z0-9_.-]*$", pipeline.Name); !match {
		return nil, status.Error(codes.FailedPrecondition, "The name of pipeline is invalid")
	}

	// Validation: Name length
	if len(pipeline.Name) > 100 {
		return nil, status.Error(codes.FailedPrecondition, "The name of pipeline has more than 100 characters")
	}

	if pipeline.Recipe != nil && pipeline.Recipe.Models != nil && len(pipeline.Recipe.Models) > 0 {
		err := s.ValidateModel(pipeline.OwnerID, pipeline.Recipe.Models)
		if err != nil {
			return &datamodel.Pipeline{}, err
		}
	}

	// TODO: Check connectors and model status to assign pipeline status

	if err := s.repository.CreatePipeline(pipeline); err != nil {
		return nil, err
	}

	dbPipeline, err := s.GetPipeline(pipeline.OwnerID, pipeline.Name)
	if err != nil {
		return nil, err
	}

	return dbPipeline, nil
}

func (s *service) ListPipeline(ownerID uuid.UUID, view pipelinePB.PipelineView, pageSize int, pageCursor string) ([]datamodel.Pipeline, string, error) {
	return s.repository.ListPipeline(ownerID, view, pageSize, pageCursor)
}

func (s *service) GetPipeline(ownerID uuid.UUID, name string) (*datamodel.Pipeline, error) {

	// Validatation: Required field
	if name == "" {
		return nil, status.Error(codes.FailedPrecondition, "The required field name is not specified")
	}

	dbPipeline, err := s.repository.GetPipeline(ownerID, name)
	if err != nil {
		return nil, err
	}

	// TODO: Use owner_id to query owner_name in mgmt-backend
	dbPipeline.FullName = fmt.Sprintf("local-user/%s", dbPipeline.Name)

	return dbPipeline, nil
}

func (s *service) UpdatePipeline(ownerID uuid.UUID, name string, updatedPipeline *datamodel.Pipeline) (*datamodel.Pipeline, error) {

	// Validatation: Required field
	if name == "" {
		return nil, status.Error(codes.FailedPrecondition, "The required field name not specify")
	}

	// Validation: Pipeline existence
	if existingPipeline, _ := s.GetPipeline(ownerID, name); existingPipeline == nil {
		return nil, status.Errorf(codes.NotFound, "Pipeline name \"%s\" is not found", updatedPipeline.Name)
	}

	// Validatation: Naming rule
	if match, _ := regexp.MatchString("^[A-Za-z0-9][a-zA-Z0-9_.-]*$", updatedPipeline.Name); !match {
		return nil, status.Error(codes.FailedPrecondition, "The updated name of pipeline is invalid")
	}

	// Validation: Name length
	if len(updatedPipeline.Name) > 100 {
		return nil, status.Error(codes.FailedPrecondition, "The updated name of pipeline has more than 100 characters")
	}

	// Validation: Model instance
	if updatedPipeline.Recipe != nil && updatedPipeline.Recipe.Models != nil && len(updatedPipeline.Recipe.Models) > 0 {
		err := s.ValidateModel(updatedPipeline.OwnerID, updatedPipeline.Recipe.Models)
		if err != nil {
			return &datamodel.Pipeline{}, err
		}
	}

	if err := s.repository.UpdatePipeline(ownerID, name, updatedPipeline); err != nil {
		return nil, err
	}

	dbPipeline, err := s.GetPipeline(updatedPipeline.OwnerID, updatedPipeline.Name)
	if err != nil {
		return nil, err
	}

	return dbPipeline, nil
}

func (s *service) DeletePipeline(ownerID uuid.UUID, name string) error {
	return s.repository.DeletePipeline(ownerID, name)
}

func (s *service) ValidateTriggerPipeline(owerID uuid.UUID, name string, pipeline *datamodel.Pipeline) error {

	// Validation: Required field
	if pipeline.Name == "" {
		return status.Errorf(codes.NotFound, "The pipeline name %s you specified is not found", name)
	}

	// Validation: Pipeline is inactivated
	if pipeline.Status == datamodel.PipelineStatus(pipelinePB.Pipeline_STATUS_INACTIVATED) {
		return status.Error(codes.FailedPrecondition, "This pipeline is inactivated")
	}

	// Validation: Pipeline is error
	if pipeline.Status == datamodel.PipelineStatus(pipelinePB.Pipeline_STATUS_ERROR) {
		return status.Error(codes.FailedPrecondition, "This pipeline has errors")
	}

	// Pipeline not belong to this requester
	// TODO: Use owner_id to query owner_name in mgmt-backend
	if !strings.Contains(pipeline.FullName, "local-user") {
		return status.Error(codes.PermissionDenied, "You are not allowed to trigger this pipeline")
	}

	// TODO: The model that pipeline used is offline

	return nil
}

func (s *service) TriggerPipeline(ownerID uuid.UUID, req *pipelinePB.TriggerPipelineRequest, pipeline *datamodel.Pipeline) (*modelPB.TriggerModelResponse, error) {

	// Check if this is a direct trigger (i.e., HTTP, gRPC source and destination connectors)
	if util.Contains(constant.ConnectionTypeDirectness, pipeline.Recipe.Source.Name) &&
		util.Contains(constant.ConnectionTypeDirectness, pipeline.Recipe.Destination.Name) {

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		m := pipeline.Recipe.Models[0]

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
			ModelName:    m.ModelName,
			InstanceName: m.InstanceName,
			Inputs:       inputs,
		})

		if err != nil {
			return nil, status.Errorf(codes.Internal, "Error model-backend %s: %v", "TriggerModel", err.Error())
		}

		return resp, nil
	}

	return nil, nil

}

func (s *service) TriggerPipelineByUpload(ownerID uuid.UUID, image bytes.Buffer, pipeline *datamodel.Pipeline) (*modelPB.TriggerModelBinaryFileUploadResponse, error) {

	// Check if this is a direct trigger (i.e., HTTP, gRPC source and destination connectors)
	if util.Contains(constant.ConnectionTypeDirectness, pipeline.Recipe.Source.Name) &&
		util.Contains(constant.ConnectionTypeDirectness, pipeline.Recipe.Destination.Name) {

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
			ModelName:    pipeline.Recipe.Models[0].ModelName,
			InstanceName: pipeline.Recipe.Models[0].InstanceName,
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "cannot send data info to server: %s", err.Error())
		}

		const chunkSize = 64 * 1024
		buf := make([]byte, chunkSize)

		for {
			n, err := image.Read(buf)
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

func (s *service) ValidateModel(ownerID uuid.UUID, selectedModels []*datamodel.Model) error {

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
			if selectedModel.ModelName == supportModel.Name {
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
			invalidErrorString += fmt.Sprintf("The model name %s with its instance %s you specified is not applicable\n", selectedModel.ModelName, selectedModel.InstanceName)
		}
	}

	if hasInvalidModel {
		return status.Error(codes.InvalidArgument, invalidErrorString)
	}

	return nil

}
