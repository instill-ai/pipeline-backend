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
	"google.golang.org/grpc/codes"

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
	ListPipeline(owner string, pageSize int64, pageToken string, isBasicView bool) ([]datamodel.Pipeline, int64, string, error)
	GetPipelineByID(id string, owner string, isBasicView bool) (*datamodel.Pipeline, error)
	GetPipelineByUID(uid uuid.UUID, owner string, isBasicView bool) (*datamodel.Pipeline, error)
	UpdatePipeline(id string, owner string, updatedPipeline *datamodel.Pipeline) (*datamodel.Pipeline, error)
	DeletePipeline(id string, owner string) error
	UpdatePipelineState(id string, owner string, state datamodel.PipelineState) (*datamodel.Pipeline, error)
	UpdatePipelineID(id string, owner string, newID string) (*datamodel.Pipeline, error)
	TriggerPipeline(req *pipelinePB.TriggerPipelineRequest, pipeline *datamodel.Pipeline) (*modelPB.TriggerModelInstanceResponse, error)
	TriggerPipelineBinaryFileUpload(fileBuf bytes.Buffer, fileLengths []uint64, pipeline *datamodel.Pipeline) (*modelPB.TriggerModelInstanceBinaryFileUploadResponse, error)
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

	mode, err := s.getMode(dbPipeline.Recipe.Source, dbPipeline.Recipe.Destination)
	if err != nil {
		return nil, err
	}

	dbPipeline.Mode = mode

	ownerRscName := dbPipeline.Owner
	if err := s.ownerNameToPermalink(&dbPipeline.Owner); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

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

	dbCreatedPipeline, err := s.GetPipelineByID(dbPipeline.ID, ownerRscName, false)
	if err != nil {
		return nil, err
	}

	return dbCreatedPipeline, nil
}

