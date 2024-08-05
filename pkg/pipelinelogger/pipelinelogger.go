package pipelinelogger

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/instill-ai/pipeline-backend/config"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/minio-go/v7/pkg/lifecycle"

	"github.com/gofrs/uuid"
	"github.com/minio/minio-go/v7"
	"gorm.io/gorm"

	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
)

type PipelineLogger struct {
	db          *gorm.DB
	minioClient *minio.Client
}

// NewPipelineLogger creates a new instance of PipelineLogger
func NewPipelineLogger(db *gorm.DB, minioClient *minio.Client) *PipelineLogger {
	return &PipelineLogger{
		db:          db,
		minioClient: minioClient,
	}
}

// LogPipelineRun creates or updates a PipelineRun in the database.
func (l *PipelineLogger) LogPipelineRun(ctx context.Context, pipelineRun *datamodel.PipelineRun) error {
	return l.db.Save(pipelineRun).Error
}

// LogComponentRun creates or updates a ComponentRun in the database.
func (l *PipelineLogger) LogComponentRun(ctx context.Context, componentRun *datamodel.ComponentRun) error {
	return l.db.Save(componentRun).Error
}

// GetPaginatedComponentsByTriggerUID retrieves a paginated list of ComponentRun by PipelineTriggerUID from the database.
func (l *PipelineLogger) GetPaginatedComponentsByTriggerUID(ctx context.Context, pipelineTriggerUID uuid.UUID, page int, pageSize int, components *[]datamodel.ComponentRun) error {
	offset := (page - 1) * pageSize
	return l.db.Where("pipeline_trigger_uid = ?", pipelineTriggerUID).Order("started_time asc").Offset(offset).Limit(pageSize).Find(components).Error
}

func checkPermission(userID, pipelineRunID string) bool {
	// TODO tillknuesting: Implement check if the user has permission to access the pipeline run.
	// Steps:

	// 1. Get the user's accessible namespaces
	//    - Query the user table to get the user's personal namespace
	//    - Query the organization_member table to get organizations the user belongs to

	// 2. Query the pipeline table to get all pipeline UIDs associated with these namespaces
	//    - Use a WHERE clause like:
	//      WHERE (namespace_type = 'users' AND namespace_id = [user's personal namespace])
	//      OR (namespace_type = 'organizations' AND namespace_id IN [list of user's org namespaces])

	// 3. Retrieve the pipeline run details using pipelineRunID
	//    - This should include the associated pipeline UID

	// 4. Check if the pipeline UID from the run is in the list of accessible pipeline UIDs
	//    - If yes, the user has permission
	//    - If no, check if the pipeline is public (may need an additional 'is_public' field in the pipeline table)

	// 5. If the pipeline is private and not in the user's accessible list:
	//    - Check for any sharing permissions (might need an additional table for this)

	return true
}

// GetPaginatedPipelineRunsWithPermissions retrieves pipeline runs with pagination and permissions
func (l *PipelineLogger) GetPaginatedPipelineRunsWithPermissions(ctx context.Context, userID, namespace string, page, pageSize int) ([]datamodel.PipelineRun, int64, error) {
	var pipelineRuns []datamodel.PipelineRun
	var totalRows int64

	offset := (page - 1) * pageSize

	// Count total rows with permissions
	err := l.db.Model(&datamodel.PipelineRun{}).
		Where("triggered_by = ? OR namespace = ?", userID, namespace).
		Count(&totalRows).Error
	if err != nil {
		return nil, 0, err
	}

	// Retrieve paginated results with permissions
	err = l.db.Preload("Components").
		Where("triggered_by = ? OR namespace = ?", userID, namespace).
		Offset(offset).Limit(pageSize).
		Find(&pipelineRuns).Error
	if err != nil {
		return nil, 0, err
	}

	// Anonymize data for non-owners
	for idx, run := range pipelineRuns {
		if run.TriggeredBy != userID {
			pipelineRuns[idx].TriggeredBy = "Anonymous"
		}
	}

	return pipelineRuns, totalRows, nil
}

