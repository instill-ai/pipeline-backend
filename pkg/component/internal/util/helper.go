package util

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gabriel-vasile/mimetype"
	"github.com/h2non/filetype"

	md "github.com/JohannesKaufmann/html-to-markdown"
	timestampPB "google.golang.org/protobuf/types/known/timestamppb"
)

func GetFileExt(fileData []byte) string {
	kind, _ := filetype.Match(fileData)
	if kind != filetype.Unknown && kind.Extension != "" {
		return kind.Extension
	}
	//fallback to DetectContentType
	mimeType := http.DetectContentType(fileData)
	return mimeType[strings.LastIndex(mimeType, "/")+1:]
}

func WriteFile(writer *multipart.Writer, fileName string, fileData []byte) error {
	part, err := writer.CreateFormFile(fileName, "file."+GetFileExt(fileData))
	if err != nil {
		return err
	}
	_, err = part.Write(fileData)
	return err
}

func WriteField(writer *multipart.Writer, key string, value string) {
	if key != "" && value != "" {
		_ = writer.WriteField(key, value)
	}
}

// ScrapeWebpageHTML scrape the HTML content of a webpage
func ScrapeWebpageHTML(doc *goquery.Document) (string, error) {
	return doc.Html()
}

// ScrapeWebpageTitle extracts and returns the title from the *goquery.Document
func ScrapeWebpageTitle(doc *goquery.Document) string {
	// Find the title tag and get its text content
	title := doc.Find("title").Text()

	// Return the trimmed title
	return strings.TrimSpace(title)
}

// ScrapeWebpageDescription extracts and returns the description from the *goquery.Document.
// If the description does not exist, an empty string is returned
// The description is found by looking for the meta tag with the name "description"
// and returning the content attribute
func ScrapeWebpageDescription(doc *goquery.Document) string {
	// Find the meta tag with the description name
	description, ok := doc.Find(`meta[name="description"]`).Attr("content")
	if !ok {
		return ""
	}
	// Return the trimmed description
	return strings.TrimSpace(description)
}

// ScrapeWebpageHTMLToMarkdown converts an HTML string to Markdown format
func ScrapeWebpageHTMLToMarkdown(html, domain string) (string, error) {
	// Initialize the markdown converter
	converter := md.NewConverter(domain, true, nil)

	// Convert the HTML to Markdown
	markdown, err := converter.ConvertString(html)
	if err != nil {
		return "", err
	}

	return markdown, nil
}

func GetDomainFromURL(urlStr string) (string, error) {
	u, err := url.Parse(urlStr)

	if err != nil {
		return "", fmt.Errorf("error when parse url: %v", err)
	}
	return u.Host, nil
}

// DecodeBase64 takes a base64-encoded blob, trims the MIME type (if present)
// and decodes the remaining bytes.
func DecodeBase64(input string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(TrimBase64Mime(input))
}

func GetFileType(base64String, filename string) (string, error) {
	parts := strings.SplitN(base64String, ";", 2)
	var typeFromBase64 string
	var typeFromFilename string
	var err error

	if len(parts) == 2 {
		contentType, _ := GetContentTypeFromBase64(base64String)
		typeFromBase64 = TransformContentTypeToFileExtension(contentType)
	}

	typeFromFilename, err = GetFileTypeByFilename(filename)
	if err != nil {
		return "", err
	}

	if typeFromBase64 == "" {
		return typeFromFilename, nil
	}

	if typeFromBase64 != typeFromFilename {
		return "", fmt.Errorf("file type mismatch")
	}

	return typeFromBase64, nil
}

func GetFileTypeByFilename(filename string) (string, error) {
	splittedString := strings.Split(filename, ".")
	if len(splittedString) != 2 {
		return "", fmt.Errorf("invalid filename")
	}
	return splittedString[1], nil
}

func GetContentTypeFromBase64(base64String string) (string, error) {
	// Remove the "data:" prefix and split at the first semicolon
	if hasDataPrefix(base64String) {
		contentType := strings.TrimPrefix(base64String, "data:")

		parts := strings.SplitN(contentType, ";", 2)
		if len(parts) != 2 {
			return "", fmt.Errorf("invalid format")
		}

		// The first part is the content type
		return parts[0], nil
	}

	b, err := base64.StdEncoding.DecodeString(base64String)
	if err != nil {
		return "", fmt.Errorf("decode base64 string: %w", err)
	}
	mimeType := strings.Split(mimetype.Detect(b).String(), ";")[0]
	return mimeType, nil
}

func GetFileBase64Content(base64String string) string {
	parts := strings.SplitN(base64String, ";", 2)
	if len(parts) == 2 {
		return strings.SplitN(parts[1], ",", 2)[1]
	}
	return base64String
}

