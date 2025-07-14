//go:generate compogen readme ./config ./README.mdx --extraContents bottom=.compogen/bottom.mdx
package openai

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	_ "embed"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util/httpclient"
	"github.com/instill-ai/pipeline-backend/pkg/component/resources/schemas"
	"github.com/instill-ai/pipeline-backend/pkg/data"
	"github.com/instill-ai/pipeline-backend/pkg/data/format"

	errorsx "github.com/instill-ai/x/errors"
)

const (
	host = "https://api.openai.com"

	TextGenerationTask    = "TASK_TEXT_GENERATION"
	TextEmbeddingsTask    = "TASK_TEXT_EMBEDDINGS"
	SpeechRecognitionTask = "TASK_SPEECH_RECOGNITION"
	TextToSpeechTask      = "TASK_TEXT_TO_SPEECH"
	TextToImageTask       = "TASK_TEXT_TO_IMAGE"

	cfgAPIKey       = "api-key"
	cfgOrganization = "organization"
	retryCount      = 10 // Note: sometime OpenAI service are not stable
)

var (
	//go:embed config/definition.yaml
	definitionYAML []byte
	//go:embed config/setup.yaml
	setupYAML []byte
	//go:embed config/tasks.yaml
	tasksYAML []byte

	once sync.Once
	comp *component
)

// Component executes queries against OpenAI.
type component struct {
	base.Component

	instillAPIKey string
}

// Init returns an initialized OpenAI component.
func Init(bc base.Component) *component {
	once.Do(func() {
		comp = &component{Component: bc}
		additionalYAMLBytes := map[string][]byte{
			"schema.yaml": schemas.SchemaYAML,
		}
		err := comp.LoadDefinition(definitionYAML, setupYAML, tasksYAML, nil, additionalYAMLBytes)
		if err != nil {
			panic(err)
		}
	})

	return comp
}

// WithInstillCredentials loads Instill credentials into the component, which
// can be used to configure it with globally defined parameters instead of with
// user-defined credential values.
func (c *component) WithInstillCredentials(s map[string]any) *component {
	c.instillAPIKey = base.ReadFromGlobalConfig(cfgAPIKey, s)
	return c
}

// CreateExecution initializes a component executor that can be used in a
// pipeline trigger.
func (c *component) CreateExecution(x base.ComponentExecution) (base.IExecution, error) {
	resolvedSetup, resolved, err := c.resolveSetup(x.Setup)
	if err != nil {
		return nil, err
	}

	x.Setup = resolvedSetup

	return &execution{
		ComponentExecution:     x,
		usesInstillCredentials: resolved,
	}, nil
}

// resolveSetup checks whether the component is configured to use the Instill
// credentials injected during initialization and, if so, returns a new setup
// with the secret credential values.
func (c *component) resolveSetup(setup *structpb.Struct) (*structpb.Struct, bool, error) {
	if setup == nil || setup.Fields == nil {
		setup = &structpb.Struct{Fields: map[string]*structpb.Value{}}
	}
	if v, ok := setup.GetFields()[cfgAPIKey]; ok {
		apiKey := v.GetStringValue()
		if apiKey != "" && apiKey != base.SecretKeyword {
			return setup, false, nil
		}
	}

	if c.instillAPIKey == "" {
		return nil, false, base.NewUnresolvedCredential(cfgAPIKey)
	}

	setup.GetFields()[cfgAPIKey] = structpb.NewStringValue(c.instillAPIKey)
	return setup, true, nil
}

type execution struct {
	base.ComponentExecution
	usesInstillCredentials bool
}

func (e *execution) UsesInstillCredentials() bool {
	return e.usesInstillCredentials
}

