//go:generate compogen readme ./config ./README.mdx
package stabilityai

import (
	"context"
	"fmt"
	"sync"

	_ "embed"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util/httpclient"
)

const (
	host = "https://api.stability.ai"

	TextToImageTask  = "TASK_TEXT_TO_IMAGE"
	ImageToImageTask = "TASK_IMAGE_TO_IMAGE"

	cfgAPIKey = "api-key"
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

// Component executes queries against StabilityAI.
type component struct {
	base.Component

	instillAPIKey string
}

// Init returns an initialized StabilityAI component.
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

// WithInstillCredentials loads Instill credentials into the component, which
// can be used to configure it with globally defined parameters instead of with
// user-defined credential values.
func (c *component) WithInstillCredentials(s map[string]any) *component {
	c.instillAPIKey = base.ReadFromGlobalConfig(cfgAPIKey, s)
	return c
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

// CreateExecution initializes a component executor that can be used in a
// pipeline trigger.
func (c *component) CreateExecution(x base.ComponentExecution) (base.IExecution, error) {
	resolvedSetup, resolved, err := c.resolveSetup(x.Setup)
	if err != nil {
		return nil, err
	}

	x.Setup = resolvedSetup
	client := newClient(x.Setup, c.GetLogger())

	e := &execution{
		ComponentExecution:     x,
		usesInstillCredentials: resolved,
		client:                 client,
	}

	switch x.Task {
	case TextToImageTask:
		e.execute = e.handleTextToImage
	case ImageToImageTask:
		e.execute = e.handleImageToImage
	default:
		return nil, fmt.Errorf("unsupported task: %s", x.Task)
	}
	return e, nil
}

type execution struct {
	base.ComponentExecution
	usesInstillCredentials bool
	client                 *httpclient.Client
	execute                func(context.Context, *base.Job) error
}

func (e *execution) UsesInstillCredentials() bool {
	return e.usesInstillCredentials
}

func (e *execution) Execute(ctx context.Context, jobs []*base.Job) error {
	return base.ConcurrentExecutor(ctx, jobs, e.execute)
}

// Test checks the component state.
func (c *component) Test(sysVars map[string]any, setup *structpb.Struct) error {
	var engines []Engine
	req := newClient(setup, c.GetLogger()).R().SetResult(&engines)

	if _, err := req.Get(listEnginesPath); err != nil {
		return err
	}

	if len(engines) == 0 {
		return fmt.Errorf("no engines")
	}

	return nil
}
