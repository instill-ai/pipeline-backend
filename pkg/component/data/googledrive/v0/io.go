package googledrive

type readFileInput struct {
	SharedLink string `instill:"shared-link"`
}

type readFileOutput struct {
	File file `instill:"file"`
}

type file struct {
	ID             string `instill:"id"`
	Name           string `instill:"name"`
	Content        string `instill:"content"`
	CreatedTime    string `instill:"created-time"`
	ModifiedTime   string `instill:"modified-time"`
	Size           int64  `instill:"size"`
	MimeType       string `instill:"mime-type"`
	Md5Checksum    string `instill:"md5-checksum,omitempty"`
	Version        int64  `instill:"version"`
	WebViewLink    string `instill:"web-view-link"`
	WebContentLink string `instill:"web-content-link,omitempty"`
}

type readFolderInput struct {
	SharedLink  string `instill:"shared-link"`
	ReadContent bool   `instill:"read-content"`
}

type readFolderOutput struct {
	Files []*file `instill:"files"`
}