func (e *execution) worker(ctx context.Context, client *httpclient.Client, job *base.Job) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("panic: %+v", r)
			job.Error.Error(ctx, fmt.Errorf("panic: %+v", r))
			return
		}
	}()

	switch e.Task {
	case TextGenerationTask:
		client.SetTimeout(30 * time.Minute)
		inputStruct := taskTextGenerationInput{}
		err := job.Input.ReadData(ctx, &inputStruct)
		if err != nil {
			job.Error.Error(ctx, err)
			return
		}

		messages := []interface{}{}

		if inputStruct.ChatHistory != nil {
			for _, chat := range inputStruct.ChatHistory {
				switch chat.Role {
				case "user":
					cs := make([]Content, len(chat.Content))
					for i, c := range chat.Content {
						cs[i] = Content{
							Type: c.Type,
						}
						if c.Type == "text" {
							cs[i].Text = c.Text
						} else {
							cs[i].ImageURL = &ImageURL{
								c.ImageURL.URL,
							}
						}
					}
					messages = append(messages, multiModalMessage{Role: chat.Role, Content: cs})
				case "assistant":
					content := ""
					for _, c := range chat.Content {
						// OpenAI doesn't support multi-modal content for
						// non-user roles.
						if c.Type == "text" {
							content = *c.Text
						}
					}
					messages = append(messages, message{Role: chat.Role, Content: content})
				}
			}
		}
		if inputStruct.SystemMessage != nil {
			messages = append(messages, message{Role: "system", Content: *inputStruct.SystemMessage})
		}
		userContents := []Content{}
		userContents = append(userContents, Content{Type: "text", Text: &inputStruct.Prompt})
		for _, image := range inputStruct.Images {

			width := image.Width().Integer()
			height := image.Height().Integer()

			newWidth, newHeight := resizeImage(width, height)

			// Only resize if dimensions changed
			if newWidth != width || newHeight != height {
				image, err = image.Resize(newWidth, newHeight)
				if err != nil {
					job.Error.Error(ctx, err)
					return
				}
			}
			i, err := image.DataURI()
			if err != nil {
				job.Error.Error(ctx, err)
				return
			}
			userContents = append(userContents, Content{Type: "image_url", ImageURL: &ImageURL{URL: i.String()}})
		}
		messages = append(messages, multiModalMessage{Role: "user", Content: userContents})

		tools := make([]toolReqStruct, len(inputStruct.Tools))
		for i, tool := range inputStruct.Tools {
			params := make(map[string]any)
			for k, v := range tool.Function.Parameters {
				params[k], err = v.ToJSONValue()
				if err != nil {
					job.Error.Error(ctx, err)
					return
				}
			}
			tools[i] = toolReqStruct{
				Type: "function",
				Function: functionReqStruct{
					Name:        tool.Function.Name,
					Parameters:  params,
					Strict:      tool.Function.Strict,
					Description: tool.Function.Description,
				},
			}
		}

		var toolChoice any
		switch choice := inputStruct.ToolChoice.(type) {
		case data.Map:
			toolChoice, err = choice.ToJSONValue()
			if err != nil {
				job.Error.Error(ctx, err)
				return
			}
		case format.String:
			toolChoice = choice.String()
		}

		body := textCompletionReq{
			Messages:         messages,
			Model:            inputStruct.Model,
			MaxTokens:        inputStruct.MaxTokens,
			Temperature:      inputStruct.Temperature,
			N:                inputStruct.N,
			TopP:             inputStruct.TopP,
			PresencePenalty:  inputStruct.PresencePenalty,
			FrequencyPenalty: inputStruct.FrequencyPenalty,
			Stream:           true,
			StreamOptions: &streamOptions{
				IncludeUsage: true,
			},
		}

		if inputStruct.Prediction != nil {
			body.Prediction = &predictionReqStruct{
				Type:    "content",
				Content: inputStruct.Prediction.Content,
			}
		}
		if len(tools) > 0 {
			body.Tools = tools
		}
		if toolChoice != nil {
			body.ToolChoice = toolChoice
		}

		if inputStruct.ResponseFormat != nil {
			body.ResponseFormat = &responseFormatReqStruct{
				Type: inputStruct.ResponseFormat.Type,
			}
			if inputStruct.ResponseFormat.Type == "json_schema" {
				sch := map[string]any{}
				if inputStruct.ResponseFormat.JSONSchema != "" {
					err = json.Unmarshal([]byte(inputStruct.ResponseFormat.JSONSchema), &sch)
					if err != nil {
						job.Error.Error(ctx, err)
						return
					}
					body.ResponseFormat = &responseFormatReqStruct{
						Type:       inputStruct.ResponseFormat.Type,
						JSONSchema: sch,
					}
				}
			}
		}

		req := client.SetDoNotParseResponse(true).R().SetBody(body)
		restyResp, err := req.Post(completionsPath)
		if err != nil {
			job.Error.Error(ctx, err)
			return
		}

		if restyResp.StatusCode() != 200 {
			rawBody := restyResp.RawBody()
			defer rawBody.Close()
			bodyBytes, err := io.ReadAll(rawBody)
			job.Error.Error(ctx, fmt.Errorf("send request to openai error with error code: %d, msg %s, %s", restyResp.StatusCode(), bodyBytes, err))
			return
		}
		scanner := bufio.NewScanner(restyResp.RawResponse.Body)

		outputStruct := taskTextGenerationOutput{}
		toolCalls := make(map[int]*toolCall)

		u := usage{}
		count := 0
		for scanner.Scan() {
			res := scanner.Text()

			if len(res) == 0 {
				continue
			}
			res = strings.Replace(res, "data: ", "", 1)

			// Note: Since we haven't provided delta updates for the
			// messages, we're reducing the number of event streams by
			// returning the response every ten iterations.
			if count == 3 || res == "[DONE]" {
				err = job.Output.WriteData(ctx, outputStruct)
				if err != nil {
					job.Error.Error(ctx, err)
					return
				}
				if res == "[DONE]" {
					break
				}
				count = 0
			}

			count += 1
			response := &textCompletionStreamResp{}
			err = json.Unmarshal([]byte(res), response)
			if err != nil {
				job.Error.Error(ctx, err)
				return
			}

			for _, c := range response.Choices {
				responseIdx := c.Index
				if len(outputStruct.Texts) <= responseIdx {
					outputStruct.Texts = append(outputStruct.Texts, "")
				}
				outputStruct.Texts[responseIdx] += c.Delta.Content

				// Collect tool calls
				for _, t := range c.Delta.ToolCalls {
					if _, exists := toolCalls[t.Index]; !exists {
						toolCalls[t.Index] = &toolCall{
							Type: t.Type,
							Function: functionCall{
								Name:      t.Function.Name,
								Arguments: t.Function.Arguments,
							},
						}
					} else {
						// Append arguments for existing tool call
						toolCalls[t.Index].Function.Arguments += t.Function.Arguments
					}
				}
			}

			u = usage{
				PromptTokens:     response.Usage.PromptTokens,
				CompletionTokens: response.Usage.CompletionTokens,
				TotalTokens:      response.Usage.TotalTokens,
				PromptTokenDetails: &promptTokenDetails{
					AudioTokens:  response.Usage.PromptTokenDetails.AudioTokens,
					CachedTokens: response.Usage.PromptTokenDetails.CachedTokens,
				},
				CompletionTokenDetails: &completionTokenDetails{
					ReasoningTokens:          response.Usage.CompletionTokenDetails.ReasoningTokens,
					AudioTokens:              response.Usage.CompletionTokenDetails.AudioTokens,
					AcceptedPredictionTokens: response.Usage.CompletionTokenDetails.AcceptedPredictionTokens,
					RejectedPredictionTokens: response.Usage.CompletionTokenDetails.RejectedPredictionTokens,
				},
			}
		}

		// Convert collected tool calls to output format
		for _, tc := range toolCalls {
			outputStruct.ToolCalls = append(outputStruct.ToolCalls, toolCall{
				Type: tc.Type,
				Function: functionCall{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			})
		}

		outputStruct.Usage = u
		err = job.Output.WriteData(ctx, outputStruct)
		if err != nil {
			job.Error.Error(ctx, err)
			return
		}

	case SpeechRecognitionTask:

		inputStruct := taskSpeechRecognitionInput{}
		err := job.Input.ReadData(ctx, &inputStruct)
		if err != nil {
			job.Error.Error(ctx, err)
			return
		}

		audioBytes, err := inputStruct.Audio.Binary()
		if err != nil {
			job.Error.Error(ctx, err)
			return
		}

		data, ct, err := getBytes(AudioTranscriptionReq{
			File:        audioBytes.ByteArray(),
			Model:       inputStruct.Model,
			Prompt:      inputStruct.Prompt,
			Language:    inputStruct.Language,
			Temperature: inputStruct.Temperature,

			// Verbosity is passed to extract result duration.
			ResponseFormat: "verbose_json",
		})
		if err != nil {
			job.Error.Error(ctx, err)
			return
		}

		resp := AudioTranscriptionResp{}
		req := client.R().SetBody(data).SetResult(&resp).SetHeader("Content-Type", ct)
		if _, err := req.Post(transcriptionsPath); err != nil {
			job.Error.Error(ctx, err)
			return
		}

		outputStruct := taskSpeechRecognitionOutput{
			Text: resp.Text,
		}
		err = job.Output.WriteData(ctx, outputStruct)
		if err != nil {
			job.Error.Error(ctx, err)
			return
		}

	case TextToSpeechTask:
		inputStruct := taskTextToSpeechInput{}
		err := job.Input.ReadData(ctx, &inputStruct)
		if err != nil {
			job.Error.Error(ctx, err)
			return
		}

		req := client.R().SetBody(TextToSpeechReq{
			Input:          inputStruct.Text,
			Model:          inputStruct.Model,
			Voice:          inputStruct.Voice,
			ResponseFormat: inputStruct.ResponseFormat,
			Speed:          inputStruct.Speed,
		})

		resp, err := req.Post(createSpeechPath)
		if err != nil {
			job.Error.Error(ctx, err)
			return
		}

		audio, err := data.NewAudioFromBytes(resp.Body(), "audio/wav", "")
		if err != nil {
			job.Error.Error(ctx, err)
			return
		}

		outputStruct := taskTextToSpeechOutput{
			Audio: audio,
		}

		err = job.Output.WriteData(ctx, outputStruct)
		if err != nil {
			job.Error.Error(ctx, err)
			return
		}

	case TextToImageTask:

		inputStruct := taskTextToImageInput{}
		err := job.Input.ReadData(ctx, &inputStruct)
		if err != nil {
			job.Error.Error(ctx, err)
			return
		}

		resp := ImageGenerationsResp{}
		req := client.R().SetBody(ImageGenerationsReq{
			Model:          inputStruct.Model,
			Prompt:         inputStruct.Prompt,
			Quality:        inputStruct.Quality,
			Size:           inputStruct.Size,
			Style:          inputStruct.Style,
			N:              inputStruct.N,
			ResponseFormat: "b64_json",
		}).SetResult(&resp)

		if _, err := req.Post(imgGenerationPath); err != nil {
			job.Error.Error(ctx, err)
			return
		}

		results := []imageGenerationsOutputResult{}
		for _, d := range resp.Data {
			b, err := base64.StdEncoding.DecodeString(util.TrimBase64Mime(d.Image))
			if err != nil {
				job.Error.Error(ctx, err)
				return
			}
			img, err := data.NewImageFromBytes(b, data.PNG, "")
			if err != nil {
				job.Error.Error(ctx, err)
				return
			}
			results = append(results, imageGenerationsOutputResult{
				Image:         img,
				RevisedPrompt: d.RevisedPrompt,
			})
		}
		outputStruct := taskTextToImageOutput{
			Results: results,
		}

		err = job.Output.WriteData(ctx, outputStruct)
		if err != nil {
			job.Error.Error(ctx, err)
			return
		}

	default:
		job.Error.Error(ctx, errorsx.AddMessage(
			fmt.Errorf("not supported task: %s", e.Task),
			fmt.Sprintf("%s task is not supported.", e.Task),
		))
		return
	}
}

