package service

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/instill-ai/pipeline-backend/internal/resource"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/logger"

	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
	mgmtPB "github.com/instill-ai/protogen-go/vdp/mgmt/v1alpha"
	modelPB "github.com/instill-ai/protogen-go/vdp/model/v1alpha"
	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1alpha"
)

func (s *service) recipeNameToPermalink(owner *mgmtPB.User, recipeRscName *datamodel.Recipe) (*datamodel.Recipe, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	recipePermalink := datamodel.Recipe{}

	// Source connector
	getSrcConnResp, err := s.connectorPublicServiceClient.GetSourceConnector(InjectOwnerToContext(ctx, owner),
		&connectorPB.GetSourceConnectorRequest{
			Name: recipeRscName.Source,
		})
	if err != nil {
		return nil, fmt.Errorf("[connector-backend] Error %s at source-connectors/%s: %s", "GetSourceConnector", recipeRscName.Source, err)
	}

	srcColID, err := resource.GetCollectionID(recipeRscName.Source)
	if err != nil {
		return nil, err
	}

	recipePermalink.Source = srcColID + "/" + getSrcConnResp.GetSourceConnector().GetUid()

	// Destination connector
	getDstConnResp, err := s.connectorPublicServiceClient.GetDestinationConnector(InjectOwnerToContext(ctx, owner),
		&connectorPB.GetDestinationConnectorRequest{
			Name: recipeRscName.Destination,
		})
	if err != nil {
		return nil, fmt.Errorf("[connector-backend] Error %s at destination-connectors/%s: %s", "GetDestinationConnector", recipeRscName.Destination, err)
	}

	dstColID, err := resource.GetCollectionID(recipeRscName.Destination)
	if err != nil {
		return nil, err
	}

	recipePermalink.Destination = dstColID + "/" + getDstConnResp.GetDestinationConnector().GetUid()

	// Model
	recipePermalink.Models = make([]string, len(recipeRscName.Models))
	for idx, modelRscName := range recipeRscName.Models {
		getModelResp, err := s.modelPublicServiceClient.GetModel(InjectOwnerToContext(ctx, owner),
			&modelPB.GetModelRequest{
				Name: modelRscName,
			})
		if err != nil {
			return nil, fmt.Errorf("[model-backend] Error %s at models/%s: %s", "GetModel", modelRscName, err)
		}

		modelColID, err := resource.GetCollectionID(modelRscName)
		if err != nil {
			return nil, err
		}

		recipePermalink.Models[idx] = modelColID + "/" + getModelResp.GetModel().GetUid()
	}

	return &recipePermalink, nil
}

func (s *service) recipePermalinkToNameAdmin(recipePermalink *datamodel.Recipe) (*datamodel.Recipe, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	recipeRscName := datamodel.Recipe{}

	// Source connector
	lookUpSrcConnResp, err := s.connectorPrivateServiceClient.LookUpSourceConnectorAdmin(ctx,
		&connectorPB.LookUpSourceConnectorAdminRequest{
			Permalink: recipePermalink.Source,
		})
	if err != nil {
		return nil, fmt.Errorf("[connector-backend] Error %s at source-connectors/%s: %s", "LookUpSourceConnector", recipePermalink.Source, err)
	}

	srcColID, err := resource.GetCollectionID(recipePermalink.Source)
	if err != nil {
		return nil, err
	}

	recipeRscName.Source = srcColID + "/" + lookUpSrcConnResp.GetSourceConnector().GetId()

	// Destination connector
	lookUpDstConnResp, err := s.connectorPrivateServiceClient.LookUpDestinationConnectorAdmin(ctx,
		&connectorPB.LookUpDestinationConnectorAdminRequest{
			Permalink: recipePermalink.Destination,
		})
	if err != nil {
		return nil, fmt.Errorf("[connector-backend] Error %s at destination-connectors/%s: %s", "LookUpDestinationConnector", recipePermalink.Destination, err)
	}

	dstColID, err := resource.GetCollectionID(recipePermalink.Destination)
	if err != nil {
		return nil, err
	}

	recipeRscName.Destination = dstColID + "/" + lookUpDstConnResp.GetDestinationConnector().GetId()

	// Models
	recipeRscName.Models = make([]string, len(recipePermalink.Models))
	for idx, modelRscPermalink := range recipePermalink.Models {

		lookUpModelResp, err := s.modelPrivateServiceClient.LookUpModelAdmin(ctx,
			&modelPB.LookUpModelAdminRequest{
				Permalink: modelRscPermalink,
			})
		if err != nil {
			return nil, fmt.Errorf("[model-backend] Error %s at models/%s: %s", "LookUpModel", modelRscPermalink, err)
		}

		modelColID, err := resource.GetCollectionID(modelRscPermalink)
		if err != nil {
			return nil, err
		}

		recipeRscName.Models[idx] = modelColID + "/" + lookUpModelResp.Model.GetId()
	}

	return &recipeRscName, nil
}

