package googledrive

import (
	"context"
	"fmt"
	"testing"

	"github.com/gojuno/minimock/v3"
	"google.golang.org/api/drive/v3"
	"google.golang.org/protobuf/types/known/structpb"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/mock"
)

var (
	fakeID = "1SwuLagPDCuk04_EIV1qj_pzSaVX3ddEA"

	sharedSheetLink     = fmt.Sprintf("https://drive.google.com/file/d/%s/view?usp=drivesdk", fakeID)
	webViewSheetLink    = fmt.Sprintf("https://docs.google.com/spreadsheets/d/%s/edit?usp=sharing", fakeID)
	webContentSheetLink = fmt.Sprintf("https://drive.google.com/uc?id=%s&export=download", fakeID)

	sharedDocLink     = fmt.Sprintf("https://docs.google.com/document/d/%s/edit?usp=drivesdk", fakeID)
	webViewDocLink    = fmt.Sprintf("https://docs.google.com/document/d/%s/edit?usp=drivesdk", fakeID)
	webContentDocLink = ""

	sharedSlideLink     = fmt.Sprintf("https://docs.google.com/presentation/d/%s/edit?usp=drivesdk", fakeID)
	webViewSlideLink    = fmt.Sprintf("https://docs.google.com/presentation/d/%s/edit?usp=drivesdk", fakeID)
	webContentSlideLink = ""

	sharedFileLink     = fmt.Sprintf("https://drive.google.com/file/d/%s/view?usp=drivesdk", fakeID)
	webViewFileLink    = fmt.Sprintf("https://drive.google.com/file/d/%s/view?usp=drivesdk", fakeID)
	webContentFileLink = fmt.Sprintf("https://drive.google.com/uc?id=%s&export=download", fakeID)

	sharedFolderLink = fmt.Sprintf("https://drive.google.com/drive/folders/%s?usp=drive_link", fakeID)
)

// To integrate the unit test methodology, we will fake the http server to return the expected response rather than mocking the interface.

