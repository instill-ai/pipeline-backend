package googledrive

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

type readFileInput struct {
	SharedLink string `json:"shared-link"`
}

type readFileOutput struct {
	File file `json:"file"`
}

type file struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Content        string `json:"content"`
	CreatedTime    string `json:"created-time"`
	ModifiedTime   string `json:"modified-time"`
	Size           int64  `json:"size"`
	MimeType       string `json:"mime-type"`
	Md5Checksum    string `json:"md5-checksum,omitempty"`
	Version        int64  `json:"version"`
	WebViewLink    string `json:"web-view-link"`
	WebContentLink string `json:"web-content-link,omitempty"`
}

func (e *execution) readFile(input *structpb.Struct, job *base.Job, ctx context.Context) (*structpb.Struct, error) {

	inputStruct := readFileInput{}

	err := base.ConvertFromStructpb(input, &inputStruct)

	if err != nil {
		return nil, fmt.Errorf("convert input to struct: %w", err)
	}

	fileUID, err := extractUIDFromSharedLink(inputStruct.SharedLink)

	if err != nil {
		return nil, fmt.Errorf("extract UID from Google Drive link: %w", err)
	}

	file, err := e.service.readFile(fileUID)

	if err != nil {
		return nil, fmt.Errorf("read file from Google Drive: %w", err)
	}

	output := readFileOutput{
		File: *file,
	}

	outputStruct, err := base.ConvertToStructpb(output)

	if err != nil {
		return nil, fmt.Errorf("convert output to struct: %w", err)
	}

	return outputStruct, nil
}

type readFolderInput struct {
	SharedLink  string `json:"shared-link"`
	ReadContent bool   `json:"read-content"`
}

type readFolderOutput struct {
	Files []*file `json:"files"`
}

func (e *execution) readFolder(input *structpb.Struct, job *base.Job, ctx context.Context) (*structpb.Struct, error) {
	inputStruct := readFolderInput{}

	err := base.ConvertFromStructpb(input, &inputStruct)

	if err != nil {
		return nil, fmt.Errorf("convert input to struct: %w", err)
	}

	folderUID, err := extractUIDFromSharedLink(inputStruct.SharedLink)

	if err != nil {
		return nil, fmt.Errorf("extract UID from Google Drive link: %w", err)
	}

	files, err := e.service.readFolder(folderUID, inputStruct.ReadContent)

	if err != nil {
		return nil, fmt.Errorf("read folder from Google Drive: %w", err)
	}

	output := readFolderOutput{
		Files: files,
	}

	outputStruct, err := base.ConvertToStructpb(output)

	if err != nil {
		return nil, fmt.Errorf("convert output to struct: %w", err)
	}

	return outputStruct, nil
}

// Now, we support the following types of Google Drive links:
// 1. Folder: https://drive.google
// 2. File: https://drive.google.com/file/d/
// 3. Spreadsheet: https://docs.google.com/spreadsheets/d/
// 4. Document: https://docs.google.com/document/d/
// 5. Presentation: https://docs.google.com/presentation/d/
// 6. Colab: https://colab.research.google.com/drive/
// So, it means the Google Form, Google Map and other types of links are not supported
func extractUIDFromSharedLink(driveLink string) (string, error) {
	patterns := map[string]string{
		"file":         "/file/d/",
		"folder":       "/drive/folders/",
		"spreadsheet":  "/spreadsheets/d/",
		"document":     "/document/d/",
		"presentation": "/presentation/d/",
		"colab":        "colab.research.google.com/drive/",
	}

	// Iterate over the patterns to find a match
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
