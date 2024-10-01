//go:generate compogen readme ./config ./README.mdx
package restapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"

	_ "embed"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/x/errmsg"

	pb "github.com/instill-ai/protogen-go/vdp/pipeline/v1beta"
)

const (
	taskGet     = "TASK_GET"
	taskPost    = "TASK_POST"
	taskPatch   = "TASK_PATCH"
	taskPut     = "TASK_PUT"
	taskDelete  = "TASK_DELETE"
	taskHead    = "TASK_HEAD"
	taskOptions = "TASK_OPTIONS"
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

	taskMethod = map[string]string{
		taskGet:     http.MethodGet,
		taskPost:    http.MethodPost,
		taskPatch:   http.MethodPatch,
		taskPut:     http.MethodPut,
		taskDelete:  http.MethodDelete,
		taskHead:    http.MethodHead,
		taskOptions: http.MethodOptions,
	}
)

type component struct {
	base.Component
}

type execution struct {
	base.ComponentExecution
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

func (c *component) CreateExecution(x base.ComponentExecution) (base.IExecution, error) {
	return &execution{
		ComponentExecution: x,
	}, nil
}

func getAuthentication(setup *structpb.Struct) (authentication, error) {
	auth := setup.GetFields()["authentication"].GetStructValue()
	authType := auth.GetFields()["auth-type"].GetStringValue()

	switch authType {
	case string(noAuthType):
		authStruct := noAuth{}
		err := base.ConvertFromStructpb(auth, &authStruct)
		if err != nil {
			return nil, err
		}
		return authStruct, nil
	case string(basicAuthType):
		authStruct := basicAuth{}
		err := base.ConvertFromStructpb(auth, &authStruct)
		if err != nil {
			return nil, err
		}
		return authStruct, nil
	case string(apiKeyType):
		authStruct := apiKeyAuth{}
		err := base.ConvertFromStructpb(auth, &authStruct)
		if err != nil {
			return nil, err
		}
		return authStruct, nil
	case string(bearerTokenType):
		authStruct := bearerTokenAuth{}
		err := base.ConvertFromStructpb(auth, &authStruct)
		if err != nil {
			return nil, err
		}
		return authStruct, nil
	default:
		return nil, errors.New("invalid authentication type")
	}
}

func (e *execution) Execute(ctx context.Context, jobs []*base.Job) error {

	method, ok := taskMethod[e.Task]
	if !ok {
		return errmsg.AddMessage(
			fmt.Errorf("not supported task: %s", e.Task),
			fmt.Sprintf("%s task is not supported.", e.Task),
		)
	}

	for _, job := range jobs {
		input, err := job.Input.Read(ctx)
		if err != nil {
			job.Error.Error(ctx, err)
			continue
		}
		taskIn := TaskInput{}
		taskOut := TaskOutput{}

		if err := base.ConvertFromStructpb(input, &taskIn); err != nil {
			job.Error.Error(ctx, err)
			continue
		}

		// We may have different url in batch.
		client, err := newClient(e.Setup, e.GetLogger())
		if err != nil {
			job.Error.Error(ctx, err)
			continue
		}

		// An API error is a valid output in this component.
		req := client.R().SetResult(&taskOut.Body).SetError(&taskOut.Body)
		if taskIn.Body != nil {
			req.SetBody(taskIn.Body)
		}

		resp, err := req.Execute(method, taskIn.EndpointURL)
		if err != nil {
			job.Error.Error(ctx, err)
			continue
		}

		taskOut.StatusCode = resp.StatusCode()
		taskOut.Header = resp.Header()

		if taskOut.Body == nil {
			// Maintain a JSON structure for the output to avoid frontend render overhead.
			taskOut.Body = map[string]interface{}{
				"response": resp.String(),
			}
		}

		output, err := base.ConvertToStructpb(taskOut)
		if err != nil {
			job.Error.Error(ctx, err)
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

func (c *component) Test(sysVars map[string]any, setup *structpb.Struct) error {
	// we don't need to validate the setup since no url setting here
	return nil
}

// Generate the model_name enum based on the task
func (c *component) GetDefinition(sysVars map[string]any, compConfig *base.ComponentConfig) (*pb.ComponentDefinition, error) {
	oriDef, err := c.Component.GetDefinition(nil, nil)
	if err != nil {
		return nil, err
	}
	if sysVars == nil && compConfig == nil {
		return oriDef, nil
	}

	def := proto.Clone(oriDef).(*pb.ComponentDefinition)
	if compConfig == nil {
		return def, nil
	}
	if compConfig.Task == "" {
		return def, nil
	}
	if _, ok := compConfig.Input["output-body-schema"]; !ok {
		return def, nil
	}

	if s, ok := compConfig.Input["output-body-schema"].(string); ok {
		sch := &structpb.Struct{}
		_ = json.Unmarshal([]byte(s), sch)
		spec := def.Spec.DataSpecifications[compConfig.Task]
		spec.Output = sch
	}

	return def, nil
}
