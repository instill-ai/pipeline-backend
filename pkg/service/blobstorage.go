// TODO:
// We should arrange the logic for blob storage in the pipeline-backend.
// Now, we use blob storage in worker and service. The logic are close but not the same.
// We should refactor the logic to make it more compact and easier to maintain for worker and service.
// This will be addressed in ins-7091

package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"mime"
	"strings"
	"time"

	"github.com/gabriel-vasile/mimetype"
	"google.golang.org/grpc/metadata"

	"github.com/instill-ai/pipeline-backend/pkg/recipe"
	"github.com/instill-ai/pipeline-backend/pkg/resource"
	"github.com/instill-ai/pipeline-backend/pkg/utils"

	artifactpb "github.com/instill-ai/protogen-go/artifact/artifact/v1alpha"
	miniox "github.com/instill-ai/x/minio"
)

func (s *service) uploadBlobAndGetDownloadURL(ctx context.Context, data string, ns resource.Namespace, expiryRule miniox.ExpiryRule) (string, error) {
	mimeType, err := getMimeType(data)
	if err != nil {
		return "", fmt.Errorf("get mime type: %w", err)
	}
	artifactClient := s.artifactPublicServiceClient

	vars, err := recipe.GenerateSystemVariables(ctx, recipe.SystemVariables{})

	if err != nil {
		return "", fmt.Errorf("generate system variables: %w", err)
	}

	ctx = metadata.NewOutgoingContext(ctx, utils.GetRequestMetadata(vars))

	timestamp := time.Now().Format(time.RFC3339)
	objectName := fmt.Sprintf("%s/%s%s", ns.NsUID.String(), timestamp, getFileExtension(mimeType))

	resp, err := artifactClient.GetObjectUploadURL(ctx, &artifactpb.GetObjectUploadURLRequest{
		NamespaceId:      ns.NsID,
		ObjectName:       objectName,
		ObjectExpireDays: int32(expiryRule.ExpirationDays),
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

	err = utils.UploadBlobData(ctx, uploadURL, mimeType, b, s.log)
	if err != nil {
		return "", fmt.Errorf("upload blob data: %w", err)
	}

	respDownloadURL, err := artifactClient.GetObjectDownloadURL(ctx, &artifactpb.GetObjectDownloadURLRequest{
		NamespaceId: ns.NsID,
		ObjectUid:   resp.GetObject().GetUid(),
	})
	if err != nil {
		return "", fmt.Errorf("get object download url: %w", err)
	}

	return respDownloadURL.GetDownloadUrl(), nil
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
