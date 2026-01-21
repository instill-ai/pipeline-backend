package instillartifact

// TODO: Make the test against the fake server rather than mocking the client interface.

import (
	"fmt"
	"testing"

	"github.com/frankban/quicktest"
	"github.com/gofrs/uuid"
	"github.com/gojuno/minimock/v3"

	timestamppb "google.golang.org/protobuf/types/known/timestamppb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/mock"

	artifactpb "github.com/instill-ai/protogen-go/artifact/v1alpha"
)

func Test_getFilesMetadata(t *testing.T) {
	c := quicktest.New(t)
	mc := minimock.NewController(t)

	c.Run("get files metadata", func(c *quicktest.C) {
		component := Init(base.Component{})

		sysVar := map[string]interface{}{
			"__ARTIFACT_BACKEND":       "http://localhost:8082",
			"__PIPELINE_USER_UID":      "fakeUser",
			"__PIPELINE_REQUESTER_UID": "fakeRequester",
		}

		e := &execution{
			ComponentExecution: base.ComponentExecution{Component: component, SystemVariables: sysVar, Setup: nil, Task: taskGetFilesMetadata},
		}

		e.execute = e.getFilesMetadata

		input := GetFilesMetadataInput{
			Namespace:       "fakeNs",
			KnowledgeBaseID: "fakeID",
		}

		inputStruct, _ := base.ConvertToStructpb(input)

		clientMock := mock.NewArtifactPublicServiceClientMock(mc)

		filter := `knowledgeBaseId="fakeID"`
		clientMock.ListFilesMock.
			Expect(minimock.AnyContext, &artifactpb.ListFilesRequest{
				Parent: "namespaces/fakeNs",
				Filter: &filter}).
			Times(1).
			Return(&artifactpb.ListFilesResponse{
				Files: []*artifactpb.File{
					{
						Id:          "fakeFileID",
						DisplayName: "fakeFileName",
						Type:        artifactpb.File_TYPE_PDF,
						Size:        1,
						CreateTime: &timestamppb.Timestamp{
							Seconds: 1,
							Nanos:   1,
						},
						UpdateTime: &timestamppb.Timestamp{
							Seconds: 1,
							Nanos:   1,
						},
					},
				},
			}, nil)

		e.client = clientMock
		e.connection = fakeConnection{}

		output, err := e.execute(inputStruct)

		c.Assert(err, quicktest.IsNil)

		var outputStruct GetFilesMetadataOutput
		err = base.ConvertFromStructpb(output, &outputStruct)

		c.Assert(err, quicktest.IsNil)

		c.Assert(len(outputStruct.Files), quicktest.Equals, 1)
		c.Assert(outputStruct.Files[0].FileUID, quicktest.Equals, "fakeFileID")
		c.Assert(outputStruct.Files[0].FileType, quicktest.Equals, "TYPE_PDF")
		c.Assert(outputStruct.Files[0].FileName, quicktest.Equals, "fakeFileName")
		c.Assert(outputStruct.Files[0].Size, quicktest.Equals, int64(1))
		c.Assert(outputStruct.Files[0].CreateTime, quicktest.Equals, "1970-01-01T00:00:01Z")
		c.Assert(outputStruct.Files[0].UpdateTime, quicktest.Equals, "1970-01-01T00:00:01Z")

	})
}

func Test_getChunksMetadata(t *testing.T) {
	c := quicktest.New(t)
	mc := minimock.NewController(t)

	c.Run("get chunks metadata", func(c *quicktest.C) {
		component := Init(base.Component{})

		sysVar := map[string]interface{}{
			"__ARTIFACT_BACKEND":       "http://localhost:8082",
			"__PIPELINE_USER_UID":      "fakeUser",
			"__PIPELINE_REQUESTER_UID": "fakeRequester",
		}

		e := &execution{
			ComponentExecution: base.ComponentExecution{Component: component, SystemVariables: sysVar, Setup: nil, Task: taskGetChunksMetadata},
		}

		e.execute = e.getChunksMetadata

		input := GetChunksMetadataInput{
			Namespace:       "fakeNs",
			KnowledgeBaseID: "fakeID",
			FileUID:         "fakeFileID",
		}

		inputStruct, _ := base.ConvertToStructpb(input)

		clientMock := mock.NewArtifactPublicServiceClientMock(mc)

		clientMock.ListChunksMock.Expect(minimock.AnyContext, &artifactpb.ListChunksRequest{
			Parent: "namespaces/fakeNs/files/fakeFileID",
		}).Times(1).Return(&artifactpb.ListChunksResponse{
			Chunks: []*artifactpb.Chunk{
				{
					Id:          "fakeChunkID",
					Retrievable: true,
					Tokens:      1,
					CreateTime: &timestamppb.Timestamp{
						Seconds: 1,
						Nanos:   1,
					},
					OriginalFileId: "fakeFileID",
				},
			},
		}, nil)

		e.client = clientMock
		e.connection = fakeConnection{}

		output, err := e.execute(inputStruct)

		c.Assert(err, quicktest.IsNil)

		var outputStruct GetChunksMetadataOutput
		err = base.ConvertFromStructpb(output, &outputStruct)

		c.Assert(err, quicktest.IsNil)

		c.Assert(len(outputStruct.Chunks), quicktest.Equals, 1)

		c.Assert(outputStruct.Chunks[0].ChunkUID, quicktest.Equals, "fakeChunkID")
		c.Assert(outputStruct.Chunks[0].Retrievable, quicktest.Equals, true)
		c.Assert(outputStruct.Chunks[0].StartPosition, quicktest.Equals, uint32(0))
		c.Assert(outputStruct.Chunks[0].EndPosition, quicktest.Equals, uint32(0))
		c.Assert(outputStruct.Chunks[0].TokenCount, quicktest.Equals, uint32(1))
		c.Assert(outputStruct.Chunks[0].CreateTime, quicktest.Equals, "1970-01-01T00:00:01Z")
		c.Assert(outputStruct.Chunks[0].OriginalFileUID, quicktest.Equals, "fakeFileID")

	})

}

