package instillartifact

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"

	artifactpb "github.com/instill-ai/protogen-go/artifact/v1alpha"
)

func (e *execution) getFilesMetadata(input *structpb.Struct) (*structpb.Struct, error) {

	inputStruct := GetFilesMetadataInput{}

	err := base.ConvertFromStructpb(input, &inputStruct)
	if err != nil {
		return nil, fmt.Errorf("failed to convert input to struct: %w", err)
	}

	artifactClient := e.client

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	ctx = metadata.NewOutgoingContext(ctx, getRequestMetadata(e.SystemVariables))

	filter := fmt.Sprintf("knowledgeBaseId=\"%s\"", inputStruct.KnowledgeBaseID)
	filesRes, err := artifactClient.ListFiles(ctx, &artifactpb.ListFilesRequest{
		Parent: fmt.Sprintf("namespaces/%s", inputStruct.Namespace),
		Filter: &filter,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	output := GetFilesMetadataOutput{
		Files: []FileOutput{},
	}

	output.setOutput(filesRes)

	for filesRes != nil && filesRes.NextPageToken != "" {
		nextToken := filesRes.NextPageToken
		filesRes, err = artifactClient.ListFiles(ctx, &artifactpb.ListFilesRequest{
			Parent:    fmt.Sprintf("namespaces/%s", inputStruct.Namespace),
			Filter:    &filter,
			PageToken: &nextToken,
		})

		if err != nil {
			return nil, fmt.Errorf("failed to list files: %w", err)
		}

		output.setOutput(filesRes)
	}

	return base.ConvertToStructpb(output)
}

func (output *GetFilesMetadataOutput) setOutput(filesRes *artifactpb.ListFilesResponse) {
	for _, filePB := range filesRes.Files {
		output.Files = append(output.Files, FileOutput{
			FileUID:    filePB.Id,
			FileName:   filePB.DisplayName,
			FileType:   filePB.Type.String(),
			CreateTime: filePB.CreateTime.AsTime().Format(time.RFC3339),
			UpdateTime: filePB.UpdateTime.AsTime().Format(time.RFC3339),
			Size:       filePB.Size,
		})
	}
}
