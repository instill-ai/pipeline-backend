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

type UploadFileInput struct {
	Options UploadData `json:"options"`
}

type UploadData struct {
	Option    string `json:"option"`
	Namespace string `json:"namespace"`
	CatalogID string `json:"catalog-id"`
	// Base64 encoded file content
	File        string   `json:"file"`
	FileName    string   `json:"file-name"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
}

func (input *UploadFileInput) isNewCatalog() bool {
	return input.Options.Option == "create new catalog"
}

type UploadFileOutput struct {
	File   FileOutput `json:"file"`
	Status bool       `json:"status"`
}

type FileOutput struct {
	FileUID    string `json:"file-uid"`
	FileName   string `json:"file-name"`
	FileType   string `json:"file-type"`
	CreateTime string `json:"create-time"`
	UpdateTime string `json:"update-time"`
	Size       int64  `json:"size"`
	CatalogID  string `json:"catalog-id"`
}

type Connection interface {
	Close() error
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

		fmt.Println("found", found)
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

type UploadFilesInput struct {
	Options UploadMultipleData `json:"options"`
}

type UploadMultipleData struct {
	Option    string `json:"option"`
	Namespace string `json:"namespace"`
	CatalogID string `json:"catalog-id"`
	// Base64 encoded file content
	Files       []string `json:"files"`
	FileNames   []string `json:"file-names"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
}

type UploadFilesOutput struct {
	Files  []FileOutput `json:"files"`
	Status bool         `json:"status"`
}

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

type GetFilesMetadataInput struct {
	Namespace string `json:"namespace"`
	CatalogID string `json:"catalog-id"`
}

type GetFilesMetadataOutput struct {
	Files []FileOutput `json:"files"`
}

