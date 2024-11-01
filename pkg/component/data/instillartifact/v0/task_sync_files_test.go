package instillartifact

// TODO: Add more test cases for error handling.

import (
	"context"
	"encoding/json"
	"net"
	"testing"

	"github.com/gojuno/minimock/v3"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/mock"

	artifactPB "github.com/instill-ai/protogen-go/artifact/artifact/v1alpha"
)

var (
	namespace = "namespace123"
	catalogID = "catalog-id123"
	// To be uploaded file
	catalogFileUID = "catalog-file-uid123"
	fileID         = "file-id123"
	fileName       = "file-name123.pdf"
	// To be updated file
	catalogFileUID2 = "catalog-file-uid456"
	fileID2         = "file-id456"
	fileName2       = "file-name456.pdf"
	fakeInput       = map[string]interface{}{
		"namespace":  namespace,
		"catalog-id": catalogID,
		"third-party-files": []map[string]interface{}{
			{
				"id":               fileID,
				"name":             fileName,
				"content":          "dGVzdDEyMw==",
				"created-time":     "2021-08-04T00:00:00Z",
				"modified-time":    "2021-08-04T00:00:00Z",
				"size":             123,
				"mime-type":        "text/plain",
				"md5-checksum":     "30bd471c9356ac2397fe96491e83f470",
				"version":          1,
				"web-view-link":    "https://drive.google.com/file/d/fakeuuid/view?usp=drivesdk",
				"web-content-link": "https://drive.google.com/uc?id=fakeuuid&export=download",
			},
			{
				"id":               fileID2,
				"name":             fileName2,
				"content":          "xGVzdDEyMw==",
				"created-time":     "2021-08-02T00:00:00Z",
				"modified-time":    "2021-08-03T00:00:00Z",
				"size":             123,
				"mime-type":        "text/plain",
				"md5-checksum":     "30bd471c9356ac2397fe96491e83f471",
				"version":          2,
				"web-view-link":    "https://drive.google.com/file/d/fakeuuid2/view?usp=drivesdk",
				"web-content-link": "https://drive.google.com/uc?id=fakeuuid2&export=download",
			},
		},
	}
)

// Test_ExecuteSyncFiles tests the Execute method of the component with the taskSyncFiles task.
func Test_ExecuteSyncFiles(t *testing.T) {
	c := qt.New(t)
	ctx := context.Background()
	mc := minimock.NewController(c)

	grpcServer := grpc.NewServer()
	testServer := mock.NewArtifactPublicServiceServerMock(mc)

	setListCatalogsMock(testServer, c)
	setListCatalogFilesMock(testServer, c)
	setDeleteCatalogFileMock(testServer, c)
	setUploadCatalogFileMock(testServer, c)
	setProcessCatalogFilesMock(testServer, c)

	artifactPB.RegisterArtifactPublicServiceServer(grpcServer, testServer)
	lis, err := net.Listen("tcp", ":0")
	c.Assert(err, qt.IsNil)

	go func() {
		err := grpcServer.Serve(lis)
		c.Assert(err, qt.IsNil)
	}()
	defer grpcServer.Stop()

	mockAddress := lis.Addr().String()

	bc := base.Component{}

	comp := Init(bc)

	task := taskSyncFiles

	x, err := comp.CreateExecution(base.ComponentExecution{
		Task: task,
		SystemVariables: map[string]any{
			"__ARTIFACT_BACKEND":              mockAddress,
			"__PIPELINE_HEADER_AUTHORIZATION": "Bearer inst token",
			"__PIPELINE_USER_UID":             "user1",
			"__PIPELINE_REQUESTER_UID":        "requester1",
		},
	})

	c.Assert(err, qt.IsNil)

	jsonData, err := json.Marshal(fakeInput)

	c.Assert(err, qt.IsNil)

	pbIn := new(structpb.Struct)

	err = protojson.Unmarshal(jsonData, pbIn)

	c.Assert(err, qt.IsNil)

	ir, ow, eh, job := mock.GenerateMockJob(c)

	ir.ReadMock.Return(pbIn, nil)
	ow.WriteMock.Set(func(ctx context.Context, output *structpb.Struct) error {
		uploadedFiles, ok := output.Fields["uploaded-files"]

		mock.Equal(ok, true)
		mock.Equal(len(uploadedFiles.GetListValue().Values), 1)

		updatedFiles, ok := output.Fields["updated-files"]

		mock.Equal(ok, true)
		mock.Equal(len(updatedFiles.GetListValue().Values), 1)

		failureFiles, ok := output.Fields["failure-files"]

		mock.Equal(ok, true)
		mock.Equal(len(failureFiles.GetListValue().Values), 0)

		errorMessages, ok := output.Fields["error-messages"]

		mock.Equal(ok, true)
		mock.Equal(len(errorMessages.GetListValue().Values), 0)

		status, ok := output.Fields["status"]

		mock.Equal(ok, true)
		mock.Equal(status.GetBoolValue(), true)

		return nil
	})

	eh.ErrorMock.Optional().Set(func(ctx context.Context, err error) {
		mock.Nil(err)
	})

	err = x.Execute(ctx, []*base.Job{job})
	c.Assert(err, qt.IsNil)
}

