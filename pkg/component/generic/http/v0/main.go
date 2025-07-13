//go:generate compogen readme ./config ./README.mdx
package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"

	_ "embed"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util/httpclient"
	"github.com/instill-ai/pipeline-backend/pkg/data"

	pipelinepb "github.com/instill-ai/protogen-go/pipeline/pipeline/v1beta"
	errorsx "github.com/instill-ai/x/errors"
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
	//go:embed config/definition.yaml
	definitionYAML []byte
	//go:embed config/setup.yaml
	setupYAML []byte
	//go:embed config/tasks.yaml
	tasksYAML []byte

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
		err := comp.LoadDefinition(definitionYAML, setupYAML, tasksYAML, nil, nil)
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
		return nil, errorsx.AddMessage(
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

// validateURL checks the component's input is a valid URL. This component only
// accepts requests to *publicly available* endpoints. Any call to the internal
// network will produce an error.
func (e *execution) validateURL(endpointURL string) error {
	parsedURL, err := url.Parse(endpointURL)
	if err != nil {
		return errorsx.AddMessage(
			fmt.Errorf("parsing endpoint URL: %w", err),
			"Couldn't parse the endpoint URL as a valid URI reference",
		)
	}

	host := parsedURL.Hostname()
	if host == "" {
		err := fmt.Errorf("missing hostname")
		return errorsx.AddMessage(err, "Endpoint URL must have a hostname")
	}

	var whitelistedHosts = []string{
		// Pipeline's public port is exposed to call pipelines from pipelines.
		// When a `pipeline` component is implemented, this won't be necessary.
		fmt.Sprintf("%s:%d", config.Config.Server.InstanceID, config.Config.Server.PublicPort),
		// Model's public port is exposed until the model component allows
		// triggering models in the custom mode.
		fmt.Sprintf("%s:%d", config.Config.ModelBackend.Host, config.Config.ModelBackend.PublicPort),
	}
	// Certain pipelines used by artifact-backend need to trigger pipelines and
	// models via this component.
	// TODO jvallesm: Remove this after INS-8119 is completed.
	if slices.Contains(whitelistedHosts, parsedURL.Host) {
		return nil
	}

	// Get IP addresses for the host
	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("looking up IP: %w", err)
	}

	// Check if any resolved IP is in private ranges
	for _, ip := range ips {
		if ip.IsPrivate() || ip.IsLoopback() {
			return errorsx.AddMessage(
				fmt.Errorf("endpoint URL resolves to private/internal IP address"),
				"URL must point to a publicly available endpoint (no private/internal addresses)",
			)
		}
	}

	return nil
}

func (e *execution) executeHTTP(ctx context.Context, job *base.Job) error {
	in := httpInput{}
	if err := job.Input.ReadData(ctx, &in); err != nil {
		return fmt.Errorf("reading input data: %w", err)
	}

	if err := e.validateURL(in.EndpointURL); err != nil {
		return fmt.Errorf("validating URL: %w", err)
	}

	// An API error is a valid output in this component.
	req := e.client.R()
	if in.Body != nil {
		jsonValue, err := in.Body.ToJSONValue()
		if err != nil {
			return fmt.Errorf("failed to convert body to JSON value: %w", err)
		}
		req.SetBody(jsonValue)
	}

	for k, h := range in.Header {
		req.SetHeader(k, strings.Join(h, ","))
	}

	resp, err := req.Execute(taskMethod[e.Task], in.EndpointURL)
	if err != nil {
		return fmt.Errorf("executing HTTP request: %w", err)
	}

	// Try to parse response as JSON first
	contentType := resp.Header().Get("Content-Type")

	// Try to parse response based on content type
	switch {
	case strings.Contains(contentType, "application/json"):
		var jsonBody any
		if err := json.Unmarshal(resp.Body(), &jsonBody); err != nil {
			return fmt.Errorf("parsing JSON response: %w", err)
		}
		value, err := data.NewValue(jsonBody)
		if err != nil {
			return fmt.Errorf("converting JSON response to format.Value: %w", err)
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
		value, err := data.NewBinaryFromBytes(resp.Body(), contentType, filename)
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
func (c *component) GetDefinition(sysVars map[string]any, compConfig *base.ComponentConfig) (*pipelinepb.ComponentDefinition, error) {
	oriDef, err := c.Component.GetDefinition(nil, nil)
	if err != nil {
		return nil, err
	}
	if sysVars == nil && compConfig == nil {
		return oriDef, nil
	}

	def := proto.Clone(oriDef).(*pipelinepb.ComponentDefinition)
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
