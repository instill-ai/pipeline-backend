//go:generate compogen readme ./config ./README.mdx
package huggingface

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sync"

	_ "embed"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util"
	"github.com/instill-ai/x/errmsg"
)

const (
	textGenerationTask         = "TASK_TEXT_GENERATION"
	textToImageTask            = "TASK_TEXT_TO_IMAGE"
	fillMaskTask               = "TASK_FILL_MASK"
	summarizationTask          = "TASK_SUMMARIZATION"
	textClassificationTask     = "TASK_TEXT_CLASSIFICATION"
	tokenClassificationTask    = "TASK_TOKEN_CLASSIFICATION"
	translationTask            = "TASK_TRANSLATION"
	zeroShotClassificationTask = "TASK_ZERO_SHOT_CLASSIFICATION"
	featureExtractionTask      = "TASK_FEATURE_EXTRACTION"
	questionAnsweringTask      = "TASK_QUESTION_ANSWERING"
	tableQuestionAnsweringTask = "TASK_TABLE_QUESTION_ANSWERING"
	sentenceSimilarityTask     = "TASK_SENTENCE_SIMILARITY"
	conversationalTask         = "TASK_CONVERSATIONAL"
	imageClassificationTask    = "TASK_IMAGE_CLASSIFICATION"
	imageSegmentationTask      = "TASK_IMAGE_SEGMENTATION"
	objectDetectionTask        = "TASK_OBJECT_DETECTION"
	imageToTextTask            = "TASK_IMAGE_TO_TEXT"
	speechRecognitionTask      = "TASK_SPEECH_RECOGNITION"
	audioClassificationTask    = "TASK_AUDIO_CLASSIFICATION"
)

var (
	//go:embed config/definition.yaml
	definitionYAML []byte
	//go:embed config/setup.yaml
	setupYAML []byte
	//go:embed config/tasks.yaml
	tasksYAML []byte
	once      sync.Once
	comp      *component
)

type component struct {
	base.Component
}

type execution struct {
	base.ComponentExecution
}

func Init(bc base.Component) *component {
	once.Do(func() {
		comp = &component{Component: bc}
		err := comp.LoadDefinition(definitionYAML, setupYAML, tasksYAML, nil, nil)
		if err != nil {
			panic(err)
		}
	})
	return comp
}

// CreateExecution initializes a component executor that can be used in a
// pipeline trigger.
func (c *component) CreateExecution(x base.ComponentExecution) (base.IExecution, error) {
	return &execution{ComponentExecution: x}, nil
}

func getAPIKey(setup *structpb.Struct) string {
	return setup.GetFields()["api-key"].GetStringValue()
}

func getBaseURL(setup *structpb.Struct) string {
	return setup.GetFields()["base-url"].GetStringValue()
}

func isCustomEndpoint(setup *structpb.Struct) bool {
	return setup.GetFields()["is-custom-endpoint"].GetBoolValue()
}

func wrapSliceInStruct(data []byte, key string) (*structpb.Struct, error) {
	var list []any
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, err
	}

	results, err := structpb.NewList(list)
	if err != nil {
		return nil, err
	}

	return &structpb.Struct{
		Fields: map[string]*structpb.Value{
			key: structpb.NewListValue(results),
		},
	}, nil
}