func Test_Execute_ReadFile(t *testing.T) {

	c := qt.New(t)
	mc := minimock.NewController(c)

	ctx := context.Background()

	testcases := []struct {
		name string

		in            readFileInput
		fakeDriveFile *drive.File
		want          readFileOutput
		wantErr       string
	}{
		{
			name: "ok - read CSV file with file extension",
			in: readFileInput{
				SharedLink: sharedSheetLink,
			},
			fakeDriveFile: &drive.File{
				Id:             fakeID,
				Name:           "testdata.csv",
				CreatedTime:    "2021-08-09T20:25:02.312Z",
				ModifiedTime:   "2021-09-17T16:58:37.924Z",
				Size:           0,
				MimeType:       "application/vnd.google-apps.spreadsheet",
				Md5Checksum:    "7db67eab9238f9a63df30f570fda2bac",
				Version:        0,
				WebViewLink:    webViewSheetLink,
				WebContentLink: webContentSheetLink,
			},
			want: readFileOutput{
				File: file{
					ID:             fakeID,
					Name:           "testdata.csv",
					Content:        "fake content",
					CreatedTime:    "2021-08-09T20:25:02.312Z",
					ModifiedTime:   "2021-09-17T16:58:37.924Z",
					Size:           0,
					MimeType:       "application/vnd.google-apps.spreadsheet",
					Md5Checksum:    "7db67eab9238f9a63df30f570fda2bac",
					Version:        0,
					WebViewLink:    webViewSheetLink,
					WebContentLink: webContentSheetLink,
				},
			},
		},
		{
			name: "ok - read CSV file without file extension",
			in: readFileInput{
				SharedLink: sharedSheetLink,
			},
			fakeDriveFile: &drive.File{
				Id:             fakeID,
				Name:           "testdata",
				CreatedTime:    "2021-08-09T20:25:02.312Z",
				ModifiedTime:   "2021-09-17T16:58:37.924Z",
				Size:           0,
				MimeType:       "application/vnd.google-apps.spreadsheet",
				Md5Checksum:    "7db67eab9238f9a63df30f570fda2bac",
				Version:        0,
				WebViewLink:    webViewSheetLink,
				WebContentLink: webContentSheetLink,
			},
			want: readFileOutput{
				File: file{
					ID:             fakeID,
					Name:           "testdata.csv",
					Content:        "fake content",
					CreatedTime:    "2021-08-09T20:25:02.312Z",
					ModifiedTime:   "2021-09-17T16:58:37.924Z",
					Size:           0,
					MimeType:       "application/vnd.google-apps.spreadsheet",
					Md5Checksum:    "7db67eab9238f9a63df30f570fda2bac",
					Version:        0,
					WebViewLink:    webViewSheetLink,
					WebContentLink: webContentSheetLink,
				},
			},
		},
		{
			name: "ok - read file Google doc file",
			in: readFileInput{
				SharedLink: sharedDocLink,
			},
			fakeDriveFile: &drive.File{
				Id:             fakeID,
				Name:           "testdata",
				CreatedTime:    "2021-08-09T20:25:02.312Z",
				ModifiedTime:   "2021-09-17T16:58:37.924Z",
				Size:           0,
				MimeType:       "application/vnd.google-apps.document",
				Md5Checksum:    "7db67eab9238f9a63df30f570fda2bac",
				Version:        0,
				WebViewLink:    webViewDocLink,
				WebContentLink: webContentDocLink,
			},
			want: readFileOutput{
				File: file{
					ID:           fakeID,
					Name:         "testdata.pdf",
					Content:      "fake content",
					CreatedTime:  "2021-08-09T20:25:02.312Z",
					ModifiedTime: "2021-09-17T16:58:37.924Z",
					Size:         0,
					MimeType:     "application/vnd.google-apps.document",
					Md5Checksum:  "7db67eab9238f9a63df30f570fda2bac",
					Version:      0,
					WebViewLink:  webViewDocLink,
				},
			},
		},
		{
			name: "ok - read file Google slide file",
			in: readFileInput{
				SharedLink: sharedSlideLink,
			},
			fakeDriveFile: &drive.File{
				Id:             fakeID,
				Name:           "testdata",
				CreatedTime:    "2021-08-09T20:25:02.312Z",
				ModifiedTime:   "2021-09-17T16:58:37.924Z",
				Size:           0,
				MimeType:       "application/vnd.google-apps.presentation",
				Md5Checksum:    "7db67eab9238f9a63df30f570fda2bac",
				Version:        0,
				WebViewLink:    webViewSlideLink,
				WebContentLink: webContentSlideLink,
			},
			want: readFileOutput{
				File: file{
					ID:           fakeID,
					Name:         "testdata.pdf",
					Content:      "fake content",
					CreatedTime:  "2021-08-09T20:25:02.312Z",
					ModifiedTime: "2021-09-17T16:58:37.924Z",
					Size:         0,
					MimeType:     "application/vnd.google-apps.presentation",
					Md5Checksum:  "7db67eab9238f9a63df30f570fda2bac",
					Version:      0,
					WebViewLink:  webViewSlideLink,
				},
			},
		},
		{
			name: "ok - read file",
			in: readFileInput{
				SharedLink: sharedFileLink,
			},
			fakeDriveFile: &drive.File{
				Id:             fakeID,
				Name:           "testdata.png",
				CreatedTime:    "2021-08-09T20:25:02.312Z",
				ModifiedTime:   "2021-09-17T16:58:37.924Z",
				Size:           0,
				MimeType:       "image/jpeg",
				Md5Checksum:    "7db67eab9238f9a63df30f570fda2bac",
				Version:        0,
				WebViewLink:    webViewFileLink,
				WebContentLink: webContentFileLink,
			},
			want: readFileOutput{
				File: file{
					ID:             fakeID,
					Name:           "testdata.png",
					Content:        "fake content",
					CreatedTime:    "2021-08-09T20:25:02.312Z",
					ModifiedTime:   "2021-09-17T16:58:37.924Z",
					Size:           0,
					MimeType:       "image/jpeg",
					Md5Checksum:    "7db67eab9238f9a63df30f570fda2bac",
					Version:        0,
					WebViewLink:    webViewFileLink,
					WebContentLink: webContentFileLink,
				},
			},
		},
		{
			name: "nok - read file with invalid shared link",
			in: readFileInput{
				SharedLink: sharedFolderLink,
			},
			wantErr: "the input link is a folder link, please use the read-folder operation",
		},
	}

	bc := base.Component{}
	component := Init(bc)

	secrets := map[string]interface{}{
		"oauthclientid":     "fake-client-id",
		"oauthclientsecret": "fake-client-secret",
	}

	component.WithOAuthConfig(secrets)

	setup := map[string]any{
		"refresh-token": "fake-refresh-token",
	}

	setupStruct, err := structpb.NewStruct(setup)

	c.Assert(err, qt.IsNil)

	for _, tc := range testcases {
		c.Run(tc.name, func(c *qt.C) {
			exec, err := component.CreateExecution(base.ComponentExecution{
				Component: component,
				Task:      taskReadFile,
				Setup:     setupStruct,
			})

			c.Assert(err, qt.IsNil)

			mockDriveService := mock.NewIDriveServiceMock(mc)
			exec.(*execution).service = mockDriveService

			fakeDriveFile := tc.fakeDriveFile
			fakeContent := "fake content"

			if tc.wantErr == "" {
				mockDriveService.ReadFileMock.
					Expect(fakeID).
					Return(fakeDriveFile, &fakeContent, nil)
			}

			ir, ow, eh, job := mock.GenerateMockJob(c)

			ir.ReadDataMock.Set(func(ctx context.Context, input any) error {
				switch input := input.(type) {
				case *readFileInput:
					*input = tc.in
				}
				return nil
			})

			ow.WriteDataMock.Optional().Set(func(ctx context.Context, output any) (err error) {
				switch output := output.(type) {
				case *readFileOutput:
					c.Assert(output, qt.DeepEquals, &tc.want)
				}
				return nil
			})

			eh.ErrorMock.Optional().Set(func(ctx context.Context, err error) {
				if tc.wantErr != "" {
					c.Assert(err, qt.ErrorMatches, tc.wantErr)
				} else {
					c.Assert(err, qt.IsNil)
				}
			})

			err = exec.Execute(ctx, []*base.Job{job})
			c.Assert(err, qt.IsNil)

		})

	}
}