func TransformContentTypeToFileExtension(contentType string) string {
	// https://gist.github.com/AshHeskes/6038140
	// We can integrate more Content-Type to file extension mappings in the future
	switch contentType {
	case "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
		return "docx"
	case "application/msword":
		return "doc"
	case "application/vnd.openxmlformats-officedocument.presentationml.presentation":
		return "pptx"
	case "application/vnd.ms-powerpoint":
		return "ppt"
	case "text/html":
		return "html"
	case "application/pdf":
		return "pdf"
	case "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":
		return "xlsx"
	case "application/vnd.ms-excel":
		return "xls"
	case "text/csv":
		return "csv"
	}
	return ""
}

func StripProtocolFromURL(url string) string {
	index := strings.Index(url, "://")
	if index > 0 {
		return url[strings.Index(url, "://")+3:]
	}
	return url
}

func GetHeaderAuthorization(vars map[string]any) string {
	if v, ok := vars["__PIPELINE_HEADER_AUTHORIZATION"]; ok {
		return v.(string)
	}
	return ""
}
func GetInstillUserUID(vars map[string]any) string {
	return vars["__PIPELINE_USER_UID"].(string)
}

func GetInstillRequesterUID(vars map[string]any) string {
	return vars["__PIPELINE_REQUESTER_UID"].(string)
}

func ConvertDataFrameToMarkdownTable(rows [][]string) string {
	var sb strings.Builder

	sb.WriteString("|")
	for _, colCell := range rows[0] {
		sb.WriteString(fmt.Sprintf(" %s |", colCell))
	}
	sb.WriteString("\n")

	sb.WriteString("|")
	for range rows[0] {
		sb.WriteString(" --- |")
	}
	sb.WriteString("\n")

	for _, row := range rows[1:] {
		sb.WriteString("|")
		for _, colCell := range row {
			sb.WriteString(fmt.Sprintf(" %s |", colCell))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func InSlice(slice []string, item string) bool {
	for _, i := range slice {
		if i == item {
			return true
		}
	}
	return false
}

func GetDataURL(base64Image string) string {

	if hasDataPrefix(base64Image) {
		return base64Image
	}

	b, err := base64.StdEncoding.DecodeString(TrimBase64Mime(base64Image))

	if err != nil {
		return base64Image
	}

	dataURL := fmt.Sprintf("data:%s;base64,%s", mimetype.Detect(b).String(), TrimBase64Mime(base64Image))

	return dataURL
}

func hasDataPrefix(base64Image string) bool {
	return strings.HasPrefix(base64Image, "data:")
}

var pythonInterpreter string = "/opt/venv/bin/python"

func ExecutePythonCode(pythonCode string, params map[string]interface{}) ([]byte, error) {

	paramsJSON, err := json.Marshal(params)

	if err != nil {
		return nil, fmt.Errorf("failed to marshal params: %w", err)
	}

	cmdRunner := exec.Command(pythonInterpreter, "-c", pythonCode)

	stdin, err := cmdRunner.StdinPipe()

	if err != nil {
		return nil, fmt.Errorf("failed to get stdin pipe: %w", err)
	}

	errChan := make(chan error, 1)
	go func() {
		defer stdin.Close()
		_, err := stdin.Write(paramsJSON)
		if err != nil {
			errChan <- err
			return
		}
		errChan <- nil
	}()

	outputBytes, err := cmdRunner.Output()

	if err != nil {
		errorStr := string(outputBytes)
		return nil, fmt.Errorf("failed to run python script: %w, %s", err, errorStr)
	}

	writeErr := <-errChan
	if writeErr != nil {
		return nil, fmt.Errorf("failed to write to stdin: %w", writeErr)
	}

	return outputBytes, nil
}
func TrimBase64Mime(b64 string) string {
	splitB64 := strings.Split(b64, ",")
	return splitB64[len(splitB64)-1]
}

func FormatToISO8601(ts *timestampPB.Timestamp) string {
	return ts.AsTime().UTC().Format(time.RFC3339)
}

// UnixToISO8601 converts a Unix timestamp to an ISO8601 formatted string
func UnixToISO8601(unix int64) string {
	return time.Unix(unix, 0).UTC().Format(time.RFC3339)
}

// return the extension of the file from the base64 string, in the "jpeg" , "png" format, check with provided header
func GetBase64FileExtension(b64 string) string {
	splitB64 := strings.Split(b64, ",")
	header := splitB64[0]
	header = strings.TrimPrefix(header, "data:")
	header = strings.TrimSuffix(header, ";base64")
	mtype, _, err := mime.ParseMediaType(header)
	if err != nil {
		return err.Error()
	}
	return strings.Split(mtype, "/")[1]
}
