// TODO:
// We should arrange the logic for blob storage in the pipeline-backend.
// Now, we use blob storage in worker and service. The logic are close but not the same.
// We should refactor the logic to make it more compact and easier to maintain for worker and service.
// This will be addressed in ins-7091

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
	miniox "github.com/instill-ai/x/minio"
)

// UploadBlobParams contains the information and dependencies to upload blob
// data owned by a namespace and obtain a download URL.
type UploadBlobParams struct {
	NamespaceID    string
	NamespaceUID   uuid.UUID
	ExpiryRule     miniox.ExpiryRule
	Logger         *zap.Logger
	ArtifactClient *artifactpb.ArtifactPublicServiceClient
}

// UploadBlobDataAndReplaceWithURLs uploads the unstructured data in the
// structs to minio and replaces the data with the URL.
// Before calling this function, ctx should have been set with the request
// metadata.
func UploadBlobDataAndReplaceWithURLs(ctx context.Context, dataStructs []*structpb.Struct, params UploadBlobParams) ([]*structpb.Struct, error) {
	updatedDataStructs := make([]*structpb.Struct, len(dataStructs))
	for i, dataStruct := range dataStructs {
		updatedDataStruct, err := uploadBlobDataAndReplaceWithURL(ctx, dataStruct, params)
		if err != nil {
			// Note: we don't want to fail the whole process if one of the data structs fails to upload.
			updatedDataStructs[i] = dataStruct
		} else {
			updatedDataStructs[i] = updatedDataStruct
		}
	}
	return updatedDataStructs, nil
}

func uploadBlobDataAndReplaceWithURL(ctx context.Context, dataStruct *structpb.Struct, params UploadBlobParams) (*structpb.Struct, error) {
	for key, value := range dataStruct.GetFields() {
		updatedValue, err := processValue(ctx, params, value)
		if err == nil {
			dataStruct.GetFields()[key] = updatedValue
		}
	}

	return dataStruct, nil
}

func processValue(ctx context.Context, params UploadBlobParams, value *structpb.Value) (*structpb.Value, error) {
	switch v := value.GetKind().(type) {
	case *structpb.Value_StringValue:
		if isUnstructuredData(v.StringValue) {
			downloadURL, err := uploadBlobAndGetDownloadURL(ctx, v.StringValue, params)
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
			updatedStructData, err := uploadBlobDataAndReplaceWithURL(ctx, structData, params)
			// Note: we don't want to fail the whole process if one of the data structs fails to upload.
			if err == nil {
				return &structpb.Value{Kind: &structpb.Value_StructValue{StructValue: updatedStructData}}, nil
			}
		}
	}

	return value, nil
}

func processList(ctx context.Context, params UploadBlobParams, list *structpb.ListValue) (*structpb.ListValue, error) {
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

func uploadBlobAndGetDownloadURL(ctx context.Context, data string, params UploadBlobParams) (string, error) {
	mimeType, err := getMimeType(data)
	if err != nil {
		return "", fmt.Errorf("get mime type: %w", err)
	}

	uid, err := uuid.NewV4()
	if err != nil {
		return "", fmt.Errorf("generate uuid: %w", err)
	}

	objectName := fmt.Sprintf("%s/%s%s", params.NamespaceUID.String(), uid.String(), getFileExtension(mimeType))

	artifactClient := *params.ArtifactClient
	resp, err := artifactClient.GetObjectUploadURL(ctx, &artifactpb.GetObjectUploadURLRequest{
		NamespaceId:      params.NamespaceID,
		ObjectName:       objectName,
		ObjectExpireDays: int32(params.ExpiryRule.ExpirationDays),
	})

	if err != nil {
		return "", fmt.Errorf("get upload url: %w", err)
	}

	uploadURL := resp.GetUploadUrl()
	data = removePrefix(data)
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