func Test_getFileInMarkdown(t *testing.T) {
	c := quicktest.New(t)
	mc := minimock.NewController(t)

	c.Run("get file in markdown", func(c *quicktest.C) {
		component := Init(base.Component{})

		sysVar := map[string]interface{}{
			"__ARTIFACT_BACKEND":       "http://localhost:8082",
			"__PIPELINE_USER_UID":      "fakeUser",
			"__PIPELINE_REQUESTER_UID": "fakeRequester",
		}

		e := &execution{
			ComponentExecution: base.ComponentExecution{Component: component, SystemVariables: sysVar, Setup: nil, Task: taskGetFileInMarkdown},
		}

		e.execute = e.getFileInMarkdown

		input := GetFileInMarkdownInput{
			Namespace:       "fakeNs",
			KnowledgeBaseID: "fakeID",
			FileUID:         "fakeFileID",
		}

		inputStruct, _ := base.ConvertToStructpb(input)

		clientMock := mock.NewArtifactPublicServiceClientMock(mc)

		// Mock GetFile to return file metadata with empty derived_resource_uri
		// (in real usage, it would have a MinIO URL, but for test we'll return empty content)
		clientMock.GetFileMock.Expect(minimock.AnyContext, &artifactpb.GetFileRequest{
			Name: "namespaces/fakeNs/files/fakeFileID",
			View: artifactpb.File_VIEW_CONTENT.Enum(),
		}).Times(1).Return(&artifactpb.GetFileResponse{
			File: &artifactpb.File{
				Id:          "fakeFileID",
				DisplayName: "fakeFileName",
				CreateTime: &timestamppb.Timestamp{
					Seconds: 1,
					Nanos:   1,
				},
				UpdateTime: &timestamppb.Timestamp{
					Seconds: 1,
					Nanos:   1,
				},
			},
			DerivedResourceUri: new(string),
		}, nil)

		e.client = clientMock
		e.connection = fakeConnection{}

		output, err := e.execute(inputStruct)

		c.Assert(err, quicktest.IsNil)

		var outputStruct GetFileInMarkdownOutput

		err = base.ConvertFromStructpb(output, &outputStruct)

		c.Assert(err, quicktest.IsNil)

		c.Assert(outputStruct.OriginalFileUID, quicktest.Equals, "fakeFileID")
		c.Assert(outputStruct.Content, quicktest.Equals, "") // Empty since no derived_resource_uri
		c.Assert(outputStruct.CreateTime, quicktest.Equals, "1970-01-01T00:00:01Z")
		c.Assert(outputStruct.UpdateTime, quicktest.Equals, "1970-01-01T00:00:01Z")

	})

}

