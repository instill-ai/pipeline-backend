package googledriveclient

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"

	"google.golang.org/api/drive/v3"
)

// IDriveService is an interface for interacting with Google Drive.
type IDriveService interface {
	ReadFile(fileUID string) (*drive.File, *string, error)
	ReadFolder(folderUID string, readContent bool) ([]*drive.File, []*string, error)
}

// DriveService is a struct that implements IDriveService.
type DriveService struct {
	// Service is the Google Drive service.
	Service *drive.Service
}

// ReadFile reads a file from Google Drive and get the file content passed as base64.
func (d *DriveService) ReadFile(fileUID string) (*drive.File, *string, error) {

	srv := d.Service

	driveFile, err := srv.Files.Get(fileUID).
		// We will need to confirm if we want to support all drives.
		// By setting SupportsAllDrives to true, the API can return the file from the shared drive, which is not owned by the user but shared with the user.
		SupportsAllDrives(true).
		Fields("id, name, createdTime, modifiedTime, size, mimeType, md5Checksum, version, webViewLink, webContentLink").
		Do()

	if err != nil {
		return nil, nil, fmt.Errorf("fetch fetch metadata of file: %w", err)
	}

	base64Content, err := readFileContent(srv, driveFile)

	if err != nil {
		return nil, nil, fmt.Errorf("read file content: %w", err)
	}

	return driveFile, &base64Content, nil
}

// ReadFolder reads a folder from Google Drive and get the files in the folder. If readContent is true, the file content will be passed as base64.
func (d *DriveService) ReadFolder(folderUID string, readContent bool) ([]*drive.File, []*string, error) {
	srv := d.Service

	q := fmt.Sprintf("'%s' in parents", folderUID)

	var allFiles []*drive.File

	pageToken := ""

	for {
		fileList, err := srv.Files.List().
			Q(q).
			// To fetch file from shared drive, we need to set SupportsAllDrives to true.
			SupportsAllDrives(true).
			// To fetch file from shared drive, we need to set IncludeItemsFromAllDrives to true.
			IncludeItemsFromAllDrives(true).
			Fields("files(id, name, createdTime, modifiedTime, size, mimeType, md5Checksum, version, webViewLink, webContentLink)").
			PageToken(pageToken).
			Do()

		if err != nil {
			return nil, nil, fmt.Errorf("fetch metadata of files: %w", err)
		}

		allFiles = append(allFiles, fileList.Files...)

		pageToken = fileList.NextPageToken

		if pageToken == "" {
			break
		}
	}

	files := make([]*drive.File, 0, len(allFiles))

	for i, f := range allFiles {
		files[i] = f
	}

	if !readContent {
		return files, nil, nil
	}

	contents := make([]*string, 0, len(allFiles))

	for i, f := range allFiles {
		content, err := readFileContent(srv, f)

		if err != nil {
			return nil, nil, fmt.Errorf("read file content: %w", err)
		}

		contents[i] = &content
	}

	return files, contents, nil

}

// Google Drive API only can support downloading the binary data.
// So, when the file is not binary, we need to export the file as PDF/CSV first.
// For example:
// Google Sheets -> Export as CSV
// Google Slides -> Export as PDF
// Google Docs -> Export as PDF
func readFileContent(srv *drive.Service, driveFile *drive.File) (string, error) {
	exportFormat := exportFormat(driveFile)

	var resp *http.Response
	var err error
	if exportFormat == "" {
		resp, err = srv.Files.Get(driveFile.Id).SupportsAllDrives(true).Download()
		if err != nil {
			return "", fmt.Errorf("download file: %w", err)
		}
	} else {
		resp, err = srv.Files.Export(driveFile.Id, exportFormat).Download()
		if err != nil {
			return "", fmt.Errorf("export file: %w", err)
		}
	}

	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)

	if err != nil {
		return "", fmt.Errorf("read file content: %w", err)
	}

	return base64.StdEncoding.EncodeToString(b), nil
}

func exportFormat(file *drive.File) string {
	switch file.MimeType {
	case "application/vnd.google-apps.spreadsheet":
		return "text/csv"
	case "application/vnd.google-apps.presentation", "application/vnd.google-apps.document":
		return "application/pdf"
	default:
		return ""
	}
}