func (s *service) recipePermalinkToName(owner *mgmtPB.User, recipePermalink *datamodel.Recipe) (*datamodel.Recipe, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	recipeRscName := datamodel.Recipe{}

	// Source connector
	lookUpSrcConnResp, err := s.connectorPublicServiceClient.LookUpSourceConnector(InjectOwnerToContext(ctx, owner),
		&connectorPB.LookUpSourceConnectorRequest{
			Permalink: recipePermalink.Source,
		})
	if err != nil {
		return nil, fmt.Errorf("[connector-backend] Error %s at source-connectors/%s: %s", "LookUpSourceConnector", recipePermalink.Source, err)
	}

	srcColID, err := resource.GetCollectionID(recipePermalink.Source)
	if err != nil {
		return nil, err
	}

	recipeRscName.Source = srcColID + "/" + lookUpSrcConnResp.GetSourceConnector().GetId()

	// Destination connector
	lookUpDstConnResp, err := s.connectorPublicServiceClient.LookUpDestinationConnector(InjectOwnerToContext(ctx, owner),
		&connectorPB.LookUpDestinationConnectorRequest{
			Permalink: recipePermalink.Destination,
		})
	if err != nil {
		return nil, fmt.Errorf("[connector-backend] Error %s at destination-connectors/%s: %s", "LookUpDestinationConnector", recipePermalink.Destination, err)
	}

	dstColID, err := resource.GetCollectionID(recipePermalink.Destination)
	if err != nil {
		return nil, err
	}

	recipeRscName.Destination = dstColID + "/" + lookUpDstConnResp.GetDestinationConnector().GetId()

	// Models
	recipeRscName.Models = make([]string, len(recipePermalink.Models))
	for idx, modelRscPermalink := range recipePermalink.Models {

		lookUpModelResp, err := s.modelPublicServiceClient.LookUpModel(InjectOwnerToContext(ctx, owner),
			&modelPB.LookUpModelRequest{
				Permalink: modelRscPermalink,
			})
		if err != nil {
			return nil, fmt.Errorf("[model-backend] Error %s at models/%s: %s", "LookUpModel", modelRscPermalink, err)
		}

		modelColID, err := resource.GetCollectionID(modelRscPermalink)
		if err != nil {
			return nil, err
		}

		recipeRscName.Models[idx] = modelColID + "/" + lookUpModelResp.Model.GetId()
	}

	return &recipeRscName, nil
}

func cvtModelTaskOutputToPipelineTaskOutput(modelTaskOutputs []*modelPB.TaskOutput) []*pipelinePB.TaskOutput {

	logger, _ := logger.GetZapLogger()

	var pipelineTaskOutputs []*pipelinePB.TaskOutput
	for _, taskOutput := range modelTaskOutputs {
		switch v := taskOutput.Output.(type) {
		case *modelPB.TaskOutput_Classification:
			pipelineTaskOutputs = append(pipelineTaskOutputs, &pipelinePB.TaskOutput{
				Output: &pipelinePB.TaskOutput_Classification{
					Classification: proto.Clone(v.Classification).(*modelPB.ClassificationOutput),
				},
			})
		case *modelPB.TaskOutput_Detection:
			pipelineTaskOutputs = append(pipelineTaskOutputs, &pipelinePB.TaskOutput{
				Output: &pipelinePB.TaskOutput_Detection{
					Detection: proto.Clone(v.Detection).(*modelPB.DetectionOutput),
				},
			})
		case *modelPB.TaskOutput_Keypoint:
			pipelineTaskOutputs = append(pipelineTaskOutputs, &pipelinePB.TaskOutput{
				Output: &pipelinePB.TaskOutput_Keypoint{
					Keypoint: proto.Clone(v.Keypoint).(*modelPB.KeypointOutput),
				},
			})
		case *modelPB.TaskOutput_Ocr:
			pipelineTaskOutputs = append(pipelineTaskOutputs, &pipelinePB.TaskOutput{
				Output: &pipelinePB.TaskOutput_Ocr{
					Ocr: proto.Clone(v.Ocr).(*modelPB.OcrOutput),
				},
			})
		case *modelPB.TaskOutput_InstanceSegmentation:
			pipelineTaskOutputs = append(pipelineTaskOutputs, &pipelinePB.TaskOutput{
				Output: &pipelinePB.TaskOutput_InstanceSegmentation{
					InstanceSegmentation: proto.Clone(v.InstanceSegmentation).(*modelPB.InstanceSegmentationOutput),
				},
			})
		case *modelPB.TaskOutput_SemanticSegmentation:
			pipelineTaskOutputs = append(pipelineTaskOutputs, &pipelinePB.TaskOutput{
				Output: &pipelinePB.TaskOutput_SemanticSegmentation{
					SemanticSegmentation: proto.Clone(v.SemanticSegmentation).(*modelPB.SemanticSegmentationOutput),
				},
			})
		case *modelPB.TaskOutput_TextToImage:
			pipelineTaskOutputs = append(pipelineTaskOutputs, &pipelinePB.TaskOutput{
				Output: &pipelinePB.TaskOutput_TextToImage{
					TextToImage: proto.Clone(v.TextToImage).(*modelPB.TextToImageOutput),
				},
			})
		case *modelPB.TaskOutput_TextGeneration:
			pipelineTaskOutputs = append(pipelineTaskOutputs, &pipelinePB.TaskOutput{
				Output: &pipelinePB.TaskOutput_TextGeneration{
					TextGeneration: proto.Clone(v.TextGeneration).(*modelPB.TextGenerationOutput),
				},
			})
		case *modelPB.TaskOutput_Unspecified:
			pipelineTaskOutputs = append(pipelineTaskOutputs, &pipelinePB.TaskOutput{
				Output: &pipelinePB.TaskOutput_Unspecified{
					Unspecified: proto.Clone(v.Unspecified).(*modelPB.UnspecifiedOutput),
				},
			})
		default:
			logger.Error("AI task type is not defined")
		}
	}

	return pipelineTaskOutputs
}

func ConvertResourceUIDToControllerResourcePermalink(resourceUID string, resourceType string) string {
	resourcePermalink := fmt.Sprintf("resources/%s/types/%s", resourceUID, resourceType)

	return resourcePermalink
}
