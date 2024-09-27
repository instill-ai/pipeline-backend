package googlecloudstorage

import (
	"context"
	"fmt"
	"io"
	"strings"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
)

type ReadInput struct {
	BucketName               string
	Delimiter                string
	Prefix                   string
	Versions                 bool
	StartOffset              string
	EndOffset                string
	IncludeTrailingDelimiter bool
	MatchGlob                string
	IncludeFoldersAsPrefixes bool
}

type ReadOutput struct {
	TextObjects     []TextObject     `json:"text-objects"`
	ImageObjects    []ImageObject    `json:"image-objects"`
	DocumentObjects []DocumentObject `json:"document-objects"`
	AudioObjects    []AudioObject    `json:"audio-objects"`
	VideoObjects    []VideoObject    `json:"video-objects"`
}

type TextObject struct {
	Data       string     `json:"data"`
	Attributes Attributes `json:"attributes"`
}

type ImageObject struct {
	Data       string     `json:"data"`
	Attributes Attributes `json:"attributes"`
}

type DocumentObject struct {
	Data       string     `json:"data"`
	Attributes Attributes `json:"attributes"`
}

type AudioObject struct {
	Data       string     `json:"data"`
	Attributes Attributes `json:"attributes"`
}

type VideoObject struct {
	Data       string     `json:"data"`
	Attributes Attributes `json:"attributes"`
}

type Attributes struct {
	Name               string            `json:"name"`
	ContentType        string            `json:"content-type"`
	ContentLanguage    string            `json:"content-language"`
	Owner              string            `json:"owner"`
	Size               int64             `json:"size"`
	ContentEncoding    string            `json:"content-encoding"`
	ContentDisposition string            `json:"content-disposition"`
	MD5                []byte            `json:"md5"`
	MediaLink          string            `json:"media-link"`
	Metadata           map[string]string `json:"metadata"`
	StorageClass       string            `json:"storage-class"`
}

func readObjects(input ReadInput, client *storage.Client, ctx context.Context) (ReadOutput, error) {
	bucketName := input.BucketName
	query := &storage.Query{
		Delimiter:                input.Delimiter,
		Prefix:                   input.Prefix,
		Versions:                 input.Versions,
		StartOffset:              input.StartOffset,
		EndOffset:                input.EndOffset,
		IncludeTrailingDelimiter: input.IncludeTrailingDelimiter,
		MatchGlob:                input.MatchGlob,
		IncludeFoldersAsPrefixes: input.IncludeFoldersAsPrefixes,
	}

	it := client.Bucket(bucketName).Objects(ctx, query)

	output := ReadOutput{
		TextObjects:     []TextObject{},
		ImageObjects:    []ImageObject{},
		DocumentObjects: []DocumentObject{},
		AudioObjects:    []AudioObject{},
		VideoObjects:    []VideoObject{},
	}

	for {
		attrs, err := it.Next()

		if err == iterator.Done {
			break
		}

		rc, err := client.Bucket(bucketName).Object(attrs.Name).NewReader(ctx)

		if err != nil {
			return output, fmt.Errorf("readObjects: %v", err)
		}
		defer rc.Close()

		b, err := io.ReadAll(rc)
		if err != nil {
			return output, fmt.Errorf("readObjects: %v", err)
		}

		attribute := Attributes{
			Name:               attrs.Name,
			ContentType:        attrs.ContentType,
			ContentLanguage:    attrs.ContentLanguage,
			Owner:              attrs.Owner,
			Size:               attrs.Size,
			ContentEncoding:    attrs.ContentEncoding,
			ContentDisposition: attrs.ContentDisposition,
			MD5:                attrs.MD5,
			MediaLink:          attrs.MediaLink,
			Metadata:           attrs.Metadata,
			StorageClass:       attrs.StorageClass,
		}

		if attrs.Metadata == nil {
			attribute.Metadata = map[string]string{}
		}

		if strings.Contains(attrs.ContentType, "text") {
			textObject := TextObject{
				Data:       string(b),
				Attributes: attribute,
			}
			output.TextObjects = append(output.TextObjects, textObject)
		} else if strings.Contains(attrs.ContentType, "image") {
			imageObject := ImageObject{
				Data:       string(b),
				Attributes: attribute,
			}
			output.ImageObjects = append(output.ImageObjects, imageObject)
		} else if strings.Contains(attrs.ContentType, "audio") {
			audioObject := AudioObject{
				Data:       string(b),
				Attributes: attribute,
			}
			output.AudioObjects = append(output.AudioObjects, audioObject)
		} else if strings.Contains(attrs.ContentType, "video") {
			videoObject := VideoObject{
				Data:       string(b),
				Attributes: attribute,
			}
			output.VideoObjects = append(output.VideoObjects, videoObject)
		} else {
			documentObject := DocumentObject{
				Data:       string(b),
				Attributes: attribute,
			}
			output.DocumentObjects = append(output.DocumentObjects, documentObject)
		}
	}
	return output, nil
}
