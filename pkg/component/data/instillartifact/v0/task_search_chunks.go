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

	artifactpb "github.com/instill-ai/protogen-go/artifact/artifact/v1alpha"
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

	var fileMediaType artifactpb.FileMediaType
	var contentType artifactpb.ContentType

	switch inputStruct.FileMediaType {
	case "document":
		fileMediaType = artifactpb.FileMediaType_FILE_MEDIA_TYPE_DOCUMENT
	case "image":
		fileMediaType = artifactpb.FileMediaType_FILE_MEDIA_TYPE_IMAGE
	case "audio":
		fileMediaType = artifactpb.FileMediaType_FILE_MEDIA_TYPE_AUDIO
	case "video":
		fileMediaType = artifactpb.FileMediaType_FILE_MEDIA_TYPE_VIDEO
	default:
		fileMediaType = artifactpb.FileMediaType_FILE_MEDIA_TYPE_UNSPECIFIED
	}

	switch inputStruct.ContentType {
	case "chunk":
		contentType = artifactpb.ContentType_CONTENT_TYPE_CHUNK
	case "summary":
		contentType = artifactpb.ContentType_CONTENT_TYPE_SUMMARY
	case "augmented":
		contentType = artifactpb.ContentType_CONTENT_TYPE_AUGMENTED
	default:
		contentType = artifactpb.ContentType_CONTENT_TYPE_UNSPECIFIED
	}

	req := &artifactpb.SimilarityChunksSearchRequest{
		NamespaceId:   inputStruct.Namespace,
		CatalogId:     inputStruct.CatalogID,
		TextPrompt:    inputStruct.TextPrompt,
		TopK:          inputStruct.TopK,
		FileMediaType: fileMediaType,
		ContentType:   contentType,
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
		req.FileUids = fileUIDs
	}

	searchRes, err := artifactClient.SimilarityChunksSearch(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to search chunks: %w", err)
	}

	output := SearchChunksOutput{
		Chunks: make([]SimilarityChunk, 0, len(searchRes.GetSimilarChunks())),
	}

	for _, chunkPB := range searchRes.GetSimilarChunks() {
		chunk := SimilarityChunk{
			ChunkUID:        chunkPB.GetChunkUid(),
			SimilarityScore: chunkPB.GetSimilarityScore(),
			TextContent:     chunkPB.GetTextContent(),
			SourceFileName:  chunkPB.GetSourceFile(),
			SourceFileUID:   chunkPB.GetChunkMetadata().GetOriginalFileUid(),
			ContentType:     chunkPB.GetChunkMetadata().GetContentType().String(),
		}

		ref := chunkPB.GetChunkMetadata().GetReference()
		if ref != nil {
			chunk.Reference = &ChunkReference{
				Start: FilePosition{
					Unit:        strings.TrimPrefix(ref.GetStart().GetUnit().String(), "UNIT_"),
					Coordinates: ref.GetStart().GetCoordinates(),
				},
				End: FilePosition{
					Unit:        strings.TrimPrefix(ref.GetEnd().GetUnit().String(), "UNIT_"),
					Coordinates: ref.GetEnd().GetCoordinates(),
				},
			}
		}

		output.Chunks = append(output.Chunks, chunk)
	}

	return base.ConvertToStructpb(output)
}
