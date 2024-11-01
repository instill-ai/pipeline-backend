package instillartifact

import (
	"context"
	"fmt"
	"strings"
	"time"

	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util"

	artifactPB "github.com/instill-ai/protogen-go/artifact/artifact/v1alpha"
)

const (
	googleDrive = "googleDrive"

	//The keys below are the single source of truth for the external metadata from catalog.
	// We use these keys to judge the uploading and overwriting files.
	idKey           = "id"
	modifiedTimeKey = "modified-time"
	sourceLinkKey   = "web-view-link"

	// We store these keys for the third-party metadata as the future extension.
	nameKey           = "name"
	createTimeKey     = "created-time"
	sizeKey           = "size"
	mimeTypeKey       = "mime-type"
	md5Key            = "md5-checksum"
	versionKey        = "version"
	webContentLinkKey = "web-content-link"
)

var (
	googleDriveLinkPrefixes = []string{
		"https://drive.google",
		"https://drive.google.com/file/d/",
		"https://docs.google.com/spreadsheets/d/",
		"https://docs.google.com/document/d/",
		"https://docs.google.com/presentation/d/",
	}
)

func (e *execution) syncFiles(input *structpb.Struct) (*structpb.Struct, error) {
	inputStruct := SyncFilesInput{}

	err := base.ConvertFromStructpb(input, &inputStruct)

	if err != nil {
		return nil, fmt.Errorf("convert input to struct: %w", err)
	}

	artifactClient, connection := e.client, e.connection

	defer connection.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	ctx = metadata.NewOutgoingContext(ctx, getRequestMetadata(e.SystemVariables))

	catalogs, err := artifactClient.ListCatalogs(ctx, &artifactPB.ListCatalogsRequest{
		NamespaceId: inputStruct.Namespace,
	})

	if err != nil {
		return nil, fmt.Errorf("list catalogs: %w", err)
	}

	found := false
	for _, catalog := range catalogs.Catalogs {
		if catalog.Name == inputStruct.CatalogID {
			found = true
			break
		}
	}

	if !found {
		_, err = artifactClient.CreateCatalog(ctx, &artifactPB.CreateCatalogRequest{
			NamespaceId: inputStruct.Namespace,
			Name:        inputStruct.CatalogID,
		})

		if err != nil {
			return nil, fmt.Errorf("create catalog: %w", err)
		}
	}

	var catalogFiles []*artifactPB.File
	var nextToken string

	syncSource := getSource(inputStruct.ThirdPartyFiles[0].WebViewLink)

	for {
		filesRes, err := artifactClient.ListCatalogFiles(ctx, &artifactPB.ListCatalogFilesRequest{
			NamespaceId: inputStruct.Namespace,
			CatalogId:   inputStruct.CatalogID,
			PageToken:   nextToken,
		})
		if err != nil {
			return nil, fmt.Errorf("list catalog files: %w", err)
		}
		for _, file := range filesRes.Files {
			if getSourceFromExternalMetadata(file) == syncSource {
				catalogFiles = append(catalogFiles, file)
			}
		}
		if filesRes.NextPageToken == "" {
			break
		}
		nextToken = filesRes.NextPageToken
	}

	// Delete Files when
	// 1. File is not in the third-party-files but in the catalog
	// 2. File is in the third-party-files and in the catalog, but the third-party modified-time is newer than the catalog's third-party's modified-time
	toBeDeleteFileUIDs := []string{}
	toBeUpdateFiles := []ThirdPartyFile{}

	// Delete file and update file section
	for _, catalogFile := range catalogFiles {
		found := false
		for _, syncFile := range inputStruct.ThirdPartyFiles {
			if getIDFromExternalMetadata(catalogFile) == syncFile.ID {
				catalogFileModifiedTime, err := getModifiedTimeFromExternalMetadata(catalogFile)
				if err != nil {
					return nil, fmt.Errorf("get third-party modified time from catalog file: %w", err)
				}
				syncFileModifiedTime, err := time.Parse(time.RFC3339, syncFile.ModifiedTime)

				if err != nil {
					return nil, fmt.Errorf("parse modified time in sync file: %w", err)
				}

				// It means overwrite the file here.
				if syncFileModifiedTime.After(catalogFileModifiedTime) {
					toBeDeleteFileUIDs = append(toBeDeleteFileUIDs, catalogFile.FileUid)
					toBeUpdateFiles = append(toBeUpdateFiles, syncFile)
				}
				found = true
				break
			}
		}
		if !found {
			toBeDeleteFileUIDs = append(toBeDeleteFileUIDs, catalogFile.FileUid)
		}
	}

	// New file section
	toBeUploadFiles := []ThirdPartyFile{}
	for _, syncFile := range inputStruct.ThirdPartyFiles {
		found := false
		for _, catalogFile := range catalogFiles {
			if getIDFromExternalMetadata(catalogFile) == syncFile.ID {
				found = true
				break
			}
		}
		if !found {
			toBeUploadFiles = append(toBeUploadFiles, syncFile)
		}
	}

	for _, fileUID := range toBeDeleteFileUIDs {
		_, err = artifactClient.DeleteCatalogFile(ctx, &artifactPB.DeleteCatalogFileRequest{
			FileUid: fileUID,
		})
		if err != nil {
			return nil, fmt.Errorf("delete catalog file: %w", err)
		}
	}

	outputStruct := SyncFilesOutput{
		UploadedFiles: []FileOutput{},
		UpdatedFiles:  []FileOutput{},
		FailureFiles:  []ThirdPartyFile{},
		ErrorMessages: []string{},
	}

	toBeProcessFileUIDs := []string{}
	for _, syncFile := range toBeUploadFiles {

		uploadingFile, err := buildUploadingFile(syncFile)

		if err != nil {
			outputStruct.FailureFiles = append(outputStruct.FailureFiles, syncFile)
			outputStruct.ErrorMessages = append(outputStruct.ErrorMessages, fmt.Sprintf("build uploading file: %v", err))
			continue
		}

		uploadRes, err := artifactClient.UploadCatalogFile(ctx, &artifactPB.UploadCatalogFileRequest{
			NamespaceId: inputStruct.Namespace,
			CatalogId:   inputStruct.CatalogID,
			File:        uploadingFile,
		})

		if err != nil {
			outputStruct.FailureFiles = append(outputStruct.FailureFiles, syncFile)
			outputStruct.ErrorMessages = append(outputStruct.ErrorMessages, fmt.Sprintf("upload file: %v", err))
			continue
		}

		toBeProcessFileUIDs = append(toBeProcessFileUIDs, uploadRes.File.FileUid)

		outputStruct.UploadedFiles = append(outputStruct.UploadedFiles, FileOutput{
			FileUID:    uploadRes.File.FileUid,
			FileName:   uploadRes.File.Name,
			FileType:   uploadRes.File.Type.String(),
			CreateTime: util.FormatToISO8601(uploadRes.File.CreateTime),
			UpdateTime: util.FormatToISO8601(uploadRes.File.UpdateTime),
			Size:       uploadRes.File.Size,
			CatalogID:  inputStruct.CatalogID,
		})
	}

	for _, syncFile := range toBeUpdateFiles {
		uploadingFile, err := buildUploadingFile(syncFile)
		if err != nil {
			outputStruct.FailureFiles = append(outputStruct.FailureFiles, syncFile)
			outputStruct.ErrorMessages = append(outputStruct.ErrorMessages, fmt.Sprintf("build uploading file: %v", err))
			continue
		}

		uploadRes, err := artifactClient.UploadCatalogFile(ctx, &artifactPB.UploadCatalogFileRequest{
			NamespaceId: inputStruct.Namespace,
			CatalogId:   inputStruct.CatalogID,
			File:        uploadingFile,
		})

		if err != nil {
			outputStruct.FailureFiles = append(outputStruct.FailureFiles, syncFile)
			outputStruct.ErrorMessages = append(outputStruct.ErrorMessages, fmt.Sprintf("upload file: %v", err))
			continue
		}

		toBeProcessFileUIDs = append(toBeProcessFileUIDs, uploadRes.File.FileUid)

		outputStruct.UpdatedFiles = append(outputStruct.UpdatedFiles, FileOutput{
			FileUID:    uploadRes.File.FileUid,
			FileName:   uploadRes.File.Name,
			FileType:   uploadRes.File.Type.String(),
			CreateTime: util.FormatToISO8601(uploadRes.File.CreateTime),
			UpdateTime: util.FormatToISO8601(uploadRes.File.UpdateTime),
			Size:       uploadRes.File.Size,
			CatalogID:  inputStruct.CatalogID,
		})
	}
	if len(toBeProcessFileUIDs) > 0 {
		_, err = artifactClient.ProcessCatalogFiles(ctx, &artifactPB.ProcessCatalogFilesRequest{
			FileUids: toBeProcessFileUIDs,
		})

		if err == nil {
			outputStruct.Status = true
		}
	}

	return base.ConvertToStructpb(outputStruct)
}