func chunk(items []*base.Job, batchSize int) (chunks [][]*base.Job) {
	for batchSize < len(items) {
		items, chunks = items[batchSize:], append(chunks, items[0:batchSize:batchSize])
	}
	return append(chunks, items)
}

func (e *execution) executeEmbedding(ctx context.Context, client *httpclient.Client, jobs []*base.Job) {

	texts := make([]string, len(jobs))
	dimensions := 0
	model := ""
	for idx, job := range jobs {
		inputStruct := taskTextEmbeddingsInput{}
		err := job.Input.ReadData(ctx, &inputStruct)
		if err != nil {
			job.Error.Error(ctx, err)
			return
		}
		texts[idx] = inputStruct.Text

		if idx == 0 {
			// Note: Currently, we assume that all data in the batch uses the same
			// model and dimension settings. We need to add a check for this:
			// if the model or dimensions differ, we should separate them into
			// different inference groups.
			dimensions = inputStruct.Dimensions
			model = inputStruct.Model
		}
	}

	resp := TextEmbeddingsResp{}

	var reqParams TextEmbeddingsReq
	if dimensions == 0 {
		reqParams = TextEmbeddingsReq{
			Model: model,
			Input: texts,
		}
	} else {
		reqParams = TextEmbeddingsReq{
			Model:      model,
			Input:      texts,
			Dimensions: dimensions,
		}
	}

	req := client.R().SetBody(reqParams).SetResult(&resp)
	if _, err := req.Post(embeddingsPath); err != nil {
		for _, job := range jobs {
			job.Error.Error(ctx, err)
		}
		return
	}

	for idx, job := range jobs {
		outputStruct := taskTextEmbeddingsOutput{
			Embedding: resp.Data[idx].Embedding,
		}
		err := job.Output.WriteData(ctx, outputStruct)
		if err != nil {
			job.Error.Error(ctx, err)
			return
		}
	}
}

