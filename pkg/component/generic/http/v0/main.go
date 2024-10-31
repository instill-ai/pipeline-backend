//go:generate compogen readme ./config ./README.mdx
package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"strings"
	"sync"

	_ "embed"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util/httpclient"
	"github.com/instill-ai/pipeline-backend/pkg/data"
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

	client  *httpclient.Client
	execute func(context.Context, *base.Job) error
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

	// We may have different url in batch.
	client, err := newClient(x.Setup, x.GetLogger())
	if err != nil {
		return nil, err
	}
	e := &execution{ComponentExecution: x, client: client}

	switch x.Task {
	case taskGet, taskPost, taskPatch, taskPut, taskDelete, taskHead, taskOptions:
		e.execute = e.executeHTTP
	default:
		return nil, errmsg.AddMessage(
			fmt.Errorf("not supported task: %s", x.Task),
			fmt.Sprintf("%s task is not supported.", x.Task),
		)
	}
	return e, nil

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

	return base.ConcurrentExecutor(ctx, jobs, e.execute)
}

func (e *execution) executeHTTP(ctx context.Context, job *base.Job) error {

	in := httpInput{}
	if err := job.Input.ReadData(ctx, &in); err != nil {
		return err
	}

	// An API error is a valid output in this component.
	req := e.client.R()
	if in.Body != nil {
		req.SetBody(in.Body)
	}

	resp, err := req.Execute(taskMethod[e.Task], in.EndpointURL)
	if err != nil {
		return err
	}

	// Try to parse response as JSON first
	contentType := resp.Header().Get("Content-Type")

	// Try to parse response based on content type
	switch {
	case strings.Contains(contentType, "application/json"):
		var jsonBody any
		if err := json.Unmarshal(resp.Body(), &jsonBody); err != nil {
			return fmt.Errorf("failed to parse JSON response: %w", err)
		}
		value, err := data.NewValue(jsonBody)
		if err != nil {
			return fmt.Errorf("failed to convert JSON response to format.Value: %w", err)
		}
		out := httpOutput{
			StatusCode: resp.StatusCode(),
			Header:     resp.Header(),
			Body:       value,
		}
		return job.Output.WriteData(ctx, out)

	case strings.HasPrefix(contentType, "text/"):
		textBody := string(resp.Body())
		value := data.NewString(textBody)
		out := httpOutput{
			StatusCode: resp.StatusCode(),
			Header:     resp.Header(),
			Body:       value,
		}
		return job.Output.WriteData(ctx, out)

	default:
		// Get filename from Content-Disposition header if present
		filename := ""
		if cd := resp.Header().Get("Content-Disposition"); cd != "" {
			if _, params, err := mime.ParseMediaType(cd); err == nil {
				if fn, ok := params["filename"]; ok {
					filename = fn
				}
			}
		}
		// For other content types, return raw bytes wrapped in format.Value
		value, err := data.NewFileFromBytes(resp.Body(), contentType, filename)
		if err != nil {
			return fmt.Errorf("failed to convert response to format.Value: %w", err)
		}
		out := httpOutput{
			StatusCode: resp.StatusCode(),
			Header:     resp.Header(),
			Body:       value,
		}

		return job.Output.WriteData(ctx, out)
	}
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
