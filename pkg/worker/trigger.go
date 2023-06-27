package worker

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/go-redis/redis/v9"
	"github.com/gogo/status"
	"github.com/instill-ai/pipeline-backend/internal/resource"
	"github.com/instill-ai/pipeline-backend/pkg/utils"
	modelPB "github.com/instill-ai/protogen-go/model/model/v1alpha"
	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1alpha"
	"google.golang.org/grpc/codes"
)

func Trigger(mClient modelPB.ModelPublicServiceClient, rClient *redis.Client, taskInputs []*modelPB.TaskInput, dataMappingIndices []string, model string, ownerPermalink string) (*pipelinePB.ModelOutput, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	resp, err := mClient.TriggerModel(utils.InjectOwnerToContextWithOwnerPermalink(ctx, ownerPermalink), &modelPB.TriggerModelRequest{
		Name:       model,
		TaskInputs: taskInputs,
	})
	if err != nil {
		return nil, err
	}

	taskOutputs := utils.CvtModelTaskOutputToPipelineTaskOutput(ctx, resp.TaskOutputs)
	for idx, taskOutput := range taskOutputs {
		taskOutput.Index = dataMappingIndices[idx]
	}

	modelOutput := &pipelinePB.ModelOutput{
		Model:       model,
		Task:        resp.Task,
		TaskOutputs: taskOutputs,
	}

	// Increment trigger image numbers
	uid, err := resource.GetPermalinkUID(ownerPermalink)
	if err != nil {
		return nil, err
	}
	if strings.HasPrefix(ownerPermalink, "users/") {
		rClient.IncrBy(context.Background(), fmt.Sprintf("user:%s:trigger.num", uid), int64(len(taskInputs)))
	} else if strings.HasPrefix(ownerPermalink, "orgs/") {
		rClient.IncrBy(context.Background(), fmt.Sprintf("org:%s:trigger.num", uid), int64(len(taskInputs)))
	}

	return modelOutput, nil
}

func TriggerBinaryFileUpload(mClient modelPB.ModelPublicServiceClient, rClient *redis.Client, task modelPB.Model_Task, input interface{}, dataMappingIndices []string, model string, ownerPermalink string) (*pipelinePB.ModelOutput, error) {

	var modelOutput *pipelinePB.ModelOutput
	var err error

	switch task {
	case modelPB.Model_TASK_CLASSIFICATION,
		modelPB.Model_TASK_DETECTION,
		modelPB.Model_TASK_KEYPOINT,
		modelPB.Model_TASK_OCR,
		modelPB.Model_TASK_INSTANCE_SEGMENTATION,
		modelPB.Model_TASK_SEMANTIC_SEGMENTATION:
		modelOutput, err = TriggerImageTask(mClient, rClient, task, input, dataMappingIndices, model, ownerPermalink)
		if err != nil {
			return nil, status.Errorf(codes.Internal, err.Error())
		}
	case modelPB.Model_TASK_TEXT_TO_IMAGE,
		modelPB.Model_TASK_TEXT_GENERATION:
		modelOutput, err = TriggerTextTask(mClient, rClient, task, input, dataMappingIndices, model, ownerPermalink)
		if err != nil {
			return nil, status.Errorf(codes.Internal, err.Error())
		}
	}

	return modelOutput, nil

}

