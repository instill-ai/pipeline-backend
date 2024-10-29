package googledrive

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func (e *execution) readFolder(input *structpb.Struct, job *base.Job, ctx context.Context) (*structpb.Struct, error) {
	inputStruct := readFolderInput{}

	err := base.ConvertFromStructpb(input, &inputStruct)

	if err != nil {
		return nil, fmt.Errorf("convert input to struct: %w", err)
	}

	if !isFolder(inputStruct.SharedLink) {
		return nil, fmt.Errorf("the input link is not a folder link, please check the link")
	}

	folderUID, err := extractUIDFromSharedLink(inputStruct.SharedLink)

	if err != nil {
		return nil, fmt.Errorf("extract UID from Google Drive link: %w", err)
	}

	driveFiles, contents, err := e.service.ReadFolder(folderUID, inputStruct.ReadContent)

	if err != nil {
		return nil, fmt.Errorf("read folder from Google Drive: %w", err)
	}

	files := make([]*file, len(driveFiles))

	for i, driveFile := range driveFiles {
		file := convertDriveFileToComponentFile(driveFile)
		if inputStruct.ReadContent {
			file.Content = *contents[i]
		}
		files[i] = file
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
