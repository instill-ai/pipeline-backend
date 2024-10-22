package googledrive

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
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

func Test_Execute_ReadFile(t *testing.T) {

	c := qt.New(t)
	mc := minimock.NewController(c)

	ctx := context.Background()

	testcases := []struct {
		name string

		in            map[string]any
		fakeDriveFile *drive.File
		want          map[string]any
		wantErr       string
	}{
		{
			name: "ok - read CSV file with file extension",
			in: map[string]any{
				"shared-link": sharedSheetLink,
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
			want: map[string]any{
				"file": map[string]any{
					"id":               fakeID,
					"name":             "testdata.csv",
					"content":          "fake content",
					"created-time":     "2021-08-09T20:25:02.312Z",
					"modified-time":    "2021-09-17T16:58:37.924Z",
					"size":             0,
					"mime-type":        "application/vnd.google-apps.spreadsheet",
					"md5-checksum":     "7db67eab9238f9a63df30f570fda2bac",
					"version":          0,
					"web-view-link":    webViewSheetLink,
					"web-content-link": webContentSheetLink,
				},
			},
		},
		{
			name: "ok - read CSV file without file extension",
			in: map[string]any{
				"shared-link": sharedSheetLink,
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
			want: map[string]any{
				"file": map[string]any{
					"id":               fakeID,
					"name":             "testdata.csv",
					"content":          "fake content",
					"created-time":     "2021-08-09T20:25:02.312Z",
					"modified-time":    "2021-09-17T16:58:37.924Z",
					"size":             0,
					"mime-type":        "application/vnd.google-apps.spreadsheet",
					"md5-checksum":     "7db67eab9238f9a63df30f570fda2bac",
					"version":          0,
					"web-view-link":    webViewSheetLink,
					"web-content-link": webContentSheetLink,
				},
			},
		},
		{
			name: "ok - read file Google doc file",
			in: map[string]any{
				"shared-link": sharedDocLink,
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
			want: map[string]any{
				"file": map[string]any{
					"id":            fakeID,
					"name":          "testdata.pdf",
					"content":       "fake content",
					"created-time":  "2021-08-09T20:25:02.312Z",
					"modified-time": "2021-09-17T16:58:37.924Z",
					"size":          0,
					"mime-type":     "application/vnd.google-apps.document",
					"md5-checksum":  "7db67eab9238f9a63df30f570fda2bac",
					"version":       0,
					"web-view-link": webViewDocLink,
				},
			},
		},
		{
			name: "ok - read file Google slide file",
			in: map[string]any{
				"shared-link": sharedSlideLink,
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
			want: map[string]any{
				"file": map[string]any{
					"id":            fakeID,
					"name":          "testdata.pdf",
					"content":       "fake content",
					"created-time":  "2021-08-09T20:25:02.312Z",
					"modified-time": "2021-09-17T16:58:37.924Z",
					"size":          0,
					"mime-type":     "application/vnd.google-apps.presentation",
					"md5-checksum":  "7db67eab9238f9a63df30f570fda2bac",
					"version":       0,
					"web-view-link": webViewSlideLink,
				},
			},
		},
		{
			name: "ok - read file",
			in: map[string]any{
				"shared-link": sharedFileLink,
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
			want: map[string]any{
				"file": map[string]any{
					"id":               fakeID,
					"name":             "testdata.png",
					"content":          "fake content",
					"created-time":     "2021-08-09T20:25:02.312Z",
					"modified-time":    "2021-09-17T16:58:37.924Z",
					"size":             0,
					"mime-type":        "image/jpeg",
					"md5-checksum":     "7db67eab9238f9a63df30f570fda2bac",
					"version":          0,
					"web-view-link":    webViewFileLink,
					"web-content-link": webContentFileLink,
				},
			},
		},
		{
			name: "nok - read file with invalid shared link",
			in: map[string]any{
				"shared-link": sharedFolderLink,
			},
			wantErr: "the input link is a folder link, please use the read-folder operation",
		},
	}

	bc := base.Component{}
	component := Init(bc)

	b, err := os.ReadFile("testdata/credentials.json")

	c.Assert(err, qt.IsNil)

	secrets := map[string]interface{}{
		"oauthcredentials": base64.StdEncoding.EncodeToString(b),
	}

	component = component.WithOAuthCredentials(secrets)

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

			pbIn, err := structpb.NewStruct(tc.in)
			c.Assert(err, qt.IsNil)

			ir, ow, eh, job := mock.GenerateMockJob(c)
			ir.ReadMock.Return(pbIn, nil)

			ow.WriteMock.Optional().Set(func(ctx context.Context, output *structpb.Struct) (err error) {
				gotJSON, err := output.MarshalJSON()

				c.Check(err, qt.IsNil)
				c.Check(gotJSON, qt.JSONEquals, tc.want)

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

func Test_Execute_ReadFolder(t *testing.T) {
	c := qt.New(t)
	mc := minimock.NewController(c)

	ctx := context.Background()

	testcases := []struct {
		name string

		in             map[string]any
		fakeDriveFiles []*drive.File
		fakeContents   []*string
		want           map[string]any
		wantErr        string
	}{
		{
			name: "ok - read folder with content",
			in: map[string]any{
				"shared-link":  sharedFolderLink,
				"read-content": true,
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
			want: map[string]any{
				"files": []map[string]any{
					{
						"id":               fakeID,
						"name":             "testdata.csv",
						"content":          "fake content",
						"created-time":     "2021-08-09T20:25:02.312Z",
						"modified-time":    "2021-09-17T16:58:37.924Z",
						"size":             0,
						"mime-type":        "application/vnd.google-apps.spreadsheet",
						"md5-checksum":     "7db67eab9238f9a63df30f570fda2bac",
						"version":          0,
						"web-view-link":    webViewSheetLink,
						"web-content-link": webContentSheetLink,
					},
				},
			},
		},
		{
			name: "ok - read folder without content",
			in: map[string]any{
				"shared-link":  sharedFolderLink,
				"read-content": false,
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
			want: map[string]any{
				"files": []map[string]any{
					{
						"id":               fakeID,
						"name":             "testdata.csv",
						"content":          "",
						"created-time":     "2021-08-09T20:25:02.312Z",
						"modified-time":    "2021-09-17T16:58:37.924Z",
						"size":             0,
						"mime-type":        "application/vnd.google-apps.spreadsheet",
						"md5-checksum":     "7db67eab9238f9a63df30f570fda2bac",
						"version":          0,
						"web-view-link":    webViewSheetLink,
						"web-content-link": webContentSheetLink,
					},
				},
			},
		},
		{
			name: "nok - read file",
			in: map[string]any{
				"shared-link":  sharedSheetLink,
				"read-content": false,
			},
			wantErr: "the input link is not a folder link, please check the link",
		},
	}

	bc := base.Component{}
	component := Init(bc)

	b, err := os.ReadFile("testdata/credentials.json")

	c.Assert(err, qt.IsNil)

	secrets := map[string]interface{}{
		"oauthcredentials": base64.StdEncoding.EncodeToString(b),
	}

	component = component.WithOAuthCredentials(secrets)

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

			readContent := tc.in["read-content"].(bool)

			if tc.wantErr == "" {
				mockDriveService.ReadFolderMock.
					Expect(fakeID, readContent).
					Return(fakeDriveFiles, fakeContents, nil)
			}

			pbIn, err := structpb.NewStruct(tc.in)
			c.Assert(err, qt.IsNil)

			ir, ow, eh, job := mock.GenerateMockJob(c)
			ir.ReadMock.Return(pbIn, nil)

			ow.WriteMock.Optional().Set(func(ctx context.Context, output *structpb.Struct) (err error) {
				gotJSON, err := output.MarshalJSON()

				c.Check(err, qt.IsNil)
				c.Check(gotJSON, qt.JSONEquals, tc.want)

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

	b, err := os.ReadFile("testdata/credentials.json")

	c.Assert(err, qt.IsNil)

	secrets := map[string]interface{}{
		"oauthcredentials": base64.StdEncoding.EncodeToString(b),
	}

	component = component.WithOAuthCredentials(secrets)

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