func TriggerImageTask(mClient modelPB.ModelPublicServiceClient, rClient *redis.Client, task modelPB.Model_Task, input interface{}, dataMappingIndices []string, model string, ownerPermalink string) (*pipelinePB.ModelOutput, error) {
	imageInput := input.(*utils.ImageInput)

	// TODO: async call model-backend
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	stream, err := mClient.TriggerModelBinaryFileUpload(utils.InjectOwnerToContextWithOwnerPermalink(ctx, ownerPermalink))
	defer func() {
		_ = stream.CloseSend()
	}()

	if err != nil {
		return nil, status.Errorf(codes.Internal, "[model-backend] Error %s at model %s: cannot init stream: %v", "TriggerModelBinaryFileUpload", model, err.Error())
	}
	var triggerRequest modelPB.TriggerModelBinaryFileUploadRequest
	switch task {
	case modelPB.Model_TASK_CLASSIFICATION:
		triggerRequest = modelPB.TriggerModelBinaryFileUploadRequest{
			Name: model,
			TaskInput: &modelPB.TaskInputStream{
				Input: &modelPB.TaskInputStream_Classification{
					Classification: &modelPB.ClassificationInputStream{
						FileLengths: imageInput.FileLengths,
					},
				},
			},
		}
	case modelPB.Model_TASK_DETECTION:
		triggerRequest = modelPB.TriggerModelBinaryFileUploadRequest{
			Name: model,
			TaskInput: &modelPB.TaskInputStream{
				Input: &modelPB.TaskInputStream_Detection{
					Detection: &modelPB.DetectionInputStream{
						FileLengths: imageInput.FileLengths,
					},
				},
			},
		}
	case modelPB.Model_TASK_KEYPOINT:
		triggerRequest = modelPB.TriggerModelBinaryFileUploadRequest{
			Name: model,
			TaskInput: &modelPB.TaskInputStream{
				Input: &modelPB.TaskInputStream_Keypoint{
					Keypoint: &modelPB.KeypointInputStream{
						FileLengths: imageInput.FileLengths,
					},
				},
			},
		}
	case modelPB.Model_TASK_OCR:
		triggerRequest = modelPB.TriggerModelBinaryFileUploadRequest{
			Name: model,
			TaskInput: &modelPB.TaskInputStream{
				Input: &modelPB.TaskInputStream_Ocr{
					Ocr: &modelPB.OcrInputStream{
						FileLengths: imageInput.FileLengths,
					},
				},
			},
		}

	case modelPB.Model_TASK_INSTANCE_SEGMENTATION:
		triggerRequest = modelPB.TriggerModelBinaryFileUploadRequest{
			Name: model,
			TaskInput: &modelPB.TaskInputStream{
				Input: &modelPB.TaskInputStream_InstanceSegmentation{
					InstanceSegmentation: &modelPB.InstanceSegmentationInputStream{
						FileLengths: imageInput.FileLengths,
					},
				},
			},
		}
	case modelPB.Model_TASK_SEMANTIC_SEGMENTATION:
		triggerRequest = modelPB.TriggerModelBinaryFileUploadRequest{
			Name: model,
			TaskInput: &modelPB.TaskInputStream{
				Input: &modelPB.TaskInputStream_SemanticSegmentation{
					SemanticSegmentation: &modelPB.SemanticSegmentationInputStream{
						FileLengths: imageInput.FileLengths,
					},
				},
			},
		}
	}
	if err := stream.Send(&triggerRequest); err != nil {
		return nil, status.Errorf(codes.Internal, "[model-backend] Error %s at model %s: cannot send data info to server: %v", "TriggerModelBinaryFileUploadRequest", model, err.Error())
	}
	fb := bytes.Buffer{}
	fb.Write(imageInput.Content)
	buf := make([]byte, 64*1024)
	for {
		n, err := fb.Read(buf)
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}

		var triggerRequest modelPB.TriggerModelBinaryFileUploadRequest
		switch task {
		case modelPB.Model_TASK_CLASSIFICATION:
			triggerRequest = modelPB.TriggerModelBinaryFileUploadRequest{
				TaskInput: &modelPB.TaskInputStream{
					Input: &modelPB.TaskInputStream_Classification{
						Classification: &modelPB.ClassificationInputStream{
							Content: buf[:n],
						},
					},
				},
			}
		case modelPB.Model_TASK_DETECTION:
			triggerRequest = modelPB.TriggerModelBinaryFileUploadRequest{
				TaskInput: &modelPB.TaskInputStream{
					Input: &modelPB.TaskInputStream_Detection{
						Detection: &modelPB.DetectionInputStream{
							Content: buf[:n],
						},
					},
				},
			}
		case modelPB.Model_TASK_KEYPOINT:
			triggerRequest = modelPB.TriggerModelBinaryFileUploadRequest{
				TaskInput: &modelPB.TaskInputStream{
					Input: &modelPB.TaskInputStream_Keypoint{
						Keypoint: &modelPB.KeypointInputStream{
							Content: buf[:n],
						},
					},
				},
			}
		case modelPB.Model_TASK_OCR:
			triggerRequest = modelPB.TriggerModelBinaryFileUploadRequest{
				TaskInput: &modelPB.TaskInputStream{
					Input: &modelPB.TaskInputStream_Ocr{
						Ocr: &modelPB.OcrInputStream{
							Content: buf[:n],
						},
					},
				},
			}

		case modelPB.Model_TASK_INSTANCE_SEGMENTATION:
			triggerRequest = modelPB.TriggerModelBinaryFileUploadRequest{
				TaskInput: &modelPB.TaskInputStream{
					Input: &modelPB.TaskInputStream_InstanceSegmentation{
						InstanceSegmentation: &modelPB.InstanceSegmentationInputStream{
							Content: buf[:n],
						},
					},
				},
			}
		case modelPB.Model_TASK_SEMANTIC_SEGMENTATION:
			triggerRequest = modelPB.TriggerModelBinaryFileUploadRequest{
				TaskInput: &modelPB.TaskInputStream{
					Input: &modelPB.TaskInputStream_SemanticSegmentation{
						SemanticSegmentation: &modelPB.SemanticSegmentationInputStream{
							Content: buf[:n],
						},
					},
				},
			}
		}
		if err := stream.Send(&triggerRequest); err != nil {
			return nil, status.Errorf(codes.Internal, "[model-backend] Error %s at model %s: cannot send chunk to server: %v", "TriggerModelBinaryFileUploadRequest", model, err.Error())
		}
	}

	resp, err := stream.CloseAndRecv()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "[model-backend] Error %s at model %s: cannot receive response: %v", "TriggerModelBinaryFileUploadRequest", model, err.Error())
	}

	taskOutputs := utils.CvtModelTaskOutputToPipelineTaskOutput(ctx, resp.TaskOutputs)
	for idx, taskOutput := range taskOutputs {
		taskOutput.Index = dataMappingIndices[idx]
	}

	modelOutput := &pipelinePB.ModelOutput{
		Model:       model,
		Task:        resp.Task,
		TaskOutputs: taskOutputs,
	}

	// Increment trigger image numbers
	uid, err := resource.GetPermalinkUID(ownerPermalink)
	if err != nil {
		return nil, err
	}
	if strings.HasPrefix(ownerPermalink, "users/") {
		rClient.IncrBy(ctx, fmt.Sprintf("user:%s:trigger.num", uid), int64(len(imageInput.FileLengths)))
	} else if strings.HasPrefix(ownerPermalink, "orgs/") {
		rClient.IncrBy(ctx, fmt.Sprintf("org:%s:trigger.num", uid), int64(len(imageInput.FileLengths)))
	}

	return modelOutput, nil
}