func buildUploadingFile(file ThirdPartyFile) (*artifactPB.File, error) {

	fileType, err := util.GetFileType(file.Content, file.Name)

	if err != nil {
		return nil, fmt.Errorf("get file type: %w", err)
	}

	typeString := "FILE_TYPE_" + strings.ToUpper(fileType)
	typePB := artifactPB.FileType_value[typeString]

	return &artifactPB.File{
		Name:    file.Name,
		Type:    artifactPB.FileType(typePB),
		Content: file.Content,
		ExternalMetadata: &structpb.Struct{
			Fields: map[string]*structpb.Value{
				idKey:             stringValue(file.ID),
				sourceLinkKey:     stringValue(file.WebViewLink),
				modifiedTimeKey:   stringValue(file.ModifiedTime),
				nameKey:           stringValue(file.Name),
				createTimeKey:     stringValue(file.CreatedTime),
				sizeKey:           numberValue(float64(file.Size)),
				mimeTypeKey:       stringValue(file.MimeType),
				md5Key:            stringValue(file.MD5Checksum),
				versionKey:        numberValue(float64(file.Version)),
				webContentLinkKey: stringValue(file.WebContentLink),
			},
		},
	}, nil
}

// We will extend this function for other sources in the future.
func getSource(link string) string {
	for _, prefix := range googleDriveLinkPrefixes {
		// All syncing files should have the same source.
		if strings.HasPrefix(link, prefix) {
			return googleDrive
		}
	}
	return ""
}

