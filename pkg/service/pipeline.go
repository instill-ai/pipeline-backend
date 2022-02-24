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
	pipelinePB "github.com/instill-ai/protogen-go/pipeline"
	"google.golang.org/grpc/codes"
)

type Services interface {
	CreatePipeline(pipeline model.Pipeline) (model.Pipeline, error)
	ListPipelines(query model.ListPipelineQuery) ([]model.Pipeline, uint64, uint64, error)
	GetPipelineByName(namespace string, pipelineName string) (model.Pipeline, error)
	UpdatePipeline(pipeline model.Pipeline) (model.Pipeline, error)
	DeletePipeline(namespace string, pipelineName string) error
	TriggerPipeline(namespace string, trigger pipelinePB.TriggerPipelineRequest, pipeline model.Pipeline) (interface{}, error)
	ValidateTriggerPipeline(namespace string, pipelineName string, pipeline model.Pipeline) error
	TriggerPipelineByUpload(namespace string, buf bytes.Buffer, pipeline model.Pipeline) (interface{}, error)
	ValidateModel(namespace string, selectedModel []*model.Model) error
}

type PipelineService struct {
	PipelineRepository repository.Operations
	ModelServiceClient modelPB.ModelClient
}

func NewPipelineService(r repository.Operations, modelServiceClient modelPB.ModelClient) Services {
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

func (p *PipelineService) TriggerPipeline(namespace string, trigger pipelinePB.TriggerPipelineRequest, pipeline model.Pipeline) (interface{}, error) {

	// TODO: The model that pipeline used is offline

	if temporal.IsDirect(pipeline.Recipe) {

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		vdo := pipeline.Recipe.Model[0]

		var contents []*modelPB.ImageRequest
		for _, content := range trigger.Contents {
			contents = append(contents, &modelPB.ImageRequest{
				Url:    content.Url,
				Base64: content.Base64,
			})
		}

		ret, err := p.ModelServiceClient.PredictModel(ctx, &modelPB.PredictModelImageRequest{
			Name:     vdo.Name,
			Version:  vdo.Version,
			Contents: contents,
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "cannot make inference: %s", err.Error())
		}

		fmt.Printf("%+v\n", ret)

		return ret, nil
	} else {
		return nil, nil
	}
}

func (p *PipelineService) TriggerPipelineByUpload(namespace string, image bytes.Buffer, pipeline model.Pipeline) (interface{}, error) {

	if temporal.IsDirect(pipeline.Recipe) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		stream, err := p.ModelServiceClient.PredictModelByUpload(ctx)
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

func (p *PipelineService) ValidateModel(namespace string, selectedModels []*model.Model) error {

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	supportModelResp, err := p.ModelServiceClient.ListModels(ctx, &modelPB.ListModelRequest{})
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
