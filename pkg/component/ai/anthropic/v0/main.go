//go:generate compogen readme ./config ./README.mdx --extraContents bottom=.compogen/bottom.mdx
package anthropic

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"

	_ "embed"

	"google.golang.org/protobuf/types/known/structpb"

	anthropicsdk "github.com/anthropics/anthropic-sdk-go"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util"
)

const (
	TextGenerationTask = "TASK_TEXT_GENERATION_CHAT"
	cfgAPIKey          = "api-key"
	host               = "https://api.anthropic.com"
	messagesPath       = "/v1/messages"
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

	supportedImageExtensions = []string{"jpeg", "png", "gif", "webp"}
)

type component struct {
	base.Component

	instillAPIKey string
}

type AnthropicClient interface {
	generateTextChat(request messagesReq) (messagesResp, error)
}

// These structs are used to send the request /  parse the response from the API, this following their naming convension.
// reference: https://docs.anthropic.com/en/api/messages
type messagesResp struct {
	ID         string    `json:"id"`
	Type       string    `json:"type"`
	Role       string    `json:"role"`
	Content    []content `json:"content"`
	Model      string    `json:"model"`
	StopReason string    `json:"stop_reason,omitempty"`
	Usage      usage     `json:"usage"`
}

type messagesReq struct {
	Model         string      `json:"model"`
	Messages      []message   `json:"messages"`
	MaxTokens     int         `json:"max_tokens"`
	Metadata      interface{} `json:"metadata"`
	StopSequences []string    `json:"stop_sequences,omitempty"`
	Stream        bool        `json:"stream,omitempty"`
	System        string      `json:"system,omitempty"`
	Temperature   float32     `json:"temperature,omitempty"`
	TopK          int         `json:"top_k,omitempty"`
	TopP          float32     `json:"top_p,omitempty"`
}

type message struct {
	Role    string    `json:"role"`
	Content []content `json:"content"`
}

type usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// doesn't support anthropic tools at the moment
type content struct {
	Type   string  `json:"type"`
	Text   string  `json:"text,omitempty"`
	Source *source `json:"source,omitempty"`
}

type source struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
}

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

type execution struct {
	base.ComponentExecution

	execute                func(context.Context, *base.Job) error
	client                 *anthropicsdk.Client
	usesInstillCredentials bool
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

	e := &execution{
		ComponentExecution:     x,
		client:                 newClient(getAPIKey(resolvedSetup)),
		usesInstillCredentials: resolved,
	}
	switch x.Task {
	case TextGenerationTask:
		e.execute = e.generateText
	default:
		return nil, fmt.Errorf("unsupported task")
	}

	return e, nil
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

func (e *execution) UsesInstillCredentials() bool {
	return e.usesInstillCredentials
}

func (e *execution) Execute(ctx context.Context, jobs []*base.Job) error {
	return base.ConcurrentExecutor(ctx, jobs, e.execute)
}

func (e *execution) generateText(ctx context.Context, job *base.Job) error {

	var inputStruct MessagesInput
	err := job.Input.ReadData(ctx, &inputStruct)
	if err != nil {
		job.Error.Error(ctx, err)
		return err
	}

	messageParams := anthropicsdk.MessageNewParams{
		Model:     anthropicsdk.F(inputStruct.ModelName),
		MaxTokens: anthropicsdk.Int(int64(inputStruct.MaxNewTokens)),
		System: anthropicsdk.F([]anthropicsdk.TextBlockParam{
			anthropicsdk.NewTextBlock(inputStruct.SystemMsg),
		}),
		Temperature: anthropicsdk.F(float64(inputStruct.Temperature)),
		TopK:        anthropicsdk.Int(int64(inputStruct.TopK)),
	}
	messages := []anthropicsdk.MessageParam{}

	chatHistory := inputStruct.ChatHistory

	for _, chatMessage := range chatHistory {
		blocks := []anthropicsdk.MessageParamContentUnion{}
		for _, content := range chatMessage.Content {
			if content.Type == "text" {
				blocks = append(blocks, anthropicsdk.NewTextBlock(content.Text))
			} else {
				base64 := strings.Split(content.ImageURL.URL, ",")[1]
				contentType := strings.Split(content.ImageURL.URL, ";")[0][len("data:"):]
				blocks = append(blocks, anthropicsdk.NewImageBlockBase64(contentType, base64))
			}
		}
		if chatMessage.Role == "user" {
			messages = append(messages, anthropicsdk.NewUserMessage(blocks...))
		} else {
			messages = append(messages, anthropicsdk.NewAssistantMessage(blocks...))
		}
	}

	blocks := []anthropicsdk.MessageParamContentUnion{}
	blocks = append(blocks, anthropicsdk.NewTextBlock(inputStruct.Prompt))

	promptImages := inputStruct.PromptImages
	for _, image := range promptImages {
		extension := base.GetBase64FileExtension(image)
		// check if the image extension is supported
		if !slices.Contains(supportedImageExtensions, extension) {
			job.Error.Error(ctx, err)
			return fmt.Errorf("unsupported image extension, expected one of: %v , got %s", supportedImageExtensions, extension)
		}
		blocks = append(blocks, anthropicsdk.NewImageBlockBase64(fmt.Sprintf("image/%s", extension), util.TrimBase64Mime(image)))
	}
	messages = append(messages, anthropicsdk.NewUserMessage(blocks...))
	messageParams.Messages = anthropicsdk.F(messages)

	stream := e.client.Messages.NewStreaming(ctx, messageParams)

	message := anthropicsdk.Message{}
	text := ""
	outputStruct := MessagesOutput{
		Text: text,
		Usage: messagesUsage{
			InputTokens:  0,
			OutputTokens: 0,
		},
	}

	for stream.Next() {
		event := stream.Current()
		err = message.Accumulate(event)
		if err != nil {
			job.Error.Error(ctx, err)
			return err
		}

		switch delta := event.Delta.(type) {
		case anthropicsdk.ContentBlockDeltaEventDelta:

			if delta.Text != "" {
				text += delta.Text
				outputStruct.Text = text
				err = job.Output.WriteData(ctx, outputStruct)
				if err != nil {
					job.Error.Error(ctx, err)
					return err
				}
			}
		}
	}

	outputStruct.Usage.InputTokens = int(message.Usage.InputTokens)
	outputStruct.Usage.OutputTokens = int(message.Usage.OutputTokens)

	err = job.Output.WriteData(ctx, outputStruct)
	if err != nil {
		job.Error.Error(ctx, err)
		return err
	}

	return nil
}
