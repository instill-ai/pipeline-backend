package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/gogo/status"
	"github.com/instill-ai/pipeline-backend/internal/temporal"
	model "github.com/instill-ai/pipeline-backend/pkg/model"
	"github.com/instill-ai/pipeline-backend/pkg/repository"
	modelPB "github.com/instill-ai/protogen-go/model"
	"google.golang.org/grpc/codes"
)

type PipelineService interface {
	CreatePipeline(pipeline model.Pipeline) (model.Pipeline, error)
	ListPipelines(query model.ListPipelineQuery) ([]model.Pipeline, uint64, uint64, error)
	GetPipelineByName(namespace string, pipelineName string) (model.Pipeline, error)
	UpdatePipeline(pipeline model.Pipeline) (model.Pipeline, error)
	DeletePipeline(namespace string, pipelineName string) error
	ValidateTriggerPipeline(namespace string, pipelineName string, pipeline model.Pipeline) error
	TriggerPipelineByUpload(namespace string, buf bytes.Buffer, pipeline model.Pipeline) (interface{}, error)
	ValidateModel(namespace string, selectedModel []*model.Model) error
}

type pipelineService struct {
	pipelineRepository repository.PipelineRepository
	modelServiceClient modelPB.ModelClient
}

func NewPipelineService(r repository.PipelineRepository, modelServiceClient modelPB.ModelClient) PipelineService {
	return &pipelineService{
		pipelineRepository: r,
		modelServiceClient: modelServiceClient,
	}
}

func (p *pipelineService) CreatePipeline(pipeline model.Pipeline) (model.Pipeline, error) {

	// Validate the naming rule of pipeline
	if match, _ := regexp.MatchString("^[A-Za-z0-9][a-zA-Z0-9_.-]*$", pipeline.Name); !match {
		return model.Pipeline{}, status.Error(codes.FailedPrecondition, "The name of pipeline is invalid")
	}

	// TODO: validation
	if pipeline.Name == "" {
		return model.Pipeline{}, status.Error(codes.FailedPrecondition, "The required field name not specify")
	}

	if existingPipeline, _ := p.GetPipelineByName(pipeline.Namespace, pipeline.Name); existingPipeline.Name != "" {
		return model.Pipeline{}, status.Errorf(codes.FailedPrecondition, "The name %s is existing in your namespace", pipeline.Name)
	}

	err := p.ValidateModel(pipeline.Namespace, pipeline.Recipe.Model)
	if err != nil {
		return model.Pipeline{}, err
	}

	if err := p.pipelineRepository.CreatePipeline(pipeline); err != nil {
		return model.Pipeline{}, err
	}

	if createdPipeline, err := p.GetPipelineByName(pipeline.Namespace, pipeline.Name); err != nil {
		return model.Pipeline{}, err
	} else {
		return createdPipeline, nil
	}
}

func (p *pipelineService) ListPipelines(query model.ListPipelineQuery) ([]model.Pipeline, uint64, uint64, error) {
	return p.pipelineRepository.ListPipelines(query)
}

func (p *pipelineService) GetPipelineByName(namespace string, pipelineName string) (model.Pipeline, error) {
	return p.pipelineRepository.GetPipelineByName(namespace, pipelineName)
}

func (p *pipelineService) UpdatePipeline(pipeline model.Pipeline) (model.Pipeline, error) {

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

	if err := p.pipelineRepository.UpdatePipeline(pipeline); err != nil {
		return model.Pipeline{}, err
	}

	if updatedPipeline, err := p.GetPipelineByName(pipeline.Namespace, pipeline.Name); err != nil {
		return model.Pipeline{}, err
	} else {
		return updatedPipeline, nil
	}
}

func (p *pipelineService) DeletePipeline(namespace string, pipelineName string) error {
	return p.pipelineRepository.DeletePipeline(namespace, pipelineName)
}

func (p *pipelineService) ValidateTriggerPipeline(namespace string, pipelineName string, pipeline model.Pipeline) error {

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

func (p *pipelineService) TriggerPipelineByUpload(namespace string, image bytes.Buffer, pipeline model.Pipeline) (interface{}, error) {

	if temporal.IsDirect(pipeline.Recipe) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		stream, err := p.modelServiceClient.PredictModelByUpload(ctx)
		defer stream.CloseSend()
		if err != nil {
			return nil, err
		}

		err = stream.Send(&modelPB.PredictModelRequest{
			Name:    pipeline.Recipe.Model[0].Name,
			Version: pipeline.Recipe.Model[0].Version,
		})
		if err != nil {
			status.Errorf(codes.Internal, "cannot send data info to server: %s", err.Error())
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

			err = stream.Send(&modelPB.PredictModelRequest{Content: buf[:n]})
			if err != nil {
				status.Errorf(codes.Internal, "cannot send chunk to server: %s", err)
			}
		}
		res, err := stream.CloseAndRecv()
		if err != nil {
			log.Fatal("cannot receive response: ", err)
			status.Errorf(codes.Internal, "cannot receive response: %s", err.Error())
		}

		return res, nil
	} else {
		return nil, nil
	}
}

func (p *pipelineService) ValidateModel(namespace string, selectedModels []*model.Model) error {

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	supportModelResp, err := p.modelServiceClient.ListModels(ctx, &modelPB.ListModelRequest{})
	if err != nil {
		return err
	}

	hasInvalidModel := false
	invalidErrorString := ""
	for _, selectedModel := range selectedModels {
		matchModel := false
		for _, supportModel := range supportModelResp.Models {
			if selectedModel.Name == supportModel.Name {
				for _, supportVersion := range supportModel.Versions {
					if selectedModel.Version == supportVersion.Version {
						if supportVersion.Status == modelPB.ModelStatus_ONLINE {
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
