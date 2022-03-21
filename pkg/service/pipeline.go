package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/gogo/status"
	"google.golang.org/grpc/codes"

	"github.com/instill-ai/pipeline-backend/internal/temporal"
	"github.com/instill-ai/pipeline-backend/pkg/repository"

	model "github.com/instill-ai/pipeline-backend/pkg/model"
	modelPB "github.com/instill-ai/protogen-go/model/v1alpha"
	pipelinePB "github.com/instill-ai/protogen-go/pipeline/v1alpha"
)

type Services interface {
	CreatePipeline(pipeline model.Pipeline) (model.Pipeline, error)
	ListPipelines(query model.ListPipelineQuery) ([]model.Pipeline, uint64, uint64, error)
	GetPipelineByName(namespace string, pipelineName string) (model.Pipeline, error)
	UpdatePipeline(pipeline model.Pipeline) (model.Pipeline, error)
	DeletePipeline(namespace string, pipelineName string) error
	TriggerPipeline(namespace string, trigger *pipelinePB.TriggerPipelineRequest, pipeline model.Pipeline) (*modelPB.TriggerModelResponse, error)
	ValidateTriggerPipeline(namespace string, pipelineName string, pipeline model.Pipeline) error
	TriggerPipelineByUpload(namespace string, buf bytes.Buffer, pipeline model.Pipeline) (*modelPB.TriggerModelBinaryFileUploadResponse, error)
	ValidateModel(namespace string, selectedModel []*model.Model) error
}

type PipelineService struct {
	PipelineRepository repository.Operations
	ModelServiceClient modelPB.ModelServiceClient
}

func NewPipelineService(r repository.Operations, modelServiceClient modelPB.ModelServiceClient) Services {
	return &PipelineService{
		PipelineRepository: r,
		ModelServiceClient: modelServiceClient,
	}
}

func (p *PipelineService) CreatePipeline(pipeline model.Pipeline) (model.Pipeline, error) {

	// TODO: more validation
	if pipeline.Name == "" {
		return model.Pipeline{}, status.Error(codes.FailedPrecondition, "The required field name is not specified")
	}

	// Validate the naming rule of pipeline
	if match, _ := regexp.MatchString("^[A-Za-z0-9][a-zA-Z0-9_.-]*$", pipeline.Name); !match {
		return model.Pipeline{}, status.Error(codes.FailedPrecondition, "The name of pipeline is invalid")
	}

	if len(pipeline.Name) > 100 {
		return model.Pipeline{}, status.Error(codes.FailedPrecondition, "The length of the name is greater than 100")
	}

	if existingPipeline, _ := p.GetPipelineByName(pipeline.Namespace, pipeline.Name); existingPipeline.Name != "" {
		return model.Pipeline{}, status.Errorf(codes.FailedPrecondition, "The name %s is existing in your namespace", pipeline.Name)
	}

	if pipeline.Recipe != nil && pipeline.Recipe.Model != nil && len(pipeline.Recipe.Model) > 0 {
		err := p.ValidateModel(pipeline.Namespace, pipeline.Recipe.Model)
		if err != nil {
			return model.Pipeline{}, err
		}
	}

	if err := p.PipelineRepository.CreatePipeline(pipeline); err != nil {
		return model.Pipeline{}, err
	}

	if createdPipeline, err := p.GetPipelineByName(pipeline.Namespace, pipeline.Name); err != nil {
		return model.Pipeline{}, err
	} else {
		return createdPipeline, nil
	}
}

func (p *PipelineService) ListPipelines(query model.ListPipelineQuery) ([]model.Pipeline, uint64, uint64, error) {
	return p.PipelineRepository.ListPipelines(query)
}

func (p *PipelineService) GetPipelineByName(namespace string, pipelineName string) (model.Pipeline, error) {
	return p.PipelineRepository.GetPipelineByName(namespace, pipelineName)
}

