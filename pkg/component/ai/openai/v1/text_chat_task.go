package openaiv1

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/go-resty/resty/v2"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/ai"
	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util/httpclient"
)

const (
	completionsPath = "/v1/chat/completions"
)

func (e *execution) ExecuteTextChat(ctx context.Context, job *base.Job) error {
	inputStruct := ai.TextChatInput{}

	input, err := job.Input.Read(ctx)
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}

	if err := base.ConvertFromStructpb(input, &inputStruct); err != nil {
		return fmt.Errorf("failed to convert input to TextChatInput: %w", err)
	}

	return ExecuteTextChat(ctx, inputStruct, e.client, job)
}

func ExecuteTextChat(ctx context.Context, inputStruct ai.TextChatInput, client httpclient.IClient, job *base.Job) error {

	requester := ModelRequesterFactory(inputStruct, client)

	return requester.SendChatRequest(ctx, job)
}

func ModelRequesterFactory(input ai.TextChatInput, client httpclient.IClient) IChatModelRequester {
	ChatModelList := []string{
		"gpt-3.5-turbo-16k",
		"gpt-4",
		"gpt-4-0314",
		"gpt-4-0613",
		"gpt-4-32k",
		"gpt-4-32k-0314",
		"gpt-4-32k-0613",
		"gpt-3.5-turbo-0301",
		"gpt-3.5-turbo-0125",
		"gpt-3.5-turbo-16k-0613",
	}

	model := input.Data.Model
	if util.InSlice(ChatModelList, model) {
		return &ChatModelRequester{
			Input:  input,
			Client: client,
		}
	}
	if model == "o1-preview" || model == "o1-mini" {
		return &O1ModelRequester{
			Input:  input,
			Client: client,
		}
	}
	return &SupportJSONOutputModelRequester{
		Input:  input,
		Client: client,
	}
}

type IChatModelRequester interface {
	SendChatRequest(context.Context, *base.Job) error
}

// o1-preview or o1-mini
type O1ModelRequester struct {
	Input  ai.TextChatInput
	Client httpclient.IClient
}

// When it supports streaming, the job and ctx will be used.
func (r *O1ModelRequester) SendChatRequest(ctx context.Context, job *base.Job) error {

	input := r.Input
	// Note: The o1-series models don't support streaming.
	input.Parameter.Stream = false

	chatReq := convertToTextChatReq(input)

	resp := textChatResp{}
	client := r.Client

	req := client.R().SetResult(&resp).SetBody(chatReq)

	if resp, err := req.Post(completionsPath); err != nil {
		errMsg := resp.Body()
		return fmt.Errorf("failed to send chat request: %w, %s", err, errMsg)
	}

	outputStruct := ai.TextChatOutput{}
	setOutputStruct(&outputStruct, resp)

	out, err := base.ConvertToStructpb(outputStruct)
	if err != nil {
		return err
	}
	return job.Output.Write(ctx, out)
}

// https://platform.openai.com/docs/api-reference/chat/create#chat-create-response_format
// Compatible with GPT-4o, GPT-4o mini, GPT-4 Turbo and all GPT-3.5 Turbo models newer than gpt-3.5-turbo-1106.
type SupportJSONOutputModelRequester struct {
	Input  ai.TextChatInput
	Client httpclient.IClient
}

func (r *SupportJSONOutputModelRequester) SendChatRequest(ctx context.Context, job *base.Job) error {

	input := r.Input

	chatReq := convertToTextChatReq(input)

	// Note: Add response format to the request.
	// We will need to think about how to customize input & output for standardized AI components.

	output, err := sendRequest(chatReq, r.Client, job, ctx)

	if err != nil {
		return err
	}

	out, err := base.ConvertToStructpb(output)
	if err != nil {
		return err
	}
	return job.Output.Write(ctx, out)
}

type ChatModelRequester struct {
	Input  ai.TextChatInput
	Client httpclient.IClient
}

func (r *ChatModelRequester) SendChatRequest(ctx context.Context, job *base.Job) error {

	input := r.Input

	chatReq := convertToTextChatReq(input)
	chatReq.Stream = true
	chatReq.StreamOptions = &streamOptions{
		IncludeUsage: true,
	}

	output, err := sendRequest(chatReq, r.Client, job, ctx)

	if err != nil {
		return err
	}

	out, err := base.ConvertToStructpb(output)
	if err != nil {
		return err
	}
	return job.Output.Write(ctx, out)
}

