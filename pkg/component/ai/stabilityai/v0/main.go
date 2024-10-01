//go:generate compogen readme ./config ./README.mdx
package stabilityai

import (
	"context"
	"fmt"
	"sync"

	_ "embed"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/x/errmsg"
)

const (
	host = "https://api.stability.ai"

	TextToImageTask  = "TASK_TEXT_TO_IMAGE"
	ImageToImageTask = "TASK_IMAGE_TO_IMAGE"

	cfgAPIKey = "api-key"
)

var (
	//go:embed config/definition.json
	definitionJSON []byte
	//go:embed config/setup.json
	setupJSON []byte
	//go:embed config/tasks.json
	tasksJSON []byte
	//go:embed config/stabilityai.json
	stabilityaiJSON []byte
	once            sync.Once
	comp            *component
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
		err := comp.LoadDefinition(definitionJSON, setupJSON, tasksJSON, map[string][]byte{"stabilityai.json": stabilityaiJSON})
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

	return &execution{
		ComponentExecution:     x,
		usesInstillCredentials: resolved,
	}, nil
}

type execution struct {
	base.ComponentExecution
	usesInstillCredentials bool
}

func (e *execution) UsesInstillCredentials() bool {
	return e.usesInstillCredentials
}

func (e *execution) Execute(ctx context.Context, jobs []*base.Job) error {

	client := newClient(e.Setup, e.GetLogger())

	for _, job := range jobs {
		input, err := job.Input.Read(ctx)
		if err != nil {
			job.Error.Error(ctx, err)
			continue
		}
		var output *structpb.Struct
		switch e.Task {
		case TextToImageTask:
			params, err := parseTextToImageReq(input)
			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}

			resp := ImageTaskRes{}
			req := client.R().SetResult(&resp).SetBody(params)

			if _, err := req.Post(params.path); err != nil {
				job.Error.Error(ctx, err)
				continue
			}

			output, err = textToImageOutput(resp)
			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}

		case ImageToImageTask:
			params, err := parseImageToImageReq(input)
			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}

			data, ct, err := params.getBytes()
			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}

			resp := ImageTaskRes{}
			req := client.R().SetBody(data).SetResult(&resp).SetHeader("Content-Type", ct)

			if _, err := req.Post(params.path); err != nil {
				job.Error.Error(ctx, err)
				continue
			}

			output, err = imageToImageOutput(resp)
			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}

		default:
			job.Error.Error(ctx, errmsg.AddMessage(
				fmt.Errorf("not supported task: %s", e.Task),
				fmt.Sprintf("%s task is not supported.", e.Task),
			))
			continue
		}
		err = job.Output.Write(ctx, output)
		if err != nil {
			job.Error.Error(ctx, err)
			continue
		}

	}
	return nil
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
