//go:generate compogen readme ./config ./README.mdx
package pinecone

import (
	"context"
	"sync"

	_ "embed"

	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util/httpclient"
)

const (
	taskQuery  = "TASK_QUERY"
	taskUpsert = "TASK_UPSERT"
	taskRerank = "TASK_RERANK"

	upsertPath = "/vectors/upsert"
	queryPath  = "/query"
	rerankPath = "/rerank"
)

//go:embed config/definition.json
var definitionJSON []byte

//go:embed config/setup.json
var setupJSON []byte

//go:embed config/tasks.json
var tasksJSON []byte

var once sync.Once
var comp *component

type component struct {
	base.Component
}

type execution struct {
	base.ComponentExecution
}

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

func (c *component) CreateExecution(x base.ComponentExecution) (base.IExecution, error) {
	return &execution{
		ComponentExecution: x,
	}, nil
}

// newIndexClient creates a new httpclient.Client with the index URL provided in setup
func newIndexClient(setup *structpb.Struct, logger *zap.Logger) *httpclient.Client {
	c := httpclient.New("Pinecone", getURL(setup),
		httpclient.WithLogger(logger),
		httpclient.WithEndUserError(new(errBody)),
	)

	c.SetHeader("Api-Key", getAPIKey(setup))
	c.SetHeader("User-Agent", "source_tag=instillai")

	return c
}

// newBaseClient creates a new httpclient.Client with the default Pinecone API URL.
func newBaseClient(setup *structpb.Struct, logger *zap.Logger) *httpclient.Client {
	c := httpclient.New("Pinecone", "https://api.pinecone.io",
		httpclient.WithLogger(logger),
		httpclient.WithEndUserError(new(errBody)),
	)

	c.SetHeader("Api-Key", getAPIKey(setup))
	c.SetHeader("User-Agent", "source_tag=instillai")
	return c
}

func getAPIKey(setup *structpb.Struct) string {
	return setup.GetFields()["api-key"].GetStringValue()
}

func getURL(setup *structpb.Struct) string {
	return setup.GetFields()["url"].GetStringValue()
}

func (e *execution) Execute(ctx context.Context, jobs []*base.Job) error {

	for _, job := range jobs {
		input, err := job.Input.Read(ctx)
		if err != nil {
			job.Error.Error(ctx, err)
			continue
		}
		var output *structpb.Struct
		switch e.Task {
		case taskQuery:
			req := newIndexClient(e.Setup, e.GetLogger()).R()

			inputStruct := queryInput{}
			err := base.ConvertFromStructpb(input, &inputStruct)
			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}

			// Each query request can contain only one of the parameters
			// vector, or id.
			// Ref: https://docs.pinecone.io/reference/query
			if inputStruct.ID != "" {
				inputStruct.Vector = nil
			}

			resp := queryResp{}
			req.SetResult(&resp).SetBody(inputStruct.asRequest())

			if _, err := req.Post(queryPath); err != nil {
				job.Error.Error(ctx, httpclient.WrapURLError(err))
				continue
			}

			resp = resp.filterOutBelowThreshold(inputStruct.MinScore)

			output, err = base.ConvertToStructpb(resp)
			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}
		case taskUpsert:
			req := newIndexClient(e.Setup, e.GetLogger()).R()

			v := upsertInput{}
			err := base.ConvertFromStructpb(input, &v)
			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}

			resp := upsertResp{}
			req.SetResult(&resp).SetBody(upsertReq{
				Vectors:   []vector{v.vector},
				Namespace: v.Namespace,
			})

			if _, err := req.Post(upsertPath); err != nil {
				job.Error.Error(ctx, httpclient.WrapURLError(err))
				continue
			}

			output, err = base.ConvertToStructpb(upsertOutput(resp))
			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}
		case taskRerank:
			// rerank task does not need index URL, so using the base client with the default pinecone API URL.
			req := newBaseClient(e.Setup, e.GetLogger()).R()

			// parse input struct
			inputStruct := rerankInput{}
			err := base.ConvertFromStructpb(input, &inputStruct)
			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}

			// make API request to rerank task
			resp := rerankResp{}
			req.SetResult(&resp).SetBody(inputStruct.asRequest())
			if _, err := req.Post(rerankPath); err != nil {
				job.Error.Error(ctx, httpclient.WrapURLError(err))
				continue
			}

			// convert response to output struct
			output, err = base.ConvertToStructpb(resp.toOutput())
			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}
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
	//TODO: change this
	return nil
}
