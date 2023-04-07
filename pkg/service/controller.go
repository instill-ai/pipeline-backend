package service

import (
	"context"
	"time"

	controllerPB "github.com/instill-ai/protogen-go/vdp/controller/v1alpha"
	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1alpha"
)

func (s *service) GetResourceState(pipelineName string) (*pipelinePB.Pipeline_State, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resourceName := ConvertPipelineToResourceName(pipelineName)

	resp, err := s.controllerClient.GetResource(ctx, &controllerPB.GetResourceRequest{
		Name: resourceName,
	})

	if err != nil {
		return nil, err
	}

	return resp.Resource.GetPipelineState().Enum(), nil
}

func (s *service) UpdateResourceState(pipelineName string, state pipelinePB.Pipeline_State, progress *int32) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resourceName := ConvertPipelineToResourceName(pipelineName)

	_, err := s.controllerClient.UpdateResource(ctx, &controllerPB.UpdateResourceRequest{
		Resource: &controllerPB.Resource{
			Name: resourceName,
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

func (s *service) DeleteResourceState(pipelineName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resourceName := ConvertPipelineToResourceName(pipelineName)

	_, err := s.controllerClient.DeleteResource(ctx, &controllerPB.DeleteResourceRequest{
		Name: resourceName,
	})

	if err != nil {
		return err
	}

	return nil
}