func (s *service) ListPipeline(owner string, pageSize int64, pageToken string, isBasicView bool) ([]datamodel.Pipeline, int64, string, error) {

	if err := s.ownerNameToPermalink(&owner); err != nil {
		return nil, 0, "", status.Errorf(codes.InvalidArgument, err.Error())
	}

	dbPipelines, ps, pt, err := s.repository.ListPipeline(owner, pageSize, pageToken, isBasicView)
	if err != nil {
		return nil, 0, "", err
	}

	for _, dbPipeline := range dbPipelines {
		if err := s.ownerPermalinkToName(&dbPipeline.Owner); err != nil {
			return nil, 0, "", status.Errorf(codes.Internal, err.Error())
		}
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

func (s *service) GetPipelineByID(id string, owner string, isBasicView bool) (*datamodel.Pipeline, error) {

	if err := s.ownerNameToPermalink(&owner); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	dbPipeline, err := s.repository.GetPipelineByID(id, owner, isBasicView)
	if err != nil {
		return nil, err
	}

	if err := s.ownerPermalinkToName(&dbPipeline.Owner); err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	if !isBasicView {
		if err := s.recipePermalinkToName(dbPipeline.Recipe); err != nil {
			return nil, status.Errorf(codes.Internal, err.Error())
		}
	}

	return dbPipeline, nil
}

func (s *service) GetPipelineByUID(uid uuid.UUID, owner string, isBasicView bool) (*datamodel.Pipeline, error) {

	if err := s.ownerNameToPermalink(&owner); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	dbPipeline, err := s.repository.GetPipelineByUID(uid, owner, isBasicView)
	if err != nil {
		return nil, err
	}

	if err := s.ownerPermalinkToName(&dbPipeline.Owner); err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	if !isBasicView {
		if err := s.recipePermalinkToName(dbPipeline.Recipe); err != nil {
			return nil, status.Errorf(codes.Internal, err.Error())
		}
	}

	return dbPipeline, nil
}

func (s *service) UpdatePipeline(id string, owner string, updatedPipeline *datamodel.Pipeline) (*datamodel.Pipeline, error) {

	ownerRscName := owner
	if err := s.ownerNameToPermalink(&owner); err != nil {
		return nil, err
	}

	if err := s.ownerNameToPermalink(&updatedPipeline.Owner); err != nil {
		return nil, err
	}

	// Validation: Pipeline existence
	if existingPipeline, _ := s.repository.GetPipelineByID(id, owner, true); existingPipeline == nil {
		return nil, status.Errorf(codes.NotFound, "Pipeline id \"%s\" is not found", id)
	}

	if err := s.repository.UpdatePipeline(id, owner, updatedPipeline); err != nil {
		return nil, err
	}

	dbPipeline, err := s.GetPipelineByID(updatedPipeline.ID, ownerRscName, false)
	if err != nil {
		return nil, err
	}

	return dbPipeline, nil
}

func (s *service) DeletePipeline(id string, owner string) error {
	if err := s.ownerNameToPermalink(&owner); err != nil {
		return err
	}
	return s.repository.DeletePipeline(id, owner)
}

func (s *service) UpdatePipelineState(id string, owner string, state datamodel.PipelineState) (*datamodel.Pipeline, error) {

	if state == datamodel.PipelineState(pipelinePB.Pipeline_STATE_UNSPECIFIED) {
		return nil, status.Errorf(codes.InvalidArgument, "State update with unspecified is not allowed")
	}

	ownerRscName := owner
	if err := s.ownerNameToPermalink(&owner); err != nil {
		return nil, err
	}

	dbPipeline, err := s.GetPipelineByID(id, ownerRscName, false)
	if err != nil {
		return nil, err
	}

	mode, err := s.getMode(dbPipeline.Recipe.Source, dbPipeline.Recipe.Destination)
	if err != nil {
		return nil, err
	}

	if mode == datamodel.PipelineMode(pipelinePB.Pipeline_MODE_SYNC) && state == datamodel.PipelineState(pipelinePB.Pipeline_STATE_INACTIVE) {
		return nil, status.Errorf(codes.InvalidArgument, "Pipeline id \"%s\" is in the sync mode, which is always active", dbPipeline.ID)
	}

	if err := s.repository.UpdatePipelineState(id, owner, state); err != nil {
		return nil, err
	}

	dbPipeline, err = s.GetPipelineByID(id, ownerRscName, false)
	if err != nil {
		return nil, err
	}

	return dbPipeline, nil
}

func (s *service) UpdatePipelineID(id string, owner string, newID string) (*datamodel.Pipeline, error) {

	ownerRscName := owner
	if err := s.ownerNameToPermalink(&owner); err != nil {
		return nil, err
	}

	// Validation: Pipeline existence
	if existingPipeline, _ := s.repository.GetPipelineByID(id, owner, true); existingPipeline == nil {
		return nil, status.Errorf(codes.NotFound, "Pipeline id \"%s\" is not found", id)
	}

	if err := s.repository.UpdatePipelineID(id, owner, newID); err != nil {
		return nil, err
	}

	dbPipeline, err := s.GetPipelineByID(newID, ownerRscName, false)
	if err != nil {
		return nil, err
	}

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

func (s *service) TriggerPipeline(req *pipelinePB.TriggerPipelineRequest, dbPipeline *datamodel.Pipeline) (*modelPB.TriggerModelInstanceResponse, error) {

	if err := s.ownerNameToPermalink(&dbPipeline.Owner); err != nil {
		return nil, err
	}

	// Check if this is a direct trigger (i.e., HTTP, gRPC source and destination connectors)
	if dbPipeline.Mode == datamodel.PipelineMode(pipelinePB.Pipeline_MODE_SYNC) {

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
			Name:   dbPipeline.Recipe.ModelInstances[0],
			Inputs: inputs,
		})

		if err != nil {
			return nil, status.Errorf(codes.Internal, "Error model-backend %s: %v", "TriggerModel", err.Error())
		}

		// Increment trigger image numbers
		uid, err := resource.GetPermalinkUID(dbPipeline.Owner)
		if err != nil {
			return nil, err
		}
		if strings.HasPrefix(dbPipeline.Owner, "users/") {
			s.redisClient.IncrBy(ctx, fmt.Sprintf("user:%s:trigger.image.num", uid), int64(len(inputs)))
		} else if strings.HasPrefix(dbPipeline.Owner, "orgs/") {
			s.redisClient.IncrBy(ctx, fmt.Sprintf("org:%s:trigger.image.num", uid), int64(len(inputs)))
		}

		return resp, nil
	}

	return nil, nil
}

func (s *service) TriggerPipelineBinaryFileUpload(fileBuf bytes.Buffer, fileLengths []uint64, dbPipeline *datamodel.Pipeline) (*modelPB.TriggerModelInstanceBinaryFileUploadResponse, error) {

	if err := s.ownerNameToPermalink(&dbPipeline.Owner); err != nil {
		return nil, err
	}

	// Check if this is a direct trigger (i.e., HTTP, gRPC source and destination connectors)
	if dbPipeline.Mode == datamodel.PipelineMode(pipelinePB.Pipeline_MODE_SYNC) {

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
			Name:        dbPipeline.Recipe.ModelInstances[0],
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

		return res, nil
	}

	return nil, nil

}
