package instillartifact

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"

	artifactPB "github.com/instill-ai/protogen-go/artifact/artifact/v1alpha"
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

	filesRes, err := artifactClient.ListCatalogFiles(ctx, &artifactPB.ListCatalogFilesRequest{
		NamespaceId: inputStruct.Namespace,
		CatalogId:   inputStruct.CatalogID,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list catalog files: %w", err)
	}

	output := GetFilesMetadataOutput{
		Files: []FileOutput{},
	}

	output.setOutput(filesRes)

	for filesRes != nil && filesRes.NextPageToken != "" {
		filesRes, err = artifactClient.ListCatalogFiles(ctx, &artifactPB.ListCatalogFilesRequest{
			NamespaceId: inputStruct.Namespace,
			CatalogId:   inputStruct.CatalogID,
			PageToken:   filesRes.NextPageToken,
		})

		if err != nil {
			return nil, fmt.Errorf("failed to list catalog files: %w", err)
		}

		output.setOutput(filesRes)
	}

	return base.ConvertToStructpb(output)
}

func (output *GetFilesMetadataOutput) setOutput(filesRes *artifactPB.ListCatalogFilesResponse) {
	for _, filePB := range filesRes.Files {
		output.Files = append(output.Files, FileOutput{
			FileUID:    filePB.FileUid,
			FileName:   filePB.Name,
			FileType:   artifactPB.FileType_name[int32(filePB.Type)],
			CreateTime: filePB.CreateTime.AsTime().Format(time.RFC3339),
			UpdateTime: filePB.UpdateTime.AsTime().Format(time.RFC3339),
			Size:       filePB.Size,
		})
	}
}
