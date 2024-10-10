package googledrive

import "google.golang.org/api/drive/v3"

type IDriveService interface {
	readFile()
	readFiles()
	readDrive()
}

type driveService struct {
	service *drive.Service
}

func (d *driveService) readFile() {
}

func (d *driveService) readFiles() {
}

func (d *driveService) readDrive() {
}
