package instillartifact

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util"

	artifactpb "github.com/instill-ai/protogen-go/artifact/artifact/v1alpha"
)

func (e *execution) uploadFiles(input *structpb.Struct) (*structpb.Struct, error) {

	inputStruct := UploadFilesInput{}

	err := base.ConvertFromStructpb(input, &inputStruct)
	if err != nil {
		return nil, fmt.Errorf("failed to convert input to struct: %w", err)
	}

	if len(inputStruct.Options.Files) != len(inputStruct.Options.FileNames) {
		return nil, fmt.Errorf("number of files and file names do not match")
	}

	artifactClient := e.client

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	ctx = metadata.NewOutgoingContext(ctx, getRequestMetadata(e.SystemVariables))

	if inputStruct.Options.Option == "create new catalog" {
		_, err = artifactClient.CreateKnowledgeBase(ctx, &artifactpb.CreateKnowledgeBaseRequest{
			NamespaceId: inputStruct.Options.Namespace,
			Id:          inputStruct.Options.KnowledgeBaseID,
			Description: inputStruct.Options.Description,
			Tags:        inputStruct.Options.Tags,
		})

		if err != nil {
			if strings.Contains(err.Error(), "knowledge base name already exists") {
				log.Println("Knowledge base already exists, skipping creation")
			} else {
				return nil, fmt.Errorf("failed to create new knowledge base: %w", err)
			}
		}
	}

	output := UploadFilesOutput{
		Files: []FileOutput{},
	}

	for i, file := range inputStruct.Options.Files {
		_, err := util.GetFileType(file, inputStruct.Options.FileNames[i])
		if err != nil {
			return nil, fmt.Errorf("failed to get file type: %w", err)
		}

		content := util.GetFileBase64Content(file)

		// CreateFile now handles upload and auto-triggers processing
		createRes, err := artifactClient.CreateFile(ctx, &artifactpb.CreateFileRequest{
			NamespaceId:     inputStruct.Options.Namespace,
			KnowledgeBaseId: inputStruct.Options.KnowledgeBaseID,
			File: &artifactpb.File{
				Filename: inputStruct.Options.FileNames[i],
				Content:  content,
			},
		})

		if err != nil {
			return nil, fmt.Errorf("failed to create file: %w", err)
		}

		createdFilePB := createRes.File

		output.Files = append(output.Files, FileOutput{
			FileUID:         createdFilePB.Uid,
			FileName:        createdFilePB.Filename,
			FileType:        createdFilePB.Type.String(),
			CreateTime:      createdFilePB.CreateTime.AsTime().Format(time.RFC3339),
			UpdateTime:      createdFilePB.UpdateTime.AsTime().Format(time.RFC3339),
			Size:            createdFilePB.Size,
			KnowledgeBaseID: inputStruct.Options.KnowledgeBaseID,
		})
	}

	// Files now auto-process, no need for separate ProcessCatalogFiles call
	output.Status = true

	return base.ConvertToStructpb(output)
}
