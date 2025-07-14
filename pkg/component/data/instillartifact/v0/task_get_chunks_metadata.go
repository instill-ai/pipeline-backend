package instillartifact

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"

	artifactpb "github.com/instill-ai/protogen-go/artifact/artifact/v1alpha"
)

func (e *execution) getChunksMetadata(input *structpb.Struct) (*structpb.Struct, error) {

	inputStruct := GetChunksMetadataInput{}
	err := base.ConvertFromStructpb(input, &inputStruct)
	if err != nil {
		return nil, fmt.Errorf("failed to convert input to struct: %w", err)
	}

	artifactClient := e.client

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	ctx = metadata.NewOutgoingContext(ctx, getRequestMetadata(e.SystemVariables))

	chunksRes, err := artifactClient.ListChunks(ctx, &artifactpb.ListChunksRequest{
		NamespaceId: inputStruct.Namespace,
		CatalogId:   inputStruct.CatalogID,
		FileUid:     inputStruct.FileUID,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list chunks: %w", err)
	}

	output := GetChunksMetadataOutput{
		Chunks: []ChunkOutput{},
	}

	for _, chunkPB := range chunksRes.Chunks {
		output.Chunks = append(output.Chunks, ChunkOutput{
			ChunkUID:        chunkPB.ChunkUid,
			Retrievable:     chunkPB.Retrievable,
			StartPosition:   chunkPB.StartPos,
			EndPosition:     chunkPB.EndPos,
			TokenCount:      chunkPB.Tokens,
			CreateTime:      chunkPB.CreateTime.AsTime().Format(time.RFC3339),
			OriginalFileUID: chunkPB.OriginalFileUid,
		})
	}

	return base.ConvertToStructpb(output)
}
