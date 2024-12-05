package utils

import (
	"context"
	"encoding/base64"
	"fmt"
	"mime"
	"net/url"
	"strings"

	"github.com/gabriel-vasile/mimetype"
	"github.com/gofrs/uuid"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/x/blobstorage"

	artifactpb "github.com/instill-ai/protogen-go/artifact/artifact/v1alpha"
)

// UploadBlobDataAndReplaceWithURLsParams is the parameters for the UploadBlobDataAndReplaceWithURLs function.
type UploadBlobDataAndReplaceWithURLsParams struct {
	// NamespaceID is the namespace ID.
	NamespaceID string
	// RequesterUID is the requester UID.
	RequesterUID uuid.UUID
	// DataStructs are the data structs to be uploaded.
	DataStructs []*structpb.Struct
	// Logger is the logger.
	Logger *zap.Logger
	// ArtifactClient is the artifact public service client.
	ArtifactClient *artifactpb.ArtifactPublicServiceClient
}

// UploadBlobDataAndReplaceWithURL uploads the unstructured data in the structs to minio and replaces the data with the URL.
// Before calling this function, ctx should have been set with the request metadata.
func UploadBlobDataAndReplaceWithURLs(ctx context.Context, params UploadBlobDataAndReplaceWithURLsParams) ([]*structpb.Struct, error) {
	updatedDataStructs := make([]*structpb.Struct, len(params.DataStructs))
	for i, dataStruct := range params.DataStructs {
		updatedDataStruct, err := uploadBlobDataAndReplaceWithURL(ctx, uploadBlobDataAndReplaceWithURLParams{
			namespaceID:    params.NamespaceID,
			requesterUID:   params.RequesterUID,
			dataStruct:     dataStruct,
			logger:         params.Logger,
			artifactClient: params.ArtifactClient,
		})
		if err != nil {
			// Note: we don't want to fail the whole process if one of the data structs fails to upload.
			updatedDataStructs[i] = dataStruct
		} else {
			updatedDataStructs[i] = updatedDataStruct
		}
	}
	return updatedDataStructs, nil
}

type uploadBlobDataAndReplaceWithURLParams struct {
	namespaceID    string
	requesterUID   uuid.UUID
	dataStruct     *structpb.Struct
	logger         *zap.Logger
	artifactClient *artifactpb.ArtifactPublicServiceClient
}

func uploadBlobDataAndReplaceWithURL(ctx context.Context, params uploadBlobDataAndReplaceWithURLParams) (*structpb.Struct, error) {

	dataStruct := params.dataStruct
	for key, value := range dataStruct.GetFields() {
		updatedValue, err := processValue(ctx, params, value)
		if err == nil {
			dataStruct.GetFields()[key] = updatedValue
		}
	}

	return dataStruct, nil
}

func processValue(ctx context.Context, params uploadBlobDataAndReplaceWithURLParams, value *structpb.Value) (*structpb.Value, error) {

	switch v := value.GetKind().(type) {
	case *structpb.Value_StringValue:
		if isUnstructuredData(v.StringValue) {
			downloadURL, err := UploadBlobAndGetDownloadURL(ctx, UploadBlobAndGetDownloadURLParams{
				NamespaceID:    params.namespaceID,
				RequesterUID:   params.requesterUID,
				Data:           v.StringValue,
				Logger:         params.logger,
				ArtifactClient: params.artifactClient,
			})
			if err != nil {
				return nil, err
			}
			return &structpb.Value{Kind: &structpb.Value_StringValue{StringValue: downloadURL}}, nil
		}
	case *structpb.Value_ListValue:
		listValue := v.ListValue
		updatedListValue, err := processList(ctx, params, listValue)
		if err == nil {
			return &structpb.Value{Kind: &structpb.Value_ListValue{ListValue: updatedListValue}}, nil
		}
	case *structpb.Value_StructValue:
		for _, item := range v.StructValue.GetFields() {
			structData := item.GetStructValue()
			newParams := params
			newParams.dataStruct = structData
			updatedStructData, err := uploadBlobDataAndReplaceWithURL(ctx, newParams)
			// Note: we don't want to fail the whole process if one of the data structs fails to upload.
			if err == nil {
				return &structpb.Value{Kind: &structpb.Value_StructValue{StructValue: updatedStructData}}, nil
			}
		}
	}

	return value, nil
}

func processList(ctx context.Context, params uploadBlobDataAndReplaceWithURLParams, list *structpb.ListValue) (*structpb.ListValue, error) {

	for i, item := range list.Values {
		updatedItem, err := processValue(ctx, params, item)
		if err == nil {
			list.Values[i] = updatedItem
		}
	}

	return list, nil
}

