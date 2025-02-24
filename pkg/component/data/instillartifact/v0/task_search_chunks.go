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

	artifactClient, connection := e.client, e.connection

	defer connection.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	ctx = metadata.NewOutgoingContext(ctx, getRequestMetadata(e.SystemVariables))

	searchRes, err := artifactClient.SimilarityChunksSearch(ctx, &artifactPB.SimilarityChunksSearchRequest{
		NamespaceId:   inputStruct.Namespace,
		CatalogId:     inputStruct.CatalogID,
		TextPrompt:    inputStruct.TextPrompt,
		TopK:          inputStruct.TopK,
		FileName:      inputStruct.Filename,
		FileMediaType: artifactPB.FileMediaType(artifactPB.FileMediaType_value[inputStruct.FileMediaType]),
		ContentType:   artifactPB.ContentType(artifactPB.ContentType_value[inputStruct.ContetType]),
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
		})
	}

	return base.ConvertToStructpb(output)
}
