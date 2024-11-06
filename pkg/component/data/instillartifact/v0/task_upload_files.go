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

	artifactPB "github.com/instill-ai/protogen-go/artifact/artifact/v1alpha"
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

	artifactClient, connection := e.client, e.connection
	defer connection.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	ctx = metadata.NewOutgoingContext(ctx, getRequestMetadata(e.SystemVariables))

	if inputStruct.Options.Option == "create new catalog" {
		_, err = artifactClient.CreateCatalog(ctx, &artifactPB.CreateCatalogRequest{
			NamespaceId: inputStruct.Options.Namespace,
			Name:        inputStruct.Options.CatalogID,
			Description: inputStruct.Options.Description,
			Tags:        inputStruct.Options.Tags,
		})

		if err != nil {
			if strings.Contains(err.Error(), "knowledge base name already exists") {
				log.Println("Catalog already exists, skipping creation")
			} else {
				return nil, fmt.Errorf("failed to create new catalog: %w", err)
			}
		}
	}

	output := UploadFilesOutput{
		Files: []FileOutput{},
	}

	fileUIDs := []string{}
	for i, file := range inputStruct.Options.Files {
		fileType, err := util.GetFileType(file, inputStruct.Options.FileNames[i])
		if err != nil {
			return nil, fmt.Errorf("failed to get file type: %w", err)
		}
		typeString := "FILE_TYPE_" + strings.ToUpper(fileType)

		content := util.GetFileBase64Content(file)

		typePB := artifactPB.FileType_value[typeString]
		filePB := &artifactPB.File{
			Name:    inputStruct.Options.FileNames[i],
			Type:    artifactPB.FileType(typePB),
			Content: content,
		}
		uploadRes, err := artifactClient.UploadCatalogFile(ctx, &artifactPB.UploadCatalogFileRequest{
			NamespaceId: inputStruct.Options.Namespace,
			CatalogId:   inputStruct.Options.CatalogID,
			File:        filePB,
		})

		if err != nil {
			return nil, fmt.Errorf("failed to upload file: %w", err)
		}

		uploadedFilePB := uploadRes.File

		fileUIDs = append(fileUIDs, uploadedFilePB.FileUid)

		output.Files = append(output.Files, FileOutput{
			FileUID:    uploadedFilePB.FileUid,
			FileName:   uploadedFilePB.Name,
			FileType:   artifactPB.FileType_name[int32(uploadedFilePB.Type)],
			CreateTime: uploadedFilePB.CreateTime.AsTime().Format(time.RFC3339),
			UpdateTime: uploadedFilePB.UpdateTime.AsTime().Format(time.RFC3339),
			Size:       uploadedFilePB.Size,
			CatalogID:  inputStruct.Options.CatalogID,
		})
	}

	_, err = artifactClient.ProcessCatalogFiles(ctx, &artifactPB.ProcessCatalogFilesRequest{
		FileUids: fileUIDs,
	})

	if err == nil {
		output.Status = true
	}

	return base.ConvertToStructpb(output)
}
