package minio

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"sync"
	"time"

	"github.com/gofrs/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/minio-go/v7/pkg/lifecycle"
	"go.uber.org/zap"

	"github.com/instill-ai/pipeline-backend/config"

	log "github.com/instill-ai/pipeline-backend/pkg/logger"
)

type MinioI interface {
	UploadFile(ctx context.Context, filePath string, fileContent any, fileMimeType string) (url string, objectInfo *minio.ObjectInfo, err error)
	UploadFileBytes(ctx context.Context, filePath string, fileBytes []byte, fileMimeType string) (url string, objectInfo *minio.ObjectInfo, err error)
	DeleteFile(ctx context.Context, filePath string) (err error)
	GetFile(ctx context.Context, filePath string) ([]byte, error)
	GetFilesByPaths(ctx context.Context, filePaths []string) ([]FileContent, error)
}

const Location = "us-east-1"

type Minio struct {
	client *minio.Client
	bucket string
}

func NewMinioClientAndInitBucket(ctx context.Context, cfg *config.MinioConfig) (*Minio, error) {
	logger, err := log.GetZapLogger(context.Background())
	if err != nil {
		return nil, err
	}
	logger.Info("Initializing Minio client and bucket...")

	endpoint := net.JoinHostPort(cfg.Host, cfg.Port)
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.RootUser, cfg.RootPwd, ""),
		Secure: cfg.Secure,
	})
	if err != nil {
		logger.Error("cannot connect to minio",
			zap.String("host:port", cfg.Host+":"+cfg.Port),
			zap.String("user", cfg.RootUser),
			zap.String("pwd", cfg.RootPwd), zap.Error(err))
		return nil, err
	}

	exists, err := client.BucketExists(ctx, cfg.BucketName)
	if err != nil {
		logger.Error("failed in checking BucketExists", zap.Error(err))
		return nil, err
	}
	if exists {
		logger.Info("Bucket already exists", zap.String("bucket", cfg.BucketName))
		return &Minio{client: client, bucket: cfg.BucketName}, nil
	}

	if err = client.MakeBucket(ctx, cfg.BucketName, minio.MakeBucketOptions{
		Region: Location,
	}); err != nil {
		logger.Error("creating Bucket failed", zap.Error(err))
		return nil, err
	}
	logger.Info("Successfully created bucket", zap.String("bucket", cfg.BucketName))

	lccfg := lifecycle.NewConfiguration()
	lccfg.Rules = []lifecycle.Rule{
		{
			ID:     "expire-bucket-objects",
			Status: "Enabled",
			Expiration: lifecycle.Expiration{
				Days: lifecycle.ExpirationDays(30),
			},
		},
	}
	err = client.SetBucketLifecycle(ctx, cfg.BucketName, lccfg)
	if err != nil {
		logger.Error("setting Bucket lifecycle failed", zap.Error(err))
		return nil, err
	}

	return &Minio{client: client, bucket: cfg.BucketName}, nil
}

func (m *Minio) UploadFile(ctx context.Context, filePath string, fileContent any, fileMimeType string) (url string, objectInfo *minio.ObjectInfo, err error) {
	jsonData, _ := json.Marshal(fileContent)
	return m.UploadFileBytes(ctx, filePath, jsonData, fileMimeType)
}

func (m *Minio) UploadFileBytes(ctx context.Context, filePath string, fileBytes []byte, fileMimeType string) (url string, objectInfo *minio.ObjectInfo, err error) {
	logger, err := log.GetZapLogger(ctx)
	if err != nil {
		return "", nil, err
	}
	reader := bytes.NewReader(fileBytes)

	// Create the file path with folder structure
	_, err = m.client.PutObject(ctx, m.bucket, filePath, reader, int64(len(fileBytes)), minio.PutObjectOptions{ContentType: fileMimeType})
	if err != nil {
		logger.Error("Failed to upload file to MinIO", zap.Error(err))
		return "", nil, err
	}

	// Get the object stat (metadata)
	stat, err := m.client.StatObject(ctx, m.bucket, filePath, minio.StatObjectOptions{})
	if err != nil {
		return "", nil, err
	}

	// Generate the presigned URL
	presignedURL, err := m.client.PresignedGetObject(ctx, m.bucket, filePath, time.Hour*24*7, nil)
	if err != nil {
		return "", nil, err
	}

	return presignedURL.String(), &stat, nil
}

// DeleteFile delete the file from minio
func (m *Minio) DeleteFile(ctx context.Context, filePathName string) (err error) {
	logger, err := log.GetZapLogger(ctx)
	if err != nil {
		return err
	}
	// Delete the file from MinIO
	err = m.client.RemoveObject(ctx, m.bucket, filePathName, minio.RemoveObjectOptions{})
	if err != nil {
		logger.Error("Failed to delete file from MinIO", zap.Error(err))
		return err
	}
	return nil
}

func (m *Minio) GetFile(ctx context.Context, filePathName string) ([]byte, error) {
	logger, err := log.GetZapLogger(ctx)
	if err != nil {
		return nil, err
	}

	// Get the object using the client
	object, err := m.client.GetObject(ctx, m.bucket, filePathName, minio.GetObjectOptions{})
	if err != nil {
		logger.Error("Failed to get file from MinIO", zap.Error(err))
		return nil, err
	}
	defer object.Close()

	// Read the object's content
	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(object)
	if err != nil {
		logger.Error("Failed to read file from MinIO", zap.Error(err))
		return nil, err
	}

	return buf.Bytes(), nil
}

// FileContent represents a file and its content
type FileContent struct {
	Name    string
	Content []byte
}

// GetFilesByPaths GetFiles retrieves the contents of specified files from MinIO
func (m *Minio) GetFilesByPaths(ctx context.Context, filePaths []string) ([]FileContent, error) {
	logger, err := log.GetZapLogger(ctx)
	if err != nil {
		return nil, err
	}

	var wg sync.WaitGroup
	fileCount := len(filePaths)

	errCh := make(chan error, fileCount)
	resultCh := make(chan FileContent, fileCount)

	for _, path := range filePaths {
		wg.Add(1)
		go func(filePath string) {
			defer wg.Done()

			obj, err := m.client.GetObject(ctx, m.bucket, filePath, minio.GetObjectOptions{})
			if err != nil {
				logger.Error("Failed to get object from MinIO", zap.String("path", filePath), zap.Error(err))
				errCh <- err
				return
			}
			defer obj.Close()

			var buffer bytes.Buffer
			_, err = io.Copy(&buffer, obj)
			if err != nil {
				logger.Error("Failed to read object content", zap.String("path", filePath), zap.Error(err))
				errCh <- err
				return
			}

			fileContent := FileContent{
				Name:    filePath,
				Content: buffer.Bytes(),
			}
			resultCh <- fileContent
		}(path)
	}

	wg.Wait()

	close(errCh)
	close(resultCh)

	var errs []error
	for err = range errCh {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}

	files := make([]FileContent, 0)
	for fileContent := range resultCh {
		files = append(files, fileContent)
	}

	return files, nil
}

func GenerateInputRefID() string {
	referenceUID, _ := uuid.NewV4()
	return "pipeline-runs/input/" + referenceUID.String()
}

func GenerateOutputRefID() string {
	referenceUID, _ := uuid.NewV4()
	return "pipeline-runs/output/" + referenceUID.String()
}
