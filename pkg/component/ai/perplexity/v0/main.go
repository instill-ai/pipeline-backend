//go:generate compogen readme ./config ./README.mdx --extraContents bottom=.compogen/bottom.mdx
package perplexity

import (
	"context"
	"fmt"
	"sync"

	_ "embed"

	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util/httpclient"
)

const (
	TaskChat = "TASK_CHAT"

	cfgAPIKey = "api-key"
	baseURL   = "https://api.perplexity.ai/"
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

type component struct {
	base.Component

	instillAPIKey string
}

type execution struct {
	base.ComponentExecution

	execute                func(context.Context, *base.Job) error
	usesInstillCredentials bool
}

// Init initializes the component with the provided base component.
func Init(bc base.Component) *component {
	once.Do(func() {
		comp = &component{Component: bc}
		err := comp.LoadDefinition(definitionJSON, setupJSON, tasksJSON, nil, nil)
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

	e := &execution{
		ComponentExecution:     x,
		usesInstillCredentials: resolved,
	}
	switch x.Task {
	case TaskChat:
		e.execute = e.executeTextChat
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

// UsesInstillCredentials returns whether the component is configured to use the
// Instill credentials injected during initialization.
func (e *execution) UsesInstillCredentials() bool {
	return e.usesInstillCredentials
}

// Execute runs the component's task with the provided jobs.
func (e *execution) Execute(ctx context.Context, jobs []*base.Job) error {
	return base.ConcurrentExecutor(ctx, jobs, e.execute)
}

type errBody struct {
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

// Message returns the error message from the response body.
func (e errBody) Message() string {
	return e.Error.Message
}

func newClient(setup *structpb.Struct, logger *zap.Logger) *httpclient.Client {
	c := httpclient.New("Perplexity", baseURL,
		httpclient.WithLogger(logger),
		httpclient.WithEndUserError(new(errBody)),
	)

	c.SetAuthToken(getAPIKey(setup))
	return c
}

func getAPIKey(setup *structpb.Struct) string {
	return setup.GetFields()["api-key"].GetStringValue()
}
