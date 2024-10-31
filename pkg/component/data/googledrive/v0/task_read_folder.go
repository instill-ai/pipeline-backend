package googledrive

import (
	"context"
	"fmt"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func (e *execution) readFolder(ctx context.Context, job *base.Job) error {
	inputStruct := readFolderInput{}

	err := job.Input.ReadData(ctx, inputStruct)

	if err != nil {
		return fmt.Errorf("read input data: %w", err)
	}

	if !isFolder(inputStruct.SharedLink) {
		return fmt.Errorf("the input link is not a folder link, please check the link")
	}

	folderUID, err := extractUIDFromSharedLink(inputStruct.SharedLink)

	if err != nil {
		return fmt.Errorf("extract UID from Google Drive link: %w", err)
	}

	driveFiles, contents, err := e.service.ReadFolder(folderUID, inputStruct.ReadContent)

	if err != nil {
		return fmt.Errorf("read folder from Google Drive: %w", err)
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

	err = job.Output.WriteData(ctx, output)

	if err != nil {
		return fmt.Errorf("write output data: %w", err)
	}

	return nil
}
