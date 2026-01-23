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

	artifactpb "github.com/instill-ai/protogen-go/artifact/v1alpha"
)

const (
	googleDrive = "googleDrive"

	//The keys below are the single source of truth for the external metadata from knowledge base.
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

	artifactClient := e.client

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	ctx = metadata.NewOutgoingContext(ctx, getRequestMetadata(e.SystemVariables))

	knowledgeBases, err := artifactClient.ListKnowledgeBases(ctx, &artifactpb.ListKnowledgeBasesRequest{
		Parent: fmt.Sprintf("namespaces/%s", inputStruct.Namespace),
	})

	if err != nil {
		return nil, fmt.Errorf("list knowledge bases: %w", err)
	}

	found := false
	for _, kb := range knowledgeBases.KnowledgeBases {
		if kb.Id == inputStruct.KnowledgeBaseID {
			found = true
			break
		}
	}

	if !found {
		_, err = artifactClient.CreateKnowledgeBase(ctx, &artifactpb.CreateKnowledgeBaseRequest{
			Parent: fmt.Sprintf("namespaces/%s", inputStruct.Namespace),
			KnowledgeBase: &artifactpb.KnowledgeBase{
				DisplayName: inputStruct.KnowledgeBaseID,
			},
		})

		if err != nil {
			return nil, fmt.Errorf("create knowledge base: %w", err)
		}
	}

	var files []*artifactpb.File
	var nextToken string

	syncSource := getSource(inputStruct.ThirdPartyFiles[0].WebViewLink)

	kbFilter := fmt.Sprintf("knowledgeBaseId=\"%s\"", inputStruct.KnowledgeBaseID)
	for {
		var pageTokenPtr *string
		if nextToken != "" {
			pageTokenPtr = &nextToken
		}
		filesRes, err := artifactClient.ListFiles(ctx, &artifactpb.ListFilesRequest{
			Parent:    fmt.Sprintf("namespaces/%s", inputStruct.Namespace),
			Filter:    &kbFilter,
			PageToken: pageTokenPtr,
		})
		if err != nil {
			return nil, fmt.Errorf("list files: %w", err)
		}
		for _, file := range filesRes.Files {
			if getSourceFromExternalMetadata(file) == syncSource {
				files = append(files, file)
			}
		}
		if filesRes.NextPageToken == "" {
			break
		}
		nextToken = filesRes.NextPageToken
	}

	filesMap := map[string]*artifactpb.File{}

	for _, file := range files {
		externalUID := getIDFromExternalMetadata(file)
		filesMap[externalUID] = file
	}

	thirdPartyFilesMap := map[string]ThirdPartyFile{}

	for _, syncFile := range inputStruct.ThirdPartyFiles {
		thirdPartyFilesMap[syncFile.ID] = syncFile
	}

	// Delete Files when
	// 1. File is not in the third-party-files but in the knowledge base
	// 2. File is in the third-party-files and in the knowledge base, but the third-party modified-time is newer than the knowledge base's third-party's modified-time
	toBeDeleteFileIDs := []string{}
	toBeUpdateFiles := []ThirdPartyFile{}

	// Delete file and update file section
	for fileUID, file := range filesMap {
		thirdPartyFile, ok := thirdPartyFilesMap[fileUID]

		if !ok {
			toBeDeleteFileIDs = append(toBeDeleteFileIDs, file.Id)
			continue
		}

		thirdPartyModifiedTime, err := time.Parse(time.RFC3339, thirdPartyFile.ModifiedTime)

		if err != nil {
			return nil, fmt.Errorf("parse modified time in sync file: %w", err)
		}

		fileModifiedTime, err := getModifiedTimeFromExternalMetadata(file)

		if err != nil {
			return nil, fmt.Errorf("parse modified time in file: %w", err)
		}

		// It means overwrite the file here.
		if thirdPartyModifiedTime.After(fileModifiedTime) {
			toBeDeleteFileIDs = append(toBeDeleteFileIDs, file.Id)
			toBeUpdateFiles = append(toBeUpdateFiles, thirdPartyFile)
		}
	}

	// New file section
	toBeUploadFiles := []ThirdPartyFile{}
	for syncFileUID, syncFile := range thirdPartyFilesMap {
		_, ok := filesMap[syncFileUID]
		if !ok {
			toBeUploadFiles = append(toBeUploadFiles, syncFile)
		}
	}

	for _, fileID := range toBeDeleteFileIDs {
		_, err = artifactClient.DeleteFile(ctx, &artifactpb.DeleteFileRequest{
			Name: fmt.Sprintf("namespaces/%s/files/%s", inputStruct.Namespace, fileID),
		})
		if err != nil {
			return nil, fmt.Errorf("delete file uid %s: %w", fileID, err)
		}
	}

	outputStruct := SyncFilesOutput{
		UploadedFiles: []FileOutput{},
		UpdatedFiles:  []FileOutput{},
		FailureFiles:  []ThirdPartyFile{},
		ErrorMessages: []string{},
	}

	for _, syncFile := range toBeUploadFiles {

		uploadingFile, err := buildUploadingFile(syncFile)

		if err != nil {
			outputStruct.FailureFiles = append(outputStruct.FailureFiles, syncFile)
			outputStruct.ErrorMessages = append(outputStruct.ErrorMessages, fmt.Sprintf("build uploading file: %v", err))
			continue
		}

		createRes, err := artifactClient.CreateFile(ctx, &artifactpb.CreateFileRequest{
			Parent: fmt.Sprintf("namespaces/%s", inputStruct.Namespace),
			File: &artifactpb.File{
				DisplayName:      uploadingFile.Filename,
				Content:          uploadingFile.Content,
				ExternalMetadata: uploadingFile.ExternalMetadata,
			},
			KnowledgeBase: inputStruct.KnowledgeBaseID,
		})

		if err != nil {
			outputStruct.FailureFiles = append(outputStruct.FailureFiles, syncFile)
			outputStruct.ErrorMessages = append(outputStruct.ErrorMessages, fmt.Sprintf("create file: %v", err))
			continue
		}

		outputStruct.UploadedFiles = append(outputStruct.UploadedFiles, FileOutput{
			FileUID:         createRes.File.Id,
			FileName:        createRes.File.DisplayName,
			FileType:        createRes.File.Type.String(),
			CreateTime:      util.FormatToISO8601(createRes.File.CreateTime),
			UpdateTime:      util.FormatToISO8601(createRes.File.UpdateTime),
			Size:            createRes.File.Size,
			KnowledgeBaseID: inputStruct.KnowledgeBaseID,
		})
	}

	for _, syncFile := range toBeUpdateFiles {
		uploadingFile, err := buildUploadingFile(syncFile)
		if err != nil {
			outputStruct.FailureFiles = append(outputStruct.FailureFiles, syncFile)
			outputStruct.ErrorMessages = append(outputStruct.ErrorMessages, fmt.Sprintf("build uploading file: %v", err))
			continue
		}

		createRes, err := artifactClient.CreateFile(ctx, &artifactpb.CreateFileRequest{
			Parent: fmt.Sprintf("namespaces/%s", inputStruct.Namespace),
			File: &artifactpb.File{
				DisplayName:      uploadingFile.Filename,
				Content:          uploadingFile.Content,
				ExternalMetadata: uploadingFile.ExternalMetadata,
			},
			KnowledgeBase: inputStruct.KnowledgeBaseID,
		})

		if err != nil {
			outputStruct.FailureFiles = append(outputStruct.FailureFiles, syncFile)
			outputStruct.ErrorMessages = append(outputStruct.ErrorMessages, fmt.Sprintf("create file: %v", err))
			continue
		}

		outputStruct.UpdatedFiles = append(outputStruct.UpdatedFiles, FileOutput{
			FileUID:         createRes.File.Id,
			FileName:        createRes.File.DisplayName,
			FileType:        createRes.File.Type.String(),
			CreateTime:      util.FormatToISO8601(createRes.File.CreateTime),
			UpdateTime:      util.FormatToISO8601(createRes.File.UpdateTime),
			Size:            createRes.File.Size,
			KnowledgeBaseID: inputStruct.KnowledgeBaseID,
		})
	}

	// Files now auto-process, no need for separate ProcessCatalogFiles call
	outputStruct.Status = true

	return base.ConvertToStructpb(outputStruct)
}

type uploadingFileData struct {
	Filename         string
	Content          string
	ExternalMetadata *structpb.Struct
}

func buildUploadingFile(file ThirdPartyFile) (*uploadingFileData, error) {

	_, err := util.GetFileType(file.Content, file.Name)

	if err != nil {
		return nil, fmt.Errorf("get file type: %w", err)
	}

	return &uploadingFileData{
		Filename: file.Name,
		Content:  file.Content,
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
func getSourceFromExternalMetadata(file *artifactpb.File) string {
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

func getModifiedTimeFromExternalMetadata(file *artifactpb.File) (time.Time, error) {
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

func getIDFromExternalMetadata(file *artifactpb.File) string {
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
