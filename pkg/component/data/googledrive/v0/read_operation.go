package googledrive

import (
	"context"

	"google.golang.org/protobuf/types/known/structpb"
)

type readFileInput struct {
	FileID string `json:"file-id"`
}

type readFileOutput struct {
	File file `json:"file"`
}

type file struct {
	Name         string `json:"name"`
	Content      string `json:"content"`
	CreatedTime  string `json:"created-time"`
	ModifiedTime string `json:"modified-time"`
	Size         int64  `json:"size"`
	MimeType     string `json:"mime-type"`
	Md5Checksum  string `json:"md5-checksum,omitempty"`
	Version      string `json:"version"`
}

func (e *execution) readFile(ctx context.Context, input *structpb.Struct) (*structpb.Struct, error) {
	return nil, nil
}

type readFilesInput struct {
	FileNames   []string `json:"file-names"`
	ReadContent bool     `json:"read-content"`
}

type readFilesOutput struct {
	Files []file `json:"files"`
}

func (e *execution) readFiles(ctx context.Context, input *structpb.Struct) (*structpb.Struct, error) {
	return nil, nil
}

type readDriveInput struct {
	OrderBy string `json:"order-by"`
	Limit   int    `json:"limit"`
}

type readDriveOutput struct {
	Files []file `json:"files"`
}

func (e *execution) readDrive(ctx context.Context, input *structpb.Struct) (*structpb.Struct, error) {
	return nil, nil
}