func TriggerTextTask(mClient modelPB.ModelPublicServiceClient, rClient *redis.Client, task modelPB.Model_Task, input interface{}, dataMappingIndices []string, model string, ownerPermalink string) (*pipelinePB.ModelOutput, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	stream, err := mClient.TriggerModelBinaryFileUpload(utils.InjectOwnerToContextWithOwnerPermalink(ctx, ownerPermalink))
	defer func() {
		_ = stream.CloseSend()
	}()

	if err != nil {
		return nil, status.Errorf(codes.Internal, "[model-backend] Error %s at model %s: cannot init stream: %v", "TriggerModelBinaryFileUpload", model, err.Error())
	}

	var triggerRequest modelPB.TriggerModelBinaryFileUploadRequest
	switch task {
	case modelPB.Model_TASK_TEXT_TO_IMAGE:
		textToImageInput := input.(*utils.TextToImageInput)
		triggerRequest = modelPB.TriggerModelBinaryFileUploadRequest{
			Name: model,
			TaskInput: &modelPB.TaskInputStream{
				Input: &modelPB.TaskInputStream_TextToImage{
					TextToImage: &modelPB.TextToImageInput{
						Prompt:   textToImageInput.Prompt,
						Steps:    &textToImageInput.Steps,
						CfgScale: &textToImageInput.CfgScale,
						Seed:     &textToImageInput.Seed,
						Samples:  &textToImageInput.Samples,
					},
				},
			},
		}
	case modelPB.Model_TASK_TEXT_GENERATION:
		textGenerationInput := input.(*utils.TextGenerationInput)
		triggerRequest = modelPB.TriggerModelBinaryFileUploadRequest{
			Name: model,
			TaskInput: &modelPB.TaskInputStream{
				Input: &modelPB.TaskInputStream_TextGeneration{
					TextGeneration: &modelPB.TextGenerationInput{
						Prompt:        textGenerationInput.Prompt,
						OutputLen:     &textGenerationInput.OutputLen,
						BadWordsList:  &textGenerationInput.BadWordsList,
						StopWordsList: &textGenerationInput.StopWordsList,
						Topk:          &textGenerationInput.TopK,
						Seed:          &textGenerationInput.Seed,
					},
				},
			},
		}
	}

	if err := stream.Send(&triggerRequest); err != nil {
		return nil, status.Errorf(codes.Internal, "[model-backend] Error %s at model %s: cannot send data info to server: %v", "TriggerModelBinaryFileUploadRequest", model, err.Error())
	}

	resp, err := stream.CloseAndRecv()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "[model-backend] Error %s at model %s: cannot receive response: %v", "TriggerModelBinaryFileUploadRequest", model, err.Error())
	}

	taskOutputs := utils.CvtModelTaskOutputToPipelineTaskOutput(ctx, resp.TaskOutputs)
	for idx, taskOutput := range taskOutputs {
		taskOutput.Index = dataMappingIndices[idx]
	}

	modelOutput := &pipelinePB.ModelOutput{
		Model:       model,
		Task:        resp.Task,
		TaskOutputs: taskOutputs,
	}

	// Increment trigger image numbers
	uid, err := resource.GetPermalinkUID(ownerPermalink)
	if err != nil {
		return nil, err
	}
	if strings.HasPrefix(ownerPermalink, "users/") {
		rClient.IncrBy(ctx, fmt.Sprintf("user:%s:trigger.num", uid), 1)
	} else if strings.HasPrefix(ownerPermalink, "orgs/") {
		rClient.IncrBy(ctx, fmt.Sprintf("org:%s:trigger.num", uid), 1)
	}

	return modelOutput, nil
}