func (e *execution) getFilesMetadata(input *structpb.Struct) (*structpb.Struct, error) {

	inputStruct := GetFilesMetadataInput{}

	err := base.ConvertFromStructpb(input, &inputStruct)
	if err != nil {
		return nil, fmt.Errorf("failed to convert input to struct: %w", err)
	}

	artifactClient, connection := e.client, e.connection

	defer connection.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	ctx = metadata.NewOutgoingContext(ctx, getRequestMetadata(e.SystemVariables))

	filesRes, err := artifactClient.ListCatalogFiles(ctx, &artifactPB.ListCatalogFilesRequest{
		NamespaceId: inputStruct.Namespace,
		CatalogId:   inputStruct.CatalogID,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list catalog files: %w", err)
	}

	output := GetFilesMetadataOutput{
		Files: []FileOutput{},
	}

	output.setOutput(filesRes)

	for filesRes != nil && filesRes.NextPageToken != "" {
		filesRes, err = artifactClient.ListCatalogFiles(ctx, &artifactPB.ListCatalogFilesRequest{
			NamespaceId: inputStruct.Namespace,
			CatalogId:   inputStruct.CatalogID,
			PageToken:   filesRes.NextPageToken,
		})

		if err != nil {
			return nil, fmt.Errorf("failed to list catalog files: %w", err)
		}

		output.setOutput(filesRes)
	}

	return base.ConvertToStructpb(output)

}

func (output *GetFilesMetadataOutput) setOutput(filesRes *artifactPB.ListCatalogFilesResponse) {
	for _, filePB := range filesRes.Files {
		output.Files = append(output.Files, FileOutput{
			FileUID:    filePB.FileUid,
			FileName:   filePB.Name,
			FileType:   artifactPB.FileType_name[int32(filePB.Type)],
			CreateTime: filePB.CreateTime.AsTime().Format(time.RFC3339),
			UpdateTime: filePB.UpdateTime.AsTime().Format(time.RFC3339),
			Size:       filePB.Size,
		})
	}
}

type GetChunksMetadataInput struct {
	Namespace string `json:"namespace"`
	CatalogID string `json:"catalog-id"`
	FileUID   string `json:"file-uid"`
}

type GetChunksMetadataOutput struct {
	Chunks []ChunkOutput `json:"chunks"`
}

type ChunkOutput struct {
	ChunkUID        string `json:"chunk-uid"`
	Retrievable     bool   `json:"retrievable"`
	StartPosition   uint32 `json:"start-position"`
	EndPosition     uint32 `json:"end-position"`
	TokenCount      uint32 `json:"token-count"`
	CreateTime      string `json:"create-time"`
	OriginalFileUID string `json:"original-file-uid"`
}

func (e *execution) getChunksMetadata(input *structpb.Struct) (*structpb.Struct, error) {

	inputStruct := GetChunksMetadataInput{}
	err := base.ConvertFromStructpb(input, &inputStruct)
	if err != nil {
		return nil, fmt.Errorf("failed to convert input to struct: %w", err)
	}

	artifactClient, connection := e.client, e.connection

	defer connection.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	ctx = metadata.NewOutgoingContext(ctx, getRequestMetadata(e.SystemVariables))

	chunksRes, err := artifactClient.ListChunks(ctx, &artifactPB.ListChunksRequest{
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

type GetFileInMarkdownInput struct {
	Namespace string `json:"namespace"`
	CatalogID string `json:"catalog-id"`
	FileUID   string `json:"file-uid"`
}

type GetFileInMarkdownOutput struct {
	OriginalFileUID string `json:"original-file-uid"`
	Content         string `json:"content"`
	CreateTime      string `json:"create-time"`
	UpdateTime      string `json:"update-time"`
}

func (e *execution) getFileInMarkdown(input *structpb.Struct) (*structpb.Struct, error) {

	inputStruct := GetFileInMarkdownInput{}
	err := base.ConvertFromStructpb(input, &inputStruct)
	if err != nil {
		return nil, fmt.Errorf("failed to convert input to struct: %w", err)
	}

	artifactClient, connection := e.client, e.connection

	defer connection.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	ctx = metadata.NewOutgoingContext(ctx, getRequestMetadata(e.SystemVariables))

	fileTextsRes, err := artifactClient.GetSourceFile(ctx, &artifactPB.GetSourceFileRequest{
		NamespaceId: inputStruct.Namespace,
		CatalogId:   inputStruct.CatalogID,
		FileUid:     inputStruct.FileUID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get source file: %w", err)
	}

	output := GetFileInMarkdownOutput{
		OriginalFileUID: fileTextsRes.SourceFile.OriginalFileUid,
		Content:         fileTextsRes.SourceFile.Content,
		CreateTime:      fileTextsRes.SourceFile.CreateTime.AsTime().Format(time.RFC3339),
		UpdateTime:      fileTextsRes.SourceFile.UpdateTime.AsTime().Format(time.RFC3339),
	}

	return base.ConvertToStructpb(output)
}

type SearchChunksInput struct {
	Namespace  string `json:"namespace"`
	CatalogID  string `json:"catalog-id"`
	TextPrompt string `json:"text-prompt"`
	TopK       uint32 `json:"top-k"`
}

type SearchChunksOutput struct {
	Chunks []SimilarityChunk `json:"chunks"`
}

type SimilarityChunk struct {
	ChunkUID        string  `json:"chunk-uid"`
	SimilarityScore float32 `json:"similarity-score"`
	TextContent     string  `json:"text-content"`
	SourceFileName  string  `json:"source-file-name"`
}

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
		NamespaceId: inputStruct.Namespace,
		CatalogId:   inputStruct.CatalogID,
		TextPrompt:  inputStruct.TextPrompt,
		TopK:        inputStruct.TopK,
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

type QueryInput struct {
	Namespace string `json:"namespace"`
	CatalogID string `json:"catalog-id"`
	Question  string `json:"question"`
	TopK      int32  `json:"top-k"`
}

type QueryOutput struct {
	Answer string            `json:"answer"`
	Chunks []SimilarityChunk `json:"chunks"`
}

func (e *execution) query(input *structpb.Struct) (*structpb.Struct, error) {

	inputStruct := QueryInput{}
	err := base.ConvertFromStructpb(input, &inputStruct)
	if err != nil {
		return nil, fmt.Errorf("failed to convert input to struct: %w", err)
	}

	artifactClient, connection := e.client, e.connection

	defer connection.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	ctx = metadata.NewOutgoingContext(ctx, getRequestMetadata(e.SystemVariables))

	queryRes, err := artifactClient.QuestionAnswering(ctx, &artifactPB.QuestionAnsweringRequest{
		NamespaceId: inputStruct.Namespace,
		CatalogId:   inputStruct.CatalogID,
		Question:    inputStruct.Question,
		TopK:        inputStruct.TopK,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to question answering: %w", err)
	}

	output := QueryOutput{
		Answer: queryRes.Answer,
		Chunks: []SimilarityChunk{},
	}

	for _, chunkPB := range queryRes.SimilarChunks {
		output.Chunks = append(output.Chunks, SimilarityChunk{
			ChunkUID:        chunkPB.ChunkUid,
			SimilarityScore: chunkPB.SimilarityScore,
			TextContent:     chunkPB.TextContent,
			SourceFileName:  chunkPB.SourceFile,
		})
	}

	return base.ConvertToStructpb(output)
}

type MatchFileStatusInput struct {
	Namespace string `json:"namespace"`
	CatalogID string `json:"catalog-id"`
	FileUID   string `json:"file-uid"`
}

type MatchFileStatusOutput struct {
	Succeeded bool `json:"succeeded"`
}

func (e *execution) matchFileStatus(input *structpb.Struct) (*structpb.Struct, error) {

	inputStruct := MatchFileStatusInput{}
	err := base.ConvertFromStructpb(input, &inputStruct)
	if err != nil {
		return nil, fmt.Errorf("failed to convert input to struct: %w", err)
	}

	artifactClient, connection := e.client, e.connection

	defer connection.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	ctx = metadata.NewOutgoingContext(ctx, getRequestMetadata(e.SystemVariables))

	for {
		matchRes, err := artifactClient.ListCatalogFiles(ctx, &artifactPB.ListCatalogFilesRequest{
			NamespaceId: inputStruct.Namespace,
			CatalogId:   inputStruct.CatalogID,
			Filter: &artifactPB.ListCatalogFilesFilter{
				FileUids: []string{inputStruct.FileUID},
			},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to match file status: %w", err)
		}

		if matchRes.Files[0].ProcessStatus == artifactPB.FileProcessStatus_FILE_PROCESS_STATUS_COMPLETED {
			return base.ConvertToStructpb(MatchFileStatusOutput{
				Succeeded: true,
			})
		}

		if matchRes.Files[0].ProcessStatus == artifactPB.FileProcessStatus_FILE_PROCESS_STATUS_FAILED {
			return base.ConvertToStructpb(MatchFileStatusOutput{
				Succeeded: false,
			})
		}
	}

}
