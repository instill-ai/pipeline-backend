//go:generate compogen readme ./config ./README.mdx --extraContents bottom=.compogen/bottom.mdx
package universalai

import (
	"context"
	"fmt"
	"sync"

	_ "embed"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

const (
	TextChatTask    = "TASK_CHAT"
	cfgAPIKey       = "api-key"
	cfgOrganization = "organization"
	retryCount      = 3
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

	instillAPIKey map[string]string
}

type execution struct {
	base.ComponentExecution

	usesInstillCredentials bool
	execute                func(context.Context, *base.Job) error
}

// Init returns an initialized AI component.
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

func (c *component) CreateExecution(x base.ComponentExecution) (base.IExecution, error) {

	model := getModel(x.GetSetup())
	vendor := modelVendorMap[model]

	resolvedSetup, resolved, err := c.resolveSetup(vendor, x.Setup)

	if err != nil {
		return nil, err
	}

	x.Setup = resolvedSetup

	e := &execution{
		ComponentExecution:     x,
		usesInstillCredentials: resolved,
	}

	switch x.Task {
	case TextChatTask:
		e.execute = e.executeTextChat
	default:
		return nil, fmt.Errorf("unknown task: %s", x.Task)
	}

	return e, nil
}

func (e *execution) Execute(ctx context.Context, jobs []*base.Job) error {
	return base.ConcurrentExecutor(ctx, jobs, e.execute)
}

func (e *execution) UsesInstillCredentials() bool {
	return e.usesInstillCredentials
}

// WithInstillCredentials loads Instill credentials into the component, which
// can be used to configure it with globally defined parameters instead of with
// user-defined credential values.
func (c *component) WithInstillCredentials(vendor string, s map[string]any) *component {
	if c.instillAPIKey == nil {
		c.instillAPIKey = make(map[string]string)
	}
	c.instillAPIKey[vendor] = base.ReadFromGlobalConfig(cfgAPIKey, s)
	return c
}

// resolveSetup checks whether the component is configured to use the Instill
// credentials injected during initialization and, if so, returns a new setup
// with the secret credential values.
func (c *component) resolveSetup(vendor string, setup *structpb.Struct) (*structpb.Struct, bool, error) {
	if setup == nil || setup.Fields == nil {
		setup = &structpb.Struct{Fields: map[string]*structpb.Value{}}
	}
	if v, ok := setup.GetFields()[cfgAPIKey]; ok {
		apiKey := v.GetStringValue()
		if apiKey != "" && apiKey != base.SecretKeyword {
			return setup, false, nil
		}
	}

	if c.instillAPIKey[vendor] == "" {
		return nil, false, base.NewUnresolvedCredential(cfgAPIKey)
	}

	setup.GetFields()[cfgAPIKey] = structpb.NewStringValue(c.instillAPIKey[vendor])
	return setup, true, nil
}

func getModel(setup *structpb.Struct) string {
	return setup.GetFields()["model"].GetStringValue()
}

var modelVendorMap = map[string]string{
	"o1-preview":             "openai",
	"o1-mini":                "openai",
	"gpt-4o-mini":            "openai",
	"gpt-4o":                 "openai",
	"gpt-4o-2024-05-13":      "openai",
	"gpt-4o-2024-08-06":      "openai",
	"gpt-4-turbo":            "openai",
	"gpt-4-turbo-2024-04-09": "openai",
	"gpt-4-0125-preview":     "openai",
	"gpt-4-turbo-preview":    "openai",
	"gpt-4-1106-preview":     "openai",
	"gpt-4-vision-preview":   "openai",
	"gpt-4":                  "openai",
	"gpt-4-0314":             "openai",
	"gpt-4-0613":             "openai",
	"gpt-4-32k":              "openai",
	"gpt-4-32k-0314":         "openai",
	"gpt-4-32k-0613":         "openai",
	"gpt-3.5-turbo":          "openai",
	"gpt-3.5-turbo-16k":      "openai",
	"gpt-3.5-turbo-0301":     "openai",
	"gpt-3.5-turbo-0613":     "openai",
	"gpt-3.5-turbo-1106":     "openai",
	"gpt-3.5-turbo-0125":     "openai",
	"gpt-3.5-turbo-16k-0613": "openai",
}
