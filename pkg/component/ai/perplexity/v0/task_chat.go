package perplexity

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util"
	"github.com/instill-ai/x/errmsg"
)

const (
	chatPath = "chat/completions"
)

func (e *execution) executeTextChat(ctx context.Context, job *base.Job) error {
	logger := e.GetLogger()
	client := newClient(e.Setup, logger)

	var inputStruct TextChatInput

	if err := job.Input.ReadData(ctx, &inputStruct); err != nil {
		err = fmt.Errorf("reading input data: %w", err)
		job.Error.Error(ctx, err)
		return err
	}

	chatReq, err := buildChatReq(inputStruct)

	if err != nil {
		err = fmt.Errorf("building chat request: %w", err)
		job.Error.Error(ctx, err)
		return err
	}

	req := client.SetDoNotParseResponse(true).R().SetBody(chatReq)

	restyResp, err := req.Post(chatPath)

	if restyResp.StatusCode() != 200 {
		rawBody := restyResp.RawBody()
		defer rawBody.Close()
		bodyBytes, err := io.ReadAll(rawBody)
		if err != nil {
			return fmt.Errorf("read response body: %w", err)
		}
		chatReqBytes, _ := json.Marshal(chatReq)
		logger.Error("Failed to send request to Perplexity",
			zap.Binary("response body", bodyBytes),
			zap.Int("status", restyResp.StatusCode()),
			zap.Binary("chatReq", chatReqBytes),
		)

		msg := fmt.Sprintf("Perplexity API responded with status %d", restyResp.StatusCode())
		err = errmsg.AddMessage(fmt.Errorf("perplexity responded with non-200 status"), msg)
		job.Error.Error(ctx, err)
		return err
	}

	if err != nil {
		return fmt.Errorf("sending chat request to Perplexity: %w", err)
	}

	if chatReq.Stream {
		outputStruct, err := streaming(ctx, restyResp, job)
		if err != nil {
			err = fmt.Errorf("streaming: %w", err)
			job.Error.Error(ctx, err)
			return err
		}

		if outputStruct == nil {
			err = fmt.Errorf("streaming: output struct is nil")
			job.Error.Error(ctx, err)
			return err
		}

		err = job.Output.WriteData(ctx, *outputStruct)

		if err != nil {
			err = fmt.Errorf("writing output data: %w", err)
			job.Error.Error(ctx, err)
			return err
		}

		return nil
	}

	resp := chatResp{}
	rawBody := restyResp.RawBody()

	defer rawBody.Close()
	bodyBytes, err := io.ReadAll(rawBody)

	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}

	if err = json.Unmarshal(bodyBytes, &resp); err != nil {
		return fmt.Errorf("unmarshal response body: %w", err)
	}

	outputStruct := buildOutputStruct(resp)

	err = job.Output.WriteData(ctx, outputStruct)

	if err != nil {
		err = fmt.Errorf("writing output data: %w", err)
		job.Error.Error(ctx, err)
		return err
	}

	return nil
}

func buildChatReq(inputStruct TextChatInput) (*chatReq, error) {

	var messages []message
	var userMessage message
	var userMessageCount int
	for _, inputMessage := range inputStruct.Data.Messages {
		for _, content := range inputMessage.Content {
			// In Perplexity API, it only accepts one user message, and it should be the last message in the list.
			if inputMessage.Role == "user" {
				userMessageCount++
				userMessage = message{
					Content: content.Text,
					Role:    inputMessage.Role,
				}
				continue
			}
			messages = append(messages, message{
				Content: content.Text,
				Role:    inputMessage.Role,
			})
		}
	}

	if userMessageCount != 1 {
		return nil, fmt.Errorf("expected exactly one user message, got %d", userMessageCount)
	}

	messages = append(messages, userMessage)

	return &chatReq{
		Model:               inputStruct.Data.Model,
		Messages:            messages,
		MaxTokens:           inputStruct.Parameter.MaxTokens,
		Temperature:         inputStruct.Parameter.Temperature,
		TopP:                inputStruct.Parameter.TopP,
		SearchDomainFilter:  inputStruct.Parameter.SearchDomainFilter,
		SearchRecencyFilter: inputStruct.Parameter.SearchRecencyFilter,
		TopK:                inputStruct.Parameter.TopK,
		Stream:              inputStruct.Parameter.Stream,
		PresencePenalty:     inputStruct.Parameter.PresencePenalty,
		FrequencyPenalty:    inputStruct.Parameter.FrequencyPenalty,
	}, nil
}

