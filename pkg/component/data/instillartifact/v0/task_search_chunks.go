package instillartifact

import (
	"context"
	"fmt"
	"strings"
	"time"

	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/gofrs/uuid"
	"github.com/instill-ai/pipeline-backend/pkg/component/base"

	artifactpb "github.com/instill-ai/protogen-go/artifact/v1alpha"
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

	var fileMediaType artifactpb.File_FileMediaType
	var chunkType artifactpb.Chunk_Type

	switch inputStruct.FileMediaType {
	case "document":
		fileMediaType = artifactpb.File_FILE_MEDIA_TYPE_DOCUMENT
	case "image":
		fileMediaType = artifactpb.File_FILE_MEDIA_TYPE_IMAGE
	case "audio":
		fileMediaType = artifactpb.File_FILE_MEDIA_TYPE_AUDIO
	case "video":
		fileMediaType = artifactpb.File_FILE_MEDIA_TYPE_VIDEO
	default:
		fileMediaType = artifactpb.File_FILE_MEDIA_TYPE_UNSPECIFIED
	}

	switch inputStruct.ChunkType {
	case "content":
		chunkType = artifactpb.Chunk_TYPE_CONTENT
	case "summary":
		chunkType = artifactpb.Chunk_TYPE_SUMMARY
	case "augmented":
		chunkType = artifactpb.Chunk_TYPE_AUGMENTED
	default:
		chunkType = artifactpb.Chunk_TYPE_UNSPECIFIED
	}

	req := &artifactpb.SearchChunksRequest{
		Parent:          fmt.Sprintf("namespaces/%s", inputStruct.Namespace),
		KnowledgeBaseId: inputStruct.KnowledgeBaseID,
		TextPrompt:      inputStruct.TextPrompt,
		TopK:            inputStruct.TopK,
		FileMediaType:   fileMediaType,
		Type:            chunkType,
	}

	// Validate file UID param. Empty UIDs will be filtered out, other invalid
	// strings will cause an error.
	fileUIDs := make([]string, 0, len(inputStruct.FileUIDs))
	for _, uid := range inputStruct.FileUIDs {
		if uid == "" {
			continue
		}

		if _, err := uuid.FromString(uid); err != nil {
			return nil, fmt.Errorf("invalid file UID %s: %w", uid, err)
		}

		fileUIDs = append(fileUIDs, uid)
	}

	if len(fileUIDs) > 0 {
		req.FileIds = fileUIDs
	}

	searchRes, err := artifactClient.SearchChunks(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to search chunks: %w", err)
	}

	output := SearchChunksOutput{
		Chunks: make([]SimilarityChunk, 0, len(searchRes.GetSimilarChunks())),
	}

	for _, chunkPB := range searchRes.GetSimilarChunks() {
		// chunkPB.GetChunk() returns full resource name: namespaces/{namespace}/files/{file}/chunks/{chunk}
		// chunkPB.GetFile() returns full resource name: namespaces/{namespace}/files/{file}
		chunk := SimilarityChunk{
			ChunkUID:        chunkPB.GetChunk(),
			SimilarityScore: chunkPB.GetSimilarityScore(),
			TextContent:     chunkPB.GetTextContent(),
			SourceFileName:  chunkPB.GetFile(),
			SourceFileUID:   chunkPB.GetChunkMetadata().GetOriginalFileId(),
			ContentType:     chunkPB.GetChunkMetadata().GetType().String(),
		}

		ref := chunkPB.GetChunkMetadata().GetMarkdownReference()
		if ref != nil {
			chunk.Reference = &ChunkReference{
				Start: FilePosition{
					Unit:        strings.TrimPrefix(ref.GetStart().GetUnit().String(), "REFERENCE_UNIT_"),
					Coordinates: ref.GetStart().GetCoordinates(),
				},
				End: FilePosition{
					Unit:        strings.TrimPrefix(ref.GetEnd().GetUnit().String(), "REFERENCE_UNIT_"),
					Coordinates: ref.GetEnd().GetCoordinates(),
				},
			}
		}

		output.Chunks = append(output.Chunks, chunk)
	}

	return base.ConvertToStructpb(output)
}