func sendRequest(chatReq textChatReq, client httpclient.IClient, job *base.Job, ctx context.Context) (ai.TextChatOutput, error) {

	req := client.SetDoNotParseResponse(true).R().SetBody(chatReq)

	outputStruct := ai.TextChatOutput{}
	restyResp, err := req.Post(completionsPath)

	if err != nil {
		return outputStruct, fmt.Errorf("failed to send chat request: %w", err)
	}

	if restyResp.StatusCode() != 200 {
		rawBody := restyResp.RawBody()
		defer rawBody.Close()
		bodyBytes, err := io.ReadAll(rawBody)
		return outputStruct, fmt.Errorf("send request to openai error with error code: %d, msg %s, %s", restyResp.StatusCode(), bodyBytes, err)
	}

	if chatReq.Stream {
		err = streaming(restyResp, job, ctx, &outputStruct)
		if err != nil {
			return outputStruct, fmt.Errorf("failed to stream response: %w", err)
		}

		return outputStruct, nil
	}

	resp := textChatResp{}
	rawBody := restyResp.RawBody()
	defer rawBody.Close()
	bodyBytes, err := io.ReadAll(rawBody)
	if err != nil {
		return outputStruct, fmt.Errorf("failed to read response body: %w", err)
	}

	err = json.Unmarshal(bodyBytes, &resp)
	if err != nil {
		return outputStruct, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	setOutputStruct(&outputStruct, resp)

	return outputStruct, nil
}

func streaming(resp *resty.Response, job *base.Job, ctx context.Context, outputStruct *ai.TextChatOutput) error {

	scanner := bufio.NewScanner(resp.RawResponse.Body)
	count := 0
	var err error

	for scanner.Scan() {

		res := scanner.Text()

		if len(res) == 0 {
			continue
		}

		res = strings.Replace(res, "data: ", "", 1)

		log.Printf("openai response: %s", res)

		// Note: Since we haven’t provided delta updates for the
		// messages, we’re reducing the number of event streams by
		// returning the response every ten iterations.
		if count == 10 || res == "[DONE]" {
			outputJSON, inErr := json.Marshal(outputStruct)
			if inErr != nil {
				return inErr
			}
			output := &structpb.Struct{}
			inErr = protojson.Unmarshal(outputJSON, output)
			if inErr != nil {
				return inErr
			}
			err = job.Output.Write(ctx, output)
			if err != nil {
				return err
			}
			if res == "[DONE]" {
				break
			}
			count = 0
		}

		count += 1
		response := &textChatStreamResp{}
		err = json.Unmarshal([]byte(res), response)

		if err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}

		for _, c := range response.Choices {
			// Now, there is no document to describe it.
			// But, when we test it, we found that the choices idx is not in order.
			// So, we need to get idx from the choice, and the len of the choices is always 1.
			responseIdx := c.Index

			if responseIdx >= len(outputStruct.Data.Choices) {
				outputStruct.Data.Choices = append(outputStruct.Data.Choices, ai.Choice{})
			}

			outputStruct.Data.Choices[responseIdx].Message.Content += c.Delta.Content

			if c.Delta.Role != "" {
				outputStruct.Data.Choices[responseIdx].Message.Role = c.Delta.Role
			}

			if c.FinishReason != "" {
				outputStruct.Data.Choices[responseIdx].FinishReason = c.FinishReason
			}
			outputStruct.Data.Choices[responseIdx].Index = c.Index
			outputStruct.Data.Choices[responseIdx].Created = response.Created
		}

		if response.Usage.TotalTokens > 0 {
			outputStruct.Metadata.Usage = ai.Usage{
				CompletionTokens: response.Usage.ChatTokens,
				PromptTokens:     response.Usage.PromptTokens,
				TotalTokens:      response.Usage.TotalTokens,
			}
		}
	}
	return nil
}

