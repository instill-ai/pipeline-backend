package googledrive

import "google.golang.org/api/drive/v3"

type IDriveService interface {
	readFile()
	readFolder()
}

type driveService struct {
	service *drive.Service
}

func (d *driveService) readFile() {
}

func (d *driveService) readFolder() {
}
