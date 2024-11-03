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

func (input *UploadFileInput) isNewCatalog() bool {
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

	if inputStruct.isNewCatalog() {

		catalogs, err := artifactClient.ListCatalogs(ctx, &artifactPB.ListCatalogsRequest{
			NamespaceId: inputStruct.Options.Namespace,
		})

		if err != nil {
			return nil, fmt.Errorf("failed to list catalogs: %w", err)
		}

		found := false
		for _, catalog := range catalogs.Catalogs {
			if catalog.Name == inputStruct.Options.CatalogID {
				found = true
				log.Println("Catalog already exists, skipping creation")
			}
		}

		if !found {
			_, err = artifactClient.CreateCatalog(ctx, &artifactPB.CreateCatalogRequest{
				NamespaceId: inputStruct.Options.Namespace,
				Name:        inputStruct.Options.CatalogID,
				Description: inputStruct.Options.Description,
				Tags:        inputStruct.Options.Tags,
			})
		}

		if err != nil {
			return nil, fmt.Errorf("failed to create new catalog: %w", err)
		}
	}

	output := UploadFileOutput{
		File: FileOutput{},
	}
	file := inputStruct.Options.File

	fileType, err := util.GetFileType(file, inputStruct.Options.FileName)
	if err != nil {
		return nil, fmt.Errorf("failed to get file type: %w", err)
	}
	typeString := "FILE_TYPE_" + strings.ToUpper(fileType)

	content := util.GetFileBase64Content(file)

	typePB := artifactPB.FileType_value[typeString]
	filePB := &artifactPB.File{
		Name:    inputStruct.Options.FileName,
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

	output.File = FileOutput{
		FileUID:    uploadedFilePB.FileUid,
		FileName:   uploadedFilePB.Name,
		FileType:   artifactPB.FileType_name[int32(uploadedFilePB.Type)],
		CreateTime: uploadedFilePB.CreateTime.AsTime().Format(time.RFC3339),
		UpdateTime: uploadedFilePB.UpdateTime.AsTime().Format(time.RFC3339),
		Size:       uploadedFilePB.Size,
		CatalogID:  inputStruct.Options.CatalogID,
	}

	// TODO: chuang, will need to process again in another task.
	_, err = artifactClient.ProcessCatalogFiles(ctx, &artifactPB.ProcessCatalogFilesRequest{
		FileUids: []string{uploadedFilePB.FileUid},
	})

	if err == nil {
		output.Status = true
	}

	return base.ConvertToStructpb(output)
}