func Test_searchChunks(t *testing.T) {
	c := quicktest.New(t)
	mc := minimock.NewController(t)

	c.Run("search chunks", func(c *quicktest.C) {
		component := Init(base.Component{})

		sysVar := map[string]interface{}{
			"__ARTIFACT_BACKEND":       "http://localhost:8082",
			"__PIPELINE_USER_UID":      "fakeUser",
			"__PIPELINE_REQUESTER_UID": "fakeRequester",
		}

		e := &execution{
			ComponentExecution: base.ComponentExecution{Component: component, SystemVariables: sysVar, Setup: nil, Task: taskSearchChunks},
		}

		e.execute = e.searchChunks

		input := SearchChunksInput{
			Namespace:       "fakeNs",
			KnowledgeBaseID: "fakeID",
			TextPrompt:      "fakePrompt",
			TopK:            1,
		}

		inputStruct, _ := base.ConvertToStructpb(input)

		clientMock := mock.NewArtifactPublicServiceClientMock(mc)

		fileUID := uuid.Must(uuid.NewV4())
		pageTwo := &artifactpb.File_Position{
			Unit:        artifactpb.File_Position_UNIT_PAGE,
			Coordinates: []uint32{2},
		}

		clientMock.SearchChunksMock.
			Expect(minimock.AnyContext, &artifactpb.SearchChunksRequest{
				Parent:          "namespaces/fakeNs",
				KnowledgeBaseId: "fakeID",
				TextPrompt:      "fakePrompt",
				TopK:            1,
			}).
			Times(1).
			Return(&artifactpb.SearchChunksResponse{
				SimilarChunks: []*artifactpb.SimilarityChunk{
					{
						Chunk:           "namespaces/fakeNs/files/fakeFileID/chunks/fakeChunkID",
						SimilarityScore: float32(1),
						TextContent:     "fakeContent",
						File:            "namespaces/fakeNs/files/fakeFileID",
						ChunkMetadata: &artifactpb.Chunk{
							OriginalFileId: fileUID.String(),
							MarkdownReference: &artifactpb.Chunk_Reference{
								Start: pageTwo,
								End:   pageTwo,
							},
						},
					},
				},
			}, nil)

		e.client = clientMock
		e.connection = fakeConnection{}

		output, err := e.execute(inputStruct)

		c.Assert(err, quicktest.IsNil)

		var outputStruct SearchChunksOutput
		err = base.ConvertFromStructpb(output, &outputStruct)

		c.Assert(err, quicktest.IsNil)

		c.Assert(len(outputStruct.Chunks), quicktest.Equals, 1)

		c.Assert(outputStruct.Chunks[0].ChunkUID, quicktest.Equals, "namespaces/fakeNs/files/fakeFileID/chunks/fakeChunkID")
		c.Assert(outputStruct.Chunks[0].SimilarityScore, quicktest.Equals, float32(1))
		c.Assert(outputStruct.Chunks[0].TextContent, quicktest.Equals, "fakeContent")
		c.Assert(outputStruct.Chunks[0].SourceFileName, quicktest.Equals, "namespaces/fakeNs/files/fakeFileID")
		c.Assert(outputStruct.Chunks[0].Reference.Start.Unit, quicktest.Equals, "UNIT_PAGE")
		c.Assert(outputStruct.Chunks[0].Reference.Start.Coordinates, quicktest.ContentEquals, pageTwo.Coordinates)
		c.Assert(outputStruct.Chunks[0].Reference.End.Unit, quicktest.Equals, "UNIT_PAGE")
		c.Assert(outputStruct.Chunks[0].Reference.End.Coordinates, quicktest.ContentEquals, pageTwo.Coordinates)
	})
}

func Test_matchFileStatus(t *testing.T) {
	c := quicktest.New(t)
	mc := minimock.NewController(t)

	testCases := []struct {
		name     string
		status   artifactpb.FileProcessStatus
		expected bool
	}{
		{
			name:     "process status completed",
			status:   artifactpb.FileProcessStatus_FILE_PROCESS_STATUS_COMPLETED,
			expected: true,
		},
		{
			name:     "process status failed",
			status:   artifactpb.FileProcessStatus_FILE_PROCESS_STATUS_FAILED,
			expected: false,
		},
	}

	for _, tc := range testCases {

		c.Run("match file status", func(c *quicktest.C) {
			component := Init(base.Component{})

			sysVar := map[string]interface{}{
				"__ARTIFACT_BACKEND":       "http://localhost:8082",
				"__PIPELINE_USER_UID":      "fakeUser",
				"__PIPELINE_REQUESTER_UID": "fakeRequester",
			}

			e := &execution{
				ComponentExecution: base.ComponentExecution{Component: component, SystemVariables: sysVar, Setup: nil, Task: taskMatchFileStatus},
			}

			e.execute = e.matchFileStatus

			input := MatchFileStatusInput{
				Namespace:       "fakeNs",
				KnowledgeBaseID: "fakeID",
				FileUID:         "fakeFileID",
			}

			inputStruct, _ := base.ConvertToStructpb(input)

			clientMock := mock.NewArtifactPublicServiceClientMock(mc)

			filter := fmt.Sprintf(`id="%s" AND knowledgeBaseId="%s"`, "fakeFileID", "fakeID")
			clientMock.ListFilesMock.
				Expect(minimock.AnyContext, &artifactpb.ListFilesRequest{
					Parent: "namespaces/fakeNs",
					Filter: &filter,
				}).
				Times(1).
				Return(&artifactpb.ListFilesResponse{
					Files: []*artifactpb.File{
						{
							ProcessStatus: tc.status,
						},
					},
				}, nil)

			e.client = clientMock
			e.connection = fakeConnection{}

			output, err := e.execute(inputStruct)

			c.Assert(err, quicktest.IsNil)

			var outputStruct MatchFileStatusOutput

			err = base.ConvertFromStructpb(output, &outputStruct)

			c.Assert(err, quicktest.IsNil)

			c.Assert(outputStruct.Succeeded, quicktest.Equals, tc.expected)

		})

	}

}

type fakeConnection struct{}

func (f fakeConnection) Close() error {
	return nil
}
