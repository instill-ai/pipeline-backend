package binary

import (
	"context"
	"encoding/base64"
	"mime"
	"strings"

	"github.com/gabriel-vasile/mimetype"
	"github.com/go-resty/resty/v2"
)

// Fetcher is a struct that fetches binary data from a URL.
type Fetcher struct {
	httpClient *resty.Client
}

// NewFetcher creates a new Fetcher instance.
func NewFetcher() Fetcher {
	return Fetcher{
		httpClient: resty.New().SetRetryCount(3),
	}
}

// FetchFromURL fetches binary data from a URL.
func (f *Fetcher) FetchFromURL(ctx context.Context, url string) (body []byte, contentType string, filename string, err error) {
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

	if disposition := resp.Header().Get("Content-Disposition"); disposition == "" {
		if strings.HasPrefix(disposition, "attachment") {
			if _, params, err := mime.ParseMediaType(disposition); err == nil {
				filename = params["filename"]
			}
		}
	}

	return
}

func (f *Fetcher) convertDataURIToBytes(url string) (b []byte, contentType string, filename string, err error) {
	slices := strings.Split(url, ",")
	if len(slices) == 1 {
		b, err = base64.StdEncoding.DecodeString(url)
		if err != nil {
			return
		}
		contentType = strings.Split(mimetype.Detect(b).String(), ";")[0]
	} else {
		mime := strings.Split(slices[0], ":")
		tags := ""
		contentType, tags, _ = strings.Cut(mime[1], ";")
		b, err = base64.StdEncoding.DecodeString(slices[1])
		if err != nil {
			return
		}
		for _, tag := range strings.Split(tags, ";") {
			key, value, _ := strings.Cut(tag, "=")
			if key == "filename" || key == "fileName" || key == "file-name" {
				filename = value
			}
		}
	}
	return b, contentType, filename, nil
}
