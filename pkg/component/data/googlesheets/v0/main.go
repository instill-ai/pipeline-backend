//go:generate compogen readme ./config ./README.mdx
package googlesheets

import (
	"context"
	"fmt"
	"sync"

	_ "embed"

	"golang.org/x/oauth2"
	"google.golang.org/api/drive/v2"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

const (
	authURL  = "https://accounts.google.com/o/oauth2/auth"
	tokenURL = "https://oauth2.googleapis.com/token"
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

const (
	taskCreateSpreadsheet       = "TASK_CREATE_SPREADSHEET"
	taskDeleteSpreadsheet       = "TASK_DELETE_SPREADSHEET"
	taskAddSheet                = "TASK_ADD_SHEET"
	taskDeleteSheet             = "TASK_DELETE_SHEET"
	taskCreateSpreadsheetColumn = "TASK_CREATE_SPREADSHEET_COLUMN"
	taskDeleteSpreadsheetColumn = "TASK_DELETE_SPREADSHEET_COLUMN"
	taskListRows                = "TASK_LIST_ROWS"
	taskLookupRows              = "TASK_LOOKUP_ROWS"
	taskGetRow                  = "TASK_GET_ROW"
	taskGetMultipleRows         = "TASK_GET_MULTIPLE_ROWS"
	taskInsertRow               = "TASK_INSERT_ROW"
	taskInsertMultipleRows      = "TASK_INSERT_MULTIPLE_ROWS"
	taskUpdateRow               = "TASK_UPDATE_ROW"
	taskUpdateMultipleRows      = "TASK_UPDATE_MULTIPLE_ROWS"
	taskDeleteRow               = "TASK_DELETE_ROW"
	taskDeleteMultipleRows      = "TASK_DELETE_MULTIPLE_ROWS"
)

type component struct {
	base.Component
	base.OAuthConnector
}

type execution struct {
	base.ComponentExecution
	execute      func(context.Context, *base.Job) error
	sheetService *sheets.Service
	driveService *drive.Service
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

func getsheetService(ctx context.Context, setup *structpb.Struct, c *component) (*sheets.Service, error) {
	config := &oauth2.Config{
		ClientID:     c.GetOAuthClientID(),
		ClientSecret: c.GetOAuthClientSecret(),
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

	srv, err := sheets.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	return srv, nil
}

func getDriveService(ctx context.Context, setup *structpb.Struct, c *component) (*drive.Service, error) {
	config := &oauth2.Config{
		ClientID:     c.GetOAuthClientID(),
		ClientSecret: c.GetOAuthClientSecret(),
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

// CreateExecution initializes a component executor that can be used in a
// pipeline trigger.
func (c *component) CreateExecution(x base.ComponentExecution) (base.IExecution, error) {
	ctx := context.Background()

	sheetSrv, err := getsheetService(ctx, x.Setup, c)
	if err != nil {
		return nil, fmt.Errorf("failed to get sheets service: %w", err)
	}
	driveSrv, err := getDriveService(ctx, x.Setup, c)
	if err != nil {
		return nil, fmt.Errorf("failed to get drive service: %w", err)
	}

	e := &execution{
		ComponentExecution: x,
		sheetService:       sheetSrv,
		driveService:       driveSrv,
	}

	switch x.Task {
	case taskCreateSpreadsheet:
		e.execute = e.createSpreadsheet
	case taskDeleteSpreadsheet:
		e.execute = e.deleteSpreadsheet
	case taskAddSheet:
		e.execute = e.addSheet
	case taskDeleteSheet:
		e.execute = e.deleteSheet
	case taskCreateSpreadsheetColumn:
		e.execute = e.createSpreadsheetColumn
	case taskDeleteSpreadsheetColumn:
		e.execute = e.deleteSpreadsheetColumn
	case taskListRows:
		e.execute = e.listRows
	case taskLookupRows:
		e.execute = e.lookupRows
	case taskGetRow:
		e.execute = e.getRow
	case taskGetMultipleRows:
		e.execute = e.getMultipleRows
	case taskInsertRow:
		e.execute = e.insertRow
	case taskInsertMultipleRows:
		e.execute = e.insertMultipleRows
	case taskUpdateRow:
		e.execute = e.updateRow
	case taskUpdateMultipleRows:
		e.execute = e.updateMultipleRows
	case taskDeleteRow:
		e.execute = e.deleteRow
	case taskDeleteMultipleRows:
		e.execute = e.deleteMultipleRows
	default:
		return nil, fmt.Errorf("not supported task: %s", x.Task)
	}

	return e, nil
}

// Execute executes the derived execution
func (e *execution) Execute(ctx context.Context, jobs []*base.Job) error {
	return base.ConcurrentExecutor(ctx, jobs, e.execute)
}

// SupportsOAuth checks whether the component is configured to support OAuth.
func (c *component) SupportsOAuth() bool {
	return c.OAuthConnector.SupportsOAuth()
}
