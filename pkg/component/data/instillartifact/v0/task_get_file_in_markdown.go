package instillartifact

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"

	artifactpb "github.com/instill-ai/protogen-go/artifact/artifact/v1alpha"
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

	// Get file with VIEW_CONTENT to get the markdown content URL
	fileRes, err := artifactClient.GetFile(ctx, &artifactpb.GetFileRequest{
		NamespaceId:     inputStruct.Namespace,
		KnowledgeBaseId: inputStruct.KnowledgeBaseID,
		FileId:          inputStruct.FileUID,
		View:            artifactpb.File_VIEW_CONTENT.Enum(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get file: %w", err)
	}

	// Fetch the markdown content from MinIO using the derived_resource_uri
	var content string
	if fileRes.DerivedResourceUri != nil && *fileRes.DerivedResourceUri != "" {
		resp, err := http.Get(*fileRes.DerivedResourceUri)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch markdown content: %w", err)
		}
		defer resp.Body.Close()

		contentBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read markdown content: %w", err)
		}
		content = string(contentBytes)
	}

	output := GetFileInMarkdownOutput{
		OriginalFileUID: fileRes.File.Uid,
		Content:         content,
		CreateTime:      fileRes.File.CreateTime.AsTime().Format(time.RFC3339),
		UpdateTime:      fileRes.File.UpdateTime.AsTime().Format(time.RFC3339),
	}

	return base.ConvertToStructpb(output)
}
