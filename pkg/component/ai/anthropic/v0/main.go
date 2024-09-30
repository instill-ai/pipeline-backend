//go:generate compogen readme ./config ./README.mdx --extraContents bottom=.compogen/bottom.mdx
package anthropic

import (
	"context"
	"fmt"
	"slices"
	"sync"

	_ "embed"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
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

type MessagesInput struct {
	ChatHistory  []ChatMessage `json:"chat-history"`
	MaxNewTokens int           `json:"max-new-tokens"`
	ModelName    string        `json:"model-name"`
	Prompt       string        `json:"prompt"`
	PromptImages []string      `json:"prompt-images"`
	Seed         int           `json:"seed"`
	SystemMsg    string        `json:"system-message"`
	Temperature  float32       `json:"temperature"`
	TopK         int           `json:"top-k"`
}

type ChatMessage struct {
	Role    string              `json:"role"`
	Content []MultiModalContent `json:"content"`
}

type MultiModalContent struct {
	ImageURL URL    `json:"image-url"`
	Text     string `json:"text"`
	Type     string `json:"type"`
}

type URL struct {
	URL string `json:"url"`
}

type MessagesOutput struct {
	Text  string        `json:"text"`
	Usage messagesUsage `json:"usage"`
}

type messagesUsage struct {
	InputTokens  int `json:"input-tokens"`
	OutputTokens int `json:"output-tokens"`
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

	execute                func(*structpb.Struct) (*structpb.Struct, error)
	client                 AnthropicClient
	usesInstillCredentials bool
}

// WithInstillCredentials loads Instill credentials into the component, which
// can be used to configure it with globally defined parameters instead of with
// user-defined credential values.
func (c *component) WithInstillCredentials(s map[string]any) *component {
	c.instillAPIKey = base.ReadFromGlobalConfig(cfgAPIKey, s)
	return c
}

// CreateExecution initializes a connector executor that can be used in a
// pipeline trigger.
func (c *component) CreateExecution(x base.ComponentExecution) (base.IExecution, error) {
	resolvedSetup, resolved, err := c.resolveSetup(x.Setup)
	if err != nil {
		return nil, err
	}

	x.Setup = resolvedSetup

	e := &execution{
		ComponentExecution:     x,
		client:                 newClient(getAPIKey(resolvedSetup), getBasePath(resolvedSetup), c.GetLogger()),
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
	return base.SequentialExecutor(ctx, jobs, e.execute)
}

func (e *execution) generateText(in *structpb.Struct) (*structpb.Struct, error) {

	var inputStruct MessagesInput
	err := base.ConvertFromStructpb(in, &inputStruct)
	if err != nil {
		return nil, err
	}

	prompt := inputStruct.Prompt

	messages := []message{}

	chatHistory := inputStruct.ChatHistory

	for _, chatMessage := range chatHistory {
		contents := getContents(chatMessage)
		message := message{Role: chatMessage.Role, Content: contents}
		messages = append(messages, message)
	}

	finalMessage := message{
		Role:    "user",
		Content: []content{{Type: "text", Text: prompt}},
	}

	promptImages := inputStruct.PromptImages
	for _, image := range promptImages {
		extension := base.GetBase64FileExtension(image)
		// check if the image extension is supported
		if !slices.Contains(supportedImageExtensions, extension) {
			return nil, fmt.Errorf("unsupported image extension, expected one of: %v , got %s", supportedImageExtensions, extension)
		}
		image := content{
			Type:   "image",
			Source: &source{Type: "base64", MediaType: fmt.Sprintf("image/%s", extension), Data: base.TrimBase64Mime(image)},
		}
		finalMessage.Content = append(finalMessage.Content, image)
	}

	messages = append(messages, finalMessage)

	req := messagesReq{
		Messages:    messages,
		Model:       inputStruct.ModelName,
		MaxTokens:   inputStruct.MaxNewTokens,
		System:      inputStruct.SystemMsg,
		TopK:        inputStruct.TopK,
		Temperature: float32(inputStruct.Temperature),
	}

	resp, err := e.client.generateTextChat(req)

	if err != nil {
		return nil, err
	}

	outputStruct := MessagesOutput{
		Text: "",
		Usage: messagesUsage{
			InputTokens:  resp.Usage.InputTokens,
			OutputTokens: resp.Usage.OutputTokens,
		},
	}
	for _, c := range resp.Content {
		outputStruct.Text += c.Text
	}

	output, err := base.ConvertToStructpb(outputStruct)
	if err != nil {
		return nil, err
	}
	return output, nil
}

func getContents(chatMessage ChatMessage) []content {
	contents := []content{}
	for _, multiModalContent := range chatMessage.Content {
		if multiModalContent.Type == "text" {
			contentReq := content{
				Type: "text",
				Text: multiModalContent.Text,
			}
			contents = append(contents, contentReq)
		}
	}

	return contents
}