// Mock Server functions section
var (
	catalogFileUID3 = "catalog-file-uid789"
	fileID3         = "file-id789"
	fileName3       = "file-name789.pdf"

	// To be updated file
	fakeMetadata2 = map[string]interface{}{
		"id":               fileID2,
		"name":             fileName2,
		"content":          "xGVzdDEyMw==",
		"created-time":     "2021-08-02T00:00:00Z",
		"modified-time":    "2021-08-02T00:00:00Z",
		"size":             123,
		"mime-type":        "text/plain",
		"md5-checksum":     "30bd471c9356ac2397fe96491e83f471",
		"version":          1,
		"web-view-link":    "https://drive.google.com/file/d/fakeuuid2/view?usp=drivesdk",
		"web-content-link": "https://drive.google.com/uc?id=fakeuuid2&export=download",
	}

	// To be deleted file
	fakeMetadata3 = map[string]interface{}{
		"id":               fileID3,
		"name":             fileName3,
		"content":          "xDVzdDEyMw==",
		"created-time":     "2021-08-02T00:00:00Z",
		"modified-time":    "2021-08-02T00:00:00Z",
		"size":             123,
		"mime-type":        "text/plain",
		"md5-checksum":     "30bd471c9356ac2397fe96491e83f471",
		"version":          1,
		"web-view-link":    "https://drive.google.com/file/d/fakeuuid2/view?usp=drivesdk",
		"web-content-link": "https://drive.google.com/uc?id=fakeuuid2&export=download",
	}
)

func setListCatalogsMock(s *mock.ArtifactPublicServiceServerMock, c *qt.C) {
	s.ListCatalogsMock.Set(func(ctx context.Context, in *artifactPB.ListCatalogsRequest) (*artifactPB.ListCatalogsResponse, error) {
		mock.Equal(in.NamespaceId, namespace)
		return &artifactPB.ListCatalogsResponse{
			Catalogs: []*artifactPB.Catalog{
				{
					CatalogId: catalogID,
					Name:      catalogID,
				},
			},
		}, nil
	})
}

func setListCatalogFilesMock(s *mock.ArtifactPublicServiceServerMock, c *qt.C) {
	s.ListCatalogFilesMock.Set(func(ctx context.Context, in *artifactPB.ListCatalogFilesRequest) (*artifactPB.ListCatalogFilesResponse, error) {
		mock.Equal(in.CatalogId, catalogID)

		metadataStruct2 := new(structpb.Struct)
		jsonData, err := json.Marshal(fakeMetadata2)
		mock.Nil(err)
		err = protojson.Unmarshal(jsonData, metadataStruct2)
		mock.Nil(err)

		metadataStruct3 := new(structpb.Struct)
		jsonData, err = json.Marshal(fakeMetadata3)
		mock.Nil(err)
		err = protojson.Unmarshal(jsonData, metadataStruct3)
		mock.Nil(err)

		return &artifactPB.ListCatalogFilesResponse{
			Files: []*artifactPB.File{
				{
					FileUid:          catalogFileUID2,
					ExternalMetadata: metadataStruct2,
				},
				{
					FileUid:          catalogFileUID3,
					ExternalMetadata: metadataStruct3,
				},
			},
		}, nil
	})
}

func setDeleteCatalogFileMock(s *mock.ArtifactPublicServiceServerMock, c *qt.C) {
	s.DeleteCatalogFileMock.Times(2).Set(func(ctx context.Context, in *artifactPB.DeleteCatalogFileRequest) (*artifactPB.DeleteCatalogFileResponse, error) {
		mock.Contains([]string{catalogFileUID2, catalogFileUID3}, in.FileUid)

		return &artifactPB.DeleteCatalogFileResponse{}, nil
	})

}

func setUploadCatalogFileMock(s *mock.ArtifactPublicServiceServerMock, c *qt.C) {
	s.UploadCatalogFileMock.Times(2).Set(func(ctx context.Context, in *artifactPB.UploadCatalogFileRequest) (*artifactPB.UploadCatalogFileResponse, error) {
		mock.Equal(in.NamespaceId, namespace)
		mock.Equal(in.CatalogId, catalogID)
		mock.Equal(in.File.Type, artifactPB.FileType_FILE_TYPE_PDF)
		mock.NotNil(in.File.Content)
		mock.NotNil(in.File.ExternalMetadata)

		var mockFileUID string
		if in.File.Name == fileName {
			mockFileUID = catalogFileUID
		} else if in.File.Name == fileName2 {
			mockFileUID = catalogFileUID2
		} else {
			panic("Unexpected file name")
		}

		return &artifactPB.UploadCatalogFileResponse{
			File: &artifactPB.File{
				FileUid:          mockFileUID,
				Name:             in.File.Name,
				Type:             in.File.Type,
				CreateTime:       in.File.CreateTime,
				UpdateTime:       in.File.UpdateTime,
				Size:             in.File.Size,
				ExternalMetadata: in.File.ExternalMetadata,
			},
		}, nil
	})
}

func setProcessCatalogFilesMock(s *mock.ArtifactPublicServiceServerMock, c *qt.C) {
	s.ProcessCatalogFilesMock.Set(func(ctx context.Context, in *artifactPB.ProcessCatalogFilesRequest) (*artifactPB.ProcessCatalogFilesResponse, error) {
		mock.DeepEquals(in.FileUids, []string{catalogFileUID, catalogFileUID2})

		return &artifactPB.ProcessCatalogFilesResponse{}, nil
	})
}
