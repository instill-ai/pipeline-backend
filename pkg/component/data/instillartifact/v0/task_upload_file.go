package instillartifact

import (
	"context"
	"fmt"
	"log"
	"time"

	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util"

	artifactpb "github.com/instill-ai/protogen-go/artifact/v1alpha"
)

func (input *UploadFileInput) isNewKnowledgeBase() bool {
	return input.Options.Option == "create new catalog"
}

func (e *execution) uploadFile(input *structpb.Struct) (*structpb.Struct, error) {

	inputStruct := UploadFileInput{}

	err := base.ConvertFromStructpb(input, &inputStruct)
	if err != nil {
		return nil, fmt.Errorf("failed to convert input to struct: %w", err)
	}

	artifactClient := e.client

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	ctx = metadata.NewOutgoingContext(ctx, getRequestMetadata(e.SystemVariables))

	if inputStruct.isNewKnowledgeBase() {

		knowledgeBases, err := artifactClient.ListKnowledgeBases(ctx, &artifactpb.ListKnowledgeBasesRequest{
			Parent: fmt.Sprintf("namespaces/%s", inputStruct.Options.Namespace),
		})

		if err != nil {
			return nil, fmt.Errorf("failed to list knowledge bases: %w", err)
		}

		found := false
		for _, kb := range knowledgeBases.KnowledgeBases {
			if kb.Id == inputStruct.Options.KnowledgeBaseID {
				found = true
				log.Println("Knowledge base already exists, skipping creation")
			}
		}

		if !found {
			_, err = artifactClient.CreateKnowledgeBase(ctx, &artifactpb.CreateKnowledgeBaseRequest{
				Parent: fmt.Sprintf("namespaces/%s", inputStruct.Options.Namespace),
				KnowledgeBase: &artifactpb.KnowledgeBase{
					DisplayName: inputStruct.Options.KnowledgeBaseID,
					Description: inputStruct.Options.Description,
					Tags:        inputStruct.Options.Tags,
				},
			})
		}

		if err != nil {
			return nil, fmt.Errorf("failed to create new knowledge base: %w", err)
		}
	}

	output := UploadFileOutput{
		File: FileOutput{},
	}
	file := inputStruct.Options.File

	_, err = util.GetFileType(file, inputStruct.Options.FileName)
	if err != nil {
		return nil, fmt.Errorf("failed to get file type: %w", err)
	}

	content := util.GetFileBase64Content(file)

	// CreateFile now handles upload and auto-triggers processing
	createRes, err := artifactClient.CreateFile(ctx, &artifactpb.CreateFileRequest{
		Parent: fmt.Sprintf("namespaces/%s", inputStruct.Options.Namespace),
		File: &artifactpb.File{
			DisplayName: inputStruct.Options.FileName,
			Content:     content,
		},
		KnowledgeBaseId: inputStruct.Options.KnowledgeBaseID,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}

	createdFilePB := createRes.File

	output.File = FileOutput{
		FileUID:         createdFilePB.Id,
		FileName:        createdFilePB.DisplayName,
		FileType:        createdFilePB.Type.String(),
		CreateTime:      createdFilePB.CreateTime.AsTime().Format(time.RFC3339),
		UpdateTime:      createdFilePB.UpdateTime.AsTime().Format(time.RFC3339),
		Size:            createdFilePB.Size,
		KnowledgeBaseID: inputStruct.Options.KnowledgeBaseID,
	}

	// Files now auto-process, no need for separate ProcessCatalogFiles call
	output.Status = true

	return base.ConvertToStructpb(output)
}
