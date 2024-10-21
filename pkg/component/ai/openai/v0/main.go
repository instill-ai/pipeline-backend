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
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util/httpclient"
	"github.com/instill-ai/x/errmsg"
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
	//go:embed config/definition.json
	definitionJSON []byte
	//go:embed config/setup.json
	setupJSON []byte
	//go:embed config/tasks.json
	tasksJSON []byte

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
		err := comp.LoadDefinition(definitionJSON, setupJSON, tasksJSON, nil)
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

		// If chat history is provided, add it to the messages, and ignore the system message
		if inputStruct.ChatHistory != nil {
			for _, chat := range inputStruct.ChatHistory {
				if chat.Role == "user" {
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
				} else {
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
		} else if inputStruct.SystemMessage != nil {
			// If chat history is not provided, add the system message to the messages
			messages = append(messages, message{Role: "system", Content: *inputStruct.SystemMessage})
		}
		userContents := []Content{}
		userContents = append(userContents, Content{Type: "text", Text: &inputStruct.Prompt})
		for _, image := range inputStruct.Images {
			i, err := image.DataURI("image/png")
			if err != nil {
				job.Error.Error(ctx, err)
				return
			}
			userContents = append(userContents, Content{Type: "image_url", ImageURL: &ImageURL{URL: i.String()}})
		}
		messages = append(messages, multiModalMessage{Role: "user", Content: userContents})

		// Note: The o1-series models don't support streaming.
		if inputStruct.Model == "o1-preview" || inputStruct.Model == "o1-mini" {

			body := textCompletionReq{
				Messages:         messages,
				Model:            inputStruct.Model,
				MaxTokens:        inputStruct.MaxTokens,
				Temperature:      inputStruct.Temperature,
				N:                inputStruct.N,
				TopP:             inputStruct.TopP,
				PresencePenalty:  inputStruct.PresencePenalty,
				FrequencyPenalty: inputStruct.FrequencyPenalty,
			}
			resp := textCompletionResp{}
			req := client.R().SetResult(&resp).SetBody(body)
			if _, err := req.Post(completionsPath); err != nil {
				job.Error.Error(ctx, err)
				return
			}

			outputStruct := taskTextGenerationOutput{
				Texts: []string{},
				Usage: usage(resp.Usage),
			}
			for _, c := range resp.Choices {
				outputStruct.Texts = append(outputStruct.Texts, c.Message.Content)
			}

			err = job.Output.WriteData(ctx, outputStruct)
			if err != nil {
				job.Error.Error(ctx, err)
				return
			}

		} else {
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

			// workaround, the OpenAI service can not accept this param
			if inputStruct.Model != "gpt-4-vision-preview" {
				if inputStruct.ResponseFormat != nil {
					body.ResponseFormat = &responseFormatReqStruct{
						Type: inputStruct.ResponseFormat.Type,
					}
					if inputStruct.ResponseFormat.Type == "json_schema" {
						if inputStruct.Model == "gpt-4o-mini" || inputStruct.Model == "gpt-4o-2024-08-06" {
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

						} else {
							job.Error.Error(ctx, fmt.Errorf("this model doesn't support response format: json_schema"))
							return
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

			u := usage{}
			count := 0
			for scanner.Scan() {

				res := scanner.Text()

				if len(res) == 0 {
					continue
				}
				res = strings.Replace(res, "data: ", "", 1)

				// Note: Since we haven’t provided delta updates for the
				// messages, we’re reducing the number of event streams by
				// returning the response every ten iterations.
				if count == 10 || res == "[DONE]" {
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
					// Now, there is no document to describe it.
					// But, when we test it, we found that the choices idx is not in order.
					// So, we need to get idx from the choice, and the len of the choices is always 1.
					responseIdx := c.Index
					if len(outputStruct.Texts) <= responseIdx {
						outputStruct.Texts = append(outputStruct.Texts, "")
					}
					outputStruct.Texts[responseIdx] += c.Delta.Content

				}

				u = usage{
					PromptTokens:     response.Usage.PromptTokens,
					CompletionTokens: response.Usage.CompletionTokens,
					TotalTokens:      response.Usage.TotalTokens,
				}

			}

			outputStruct.Usage = u
			err = job.Output.WriteData(ctx, outputStruct)
			if err != nil {
				job.Error.Error(ctx, err)
				return
			}
		}

	case SpeechRecognitionTask:
		input, err := job.Input.Read(ctx)
		if err != nil {
			job.Error.Error(ctx, err)
			return
		}
		inputStruct := AudioTranscriptionInput{}
		err = base.ConvertFromStructpb(input, &inputStruct)
		if err != nil {
			job.Error.Error(ctx, err)

			return
		}

		audioBytes, err := base64.StdEncoding.DecodeString(base.TrimBase64Mime(inputStruct.Audio))
		if err != nil {
			job.Error.Error(ctx, err)
			return
		}

		data, ct, err := getBytes(AudioTranscriptionReq{
			File:        audioBytes,
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

		output, err := base.ConvertToStructpb(resp)
		if err != nil {
			job.Error.Error(ctx, err)
			return
		}
		err = job.Output.Write(ctx, output)
		if err != nil {
			job.Error.Error(ctx, err)
			return
		}

	case TextToSpeechTask:
		input, err := job.Input.Read(ctx)
		if err != nil {
			job.Error.Error(ctx, err)
			return
		}
		inputStruct := TextToSpeechInput{}
		err = base.ConvertFromStructpb(input, &inputStruct)
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

		audio := base64.StdEncoding.EncodeToString(resp.Body())
		outputStruct := TextToSpeechOutput{
			Audio: fmt.Sprintf("data:audio/wav;base64,%s", audio),
		}

		output, err := base.ConvertToStructpb(outputStruct)
		if err != nil {
			job.Error.Error(ctx, err)
			return
		}
		err = job.Output.Write(ctx, output)
		if err != nil {
			job.Error.Error(ctx, err)
			return
		}

	case TextToImageTask:
		input, err := job.Input.Read(ctx)
		if err != nil {
			job.Error.Error(ctx, err)
			return
		}
		inputStruct := ImagesGenerationInput{}
		err = base.ConvertFromStructpb(input, &inputStruct)
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

		results := []ImageGenerationsOutputResult{}
		for _, data := range resp.Data {
			results = append(results, ImageGenerationsOutputResult{
				Image:         fmt.Sprintf("data:image/webp;base64,%s", data.Image),
				RevisedPrompt: data.RevisedPrompt,
			})
		}
		outputStruct := ImageGenerationsOutput{
			Results: results,
		}

		output, err := base.ConvertToStructpb(outputStruct)
		if err != nil {
			job.Error.Error(ctx, err)
			return
		}
		err = job.Output.Write(ctx, output)
		if err != nil {
			job.Error.Error(ctx, err)
			return
		}

	default:
		job.Error.Error(ctx, errmsg.AddMessage(
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
		input, err := job.Input.Read(ctx)
		if err != nil {
			job.Error.Error(ctx, err)
			return
		}
		inputStruct := TextEmbeddingsInput{}
		err = base.ConvertFromStructpb(input, &inputStruct)
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
		outputStruct := TextEmbeddingsOutput{
			Embedding: resp.Data[idx].Embedding,
		}
		output, err := base.ConvertToStructpb(outputStruct)
		if err != nil {
			job.Error.Error(ctx, err)
			return
		}
		err = job.Output.Write(ctx, output)
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
