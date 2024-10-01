//go:generate compogen readme ./config ./README.mdx
package cohere

import (
	"context"
	"fmt"
	"sync"

	_ "embed"

	"google.golang.org/protobuf/types/known/structpb"

	cohereSDK "github.com/cohere-ai/cohere-go/v2"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

const (
	TextGenerationTask = "TASK_TEXT_GENERATION_CHAT"
	TextEmbeddingTask  = "TASK_TEXT_EMBEDDINGS"
	TextRerankTask     = "TASK_TEXT_RERANKING"
	cfgAPIKey          = "api-key"
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

type CohereClient interface {
	generateTextChat(request cohereSDK.ChatRequest) (cohereSDK.NonStreamedChatResponse, error)
	generateEmbedding(request cohereSDK.EmbedRequest) (cohereSDK.EmbedResponse, error)
	generateRerank(request cohereSDK.RerankRequest) (cohereSDK.RerankResponse, error)
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
	client                 CohereClient
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
		client:                 newClient(getAPIKey(resolvedSetup), c.GetLogger()),
		usesInstillCredentials: resolved,
	}
	switch x.Task {
	case TextGenerationTask:
		e.execute = e.taskTextGeneration
	case TextEmbeddingTask:
		e.execute = e.taskEmbedding
	case TextRerankTask:
		e.execute = e.taskRerank
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