func isUnstructuredData(data string) bool {
	return strings.HasPrefix(data, "data:") && strings.Contains(data, ";base64,")
}

// UploadBlobAndGetDownloadURLParams is the parameters for the UploadBlobAndGetDownloadURL function.
type UploadBlobAndGetDownloadURLParams struct {
	// NamespaceID is the namespace ID.
	NamespaceID string
	// RequesterUID is the requester UID.
	RequesterUID uuid.UUID
	// Data is the data to be uploaded.
	Data string
	// Logger is the logger.
	Logger *zap.Logger
	// ArtifactClient is the artifact public service client.
	ArtifactClient *artifactpb.ArtifactPublicServiceClient
}

// UploadBlobAndGetDownloadURL uploads the unstructured data to minio and returns the public download URL.
func UploadBlobAndGetDownloadURL(ctx context.Context, params UploadBlobAndGetDownloadURLParams) (string, error) {
	mimeType, err := getMimeType(params.Data)
	if err != nil {
		return "", fmt.Errorf("get mime type: %w", err)
	}

	uid, err := uuid.NewV4()

	if err != nil {
		return "", fmt.Errorf("generate uuid: %w", err)
	}
	objectName := fmt.Sprintf("%s/%s%s", params.RequesterUID.String(), uid.String(), getFileExtension(mimeType))

	artifactClient := *params.ArtifactClient

	// TODO: We will need to add the expiry days for the blob data.
	// This will be addressed in ins-6857
	resp, err := artifactClient.GetObjectUploadURL(ctx, &artifactpb.GetObjectUploadURLRequest{
		NamespaceId:      params.NamespaceID,
		ObjectName:       objectName,
		ObjectExpireDays: 0,
	})

	if err != nil {
		return "", fmt.Errorf("get upload url: %w", err)
	}

	uploadURL := resp.GetUploadUrl()
	data := removePrefix(params.Data)
	b, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return "", fmt.Errorf("decode base64 string: %w", err)
	}

	err = UploadBlobData(ctx, uploadURL, mimeType, b, params.Logger)
	if err != nil {
		return "", fmt.Errorf("upload blob data: %w", err)
	}

	respDownloadURL, err := artifactClient.GetObjectDownloadURL(ctx, &artifactpb.GetObjectDownloadURLRequest{
		NamespaceId: params.NamespaceID,
		ObjectUid:   resp.GetObject().GetUid(),
	})
	if err != nil {
		return "", fmt.Errorf("get object download url: %w", err)
	}

	return respDownloadURL.GetDownloadUrl(), nil
}

// TODO: make it unexported when everything is migrated to utils package.
// UploadBlobData uploads the blob data to the given upload URL.
func UploadBlobData(ctx context.Context, uploadURL string, fileContentType string, fileBytes []byte, logger *zap.Logger) error {
	if uploadURL == "" {
		return fmt.Errorf("empty upload URL provided")
	}

	parsedURL, err := url.Parse(uploadURL)
	if err != nil {
		return fmt.Errorf("parsing upload URL: %w", err)
	}
	if config.Config.APIGateway.TLSEnabled {
		parsedURL.Scheme = "https"
	} else {
		parsedURL.Scheme = "http"
	}
	parsedURL.Host = fmt.Sprintf("%s:%d", config.Config.APIGateway.Host, config.Config.APIGateway.PublicPort)
	fullURL := parsedURL.String()

	err = blobstorage.UploadFile(ctx, logger, fullURL, fileBytes, fileContentType)

	if err != nil {
		return fmt.Errorf("uploading blob: %w", err)
	}

	return nil
}

func getMimeType(data string) (string, error) {
	var mimeType string
	if strings.HasPrefix(data, "data:") {
		contentType := strings.TrimPrefix(data, "data:")
		parts := strings.SplitN(contentType, ";", 2)
		if len(parts) == 0 {
			return "", fmt.Errorf("invalid data url")
		}
		mimeType = parts[0]
	} else {
		b, err := base64.StdEncoding.DecodeString(data)
		if err != nil {
			return "", fmt.Errorf("decode base64 string: %w", err)
		}
		mimeType = strings.Split(mimetype.Detect(b).String(), ";")[0]

	}
	return mimeType, nil
}

func getFileExtension(mimeType string) string {
	ext, err := mime.ExtensionsByType(mimeType)
	if err != nil {
		return ""
	}
	if len(ext) == 0 {
		return ""
	}
	return ext[0]
}

func removePrefix(data string) string {
	if strings.HasPrefix(data, "data:") {
		parts := strings.SplitN(data, ",", 2)
		if len(parts) == 0 {
			return ""
		}
		return parts[1]
	}
	return data
}