// Build the vendor-specific request structure
func convertToTextChatReq(input ai.TextChatInput) textChatReq {
	messages := buildMessages(input)

	params := input.Parameter
	stream := params.Stream

	textChatReq := textChatReq{
		Model:       input.Data.Model,
		Messages:    messages,
		MaxTokens:   params.MaxTokens,
		Temperature: params.Temperature,
		N:           params.N,
		TopP:        params.TopP,
		Seed:        params.Seed,
	}

	if stream {
		textChatReq.Stream = true
		textChatReq.StreamOptions = &streamOptions{
			IncludeUsage: true,
		}
		return textChatReq
	}
	return textChatReq
}

func buildMessages(input ai.TextChatInput) []interface{} {
	messages := make([]interface{}, len(input.Data.Messages))
	for i, msg := range input.Data.Messages {
		content := make([]map[string]interface{}, len(msg.Contents))
		for j, c := range msg.Contents {
			content[j] = map[string]interface{}{
				"type": c.Type,
			}
			if c.Type == "text" {
				content[j]["text"] = c.Text
			}
			if c.Type == "image-url" {
				content[j]["image_url"] = c.ImageURL
			}
			if c.Type == "image-base64" {
				content[j]["image_url"] = util.GetDataURL(c.ImageBase64)
			}
		}

		if msg.Name == "" {
			messages[i] = map[string]interface{}{
				"role":    msg.Role,
				"content": content,
			}
		} else {
			messages[i] = map[string]interface{}{
				"role":    msg.Role,
				"name":    msg.Name,
				"content": content,
			}
		}
	}

	return messages
}

func setOutputStruct(outputStruct *ai.TextChatOutput, resp textChatResp) {
	outputStruct.Data.Choices = make([]ai.Choice, len(resp.Choices))
	for i, choice := range resp.Choices {
		outputStruct.Data.Choices[i] = ai.Choice{
			FinishReason: choice.FinishReason,
			Index:        choice.Index,
			Message: ai.OutputMessage{
				Content: choice.Message.Content,
				Role:    choice.Message.Role,
			},
			Created: resp.Created,
		}
	}

	outputStruct.Metadata.Usage = ai.Usage{
		CompletionTokens: resp.Usage.ChatTokens,
		PromptTokens:     resp.Usage.PromptTokens,
		TotalTokens:      resp.Usage.TotalTokens,
	}
}

// API request and response structures
type textChatReq struct {
	Model            string                   `json:"model"`
	Messages         []interface{}            `json:"messages"`
	Temperature      *float32                 `json:"temperature,omitempty"`
	TopP             *float32                 `json:"top_p,omitempty"`
	N                *int                     `json:"n,omitempty"`
	Stop             *string                  `json:"stop,omitempty"`
	Seed             *int                     `json:"seed,omitempty"`
	MaxTokens        *int                     `json:"max_tokens,omitempty"`
	PresencePenalty  *float32                 `json:"presence_penalty,omitempty"`
	FrequencyPenalty *float32                 `json:"frequency_penalty,omitempty"`
	ResponseFormat   *responseFormatReqStruct `json:"response_format,omitempty"`
	Stream           bool                     `json:"stream"`
	StreamOptions    *streamOptions           `json:"stream_options,omitempty"`
}

type streamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

type responseFormatReqStruct struct {
	Type       string         `json:"type,omitempty"`
	JSONSchema map[string]any `json:"json_schema,omitempty"`
}

type textChatStreamResp struct {
	ID      string          `json:"id"`
	Object  string          `json:"object"`
	Created int             `json:"created"`
	Choices []streamChoices `json:"choices"`
	Usage   usageOpenAI     `json:"usage"`
}

type textChatResp struct {
	ID      string      `json:"id"`
	Object  string      `json:"object"`
	Created int         `json:"created"`
	Choices []choice    `json:"choices"`
	Usage   usageOpenAI `json:"usage"`
}

type outputMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type streamChoices struct {
	Index        int           `json:"index"`
	FinishReason string        `json:"finish_reason"`
	Delta        outputMessage `json:"delta"`
}

type choice struct {
	Index        int           `json:"index"`
	FinishReason string        `json:"finish_reason"`
	Message      outputMessage `json:"message"`
}

type usageOpenAI struct {
	PromptTokens int `json:"prompt_tokens"`
	ChatTokens   int `json:"completion_tokens"`
	TotalTokens  int `json:"total_tokens"`
}