func (p *PipelineService) UpdatePipeline(pipeline model.Pipeline) (model.Pipeline, error) {

	// TODO: validation
	if pipeline.Name == "" {
		return model.Pipeline{}, status.Error(codes.FailedPrecondition, "The required field name not specify")
	}

	if existingPipeline, _ := p.GetPipelineByName(pipeline.Namespace, pipeline.Name); existingPipeline.Name == "" {
		return model.Pipeline{}, status.Errorf(codes.NotFound, "The pipeline name %s you specified is not found", pipeline.Name)
	}

	if pipeline.Recipe != nil && pipeline.Recipe.Model != nil && len(pipeline.Recipe.Model) > 0 {
		err := p.ValidateModel(pipeline.Namespace, pipeline.Recipe.Model)
		if err != nil {
			return model.Pipeline{}, err
		}
	}

	if err := p.PipelineRepository.UpdatePipeline(pipeline); err != nil {
		return model.Pipeline{}, err
	}

	if updatedPipeline, err := p.GetPipelineByName(pipeline.Namespace, pipeline.Name); err != nil {
		return model.Pipeline{}, err
	} else {
		return updatedPipeline, nil
	}
}

func (p *PipelineService) DeletePipeline(namespace string, pipelineName string) error {
	return p.PipelineRepository.DeletePipeline(namespace, pipelineName)
}

func (p *PipelineService) ValidateTriggerPipeline(namespace string, pipelineName string, pipeline model.Pipeline) error {

	// Specified pipeline not exists
	if pipeline.Name == "" {
		return status.Errorf(codes.NotFound, "The pipeline name %s you specified is not found", pipelineName)
	}

	// Pipeline is inactive
	if !pipeline.Active {
		return status.Error(codes.FailedPrecondition, "This pipeline has been deactivated")
	}

	// Pipeline not belong to this requester
	if !strings.Contains(pipeline.FullName, namespace) {
		return status.Error(codes.PermissionDenied, "You are not allowed to trigger this pipeline")
	}

	// TODO: The model that pipeline used is offline

	return nil
}

func (p *PipelineService) TriggerPipeline(namespace string, req *pipelinePB.TriggerPipelineRequest, pipeline model.Pipeline) (*modelPB.TriggerModelResponse, error) {

	// TODO: The model that pipeline used is offline
	if temporal.IsDirect(pipeline.Recipe) {

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		m := pipeline.Recipe.Model[0]

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

		ret, err := p.ModelServiceClient.TriggerModel(ctx, &modelPB.TriggerModelRequest{
			Name:    m.Name,
			Version: m.Version,
			Inputs:  inputs,
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "cannot make inference: %s", err.Error())
		}

		return ret, nil
	}

	return nil, nil

}

func (p *PipelineService) TriggerPipelineByUpload(namespace string, image bytes.Buffer, pipeline model.Pipeline) (*modelPB.TriggerModelBinaryFileUploadResponse, error) {

	if temporal.IsDirect(pipeline.Recipe) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		stream, err := p.ModelServiceClient.TriggerModelBinaryFileUpload(ctx)
		defer func() {
			_ = stream.CloseSend()
		}()
		if err != nil {
			return nil, err
		}

		err = stream.Send(&modelPB.TriggerModelBinaryFileUploadRequest{
			Name:    pipeline.Recipe.Model[0].Name,
			Version: pipeline.Recipe.Model[0].Version,
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
	} else {
		return nil, nil
	}
}

func (p *PipelineService) ValidateModel(namespace string, selectedModels []*model.Model) error {

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	supportModelResp, err := p.ModelServiceClient.ListModel(ctx, &modelPB.ListModelRequest{})
	if err != nil {
		return err
	}

	hasInvalidModel := false
	invalidErrorString := ""
	for _, selectedModel := range selectedModels {
		matchModel := false
		for _, supportModel := range supportModelResp.Models {
			if selectedModel.Name == supportModel.Name {
				for _, supportVersion := range supportModel.ModelVersions {
					if selectedModel.Version == supportVersion.Version {
						if supportVersion.Status == modelPB.ModelVersion_STATUS_ONLINE {
							matchModel = true
							break
						}
					}
				}
			}
		}
		if !matchModel {
			hasInvalidModel = true
			invalidErrorString += fmt.Sprintf("The model %s and version %d you specified is not applicable\n", selectedModel.Name, selectedModel.Version)
		}
	}

	if hasInvalidModel {
		return status.Error(codes.InvalidArgument, invalidErrorString)
	} else {
		return nil
	}
}
