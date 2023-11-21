package service

import (
	"context"
	"time"

	"github.com/gofrs/uuid"

	"github.com/instill-ai/pipeline-backend/internal/resource"
	controllerPB "github.com/instill-ai/protogen-go/vdp/controller/v1alpha"
	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1alpha"
)

func (s *service) GetPipelineState(pipelineUID uuid.UUID) (*pipelinePB.State, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resourcePermalink := ConvertResourceUIDToControllerResourcePermalink(pipelineUID, "pipeline_releases")

	resp, err := s.controllerClient.GetResource(ctx, &controllerPB.GetResourceRequest{
		ResourcePermalink: resourcePermalink,
	})

	if err != nil {
		return nil, err
	}

	return resp.Resource.GetPipelineState().Enum(), nil
}

func (s *service) UpdatePipelineState(pipelineUID uuid.UUID, state pipelinePB.State, progress *int32) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resourcePermalink := ConvertResourceUIDToControllerResourcePermalink(pipelineUID, "pipeline_releases")

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

func (s *service) DeletePipelineState(pipelineUID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resourcePermalink := ConvertResourceUIDToControllerResourcePermalink(pipelineUID, "pipeline_releases")

	_, err := s.controllerClient.DeleteResource(ctx, &controllerPB.DeleteResourceRequest{
		ResourcePermalink: resourcePermalink,
	})

	if err != nil {
		return err
	}

	return nil
}

func (s *service) GetConnectorState(connectorUID uuid.UUID) (*pipelinePB.Connector_State, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resourcePermalink := resource.ConvertConnectorToResourceName(connectorUID.String())

	resp, err := s.controllerClient.GetResource(ctx, &controllerPB.GetResourceRequest{
		ResourcePermalink: resourcePermalink,
	})

	if err != nil {
		return nil, err
	}

	return resp.Resource.GetConnectorState().Enum(), nil
}

func (s *service) UpdateConnectorState(connectorUID uuid.UUID, state pipelinePB.Connector_State, progress *int32) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resourcePermalink := resource.ConvertConnectorToResourceName(connectorUID.String())

	_, err := s.controllerClient.UpdateResource(ctx, &controllerPB.UpdateResourceRequest{
		Resource: &controllerPB.Resource{
			ResourcePermalink: resourcePermalink,
			State: &controllerPB.Resource_ConnectorState{
				ConnectorState: state,
			},
			Progress: progress,
		},
		WorkflowId: nil,
	})

	if err != nil {
		return err
	}

	return nil
}

func (s *service) DeleteConnectorState(connectorUID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resourcePermalink := resource.ConvertConnectorToResourceName(connectorUID.String())

	_, err := s.controllerClient.DeleteResource(ctx, &controllerPB.DeleteResourceRequest{
		ResourcePermalink: resourcePermalink,
	})

	if err != nil {
		return err
	}

	return nil
}
