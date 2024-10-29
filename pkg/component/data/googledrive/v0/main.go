//go:generate compogen readme ./config ./README.mdx
package googledrive

import (
	"context"
	"fmt"
	"sync"

	_ "embed"

	"golang.org/x/oauth2"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/data/googledrive/v0/client"
	"github.com/instill-ai/x/errmsg"
)

const (
	taskReadFile         = "TASK_READ_FILE"
	taskReadFolder       = "TASK_READ_FOLDER"
	cfgOAuthClientID     = "client-id"
	cfgOAuthClientSecret = "client-secret"

	authURL  = "https://accounts.google.com/o/oauth2/auth"
	tokenURL = "https://oauth2.googleapis.com/token"
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

	instillAIClientID     string
	instillAIClientSecret string
}

type execution struct {
	base.ComponentExecution
	execute func(*structpb.Struct, *base.Job, context.Context) (*structpb.Struct, error)

	service client.IDriveService
}

// Init returns an implementation of IComponent that interacts with Google Drive.
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

// CreateExecution initializes a component executor that can be used in a
// pipeline trigger.
func (c *component) CreateExecution(x base.ComponentExecution) (base.IExecution, error) {

	ctx := context.Background()

	drive, err := getDriveService(ctx, x.Setup, c)

	if err != nil {
		return nil, fmt.Errorf("failed to get drive service: %w", err)
	}

	e := &execution{
		ComponentExecution: x,
		service:            &client.DriveService{Service: drive},
	}

	switch x.Task {
	case taskReadFile:
		e.execute = e.readFile
	case taskReadFolder:
		e.execute = e.readFolder
	default:
		return nil, errmsg.AddMessage(
			fmt.Errorf("not supported task: %s", x.Task),
			fmt.Sprintf("%s task is not supported.", x.Task),
		)
	}
	return e, nil
}

func getDriveService(ctx context.Context, setup *structpb.Struct, c *component) (*drive.Service, error) {
	config := &oauth2.Config{
		ClientID:     c.instillAIClientID,
		ClientSecret: c.instillAIClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  authURL,
			TokenURL: tokenURL,
		},
	}

	refreshToken := setup.GetFields()["refresh-token"].GetStringValue()

	tok := &oauth2.Token{
		RefreshToken: refreshToken,
	}

	client := config.Client(ctx, tok)

	srv, err := drive.NewService(ctx, option.WithHTTPClient(client))

	if err != nil {
		return nil, err
	}

	return srv, nil
}

// Execute reads the input from the job, executes the task, and writes the output
// to the job.
func (e *execution) Execute(ctx context.Context, jobs []*base.Job) error {
	return base.ConcurrentExecutor(ctx, jobs, e.execute)
}

// WithOAuthCredentials sets the OAuth credentials for the component.
func (c *component) WithOAuthCredentials(s map[string]any) *component {
	c.instillAIClientID = base.ReadFromGlobalConfig(cfgOAuthClientID, s)
	c.instillAIClientSecret = base.ReadFromGlobalConfig(cfgOAuthClientSecret, s)
	return c
}
