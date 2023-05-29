package utils

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/instill-ai/pipeline-backend/internal/resource"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/logger/otel"
	"go.opentelemetry.io/otel/trace"

	mgmtPB "github.com/instill-ai/protogen-go/vdp/mgmt/v1alpha"
	modelPB "github.com/instill-ai/protogen-go/vdp/model/v1alpha"
	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1alpha"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
)

type DefinitionType int64

const (
	DefinitionTypeUnspecified DefinitionType = 0
	SourceConnector           DefinitionType = 1
	DestinationConnector      DefinitionType = 2
	Model                     DefinitionType = 3
)

func GenOwnerPermalink(owner *mgmtPB.User) string {
	return "users/" + owner.GetUid()
}

func InjectOwnerToContext(ctx context.Context, owner *mgmtPB.User) context.Context {
	ctx = metadata.AppendToOutgoingContext(ctx, "Jwt-Sub", owner.GetUid())
	return ctx
}
func InjectOwnerToContextWithOwnerPermalink(ctx context.Context, permalink string) context.Context {
	uid, _ := resource.GetPermalinkUID(permalink)
	ctx = metadata.AppendToOutgoingContext(ctx, "Jwt-Sub", uid)
	return ctx
}

func GetDefinitionType(component *datamodel.Component) DefinitionType {
	if i := strings.Index(component.ResourceName, "/"); i >= 0 {
		switch component.ResourceName[:i] {
		case "source-connectors":
			return SourceConnector
		case "destination-connectors":
			return DestinationConnector
		case "models":
			return Model
		}
	}
	return DefinitionTypeUnspecified
}

func GetResourceFromRecipe(recipe *datamodel.Recipe, t DefinitionType) []string {
	resources := []string{}
	for _, component := range recipe.Components {
		switch GetDefinitionType(component) {
		case t:
			resources = append(resources, component.ResourceName)
		}
	}
	return resources
}

func GetModelsFromRecipe(recipe *datamodel.Recipe) []string {
	return GetResourceFromRecipe(recipe, Model)
}

func GetDestinationsFromRecipe(recipe *datamodel.Recipe) []string {
	return GetResourceFromRecipe(recipe, DestinationConnector)
}

func GetSourcesFromRecipe(recipe *datamodel.Recipe) []string {
	return GetResourceFromRecipe(recipe, SourceConnector)
}

func CvtModelTaskOutputToPipelineTaskOutput(modelTaskOutputs []*modelPB.TaskOutput) []*pipelinePB.TaskOutput {

	// logger, _ := logger.GetZapLogger()

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
			// logger.Error("AI task type is not defined")
		}
	}

	return pipelineTaskOutputs
}

type TextToImageInput struct {
	Prompt   string
	Steps    int64
	CfgScale float32
	Seed     int64
	Samples  int64
}

type TextGenerationInput struct {
	Prompt        string
	OutputLen     int64
	BadWordsList  string
	StopWordsList string
	TopK          int64
	Seed          int64
}

type ImageInput struct {
	Content     []byte
	FileNames   []string
	FileLengths []uint64
}

func ConstructAuditLog(
	span trace.Span,
	user mgmtPB.User,
	pipeline datamodel.Pipeline,
	eventName string,
	billable bool,
	metadata string,
) []byte {
	logMessage, _ := json.Marshal(otel.AuditLogMessage{
		ServiceName: "pipeline-backend",
		TraceInfo: struct {
			TraceId string
			SpanId  string
		}{
			TraceId: span.SpanContext().TraceID().String(),
			SpanId:  span.SpanContext().SpanID().String(),
		},
		UserInfo: struct {
			UserID   string
			UserUUID string
			Token    string
		}{
			UserID:   user.Id,
			UserUUID: *user.Uid,
			Token:    *user.CookieToken,
		},
		EventInfo: struct{ Name string }{
			Name: eventName,
		},
		ResourceInfo: struct {
			ResourceName  string
			ResourceUUID  string
			ResourceState string
			Billable      bool
		}{
			ResourceName:  pipeline.ID,
			ResourceUUID:  pipeline.UID.String(),
			ResourceState: pipelinePB.Pipeline_State(pipeline.State).String(),
			Billable:      billable,
		},
		Metadata: metadata,
	})

	return logMessage
}

func ConstructErrorLog(
	span trace.Span,
	statusCode int,
	errorMessage string,
) []byte {
	logMessage, _ := json.Marshal(otel.ErrorLogMessage{
		ServiceName: "pipeline-backend",
		TraceInfo: struct {
			TraceId string
			SpanId  string
		}{
			TraceId: span.SpanContext().TraceID().String(),
			SpanId:  span.SpanContext().SpanID().String(),
		},
		StatusCode:   statusCode,
		ErrorMessage: errorMessage,
	})

	return logMessage
}
