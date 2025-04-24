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

func (e *execution) searchChunks(input *structpb.Struct) (*structpb.Struct, error) {

	inputStruct := SearchChunksInput{}
	err := base.ConvertFromStructpb(input, &inputStruct)
	if err != nil {
		return nil, fmt.Errorf("failed to convert input to struct: %w", err)
	}

	artifactClient := e.client

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	ctx = metadata.NewOutgoingContext(ctx, getRequestMetadata(e.SystemVariables))

	var fileMediaType artifactPB.FileMediaType
	var contentType artifactPB.ContentType

	switch inputStruct.FileMediaType {
	case "document":
		fileMediaType = artifactPB.FileMediaType_FILE_MEDIA_TYPE_DOCUMENT
	case "image":
		fileMediaType = artifactPB.FileMediaType_FILE_MEDIA_TYPE_IMAGE
	case "audio":
		fileMediaType = artifactPB.FileMediaType_FILE_MEDIA_TYPE_AUDIO
	case "video":
		fileMediaType = artifactPB.FileMediaType_FILE_MEDIA_TYPE_VIDEO
	default:
		fileMediaType = artifactPB.FileMediaType_FILE_MEDIA_TYPE_UNSPECIFIED
	}

	switch inputStruct.ContetType {
	case "chunk":
		contentType = artifactPB.ContentType_CONTENT_TYPE_CHUNK
	case "summary":
		contentType = artifactPB.ContentType_CONTENT_TYPE_SUMMARY
	case "augmented":
		contentType = artifactPB.ContentType_CONTENT_TYPE_AUGMENTED
	default:
		contentType = artifactPB.ContentType_CONTENT_TYPE_UNSPECIFIED
	}

	searchRes, err := artifactClient.SimilarityChunksSearch(ctx, &artifactPB.SimilarityChunksSearchRequest{
		NamespaceId:   inputStruct.Namespace,
		CatalogId:     inputStruct.CatalogID,
		TextPrompt:    inputStruct.TextPrompt,
		TopK:          inputStruct.TopK,
		FileName:      inputStruct.Filename,
		FileMediaType: fileMediaType,
		ContentType:   contentType,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to search chunks: %w", err)
	}

	output := SearchChunksOutput{
		Chunks: []SimilarityChunk{},
	}

	for _, chunkPB := range searchRes.SimilarChunks {
		output.Chunks = append(output.Chunks, SimilarityChunk{
			ChunkUID:        chunkPB.ChunkUid,
			SimilarityScore: chunkPB.SimilarityScore,
			TextContent:     chunkPB.TextContent,
			SourceFileName:  chunkPB.SourceFile,
			ContentType:     chunkPB.GetChunkMetadata().GetContentType().String(),
		})
	}

	return base.ConvertToStructpb(output)
}
