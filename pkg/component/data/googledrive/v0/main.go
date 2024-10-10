//go:generate compogen readme ./config ./README.mdx
package googledrive

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sync"

	_ "embed"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/x/errmsg"
)

const (
	taskReadFile       = "TASK_READ_FILE"
	taskReadFiles      = "TASK_READ_FILES"
	taskReadDrive      = "TASK_READ_DRIVE"
	cfgOAuthCredential = "oauth-credentials"
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
	// The JSON string of OAuth credentials encoded by base64.
	instillAICredentials string
}

type execution struct {
	base.ComponentExecution
	execute func(context.Context, *structpb.Struct) (*structpb.Struct, error)

	service IDriveService
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
		service:            &driveService{service: drive},
	}

	switch x.Task {
	case taskReadFile:
		e.execute = e.readFile
	case taskReadFiles:
		e.execute = e.readFiles
	case taskReadDrive:
		e.execute = e.readDrive
	default:
		return nil, errmsg.AddMessage(
			fmt.Errorf("not supported task: %s", x.Task),
			fmt.Sprintf("%s task is not supported.", x.Task),
		)
	}
	return e, nil
}

func getDriveService(ctx context.Context, setup *structpb.Struct, c *component) (*drive.Service, error) {

	decodedBytes, err := base64.StdEncoding.DecodeString(c.instillAICredentials)

	if err != nil {
		return nil, fmt.Errorf("failed to decode Instill AI credentials: %w", err)
	}

	config, err := google.ConfigFromJSON(decodedBytes, getConfigScopes()...)

	if err != nil {
		return nil, fmt.Errorf("failed to get Google config from JSON: %w", err)
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

func getConfigScopes() []string {
	type setupConfig struct {
		InstillOAuthConfig struct {
			Scopes []string `json:"scopes"`
		} `json:"instillOAuthConfig"`
	}
	var setup setupConfig
	json.Unmarshal(setupJSON, &setup)
	return setup.InstillOAuthConfig.Scopes
}

// Need to get the scopes from the token.json that received from the OAuth2
// Temporarily, it will be same as the scopes in setup.json.
// So, we get it from setup.json first. Later, we will get it from token.json
// after we confirm how we retrieve the scopes from token.json.
func getScopes(setup *structpb.Struct) []string {
	return getConfigScopes()
}

// Execute reads the input from the job, executes the task, and writes the output
// to the job.
func (e *execution) Execute(ctx context.Context, jobs []*base.Job) error {
	for _, job := range jobs {
		input, err := job.Input.Read(ctx)
		if err != nil {
			job.Error.Error(ctx, err)
			continue
		}

		output, err := e.execute(ctx, input)
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

func (c *component) WithOAuthCredentials(s map[string]any) *component {
	c.instillAICredentials = base.ReadFromGlobalConfig(cfgOAuthCredential, s)
	return c
}