func (e *execution) Execute(ctx context.Context, jobs []*base.Job) error {

	client := newClient(e.Setup, e.GetLogger())
	client.SetRetryCount(retryCount)
	client.SetRetryWaitTime(1 * time.Second)

	switch e.Task {
	case TextEmbeddingsTask:
		// OpenAI embedding API supports batch inference, so we'll leverage it
		// directly for optimal performance.
		e.executeEmbedding(ctx, client, jobs)

	default:
		// TODO: we can encapsulate this code into a `ConcurrentExecutor`.
		// The `ConcurrentExecutor` will use goroutines to execute jobs in parallel.
		batchSize := 4
		for _, batch := range chunk(jobs, batchSize) {
			var wg sync.WaitGroup
			wg.Add(len(batch))
			for _, job := range batch {
				go func() {
					defer wg.Done()
					e.worker(ctx, client, job)
				}()
			}
			wg.Wait()
		}

	}

	return nil
}

// Test checks the component state.
func (c *component) Test(_ map[string]any, setup *structpb.Struct) error {
	models := ListModelsResponse{}
	req := newClient(setup, c.GetLogger()).R().SetResult(&models)

	if _, err := req.Get(listModelsPath); err != nil {
		return err
	}

	if len(models.Data) == 0 {
		return fmt.Errorf("no models")
	}

	return nil
}