// We will save the third-party metadata with aligning the data structure of third-party-files in tasks.json.
func getSourceFromExternalMetadata(file *artifactPB.File) string {
	externalData := file.ExternalMetadata
	if externalData == nil {
		return ""
	}
	link, ok := externalData.GetFields()[sourceLinkKey]
	if !ok {
		return ""
	}
	if link.GetStringValue() == "" {
		return ""
	}
	return getSource(link.GetStringValue())
}

func getModifiedTimeFromExternalMetadata(file *artifactPB.File) (time.Time, error) {
	externalData := file.ExternalMetadata
	if externalData == nil {
		return time.Time{}, nil
	}
	modifiedTime, ok := externalData.GetFields()[modifiedTimeKey]
	if !ok {
		return time.Time{}, nil
	}
	if modifiedTime.GetStringValue() == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339, modifiedTime.GetStringValue())
}

func getIDFromExternalMetadata(file *artifactPB.File) string {
	externalData := file.ExternalMetadata
	if externalData == nil {
		return ""
	}
	id, ok := externalData.GetFields()[idKey]
	if !ok {
		return ""
	}
	if id.GetStringValue() == "" {
		return ""
	}
	return id.GetStringValue()
}

func stringValue(s string) *structpb.Value {
	return &structpb.Value{Kind: &structpb.Value_StringValue{StringValue: s}}
}

func numberValue(n float64) *structpb.Value {
	return &structpb.Value{Kind: &structpb.Value_NumberValue{NumberValue: n}}
}