func Test_Execute_ReadFolder(t *testing.T) {
	c := qt.New(t)
	mc := minimock.NewController(c)

	ctx := context.Background()

	testcases := []struct {
		name string

		in             readFolderInput
		fakeDriveFiles []*drive.File
		fakeContents   []*string
		want           readFolderOutput
		wantErr        string
	}{
		{
			name: "ok - read folder with content",
			in: readFolderInput{
				SharedLink:  sharedFolderLink,
				ReadContent: true,
			},
			fakeDriveFiles: []*drive.File{
				{
					Id:             fakeID,
					Name:           "testdata.csv",
					CreatedTime:    "2021-08-09T20:25:02.312Z",
					ModifiedTime:   "2021-09-17T16:58:37.924Z",
					Size:           0,
					MimeType:       "application/vnd.google-apps.spreadsheet",
					Md5Checksum:    "7db67eab9238f9a63df30f570fda2bac",
					Version:        0,
					WebViewLink:    webViewSheetLink,
					WebContentLink: webContentSheetLink,
				},
			},
			fakeContents: []*string{
				stringPointer("fake content"),
			},
			want: readFolderOutput{
				Files: []*file{
					{
						ID:             fakeID,
						Name:           "testdata.csv",
						Content:        "fake content",
						CreatedTime:    "2021-08-09T20:25:02.312Z",
						ModifiedTime:   "2021-09-17T16:58:37.924Z",
						Size:           0,
						MimeType:       "application/vnd.google-apps.spreadsheet",
						Md5Checksum:    "7db67eab9238f9a63df30f570fda2bac",
						Version:        0,
						WebViewLink:    webViewSheetLink,
						WebContentLink: webContentSheetLink,
					},
				},
			},
		},
		{
			name: "ok - read folder without content",
			in: readFolderInput{
				SharedLink:  sharedFolderLink,
				ReadContent: false,
			},
			fakeDriveFiles: []*drive.File{
				{
					Id:             fakeID,
					Name:           "testdata.csv",
					CreatedTime:    "2021-08-09T20:25:02.312Z",
					ModifiedTime:   "2021-09-17T16:58:37.924Z",
					Size:           0,
					MimeType:       "application/vnd.google-apps.spreadsheet",
					Md5Checksum:    "7db67eab9238f9a63df30f570fda2bac",
					Version:        0,
					WebViewLink:    webViewSheetLink,
					WebContentLink: webContentSheetLink,
				},
			},
			fakeContents: nil,
			want: readFolderOutput{
				Files: []*file{
					{
						ID:             fakeID,
						Name:           "testdata.csv",
						Content:        "",
						CreatedTime:    "2021-08-09T20:25:02.312Z",
						ModifiedTime:   "2021-09-17T16:58:37.924Z",
						Size:           0,
						MimeType:       "application/vnd.google-apps.spreadsheet",
						Md5Checksum:    "7db67eab9238f9a63df30f570fda2bac",
						Version:        0,
						WebViewLink:    webViewSheetLink,
						WebContentLink: webContentSheetLink,
					},
				},
			},
		},
		{
			name: "nok - read file",
			in: readFolderInput{
				SharedLink:  sharedSheetLink,
				ReadContent: false,
			},
			wantErr: "the input link is not a folder link, please check the link",
		},
	}

	bc := base.Component{}
	component := Init(bc)

	secrets := map[string]interface{}{
		"oauthclientid":     "fake-client-id",
		"oauthclientsecret": "fake-client-secret",
	}

	component.WithOAuthConfig(secrets)

	setup := map[string]any{
		"refresh-token": "fake-refresh-token",
	}

	setupStruct, err := structpb.NewStruct(setup)

	c.Assert(err, qt.IsNil)

	for _, tc := range testcases {
		c.Run(tc.name, func(c *qt.C) {

			exec, err := component.CreateExecution(base.ComponentExecution{
				Component: component,
				Task:      taskReadFolder,
				Setup:     setupStruct,
			})

			c.Assert(err, qt.IsNil)

			mockDriveService := mock.NewIDriveServiceMock(mc)
			exec.(*execution).service = mockDriveService

			fakeDriveFiles := tc.fakeDriveFiles
			fakeContents := tc.fakeContents

			readContent := tc.in.ReadContent

			if tc.wantErr == "" {
				mockDriveService.ReadFolderMock.
					Expect(fakeID, readContent).
					Return(fakeDriveFiles, fakeContents, nil)
			}

			ir, ow, eh, job := mock.GenerateMockJob(c)

			ir.ReadDataMock.Set(func(ctx context.Context, input any) error {
				switch input := input.(type) {
				case *readFolderInput:
					*input = tc.in
				}
				return nil
			})

			ow.WriteDataMock.Optional().Set(func(ctx context.Context, output any) (err error) {
				switch output := output.(type) {
				case *readFolderOutput:
					c.Assert(output, qt.DeepEquals, &tc.want)
				}
				return nil
			})

			eh.ErrorMock.Optional().Set(func(ctx context.Context, err error) {
				if tc.wantErr != "" {
					c.Check(err, qt.ErrorMatches, tc.wantErr)
				} else {
					c.Check(err, qt.IsNil)
				}
			})

			err = exec.Execute(ctx, []*base.Job{job})
			c.Check(err, qt.IsNil)
		})
	}
}

func Test_CreateExecution(t *testing.T) {
	c := qt.New(t)

	testcase := struct {
		name string

		task    string
		wantErr string
	}{
		name: "nok - unsupported task",
		task: "FOOBAR",

		wantErr: "not supported task: FOOBAR",
	}

	bc := base.Component{}
	component := Init(bc)

	secrets := map[string]interface{}{
		"oauthclientid":     "fake-client-id",
		"oauthclientsecret": "fake-client-secret",
	}

	component.WithOAuthConfig(secrets)

	setup := map[string]any{
		"refresh-token": "fake-refresh-token",
	}

	setupStruct, err := structpb.NewStruct(setup)

	c.Assert(err, qt.IsNil)

	c.Run(testcase.name, func(c *qt.C) {

		_, err := component.CreateExecution(base.ComponentExecution{
			Component: component,
			Task:      testcase.task,
			Setup:     setupStruct,
		})

		c.Check(err, qt.ErrorMatches, testcase.wantErr)
	})

}

func stringPointer(s string) *string {
	return &s
}
