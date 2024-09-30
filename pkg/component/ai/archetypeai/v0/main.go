//go:generate compogen readme ./config ./README.mdx
package archetypeai

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync"

	_ "embed"

	"github.com/gofrs/uuid"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util/httpclient"
	"github.com/instill-ai/x/errmsg"
)

const (
	taskDescribe   = "TASK_DESCRIBE"
	taskSummarize  = "TASK_SUMMARIZE"
	taskUploadFile = "TASK_UPLOAD_FILE"
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
}

type execution struct {
	base.ComponentExecution

	execute func(*structpb.Struct) (*structpb.Struct, error)
	client  *httpclient.Client
}

// Init returns an implementation of IConnector that interacts with Archetype
// AI.
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
	e := &execution{
		ComponentExecution: x,
		client:             newClient(x.Setup, c.GetLogger()),
	}

	switch x.Task {
	case taskDescribe:
		e.execute = e.describe
	case taskSummarize:
		e.execute = e.summarize
	case taskUploadFile:
		e.execute = e.uploadFile
	default:
		return nil, errmsg.AddMessage(
			fmt.Errorf("not supported task: %s", x.Task),
			fmt.Sprintf("%s task is not supported.", x.Task),
		)
	}

	return e, nil
}

// Execute performs calls the Archetype AI API to execute a task.
func (e *execution) Execute(ctx context.Context, jobs []*base.Job) error {
	return base.SequentialExecutor(ctx, jobs, e.execute)
}

func (e *execution) describe(in *structpb.Struct) (*structpb.Struct, error) {
	params := fileQueryParams{}
	if err := base.ConvertFromStructpb(in, &params); err != nil {
		return nil, err
	}

	// We have a 1-1 mapping between the VDP user input and the Archetype AI
	// request. If this stops being the case in the future, we'll need a
	// describeReq structure.
	resp := describeResp{}
	req := e.client.R().SetBody(fileQueryReq(params)).SetResult(&resp)

	if _, err := req.Post(describePath); err != nil {
		return nil, err
	}

	// Archetype AI might return a 200 status even if the operation failed
	// (e.g. if the file doesn't exist).
	if resp.Status != statusCompleted {
		return nil, errmsg.AddMessage(
			fmt.Errorf("response with non-completed status"),
			fmt.Sprintf(`Archetype AI didn't complete query %s: status is "%s".`, resp.QueryID, resp.Status),
		)
	}

	frameDescriptionOutputs := make([]frameDescriptionOutput, len(resp.Response))
	for i := range resp.Response {
		frameDescriptionOutputs[i] = frameDescriptionOutput(resp.Response[i])
	}
	out, err := base.ConvertToStructpb(describeOutput{
		Descriptions: frameDescriptionOutputs,
	})
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (e *execution) summarize(in *structpb.Struct) (*structpb.Struct, error) {
	params := fileQueryParams{}
	if err := base.ConvertFromStructpb(in, &params); err != nil {
		return nil, err
	}

	// We have a 1-1 mapping between the VDP user input and the Archetype AI
	// request. If this stops being the case in the future, we'll need a
	// summarizeReq structure.
	resp := summarizeResp{}
	req := e.client.R().SetBody(fileQueryReq(params)).SetResult(&resp)

	if _, err := req.Post(summarizePath); err != nil {
		return nil, err
	}

	// Archetype AI might return a 200 status even if the operation failed
	// (e.g. if the file doesn't exist).
	if resp.Status != statusCompleted {
		return nil, errmsg.AddMessage(
			fmt.Errorf("response with non-completed status"),
			fmt.Sprintf(`Archetype AI didn't complete query %s: status is "%s".`, resp.QueryID, resp.Status),
		)
	}

	out, err := base.ConvertToStructpb(summarizeOutput{
		Response: resp.Response.ProcessedText,
	})
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (e *execution) uploadFile(in *structpb.Struct) (*structpb.Struct, error) {
	params := uploadFileParams{}
	if err := base.ConvertFromStructpb(in, &params); err != nil {
		return nil, err
	}

	resp := uploadFileResp{}
	req := e.client.R().SetResult(&resp)

	b, err := util.DecodeBase64(params.File)
	if err != nil {
		return nil, err
	}

	id, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}

	req.SetFileReader("file", id.String(), bytes.NewReader(b))
	if _, err := req.Post(uploadFilePath); err != nil {
		return nil, err
	}

	if !resp.IsValid {
		errMsg := "invalid file."
		if len(resp.Errors) > 0 {
			errMsg = strings.Join(resp.Errors, " ")
		}

		return nil, errmsg.AddMessage(
			fmt.Errorf("file upload failed"),
			fmt.Sprintf(`Couldn't complete upload: %s`, errMsg),
		)
	}

	out, err := base.ConvertToStructpb(uploadFileOutput{FileID: resp.FileID})
	if err != nil {
		return nil, err
	}

	return out, nil
}

// Test checks the connectivity of the connector.

func (c *component) Test(sysVars map[string]any, setup *structpb.Struct) error {
	// TODO Archetype AI API is not public yet. We could test the setup
	// by calling one of the endpoints used in the available tasks. However,
	// these are not designed for specifically for this purpose. When we know
	// of an endpoint that's more suited for this, it should be used instead.
	return nil
}

func getAPIKey(setup *structpb.Struct) string {
	return setup.GetFields()["api-key"].GetStringValue()
}

// getBasePath returns Archetype AI's API URL. This configuration param allows
// us to override the API the connector will point to. It isn't meant to be
// exposed to users. Rather, it can serve to test the logic against a fake
// server.
// TODO instead of having the API value hardcoded in the codebase, it should
// be read from a setup file or environment variable.
func getBasePath(setup *structpb.Struct) string {
	v, ok := setup.GetFields()["base-path"]
	if !ok {
		return host
	}
	return v.GetStringValue()
}
