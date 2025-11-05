package external

import (
	"context"
	"encoding/base64"
	"fmt"
	"mime"
	"net/url"
	"regexp"
	"strings"

	"github.com/gabriel-vasile/mimetype"
	"github.com/go-resty/resty/v2"
	"github.com/gofrs/uuid"

	"github.com/instill-ai/x/minio"
	"github.com/instill-ai/x/resource"

	artifactpb "github.com/instill-ai/protogen-go/artifact/artifact/v1alpha"
)

// BinaryFetcher is an interface that fetches binary data from a URL.
type BinaryFetcher interface {
	FetchFromURL(ctx context.Context, url string) (body []byte, contentType string, filename string, err error)
}

type binaryFetcher struct {
	httpClient *resty.Client
}

// NewBinaryFetcher creates a new BinaryFetcher instance.
func NewBinaryFetcher() BinaryFetcher {
	return &binaryFetcher{
		httpClient: resty.New().SetRetryCount(3),
	}
}

// FetchFromURL fetches binary data from a URL.
func (f *binaryFetcher) FetchFromURL(ctx context.Context, url string) (body []byte, contentType string, filename string, err error) {
	if strings.HasPrefix(url, "data:") {
		return f.convertDataURIToBytes(url)
	}

	var resp *resty.Response
	resp, err = f.httpClient.R().SetContext(ctx).Get(url)
	if err != nil {
		return
	}

	body = resp.Body()
	contentType = strings.Split(mimetype.Detect(body).String(), ";")[0]

	if disposition := resp.Header().Get("Content-Disposition"); disposition != "" {
		if _, params, err := mime.ParseMediaType(disposition); err == nil {
			filename = params["filename"]
		}
	}

	return
}

// Pattern matches: https://{domain}/v1alpha/namespaces/{namespace}/blob-urls/{uid}
// This is a deprecated pattern, we should use the presigned pattern instead.
var minioURLPattern = regexp.MustCompile(`https?://[^/]+/v1alpha/namespaces/[^/]+/blob-urls/([^/]+)$`)

// Pattern matches: https://{domain}/v1alpha/blob-urls/{encoded_presigned_url}
// This is the new pattern, we should use this instead of the deprecated pattern.
// The new design totally rely on the presigned URL provided by MinIO, without the need to get object URL from table.
var minioURLPresignedPattern = regexp.MustCompile(`https?://[^/]+/v1alpha/blob-urls/([^/]+)$`)

// ArtifactBinaryFetcher fetches binary data from a URL.
// If that URL comes from an object uploaded on Instill Artifact,
// it uses the blob storage client directly to avoid egress costs.
type artifactBinaryFetcher struct {
	binaryFetcher  *binaryFetcher
	artifactClient artifactpb.ArtifactPrivateServiceClient // By having this injected, main.go is responsible of closing the connection.
	fileGetter     *minio.FileGetter
}

// NewArtifactBinaryFetcher creates a new ArtifactBinaryFetcher instance.
func NewArtifactBinaryFetcher(ac artifactpb.ArtifactPrivateServiceClient, fg *minio.FileGetter) BinaryFetcher {
	return &artifactBinaryFetcher{
		binaryFetcher: &binaryFetcher{
			httpClient: resty.New().SetRetryCount(3),
		},
		artifactClient: ac,
		fileGetter:     fg,
	}
}

func (f *artifactBinaryFetcher) FetchFromURL(ctx context.Context, fileURL string) (b []byte, contentType string, filename string, err error) {
	if strings.HasPrefix(fileURL, "data:") {
		return f.binaryFetcher.convertDataURIToBytes(fileURL)
	}
	if matches := minioURLPattern.FindStringSubmatch(fileURL); matches != nil {
		if len(matches) < 2 {
			err = fmt.Errorf("invalid blob storage url: %s", fileURL)
			return
		}

		return f.fetchFromBlobStorage(ctx, uuid.FromStringOrNil(matches[1]))
	}
	if matches := minioURLPresignedPattern.FindStringSubmatch(fileURL); matches != nil {
		if len(matches) < 1 {
			err = fmt.Errorf("invalid blob storage url: %s", fileURL)
			return
		}
		parsedURL, err := url.Parse(fileURL)
		if err != nil {
			return nil, "", "", err
		}
		// The presigned URL is encoded in the format:
		// scheme://host/v1alpha/blob-urls/base64_encoded_presigned_url
		// Here we decode the base64 string to the presigned URL.
		base64Decoded, err := base64.URLEncoding.DecodeString(strings.Split(parsedURL.Path, "/")[3])
		if err != nil {
			return nil, "", "", err
		}

		// the decoded presigned URL is a self-contained URL that can be used
		// to upload or download the object directly.
		return f.binaryFetcher.FetchFromURL(ctx, string(base64Decoded))
	}
	return f.binaryFetcher.FetchFromURL(ctx, fileURL)
}

