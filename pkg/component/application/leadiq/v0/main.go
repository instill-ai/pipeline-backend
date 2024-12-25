//go:generate compogen readme ./config ./README.mdx --extraContents TASK_FIND_PROSPECTS=.compogen/find_prospects.mdx --extraContents bottom=.compogen/bottom.mdx
package leadiq

import (
	"context"
	"fmt"
	"regexp"
	"sync"

	_ "embed"

	"github.com/machinebox/graphql"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util/httpclient"
)

const (
	apiBaseURL        = "https://api.leadiq.com/graphql"
	TaskFindProspects = "TASK_FIND_PROSPECTS"

	cfgConnection = "api-key"
)

var (
	//go:embed config/definition.yaml
	definitionYAML []byte
	//go:embed config/setup.yaml
	setupYAML []byte
	//go:embed config/tasks.yaml
	tasksYAML []byte

	//go:embed queries/flat_advanced_search.txt
	flatAdvancedSearchQuery string
	//go:embed queries/search_people.txt
	searchPeopleQuery string

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
	httpStatusCodeCompiler *regexp.Regexp
}

// Init initializes the component with the provided base component.
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
	c.instillAPIKey = base.ReadFromGlobalConfig(cfgConnection, s)
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

	httpStatusCodeCompiler, err := regexp.Compile(`\b(\d{3})\b`)

	if err != nil {
		err = fmt.Errorf("compiling http status code regex: %w", err)
		return nil, err
	}

	e := &execution{
		ComponentExecution:     x,
		usesInstillCredentials: resolved,
		httpStatusCodeCompiler: httpStatusCodeCompiler,
	}
	switch x.Task {
	case TaskFindProspects:
		e.execute = e.executeFindProspects
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
	if v, ok := setup.GetFields()[cfgConnection]; ok {
		apiKey := v.GetStringValue()
		if apiKey != "" && apiKey != base.SecretKeyword {
			return setup, false, nil
		}
	}

	if c.instillAPIKey == "" {
		return nil, false, base.NewUnresolvedCredential(cfgConnection)
	}

	setup.GetFields()[cfgConnection] = structpb.NewStringValue(c.instillAPIKey)
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

func newClient(setup *structpb.Struct, logger *zap.Logger) *graphql.Client {
	c := httpclient.New("LeadIQ", apiBaseURL,
		httpclient.WithLogger(logger),
		httpclient.WithEndUserError(new(errBody)),
	)
	// Set the API key as a basic auth header.
	// Ideally, graphql.NewRequest should support setting headers.
	// But, it will not be used in graphql.NewRequest because it does not support setting headers.
	// So, we have to reset it again before sending the request.
	c.SetHeader("Authorization", "Basic "+getAPIKey(setup))
	return graphql.NewClient(apiBaseURL, graphql.WithHTTPClient(c.GetClient()))
}

func getAPIKey(setup *structpb.Struct) string {
	return setup.GetFields()["api-key"].GetStringValue()
}