func buildOutputStruct(resp chatResp) TextChatOutput {
	var outputData OutputData

	var choices []Choice
	for _, choice := range resp.Choices {
		choices = append(choices, Choice{
			Index:        choice.Index,
			FinishReason: choice.FinishReason,
			Message: OutputMessage{
				Content: choice.Message.Content,
				Role:    choice.Message.Role,
			},
			Created: util.UnixToISO8601(resp.Created),
		})
	}

	outputData.Choices = choices

	outputData.Citations = append(outputData.Citations, resp.Citations...)
	return TextChatOutput{
		Data: outputData,
		Metadata: Metadata{
			Usage: Usage{
				PromptTokens:     resp.Usage.PromptTokens,
				CompletionTokens: resp.Usage.CompletionTokens,
				TotalTokens:      resp.Usage.TotalTokens,
			},
		},
	}
}

func streaming(ctx context.Context, resp *resty.Response, job *base.Job) (*TextChatOutput, error) {
	scanner := bufio.NewScanner(resp.RawResponse.Body)
	count := 0
	var err error
	var outputStruct TextChatOutput

	for scanner.Scan() {
		res := scanner.Text()

		if len(res) == 0 || !strings.HasPrefix(res, "data: ") {
			continue
		}

		res = strings.Replace(res, "data: ", "", 1)
		count += 1

		response := chatResp{}
		if err = json.Unmarshal([]byte(res), &response); err != nil {
			return nil, fmt.Errorf("unmarshal streaming response: %w", err)
		}

		outputStruct = buildOutputStruct(response)

		// Note: Since we haven’t provided delta updates for the
		// messages, we’re reducing the number of event streams by
		// returning the response every ten iterations.
		if count == 10 {
			// In case duplication of credit consumption, we remove the usage from the
			// streaming output struct.
			streamingOutputStruct := outputStruct
			streamingOutputStruct.Metadata.Usage = Usage{}
			err = job.Output.WriteData(ctx, streamingOutputStruct)
			if err != nil {
				return nil, fmt.Errorf("writing streaming output data: %w", err)
			}
			count = 0
		}

	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading streaming response: %w", err)
	}

	return &outputStruct, nil
}

type chatReq struct {
	Model               string    `json:"model"`
	Messages            []message `json:"messages"`
	MaxTokens           int       `json:"max_tokens"`
	Temperature         float64   `json:"temperature"`
	TopP                float64   `json:"top_p"`
	SearchDomainFilter  []string  `json:"search_domain_filter"`
	SearchRecencyFilter string    `json:"search_recency_filter"`
	TopK                int       `json:"top_k"`
	Stream              bool      `json:"stream"`
	PresencePenalty     float64   `json:"presence_penalty"`
	FrequencyPenalty    float64   `json:"frequency_penalty"`
}

type message struct {
	Content string `json:"content"`
	Role    string `json:"role"`
}

type chatResp struct {
	ID        string   `json:"id"`
	Model     string   `json:"model"`
	Created   int64    `json:"created"`
	Usage     usage    `json:"usage"`
	Citations []string `json:"citations"`
	Object    string   `json:"object"`
	Choices   []choice `json:"choices"`
}

type usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type choice struct {
	Index        int     `json:"index"`
	FinishReason string  `json:"finish_reason"`
	Message      message `json:"message"`
	Delta        delta   `json:"delta"`
}

type delta struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
