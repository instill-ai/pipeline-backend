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

	artifactpb "github.com/instill-ai/protogen-go/artifact/v1alpha"
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
		"namespace":         namespace,
		"knowledge-base-id": catalogID,
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

	setListCatalogsMock(testServer)
	setListCatalogFilesMock(testServer)
	setDeleteCatalogFileMock(testServer)
	setUploadCatalogFileMock(testServer)
	setProcessCatalogFilesMock(testServer)

	artifactpb.RegisterArtifactPublicServiceServer(grpcServer, testServer)
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

func setListCatalogsMock(s *mock.ArtifactPublicServiceServerMock) {
	s.ListKnowledgeBasesMock.Set(func(ctx context.Context, in *artifactpb.ListKnowledgeBasesRequest) (*artifactpb.ListKnowledgeBasesResponse, error) {
		mock.Equal(in.Parent, "namespaces/"+namespace)
		return &artifactpb.ListKnowledgeBasesResponse{
			KnowledgeBases: []*artifactpb.KnowledgeBase{
				{
					Id:   catalogID,
					Name: "namespaces/" + namespace + "/knowledge-bases/" + catalogID,
				},
			},
		}, nil
	})
}

func setListCatalogFilesMock(s *mock.ArtifactPublicServiceServerMock) {
	s.ListFilesMock.Set(func(ctx context.Context, in *artifactpb.ListFilesRequest) (*artifactpb.ListFilesResponse, error) {
		mock.Equal(in.Parent, "namespaces/"+namespace)

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

		return &artifactpb.ListFilesResponse{
			Files: []*artifactpb.File{
				{
					Id:               catalogFileUID2,
					ExternalMetadata: metadataStruct2,
				},
				{
					Id:               catalogFileUID3,
					ExternalMetadata: metadataStruct3,
				},
			},
		}, nil
	})
}

func setDeleteCatalogFileMock(s *mock.ArtifactPublicServiceServerMock) {
	s.DeleteFileMock.Times(2).Set(func(ctx context.Context, in *artifactpb.DeleteFileRequest) (*artifactpb.DeleteFileResponse, error) {
		// Extract file ID from resource name: namespaces/{namespace}/files/{file}
		expectedNames := []string{
			"namespaces/" + namespace + "/files/" + catalogFileUID2,
			"namespaces/" + namespace + "/files/" + catalogFileUID3,
		}
		mock.Contains(expectedNames, in.Name)

		return &artifactpb.DeleteFileResponse{}, nil
	})

}

func setUploadCatalogFileMock(s *mock.ArtifactPublicServiceServerMock) {
	s.CreateFileMock.Times(2).Set(func(ctx context.Context, in *artifactpb.CreateFileRequest) (*artifactpb.CreateFileResponse, error) {
		mock.Equal(in.Parent, "namespaces/"+namespace)
		mock.NotNil(in.File)
		mock.NotNil(in.File.Content)
		mock.NotNil(in.File.ExternalMetadata)

		var mockFileUID string
		switch in.File.DisplayName {
		case fileName:
			mockFileUID = catalogFileUID
		case fileName2:
			mockFileUID = catalogFileUID2
		default:
			panic("Unexpected file name")
		}

		return &artifactpb.CreateFileResponse{
			File: &artifactpb.File{
				Id:               mockFileUID,
				DisplayName:      in.File.DisplayName,
				Type:             artifactpb.File_TYPE_PDF,
				CreateTime:       nil,
				UpdateTime:       nil,
				Size:             123,
				ExternalMetadata: in.File.ExternalMetadata,
			},
		}, nil
	})
}

func setProcessCatalogFilesMock(s *mock.ArtifactPublicServiceServerMock) {
	// ProcessCatalogFiles API has been removed - files now auto-process on creation
	// This function is kept empty for compatibility but does nothing
}
