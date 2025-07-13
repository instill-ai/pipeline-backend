//go:generate compogen readme ./config ./README.mdx --extraContents intro=.compogen/intro.mdx
package googledrive

import (
	"context"
	"fmt"
	"strings"
	"sync"

	_ "embed"

	"golang.org/x/oauth2"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/data/googledrive/v0/client"

	errorsx "github.com/instill-ai/x/errors"
)

const (
	taskReadFile   = "TASK_READ_FILE"
	taskReadFolder = "TASK_READ_FOLDER"

	authURL  = "https://accounts.google.com/o/oauth2/auth"
	tokenURL = "https://oauth2.googleapis.com/token"
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
)

type component struct {
	base.Component
	base.OAuthConnector
}

type execution struct {
	base.ComponentExecution
	execute func(context.Context, *base.Job) error

	service client.IDriveService
}

// Init returns an implementation of IComponent that interacts with Google Drive.
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
		return nil, errorsx.AddMessage(
			fmt.Errorf("not supported task: %s", x.Task),
			fmt.Sprintf("%s task is not supported.", x.Task),
		)
	}
	return e, nil
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

// Execute reads the input from the job, executes the task, and writes the output
// to the job.
func (e *execution) Execute(ctx context.Context, jobs []*base.Job) error {
	return base.ConcurrentExecutor(ctx, jobs, e.execute)
}

// SupportsOAuth checks whether the component is configured to support OAuth.
func (c *component) SupportsOAuth() bool {
	return c.OAuthConnector.SupportsOAuth()
}

// Now, we support the following types of Google Drive links:
// 1. Folder: https://drive.google
// 2. File: https://drive.google.com/file/d/
// 3. Spreadsheet: https://docs.google.com/spreadsheets/d/
// 4. Document: https://docs.google.com/document/d/
// 5. Presentation: https://docs.google.com/presentation/d/
// So, it means the Google Form, Google Map and other types of links are not supported
func extractUIDFromSharedLink(driveLink string) (string, error) {
	patterns := map[string]string{
		"folder":       "/drive/folders/",
		"file":         "/file/d/",
		"spreadsheet":  "/spreadsheets/d/",
		"document":     "/document/d/",
		"presentation": "/presentation/d/",
	}

	for _, pattern := range patterns {
		if strings.Contains(driveLink, pattern) {
			parts := strings.Split(driveLink, pattern)
			if len(parts) < 2 {
				return "", fmt.Errorf("invalid Google Drive link")
			}
			// Sample link: https://drive.google.com/drive/folders/xxxxxx?usp=drive_link
			// Sample link: https://drive.google.com/file/d/xxxxxx/view?usp=drive_link
			uidParts := strings.SplitN(parts[1], "?", 2)
			uidParts = strings.SplitN(uidParts[0], "/", 2)
			return uidParts[0], nil
		}
	}

	return "", fmt.Errorf("unrecognized Google Drive link format")
}

func isFolder(link string) bool {
	return strings.Contains(link, "/drive/folders/")
}

func convertDriveFileToComponentFile(driveFile *drive.File) *file {
	// Google Drive API only can support downloading the binary data.
	// So, when the file is not binary, we need to export the file as PDF/CSV first.
	// To make Google Drive Component can seamlessly work with other components, we need to add the file extension to the file name.
	fileExtension := exportFileExtension(driveFile.MimeType)
	if fileExtension != "" {
		driveFile.Name = addFileExtension(driveFile.Name, fileExtension)
	}

	return &file{
		ID:             driveFile.Id,
		Name:           driveFile.Name,
		CreatedTime:    driveFile.CreatedTime,
		ModifiedTime:   driveFile.ModifiedTime,
		Size:           driveFile.Size,
		MimeType:       driveFile.MimeType,
		Md5Checksum:    driveFile.Md5Checksum,
		Version:        driveFile.Version,
		WebViewLink:    driveFile.WebViewLink,
		WebContentLink: driveFile.WebContentLink,
	}
}

func exportFileExtension(mimeType string) string {
	switch mimeType {
	case "application/vnd.google-apps.spreadsheet":
		return ".csv"
	case "application/vnd.google-apps.presentation", "application/vnd.google-apps.document":
		return ".pdf"
	default:
		return ""
	}
}

func addFileExtension(fileName, Extension string) string {
	if !strings.HasSuffix(fileName, Extension) {
		return fileName + Extension
	}
	return fileName
}
