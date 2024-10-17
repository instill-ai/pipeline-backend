package googledrive

import (
	"context"

	"google.golang.org/protobuf/types/known/structpb"
)

type readFileInput struct {
	SharedLink string `json:"shared-link"`
}

type readFileOutput struct {
	File file `json:"file"`
}

type file struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Content        string `json:"content"`
	CreatedTime    string `json:"created-time"`
	ModifiedTime   string `json:"modified-time"`
	Size           int64  `json:"size"`
	MimeType       string `json:"mime-type"`
	Md5Checksum    string `json:"md5-checksum,omitempty"`
	Version        string `json:"version"`
	WebViewLink    string `json:"web-view-link"`
	WebContentLink string `json:"web-content-link,omitempty"`
}

func (e *execution) readFile(ctx context.Context, input *structpb.Struct) (*structpb.Struct, error) {
	return nil, nil
}

type readFolderInput struct {
	SharedLink  string `json:"shared-link"`
	ReadContent bool   `json:"read-content"`
}

type readFolderOutput struct {
	Files []file `json:"files"`
}

func (e *execution) readFolder(ctx context.Context, input *structpb.Struct) (*structpb.Struct, error) {
	return nil, nil
}
