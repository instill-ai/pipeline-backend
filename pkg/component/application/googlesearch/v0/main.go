//go:generate compogen readme ./config ./README.mdx
package googlesearch

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	_ "embed"

	"google.golang.org/api/customsearch/v1"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

const (
	taskSearch = "TASK_SEARCH"
)

//go:embed config/definition.yaml
var definitionYAML []byte

//go:embed config/setup.yaml
var setupYAML []byte

//go:embed config/tasks.yaml
var tasksYAML []byte

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
		err := comp.LoadDefinition(definitionYAML, setupYAML, tasksYAML, nil, nil)
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

// NewService creates a Google custom search service
func NewService(apiKey string) (*customsearch.Service, error) {
	return customsearch.NewService(context.Background(), option.WithAPIKey(apiKey))
}

func getAPIKey(setup *structpb.Struct) string {
	return setup.GetFields()["api-key"].GetStringValue()
}

func getSearchEngineID(setup *structpb.Struct) string {
	return setup.GetFields()["cse-id"].GetStringValue()
}

func (e *execution) Execute(ctx context.Context, jobs []*base.Job) error {

	service, err := NewService(getAPIKey(e.Setup))
	if err != nil || service == nil {
		return fmt.Errorf("error creating Google custom search service: %v", err)
	}
	cseListCall := service.Cse.List().Cx(getSearchEngineID(e.Setup))

	for _, job := range jobs {
		input, err := job.Input.Read(ctx)
		if err != nil {
			job.Error.Error(ctx, err)
			continue
		}
		switch e.Task {
		case taskSearch:

			inputStruct := SearchInput{}
			err := base.ConvertFromStructpb(input, &inputStruct)
			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}

			// Make the search request
			outputStruct, err := search(cseListCall, inputStruct)

			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}

			outputJSON, err := json.Marshal(outputStruct)
			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}
			output := &structpb.Struct{}
			err = json.Unmarshal(outputJSON, output)
			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}
			err = job.Output.Write(ctx, output)
			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}

		default:
			job.Error.Error(ctx, fmt.Errorf("not supported task: %s", e.Task))
			continue
		}
	}

	return nil
}

func (c *component) Test(sysVars map[string]any, setup *structpb.Struct) error {

	service, err := NewService(getAPIKey(setup))
	if err != nil || service == nil {
		return fmt.Errorf("error creating Google custom search service: %v", err)
	}
	if service == nil {
		return fmt.Errorf("error creating Google custom search service: %v", err)
	}
	return nil
}
