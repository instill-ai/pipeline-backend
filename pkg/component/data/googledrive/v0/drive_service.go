package googledrive

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"

	"google.golang.org/api/drive/v3"
)

type IDriveService interface {
	readFile(fileUID string) (*file, error)
	readFolder(folderUID string, readContent bool) ([]*file, error)
}

type driveService struct {
	service *drive.Service
}

func (d *driveService) readFile(fileUID string) (*file, error) {

	srv := d.service

	driveFile, err := srv.Files.Get(fileUID).
		// We will need to confirm if we want to support all drives.
		// By setting SupportsAllDrives to true, the API can return the file from the shared drive, which is not owned by the user but shared with the user.
		SupportsAllDrives(true).
		Fields("id, name, createdTime, modifiedTime, size, mimeType, md5Checksum, version, webViewLink, webContentLink").
		Do()

	if err != nil {
		return nil, fmt.Errorf("fetch fetch metadata of file: %w", err)
	}

	file := convertDriveFileToComponentFile(driveFile)

	base64Content, err := readFileContent(srv, driveFile)

	if err != nil {
		return nil, fmt.Errorf("read file content: %w", err)
	}

	file.Content = base64Content

	return file, nil
}

func (d *driveService) readFolder(folderUID string, readContent bool) ([]*file, error) {
	srv := d.service

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
			return nil, fmt.Errorf("fetch metadata of files: %w", err)
		}

		allFiles = append(allFiles, fileList.Files...)

		pageToken = fileList.NextPageToken

		if pageToken == "" {
			break
		}
	}

	files := make([]*file, 0, len(allFiles))

	for _, f := range allFiles {
		file := convertDriveFileToComponentFile(f)

		if readContent {
			base64Content, err := readFileContent(srv, f)

			if err != nil {
				return nil, fmt.Errorf("read file content: %w", err)
			}

			file.Content = base64Content
		}

		files = append(files, file)
	}

	return files, nil

}

func convertDriveFileToComponentFile(driveFile *drive.File) *file {
	return &file{
		ID:             driveFile.Id,
		Name:           driveFile.Name,
		CreatedTime:    driveFile.CreatedTime,
		ModifiedTime:   driveFile.ModifiedTime,
		Size:           driveFile.Size,
		MimeType:       driveFile.MimeType,
		Md5Checksum:    driveFile.Md5Checksum,
		Version:        driveFile.Version,
		WebViewLink:    driveFile.WebViewLink,
		WebContentLink: driveFile.WebContentLink,
	}
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

func exportFileExtension(mimeType string) string {
	switch mimeType {
	case "application/vnd.google-apps.spreadsheet":
		return ".csv"
	case "application/vnd.google-apps.presentation", "application/vnd.google-apps.document":
		return ".pdf"
	default:
		return ""
	}
}