func (f *artifactBinaryFetcher) fetchFromBlobStorage(ctx context.Context, urlUID uuid.UUID) (b []byte, contentType string, filename string, err error) {
	objectURLRes, err := f.artifactClient.GetObjectURLAdmin(ctx, &artifactpb.GetObjectURLAdminRequest{
		Uid: urlUID.String(),
	})
	if err != nil {
		return nil, "", "", err
	}

	objectUID := objectURLRes.ObjectUrl.ObjectUid

	objectRes, err := f.artifactClient.GetObjectAdmin(ctx, &artifactpb.GetObjectAdminRequest{
		Uid: objectUID,
	})

	if err != nil {
		return nil, "", "", err
	}

	// TODO: we have agreed on to add the bucket name in pipelinepb.Object
	// After the contract is updated, we have to replace it
	bucketName := "core-blob"
	objectPath := *objectRes.Object.Path

	// TODO this won't always produce a valid user UID (e.g. the jobs in the
	// worker don't have this in the context).
	// If we want a full audit of the MinIO actions (or if we want to check
	// object permissions), we need to update the signature to pass the user
	// UID explicitly.
	_, userUID := resource.GetRequesterUIDAndUserUID(ctx)
	b, _, err = f.fileGetter.GetFile(ctx, minio.GetFileParams{
		BucketName: bucketName,
		Path:       objectPath,
		UserUID:    userUID,
	})
	if err != nil {
		return nil, "", "", err
	}
	contentType = strings.Split(mimetype.Detect(b).String(), ";")[0]
	return b, contentType, objectRes.Object.Name, nil
}

func (f *binaryFetcher) convertDataURIToBytes(dataURI string) (b []byte, contentType string, filename string, err error) {
	slices := strings.Split(dataURI, ",")
	if len(slices) == 1 {
		b, err = base64.StdEncoding.DecodeString(dataURI)
		if err != nil {
			return
		}
		contentType = strings.Split(mimetype.Detect(b).String(), ";")[0]
	} else {
		// Parse the header part (before the comma)
		header := slices[0]
		if !strings.HasPrefix(header, "data:") {
			return nil, "", "", fmt.Errorf("invalid data URI format")
		}

		// Remove "data:" prefix
		header = strings.TrimPrefix(header, "data:")

		// Split by semicolon to get content type and parameters
		parts := strings.Split(header, ";")
		if len(parts) == 0 {
			return nil, "", "", fmt.Errorf("invalid data URI header")
		}

		// First part is the content type
		contentType = parts[0]

		// Parse parameters (skip the first part which is content type)
		for i := 1; i < len(parts); i++ {
			part := strings.TrimSpace(parts[i])
			if part == "" {
				continue
			}

			// Check if this is the base64 parameter
			if part == "base64" {
				continue
			}

			// Parse key=value pairs
			if strings.Contains(part, "=") {
				key, value, _ := strings.Cut(part, "=")
				key = strings.TrimSpace(key)
				value = strings.TrimSpace(value)
				if key == "filename" || key == "fileName" || key == "file-name" {
					// URL decode the filename to handle %20 and other encoded characters
					if decodedValue, err := url.QueryUnescape(value); err == nil {
						filename = decodedValue
					} else {
						filename = value // fallback to original value if decoding fails
					}
				}
			}
		}

		// Decode the base64 data
		b, err = base64.StdEncoding.DecodeString(slices[1])
		if err != nil {
			return nil, "", "", err
		}
	}

	return b, contentType, filename, nil
}
