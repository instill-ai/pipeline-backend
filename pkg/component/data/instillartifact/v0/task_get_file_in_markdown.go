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

func (e *execution) getFileInMarkdown(input *structpb.Struct) (*structpb.Struct, error) {

	inputStruct := GetFileInMarkdownInput{}
	err := base.ConvertFromStructpb(input, &inputStruct)
	if err != nil {
		return nil, fmt.Errorf("failed to convert input to struct: %w", err)
	}

	artifactClient := e.client

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	ctx = metadata.NewOutgoingContext(ctx, getRequestMetadata(e.SystemVariables))

	fileTextsRes, err := artifactClient.GetSourceFile(ctx, &artifactPB.GetSourceFileRequest{
		NamespaceId: inputStruct.Namespace,
		CatalogId:   inputStruct.CatalogID,
		FileUid:     inputStruct.FileUID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get source file: %w", err)
	}

	output := GetFileInMarkdownOutput{
		OriginalFileUID: fileTextsRes.SourceFile.OriginalFileUid,
		Content:         fileTextsRes.SourceFile.Content,
		CreateTime:      fileTextsRes.SourceFile.CreateTime.AsTime().Format(time.RFC3339),
		UpdateTime:      fileTextsRes.SourceFile.UpdateTime.AsTime().Format(time.RFC3339),
	}

	return base.ConvertToStructpb(output)
}
