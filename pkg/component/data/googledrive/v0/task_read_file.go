package googledrive

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func (e *execution) readFile(input *structpb.Struct, job *base.Job, ctx context.Context) (*structpb.Struct, error) {

	inputStruct := readFileInput{}

	err := base.ConvertFromStructpb(input, &inputStruct)

	if err != nil {
		return nil, fmt.Errorf("convert input to struct: %w", err)
	}

	if isFolder(inputStruct.SharedLink) {
		return nil, fmt.Errorf("the input link is a folder link, please use the read-folder operation")
	}

	fileUID, err := extractUIDFromSharedLink(inputStruct.SharedLink)

	if err != nil {
		return nil, fmt.Errorf("extract UID from Google Drive link: %w", err)
	}

	driveFile, content, err := e.service.ReadFile(fileUID)

	if err != nil {
		return nil, fmt.Errorf("read file from Google Drive: %w", err)
	}

	file := convertDriveFileToComponentFile(driveFile)
	file.Content = *content

	output := readFileOutput{
		File: *file,
	}

	outputStruct, err := base.ConvertToStructpb(output)

	if err != nil {
		return nil, fmt.Errorf("convert output to struct: %w", err)
	}

	return outputStruct, nil
}