func (e *execution) Execute(ctx context.Context, jobs []*base.Job) error {

	client := newClient(e.Setup, e.GetLogger())

	for _, job := range jobs {
		input, err := job.Input.Read(ctx)
		if err != nil {
			job.Error.Error(ctx, err)
			continue
		}
		path := "/"
		if !isCustomEndpoint(e.Setup) {
			path = modelsPath + input.GetFields()["model"].GetStringValue()
		}

		output := &structpb.Struct{}

		switch e.Task {
		case textGenerationTask:
			inputStruct := TextGenerationRequest{}
			if err := base.ConvertFromStructpb(input, &inputStruct); err != nil {
				job.Error.Error(ctx, err)
				continue
			}

			resp := []TextGenerationResponse{}
			req := client.R().SetBody(inputStruct).SetResult(&resp)
			if _, err := post(req, path); err != nil {
				job.Error.Error(ctx, err)
				continue
			}

			if len(resp) < 1 {
				err := fmt.Errorf("invalid response")
				job.Error.Error(ctx, errmsg.AddMessage(err, "Hugging Face didn't return any result"))
				continue
			}

			output, err = structpb.NewStruct(map[string]any{"generated-text": resp[0].GeneratedText})
			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}

		case textToImageTask:
			inputStruct := TextToImageRequest{}
			if err := base.ConvertFromStructpb(input, &inputStruct); err != nil {
				job.Error.Error(ctx, err)
				continue
			}

			req := client.R().SetBody(inputStruct)
			resp, err := post(req, path)
			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}

			rawImg := base64.StdEncoding.EncodeToString(resp.Body())
			output, err = structpb.NewStruct(map[string]any{
				"image": fmt.Sprintf("data:image/jpeg;base64,%s", rawImg),
			})
			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}

		case fillMaskTask:
			inputStruct := FillMaskRequest{}
			if err := base.ConvertFromStructpb(input, &inputStruct); err != nil {
				job.Error.Error(ctx, err)
				continue
			}

			req := client.R().SetBody(inputStruct)
			resp, err := post(req, path)
			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}

			output, err = wrapSliceInStruct(resp.Body(), "results")
			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}

		case summarizationTask:
			inputStruct := SummarizationRequest{}
			if err := base.ConvertFromStructpb(input, &inputStruct); err != nil {
				job.Error.Error(ctx, err)
				continue
			}

			resp := []SummarizationResponse{}
			req := client.R().SetBody(inputStruct).SetResult(&resp)
			if _, err := post(req, path); err != nil {
				job.Error.Error(ctx, err)
				continue
			}

			if len(resp) < 1 {
				err := fmt.Errorf("invalid response")
				job.Error.Error(ctx, errmsg.AddMessage(err, "Hugging Face didn't return any result"))
				continue
			}

			output, err = structpb.NewStruct(map[string]any{"summary-text": resp[0].SummaryText})
			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}

		case textClassificationTask:
			inputStruct := TextClassificationRequest{}
			if err := base.ConvertFromStructpb(input, &inputStruct); err != nil {
				job.Error.Error(ctx, err)
				continue
			}

			var resp [][]any
			req := client.R().SetBody(inputStruct).SetResult(&resp)
			if _, err := post(req, path); err != nil {
				job.Error.Error(ctx, err)
				continue
			}

			if len(resp) < 1 {
				err := fmt.Errorf("invalid response")
				job.Error.Error(ctx, errmsg.AddMessage(err, "Hugging Face didn't return any result"))
				continue
			}

			results, err := structpb.NewList(resp[0])
			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}

			output = &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"results": structpb.NewListValue(results),
				},
			}

		case tokenClassificationTask:
			inputStruct := TokenClassificationRequest{}
			if err := base.ConvertFromStructpb(input, &inputStruct); err != nil {
				job.Error.Error(ctx, err)
				continue
			}
			req := client.R().SetBody(inputStruct)
			resp, err := post(req, path)
			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}

			output, err = wrapSliceInStruct(resp.Body(), "results")
			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}

		case translationTask:
			inputStruct := TranslationRequest{}
			if err := base.ConvertFromStructpb(input, &inputStruct); err != nil {
				job.Error.Error(ctx, err)
				continue
			}

			resp := []TranslationResponse{}
			req := client.R().SetBody(inputStruct).SetResult(&resp)
			if _, err := post(req, path); err != nil {
				job.Error.Error(ctx, err)
				continue
			}

			if len(resp) < 1 {
				err := fmt.Errorf("invalid response")
				job.Error.Error(ctx, errmsg.AddMessage(err, "Hugging Face didn't return any result"))
				continue
			}

			output, err = structpb.NewStruct(map[string]any{"translation-text": resp[0].TranslationText})
			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}

		case zeroShotClassificationTask:
			inputStruct := ZeroShotRequest{}
			if err := base.ConvertFromStructpb(input, &inputStruct); err != nil {
				job.Error.Error(ctx, err)
				continue
			}

			req := client.R().SetBody(inputStruct)
			resp, err := post(req, path)
			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}

			if err = protojson.Unmarshal(resp.Body(), output); err != nil {
				job.Error.Error(ctx, err)
				continue
			}

		// case featureExtractionTask:
		// TODO: fix this task
		// 	inputStruct := FeatureExtractionRequest{}
		// 	err := base.ConvertFromStructpb(input, &inputStruct)
		// 	if err != nil {
		// 		return nil, err
		// 	}
		// 	jsonBody, _ := json.Marshal(inputStruct)
		// 	resp, err := doer.MakeHFAPIRequest(jsonBody, model)
		// 	if err != nil {
		// 		return nil, err
		// 	}
		// 	threeDArr := [][][]float64{}
		// 	err = json.Unmarshal(resp, &threeDArr)
		// 	if err != nil {
		// 		return nil, err
		// 	}
		// 	if len(threeDArr) <= 0 {
		// 		return nil, errors.New("invalid response")
		// 	}
		// 	nestedArr := threeDArr[0]
		// 	features := structpb.ListValue{}
		// 	features.Values = make([]*structpb.Value, len(nestedArr))
		// 	for i, innerArr := range nestedArr {
		// 		innerValues := make([]*structpb.Value, len(innerArr))
		// 		for j := range innerArr {
		// 			innerValues[j] = &structpb.Value{Kind: &structpb.Value_NumberValue{NumberValue: innerArr[j]}}
		// 		}
		// 		features.Values[i] = &structpb.Value{Kind: &structpb.Value_ListValue{ListValue: &structpb.ListValue{Values: innerValues}}}
		// 	}
		// 	output := structpb.Struct{
		// 		Fields: map[string]*structpb.Value{"feature": {Kind: &structpb.Value_ListValue{ListValue: &features}}},
		// 	}
		// 	outputs = append(outputs, &output)
		case questionAnsweringTask:
			inputStruct := QuestionAnsweringRequest{}
			if err := base.ConvertFromStructpb(input, &inputStruct); err != nil {
				job.Error.Error(ctx, err)
				continue
			}
			req := client.R().SetBody(inputStruct)
			resp, err := post(req, path)
			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}

			if err = protojson.Unmarshal(resp.Body(), output); err != nil {
				job.Error.Error(ctx, err)
				continue
			}

		case tableQuestionAnsweringTask:
			inputStruct := TableQuestionAnsweringRequest{}
			if err := base.ConvertFromStructpb(input, &inputStruct); err != nil {
				job.Error.Error(ctx, err)
				continue
			}

			req := client.R().SetBody(inputStruct)
			resp, err := post(req, path)
			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}

			if err = protojson.Unmarshal(resp.Body(), output); err != nil {
				job.Error.Error(ctx, err)
				continue
			}

		case sentenceSimilarityTask:
			inputStruct := SentenceSimilarityRequest{}
			if err := base.ConvertFromStructpb(input, &inputStruct); err != nil {
				job.Error.Error(ctx, err)
				continue
			}

			req := client.R().SetBody(inputStruct)
			resp, err := post(req, path)
			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}

			output, err = wrapSliceInStruct(resp.Body(), "scores")
			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}

		case conversationalTask:
			inputStruct := ConversationalRequest{}
			if err := base.ConvertFromStructpb(input, &inputStruct); err != nil {
				job.Error.Error(ctx, err)
				continue
			}

			resp := ConversationalResponse{}
			req := client.R().SetBody(inputStruct).SetResult(&resp)

			if _, err := post(req, path); err != nil {
				job.Error.Error(ctx, err)
				continue
			}

			output, err = base.ConvertToStructpb(resp)
			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}

		case imageClassificationTask:
			inputStruct := ImageRequest{}
			if err := base.ConvertFromStructpb(input, &inputStruct); err != nil {
				job.Error.Error(ctx, err)
				continue
			}

			b, err := base64.StdEncoding.DecodeString(util.TrimBase64Mime(inputStruct.Image))
			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}

			req := client.R().SetBody(b)
			resp, err := post(req, path)
			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}

			output, err = wrapSliceInStruct(resp.Body(), "classes")
			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}

		case imageSegmentationTask:
			inputStruct := ImageRequest{}
			if err := base.ConvertFromStructpb(input, &inputStruct); err != nil {
				job.Error.Error(ctx, err)
				continue
			}

			b, err := base64.StdEncoding.DecodeString(util.TrimBase64Mime(inputStruct.Image))
			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}

			resp := []ImageSegmentationResponse{}
			req := client.R().SetBody(b).SetResult(&resp)
			if _, err := post(req, path); err != nil {
				job.Error.Error(ctx, err)
				continue
			}

			segments := &structpb.ListValue{
				Values: make([]*structpb.Value, len(resp)),
			}

			for i := range resp {
				segment, err := structpb.NewStruct(map[string]any{
					"score": resp[i].Score,
					"label": resp[i].Label,
					"mask":  fmt.Sprintf("data:image/png;base64,%s", resp[i].Mask),
				})

				if err != nil {
					job.Error.Error(ctx, err)
					continue
				}

				segments.Values[i] = structpb.NewStructValue(segment)
			}

			output = &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"segments": structpb.NewListValue(segments),
				},
			}

		case objectDetectionTask:
			inputStruct := ImageRequest{}
			if err := base.ConvertFromStructpb(input, &inputStruct); err != nil {
				job.Error.Error(ctx, err)
				continue
			}

			b, err := base64.StdEncoding.DecodeString(util.TrimBase64Mime(inputStruct.Image))
			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}

			req := client.R().SetBody(b)
			resp, err := post(req, path)
			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}

			output, err = wrapSliceInStruct(resp.Body(), "objects")
			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}

		case imageToTextTask:
			inputStruct := ImageRequest{}
			if err := base.ConvertFromStructpb(input, &inputStruct); err != nil {
				job.Error.Error(ctx, err)
				continue
			}

			b, err := base64.StdEncoding.DecodeString(util.TrimBase64Mime(inputStruct.Image))
			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}

			resp := []ImageToTextResponse{}
			req := client.R().SetBody(b).SetResult(&resp)
			if _, err := post(req, path); err != nil {
				job.Error.Error(ctx, err)
				continue
			}

			if len(resp) < 1 {
				err := fmt.Errorf("invalid response")
				job.Error.Error(ctx, errmsg.AddMessage(err, "Hugging Face didn't return any result"))
				continue
			}

			output, err = structpb.NewStruct(map[string]any{"text": resp[0].GeneratedText})
			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}

		case speechRecognitionTask:
			inputStruct := AudioRequest{}
			if err := base.ConvertFromStructpb(input, &inputStruct); err != nil {
				job.Error.Error(ctx, err)
				continue
			}

			b, err := base64.StdEncoding.DecodeString(util.TrimBase64Mime(inputStruct.Audio))
			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}

			resp := SpeechRecognitionResponse{}
			req := client.R().SetBody(b).SetResult(&resp)

			if _, err := post(req, path); err != nil {
				job.Error.Error(ctx, err)
				continue
			}

			output, err = base.ConvertToStructpb(resp)
			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}

		case audioClassificationTask:
			inputStruct := AudioRequest{}
			if err := base.ConvertFromStructpb(input, &inputStruct); err != nil {
				job.Error.Error(ctx, err)
				continue
			}

			b, err := base64.StdEncoding.DecodeString(util.TrimBase64Mime(inputStruct.Audio))
			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}

			req := client.R().SetBody(b)
			resp, err := post(req, path)
			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}

			output, err = wrapSliceInStruct(resp.Body(), "classes")
			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}

		default:
			job.Error.Error(ctx, errmsg.AddMessage(
				fmt.Errorf("not supported task: %s", e.Task),
				fmt.Sprintf("%s task is not supported.", e.Task),
			))
			continue
		}
		err = job.Output.Write(ctx, output)
		if err != nil {
			job.Error.Error(ctx, err)
			continue
		}
	}

	return nil
}

func (c *component) Test(sysVars map[string]any, setup *structpb.Struct) error {
	req := newClient(setup, c.GetLogger()).R()
	resp, err := req.Get("")
	if err != nil {
		return err
	}

	if resp.IsError() {
		return fmt.Errorf("setup error")
	}

	return nil
}
