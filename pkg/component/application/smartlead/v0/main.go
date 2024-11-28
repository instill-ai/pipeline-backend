//go:generate compogen readme ./config ./README.mdx --extraContents intro=.compogen/intro.mdx --extraContents bottom=.compogen/bottom.mdx
package smartlead

import (
	"context"
	"fmt"
	"sync"

	_ "embed"

	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util/httpclient"
	"github.com/instill-ai/x/errmsg"
)

const (
	taskCreateCampaign       = "TASK_CREATE_CAMPAIGN"
	taskSetupCampaign        = "TASK_SETUP_CAMPAIGN"
	taskSaveSequences        = "TASK_SAVE_SEQUENCES"
	taskGetSequences         = "TASK_GET_SEQUENCES"
	taskAddLeads             = "TASK_ADD_LEADS"
	taskAddSenderEmail       = "TASK_ADD_SENDER_EMAIL"
	taskUpdateCampaignStatus = "TASK_UPDATE_CAMPAIGN_STATUS"
	taskGetCampaignMetric    = "TASK_GET_CAMPAIGN_METRIC"
	taskListLeadsStatus      = "TASK_LIST_LEADS_STATUS"

	baseURL = "https://server.smartlead.ai/api/v1/"
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
	execute func(context.Context, *base.Job) error
}

// Init initializes a Component that interacts with GitHub.
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

// CreateExecution initializes a component executor that can be used in a
// pipeline run.
func (c *component) CreateExecution(x base.ComponentExecution) (base.IExecution, error) {

	e := &execution{
		ComponentExecution: x,
	}

	switch x.Task {
	case taskCreateCampaign:
		e.execute = e.createCampaign
	case taskSetupCampaign:
		e.execute = e.setupCampaign
	case taskSaveSequences:
		e.execute = e.saveSequences
	case taskGetSequences:
		e.execute = e.getSequences
	case taskAddLeads:
		e.execute = e.addLeads
	case taskAddSenderEmail:
		e.execute = e.addSenderEmail
	case taskUpdateCampaignStatus:
		e.execute = e.updateCampaignStatus
	case taskGetCampaignMetric:
		e.execute = e.getCampaignMetric
	case taskListLeadsStatus:
		e.execute = e.listLeadsStatus
	default:
		return nil, errmsg.AddMessage(
			fmt.Errorf("not supported task: %s", x.Task),
			fmt.Sprintf("%s task is not supported.", x.Task),
		)
	}
	return e, nil

}

// Execute runs the component with the given jobs.
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

func newClient(setup *structpb.Struct, logger *zap.Logger, pathParams map[string]string) *httpclient.Client {
	c := httpclient.New("Smartlead", baseURL,
		httpclient.WithLogger(logger),
		httpclient.WithEndUserError(new(errBody)),
	)

	c.SetPathParam("apiKey", getAPIKey(setup))
	if pathParams != nil {
		c.SetPathParams(pathParams)
	}
	return c
}

func getAPIKey(setup *structpb.Struct) string {
	return setup.GetFields()["api-key"].GetStringValue()
}
