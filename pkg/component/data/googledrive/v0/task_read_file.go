package googledrive

import (
	"context"
	"fmt"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func (e *execution) readFile(ctx context.Context, job *base.Job) error {

	inputStruct := &readFileInput{}

	err := job.Input.ReadData(ctx, inputStruct)

	if err != nil {
		return fmt.Errorf("read input data: %w", err)
	}

	if isFolder(inputStruct.SharedLink) {
		return fmt.Errorf("the input link is a folder link, please use the read-folder operation")
	}

	fileUID, err := extractUIDFromSharedLink(inputStruct.SharedLink)

	if err != nil {
		return fmt.Errorf("extract UID from Google Drive link: %w", err)
	}

	driveFile, content, err := e.service.ReadFile(fileUID)

	if err != nil {
		return fmt.Errorf("read file from Google Drive: %w", err)
	}

	file := convertDriveFileToComponentFile(driveFile)
	file.Content = *content

	output := &readFileOutput{
		File: *file,
	}

	err = job.Output.WriteData(ctx, output)

	if err != nil {
		return fmt.Errorf("write output data: %w", err)
	}

	return nil
}
