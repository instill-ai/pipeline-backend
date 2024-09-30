package document

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	"image/png"
	"strings"
	"time"
	"unicode/utf8"

	"encoding/base64"

	"code.sajari.com/docconv"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util"
)

var (
	supportedByDocconvConvertMimeTypes = map[string]bool{
		"application/msword":      true,
		"application/vnd.ms-word": true,
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document":   true,
		"application/vnd.openxmlformats-officedocument.presentationml.presentation": true,
		"application/vnd.oasis.opendocument.text":                                   true,
		"application/vnd.apple.pages":                                               true,
		"application/x-iwork-pages-sffpages":                                        true,
		"application/pdf":                                                           true,
		"application/rtf":                                                           true,
		"application/x-rtf":                                                         true,
		"text/rtf":                                                                  true,
		"text/richtext":                                                             true,
		"text/html":                                                                 true,
		"text/url":                                                                  true,
		"text/xml":                                                                  true,
		"application/xml":                                                           true,
		"image/jpeg":                                                                true,
		"image/png":                                                                 true,
		"image/tif":                                                                 true,
		"image/tiff":                                                                true,
		"text/plain":                                                                true,
	}
)

// ConvertToTextInput defines the input for convert to text task
type ConvertToTextInput struct {
	// Document: Document to convert
	Document string `json:"document"`
	Filename string `json:"filename"`
}

// ConvertToTextOutput defines the output for convert to text task
type ConvertToTextOutput struct {
	// Body: Plain text converted from the document
	Body string `json:"body"`
	// Meta: Metadata extracted from the document
	Meta map[string]string `json:"meta"`
	// MSecs: Time taken to convert the document
	MSecs uint32 `json:"msecs"`
	// Error: Error message if any during the conversion process
	Error    string `json:"error"`
	Filename string `json:"filename"`
}

type converter interface {
	convert(contentType string, b []byte) (ConvertToTextOutput, error)
}

type docconvConverter struct{}

func (d docconvConverter) convert(contentType string, b []byte) (ConvertToTextOutput, error) {

	if contentType == "image/jpeg" {
		pngData, err := convertJpegToPng(b)
		if err != nil {
			return ConvertToTextOutput{}, fmt.Errorf("error converting jpeg to png: %v", err)
		}
		b = pngData
		contentType = "image/png"
	}

	res, err := docconv.Convert(bytes.NewReader(b), contentType, false)
	if err != nil {
		fmt.Println("Error converting document to text", err)
		return ConvertToTextOutput{}, err
	}

	if res.Meta == nil {
		res.Meta = map[string]string{}
	}

	return ConvertToTextOutput{
		Body:  res.Body,
		Meta:  res.Meta,
		MSecs: res.MSecs,
		Error: res.Error,
	}, nil
}

func convertJpegToPng(jpegData []byte) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(jpegData))
	if err != nil {
		return nil, err
	}

	var pngBuffer bytes.Buffer

	err = png.Encode(&pngBuffer, img)
	if err != nil {
		return nil, err
	}

	return pngBuffer.Bytes(), nil
}

type uft8EncodedFileConverter struct{}

func (m uft8EncodedFileConverter) convert(contentType string, b []byte) (ConvertToTextOutput, error) {

	before := time.Now()
	content := string(b)

	duration := time.Since(before)
	millis := duration.Milliseconds()

	metadata := map[string]string{}

	return ConvertToTextOutput{
		Body:  content,
		Meta:  metadata,
		MSecs: uint32(millis),
		Error: "",
	}, nil
}

func isSupportedByDocconvConvert(contentType string) bool {
	return supportedByDocconvConvertMimeTypes[contentType]
}

func ConvertToText(input ConvertToTextInput) (ConvertToTextOutput, error) {

	contentType, err := util.GetContentTypeFromBase64(input.Document)
	if err != nil {
		return ConvertToTextOutput{}, err
	}

	b, err := base64.StdEncoding.DecodeString(base.TrimBase64Mime(input.Document))
	if err != nil {
		return ConvertToTextOutput{}, err
	}

	// TODO: support xlsx file type with https://github.com/qax-os/excelize
	var converter converter
	if isSupportedByDocconvConvert(contentType) {
		converter = docconvConverter{}
	} else if utf8.Valid(b) {
		converter = uft8EncodedFileConverter{}
	} else {
		return ConvertToTextOutput{}, fmt.Errorf("unsupported content type")
	}

	res, err := converter.convert(contentType, b)
	if err != nil {
		return ConvertToTextOutput{}, err
	}

	if input.Filename != "" {
		filename := strings.Split(input.Filename, ".")[0] + ".txt"
		res.Filename = filename
	}

	return res, nil
}