// GetPaginatedComponentRunsByPipelineRunIDWithPermissions retrieves component runs by pipeline run ID with pagination and permissions
func (l *PipelineLogger) GetPaginatedComponentRunsByPipelineRunIDWithPermissions(ctx context.Context, userID, pipelineRunID string, page, pageSize int) ([]datamodel.ComponentRun, int64, error) {
	var componentRuns []datamodel.ComponentRun
	var totalRows int64

	offset := (page - 1) * pageSize

	// Retrieve the parent pipeline run to check permissions
	var pipelineRun datamodel.PipelineRun
	err := l.db.First(&pipelineRun, "pipeline_trigger_uid = ?", pipelineRunID).Error
	if err != nil {
		return nil, 0, err
	}

	// Check if the user has access to the pipeline run
	if !checkPermission(userID, pipelineRunID) {
		return nil, 0, fmt.Errorf("access denied")
	}

	// Count total rows
	err = l.db.Model(&datamodel.ComponentRun{}).
		Where("pipeline_trigger_uid = ?", pipelineRunID).
		Count(&totalRows).Error
	if err != nil {
		return nil, 0, err
	}

	// Retrieve paginated results
	err = l.db.Where("pipeline_trigger_uid = ?", pipelineRunID).
		Offset(offset).Limit(pageSize).
		Find(&componentRuns).Error
	if err != nil {
		return nil, 0, err
	}

	return componentRuns, totalRows, nil
}

func (l *PipelineLogger) GeneratePresignedURL(ctx context.Context, bucketName, objectName string) (string, error) {
	expiry := time.Duration(time.Hour * 24 * 7)
	presignedURL, err := l.minioClient.PresignedGetObject(context.Background(), bucketName, objectName, expiry, nil)
	if err != nil {
		return "", err
	}
	return presignedURL.String(), nil
}

func (l *PipelineLogger) createBucketIfNotExists(ctx context.Context, bucketName string) error {
	exists, err := l.minioClient.BucketExists(context.Background(), bucketName)
	if err != nil {
		return err
	}
	if !exists {
		err = l.minioClient.MakeBucket(context.Background(), bucketName, minio.MakeBucketOptions{})
		if err != nil {
			return err
		}
	}
	return nil
}

func (l *PipelineLogger) setLifecyclePolicy(ctx context.Context, bucketName string, days int) error {
	newConfiguration := lifecycle.NewConfiguration()
	newConfiguration.Rules = []lifecycle.Rule{
		{
			ID:     "expire-bucket-objects",
			Status: "Enabled",
			Expiration: lifecycle.Expiration{
				Days: lifecycle.ExpirationDays(days),
			},
		},
	}

	err := l.minioClient.SetBucketLifecycle(context.Background(), bucketName, newConfiguration)
	if err != nil {
		return fmt.Errorf("failed to set bucket lifecycle: %v", err)
	}

	return nil
}

// CreateMinioClient initializes and returns a MinIO client.
func CreateMinioClient(config config.MinioConfig) (*minio.Client, error) {
	// IPv4 only.
	endpoint := config.Host + ":" + config.Port

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(config.RootUser, config.RootPwd, ""),
		Secure: config.Secure,
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to create minio client: %v", err))
		return nil, err
	}

	return client, nil
}

func (l *PipelineLogger) UploadToMinio(objectName string, data []byte, contentType string, bucketName string) (string, minio.ObjectInfo, error) {
	ctx := context.Background()
	// Create the bucket if it doesn't exist
	err := l.createBucketIfNotExists(ctx, bucketName)
	if err != nil {
		return "", minio.ObjectInfo{}, err
	}

	// Set lifecycle policy to expire objects after 30 days
	err = l.setLifecyclePolicy(ctx, bucketName, 7)
	if err != nil {
		return "", minio.ObjectInfo{}, err
	}

	// Upload the JSON to MinIO
	_, err = l.minioClient.PutObject(ctx, bucketName, objectName,
		bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return "", minio.ObjectInfo{}, err
	}

	// Get the object stat (metadata)
	stat, err := l.minioClient.StatObject(ctx, bucketName, objectName, minio.StatObjectOptions{})
	if err != nil {
		return "", minio.ObjectInfo{}, err
	}

	// Generate the presigned URL
	presignedURL, err := l.GeneratePresignedURL(ctx, bucketName, objectName)
	if err != nil {
		return "", minio.ObjectInfo{}, err
	}
	return presignedURL, stat, nil
}
