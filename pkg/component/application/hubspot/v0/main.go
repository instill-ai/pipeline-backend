//go:generate compogen readme ./config ./README.mdx

package hubspot

import (
	"context"
	"fmt"
	"sync"

	_ "embed"

	"google.golang.org/protobuf/types/known/structpb"

	hubspot "github.com/belong-inc/go-hubspot"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

const (
	taskGetContact          = "TASK_GET_CONTACT"
	taskCreateContact       = "TASK_CREATE_CONTACT"
	taskGetDeal             = "TASK_GET_DEAL"
	taskCreateDeal          = "TASK_CREATE_DEAL"
	taskUpdateDeal          = "TASK_UPDATE_DEAL"
	taskGetCompany          = "TASK_GET_COMPANY"
	taskCreateCompany       = "TASK_CREATE_COMPANY"
	taskGetTicket           = "TASK_GET_TICKET"
	taskCreateTicket        = "TASK_CREATE_TICKET"
	taskUpdateTicket        = "TASK_UPDATE_TICKET"
	taskGetThread           = "TASK_GET_THREAD"
	taskInsertMessage       = "TASK_INSERT_MESSAGE"
	taskRetrieveAssociation = "TASK_RETRIEVE_ASSOCIATION"
	taskGetOwner            = "TASK_GET_OWNER"
	taskGetAll              = "TASK_GET_ALL"
)

var (
	//go:embed config/definition.json
	definitionJSON []byte
	//go:embed config/tasks.json
	tasksJSON []byte
	//go:embed config/setup.json
	setupJSON []byte

	once sync.Once
	comp *component
)

type component struct {
	base.Component
}

type execution struct {
	base.ComponentExecution
	client  *CustomClient
	execute func(*structpb.Struct) (*structpb.Struct, error)
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

func getToken(setup *structpb.Struct) string {
	return setup.GetFields()["token"].GetStringValue()
}

// custom client to support thread task
func hubspotNewCustomClient(setup *structpb.Struct) *CustomClient {
	client, err := NewCustomClient(hubspot.SetPrivateAppToken(getToken(setup)))

	if err != nil {
		panic(err)
	}

	return client
}

func (c *component) CreateExecution(x base.ComponentExecution) (base.IExecution, error) {

	e := &execution{
		ComponentExecution: x,
		client:             hubspotNewCustomClient(x.Setup),
	}

	switch x.Task {
	case taskGetContact:
		e.execute = e.GetContact
	case taskCreateContact:
		e.execute = e.CreateContact
	case taskGetDeal:
		e.execute = e.GetDeal
	case taskCreateDeal:
		e.execute = e.CreateDeal
	case taskUpdateDeal:
		e.execute = e.UpdateDeal
	case taskGetCompany:
		e.execute = e.GetCompany
	case taskCreateCompany:
		e.execute = e.CreateCompany
	case taskGetTicket:
		e.execute = e.GetTicket
	case taskCreateTicket:
		e.execute = e.CreateTicket
	case taskUpdateTicket:
		e.execute = e.UpdateTicket
	case taskGetThread:
		e.execute = e.GetThread
	case taskInsertMessage:
		e.execute = e.InsertMessage
	case taskRetrieveAssociation:
		e.execute = e.RetrieveAssociation
	case taskGetOwner:
		e.execute = e.GetOwner
	case taskGetAll:
		e.execute = e.GetAll
	default:
		return nil, fmt.Errorf("unsupported task")
	}

	return e, nil
}

func (e *execution) Execute(ctx context.Context, jobs []*base.Job) error {
	return base.SequentialExecutor(ctx, jobs, e.execute)
}
