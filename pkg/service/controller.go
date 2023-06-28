package service

import (
	"context"
	"time"

	"github.com/gofrs/uuid"

	controllerPB "github.com/instill-ai/protogen-go/vdp/controller/v1alpha"
	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1alpha"
)

func (s *service) GetResourceState(pipelineUID uuid.UUID) (*pipelinePB.Pipeline_State, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resourcePermalink := ConvertResourceUIDToControllerResourcePermalink(pipelineUID.String(), "pipelines")

	resp, err := s.controllerClient.GetResource(ctx, &controllerPB.GetResourceRequest{
		ResourcePermalink: resourcePermalink,
	})

	if err != nil {
		return nil, err
	}

	return resp.Resource.GetPipelineState().Enum(), nil
}

func (s *service) UpdateResourceState(pipelineUID uuid.UUID, state pipelinePB.Pipeline_State, progress *int32) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resourcePermalink := ConvertResourceUIDToControllerResourcePermalink(pipelineUID.String(), "pipelines")

	_, err := s.controllerClient.UpdateResource(ctx, &controllerPB.UpdateResourceRequest{
		Resource: &controllerPB.Resource{
			ResourcePermalink: resourcePermalink,
			State: &controllerPB.Resource_PipelineState{
				PipelineState: state,
			},
			Progress: progress,
		},
	})

	if err != nil {
		return err
	}

	return nil
}

func (s *service) DeleteResourceState(pipelineUID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resourcePermalink := ConvertResourceUIDToControllerResourcePermalink(pipelineUID.String(), "pipelines")

	_, err := s.controllerClient.DeleteResource(ctx, &controllerPB.DeleteResourceRequest{
		ResourcePermalink: resourcePermalink,
	})

	if err != nil {
		return err
	}

	return nil
}
