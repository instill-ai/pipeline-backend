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
	ListPipeline(owner string, view pipelinePB.View, pageSize int, pageToken string) ([]datamodel.Pipeline, string, error)
	GetPipelineByID(id string, owner string) (*datamodel.Pipeline, error)
	UpdatePipeline(uid uuid.UUID, owner string, updatedPipeline *datamodel.Pipeline) (*datamodel.Pipeline, error)
	DeletePipeline(uid uuid.UUID, owner string) error
	TriggerPipeline(req *pipelinePB.TriggerPipelineRequest, pipeline *datamodel.Pipeline) (*modelPB.TriggerModelInstanceResponse, error)
	TriggerPipelineBinaryFileUpload(fileBuf bytes.Buffer, fileLengths []uint64, pipeline *datamodel.Pipeline) (*modelPB.TriggerModelInstanceBinaryFileUploadResponse, error)
	ValidatePipeline(pipeline *datamodel.Pipeline) error
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

	// Validatation: name naming rule
	if match, _ := regexp.MatchString("^[A-Za-z0-9][a-zA-Z0-9_.-]*$", pipeline.ID); !match {
		return nil, status.Error(codes.FailedPrecondition, "The id of pipeline is invalid")
	}

	// Validation: name length
	if len(pipeline.ID) > 63 {
		return nil, status.Error(codes.FailedPrecondition, "The id of pipeline has more than 63 characters")
	}

	// Determine pipeline mode
	if util.Contains(constant.ConnectionTypeDirectness, pipeline.Recipe.Source) &&
		util.Contains(constant.ConnectionTypeDirectness, pipeline.Recipe.Destination) {
		if pipeline.Recipe.Source == pipeline.Recipe.Destination {
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

	dbPipeline, err := s.GetPipelineByID(pipeline.ID, pipeline.Owner)
	if err != nil {
		return nil, err
	}

	return dbPipeline, nil
}

func (s *service) ListPipeline(owner string, view pipelinePB.View, pageSize int, pageToken string) ([]datamodel.Pipeline, string, error) {
	return s.repository.ListPipeline(owner, view, pageSize, pageToken)
}

func (s *service) GetPipelineByID(id string, owner string) (*datamodel.Pipeline, error) {
	dbPipeline, err := s.repository.GetPipelineByID(id, owner)
	if err != nil {
		return nil, err
	}

	return dbPipeline, nil
}

func (s *service) UpdatePipeline(uid uuid.UUID, owner string, updatedPipeline *datamodel.Pipeline) (*datamodel.Pipeline, error) {

	// Validation: Pipeline existence
	if existingPipeline, _ := s.repository.GetPipeline(uid, owner); existingPipeline == nil {
		return nil, status.Errorf(codes.NotFound, "Pipeline id \"%s\" is not found", uid.String())
	}

	// Validatation: id naming rule
	if match, _ := regexp.MatchString("^[A-Za-z0-9][a-zA-Z0-9_.-]*$", updatedPipeline.ID); !match {
		return nil, status.Error(codes.FailedPrecondition, "The id of pipeline is invalid")
	}

	// Validation: id length
	if len(updatedPipeline.ID) > 63 {
		return nil, status.Error(codes.FailedPrecondition, "The id of pipeline has more than 63 characters")
	}

	if err := s.repository.UpdatePipeline(uid, owner, updatedPipeline); err != nil {
		return nil, err
	}

	dbPipeline, err := s.GetPipelineByID(updatedPipeline.ID, owner)
	if err != nil {
		return nil, err
	}

	return dbPipeline, nil
}

func (s *service) DeletePipeline(uid uuid.UUID, owner string) error {
	return s.repository.DeletePipeline(uid, owner)
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

func (s *service) TriggerPipeline(req *pipelinePB.TriggerPipelineRequest, pipeline *datamodel.Pipeline) (*modelPB.TriggerModelInstanceResponse, error) {

	// Check if this is a direct trigger (i.e., HTTP, gRPC source and destination connectors)
	if pipeline.Mode == datamodel.PipelineMode(pipelinePB.Pipeline_MODE_SYNC) {

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

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

		resp, err := s.modelServiceClient.TriggerModelInstance(ctx, &modelPB.TriggerModelInstanceRequest{
			Name:   pipeline.Recipe.ModelInstances[0],
			Inputs: inputs,
		})

		if err != nil {
			return nil, status.Errorf(codes.Internal, "Error model-backend %s: %v", "TriggerModel", err.Error())
		}

		return resp, nil
	}

	return nil, nil

}

func (s *service) TriggerPipelineBinaryFileUpload(fileBuf bytes.Buffer, fileLengths []uint64, pipeline *datamodel.Pipeline) (*modelPB.TriggerModelInstanceBinaryFileUploadResponse, error) {

	// Check if this is a direct trigger (i.e., HTTP, gRPC source and destination connectors)
	if pipeline.Mode == datamodel.PipelineMode(pipelinePB.Pipeline_MODE_SYNC) {

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		stream, err := s.modelServiceClient.TriggerModelInstanceBinaryFileUpload(ctx)
		defer func() {
			_ = stream.CloseSend()
		}()
		if err != nil {
			return nil, fmt.Errorf("Error model-backend %s: %v", "TriggerModelBinaryFileUpload", err.Error())
		}

		err = stream.Send(&modelPB.TriggerModelInstanceBinaryFileUploadRequest{
			Name:        pipeline.Recipe.ModelInstances[0],
			FileLengths: fileLengths,
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

			err = stream.Send(&modelPB.TriggerModelInstanceBinaryFileUploadRequest{Bytes: buf[:n]})
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
