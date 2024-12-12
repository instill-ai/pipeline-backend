// TODO:
// We should arrange the logic for blob storage in the pipeline-backend.
// Now, we use blob storage in worker and service. The logic are close but not the same.
// We should refactor the logic to make it more compact and easier to maintain for worker and service.
// This will be addressed in ins-7091

package worker

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"

	"github.com/instill-ai/pipeline-backend/pkg/data"
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
	"github.com/instill-ai/pipeline-backend/pkg/logger"
	"github.com/instill-ai/pipeline-backend/pkg/utils"

	artifactpb "github.com/instill-ai/protogen-go/artifact/artifact/v1alpha"
)

func (w *worker) uploadFileAndReplaceWithURL(ctx context.Context, param *ComponentActivityParam, value *format.Value) format.Value {
	logger, _ := logger.GetZapLogger(ctx)
	if value == nil {
		return nil
	}
	switch v := (*value).(type) {
	case format.File:
		downloadURL, err := w.uploadBlobDataAndGetDownloadURL(ctx, param, v)
		if err != nil || downloadURL == "" {
			logger.Warn("uploading blob data", zap.Error(err))
			return v
		}
		return data.NewString(downloadURL)
	case data.Array:
		newArray := make(data.Array, len(v))
		for i, item := range v {
			newArray[i] = w.uploadFileAndReplaceWithURL(ctx, param, &item)
		}
		return newArray
	case data.Map:
		newMap := make(data.Map)
		for k, v := range v {
			newMap[k] = w.uploadFileAndReplaceWithURL(ctx, param, &v)
		}
		return newMap
	default:
		return v
	}
}

func (w *worker) uploadBlobDataAndGetDownloadURL(ctx context.Context, param *ComponentActivityParam, value format.File) (string, error) {
	artifactClient := w.artifactPublicServiceClient
	requesterID := param.SystemVariables.PipelineRequesterID

	sysVarJSON := utils.StructToMap(param.SystemVariables, "json")

	ctx = metadata.NewOutgoingContext(ctx, utils.GetRequestMetadata(sysVarJSON))

	objectName := fmt.Sprintf("%s/%s", requesterID, value.Filename())

	resp, err := artifactClient.GetObjectUploadURL(ctx, &artifactpb.GetObjectUploadURLRequest{
		NamespaceId:      requesterID,
		ObjectName:       objectName,
		ObjectExpireDays: int32(param.SystemVariables.ExpiryRule.ExpirationDays),
	})

	if err != nil {
		return "", fmt.Errorf("get upload url: %w", err)
	}

	uploadURL := resp.GetUploadUrl()

	fileBytes, err := value.Binary()
	if err != nil {
		return "", fmt.Errorf("getting file bytes: %w", err)
	}

	err = utils.UploadBlobData(ctx, uploadURL, value.ContentType().String(), fileBytes.ByteArray(), w.log)
	if err != nil {
		return "", fmt.Errorf("upload blob data: %w", err)
	}

	respDownloadURL, err := artifactClient.GetObjectDownloadURL(ctx, &artifactpb.GetObjectDownloadURLRequest{
		NamespaceId: requesterID,
		ObjectUid:   resp.GetObject().GetUid(),
	})
	if err != nil {
		return "", fmt.Errorf("get object download url: %w", err)
	}

	return respDownloadURL.GetDownloadUrl(), nil
}
