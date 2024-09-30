package instillartifact

import (
	"testing"

	"github.com/frankban/quicktest"
	"github.com/gojuno/minimock/v3"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/mock"

	artifactPB "github.com/instill-ai/protogen-go/artifact/artifact/v1alpha"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
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
			Namespace: "fakeNs",
			CatalogID: "fakeID",
		}

		inputStruct, _ := base.ConvertToStructpb(input)

		clientMock := mock.NewArtifactPublicServiceClientMock(mc)

		clientMock.ListCatalogFilesMock.
			Expect(minimock.AnyContext, &artifactPB.ListCatalogFilesRequest{
				NamespaceId: "fakeNs",
				CatalogId:   "fakeID"}).
			Times(1).
			Return(&artifactPB.ListCatalogFilesResponse{
				Files: []*artifactPB.File{
					{
						FileUid: "fakeFileID",
						Name:    "fakeFileName",
						Type:    artifactPB.FileType_FILE_TYPE_PDF,
						Size:    1,
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
		c.Assert(outputStruct.Files[0].FileType, quicktest.Equals, "FILE_TYPE_PDF")
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
			Namespace: "fakeNs",
			CatalogID: "fakeID",
			FileUID:   "fakeFileID",
		}

		inputStruct, _ := base.ConvertToStructpb(input)

		clientMock := mock.NewArtifactPublicServiceClientMock(mc)

		clientMock.ListChunksMock.Expect(minimock.AnyContext, &artifactPB.ListChunksRequest{
			NamespaceId: "fakeNs",
			CatalogId:   "fakeID",
			FileUid:     "fakeFileID",
		}).Times(1).Return(&artifactPB.ListChunksResponse{
			Chunks: []*artifactPB.Chunk{
				{
					ChunkUid:    "fakeChunkID",
					Retrievable: true,
					StartPos:    0,
					EndPos:      1,
					Tokens:      1,
					CreateTime: &timestamppb.Timestamp{
						Seconds: 1,
						Nanos:   1,
					},
					OriginalFileUid: "fakeFileID",
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
		c.Assert(outputStruct.Chunks[0].EndPosition, quicktest.Equals, uint32(1))
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
			Namespace: "fakeNs",
			CatalogID: "fakeID",
			FileUID:   "fakeFileID",
		}

		inputStruct, _ := base.ConvertToStructpb(input)

		clientMock := mock.NewArtifactPublicServiceClientMock(mc)

		clientMock.GetSourceFileMock.Expect(minimock.AnyContext, &artifactPB.GetSourceFileRequest{
			NamespaceId: "fakeNs",
			CatalogId:   "fakeID",
			FileUid:     "fakeFileID",
		}).Times(1).Return(&artifactPB.GetSourceFileResponse{
			SourceFile: &artifactPB.SourceFile{
				OriginalFileUid: "fakeFileID",
				Content:         "fakeContent",
				CreateTime: &timestamppb.Timestamp{
					Seconds: 1,
					Nanos:   1,
				},
				UpdateTime: &timestamppb.Timestamp{
					Seconds: 1,
					Nanos:   1,
				},
			},
		}, nil)

		e.client = clientMock
		e.connection = fakeConnection{}

		output, err := e.execute(inputStruct)

		c.Assert(err, quicktest.IsNil)

		var outputStruct GetFileInMarkdownOutput

		err = base.ConvertFromStructpb(output, &outputStruct)

		c.Assert(err, quicktest.IsNil)

		c.Assert(outputStruct.OriginalFileUID, quicktest.Equals, "fakeFileID")
		c.Assert(outputStruct.Content, quicktest.Equals, "fakeContent")
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
			Namespace:  "fakeNs",
			CatalogID:  "fakeID",
			TextPrompt: "fakePrompt",
			TopK:       1,
		}

		inputStruct, _ := base.ConvertToStructpb(input)

		clientMock := mock.NewArtifactPublicServiceClientMock(mc)

		clientMock.SimilarityChunksSearchMock.
			Expect(minimock.AnyContext, &artifactPB.SimilarityChunksSearchRequest{
				NamespaceId: "fakeNs",
				CatalogId:   "fakeID",
				TextPrompt:  "fakePrompt",
				TopK:        1,
			}).
			Times(1).
			Return(&artifactPB.SimilarityChunksSearchResponse{
				SimilarChunks: []*artifactPB.SimilarityChunk{
					{
						ChunkUid:        "fakeChunkID",
						SimilarityScore: float32(1),
						TextContent:     "fakeContent",
						SourceFile:      "fakeFile",
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

		c.Assert(outputStruct.Chunks[0].ChunkUID, quicktest.Equals, "fakeChunkID")
		c.Assert(outputStruct.Chunks[0].SimilarityScore, quicktest.Equals, float32(1))
		c.Assert(outputStruct.Chunks[0].TextContent, quicktest.Equals, "fakeContent")
		c.Assert(outputStruct.Chunks[0].SourceFileName, quicktest.Equals, "fakeFile")
	})
}

func Test_query(t *testing.T) {
	c := quicktest.New(t)
	mc := minimock.NewController(t)

	c.Run("query", func(c *quicktest.C) {
		component := Init(base.Component{})

		sysVar := map[string]interface{}{
			"__ARTIFACT_BACKEND":       "http://localhost:8082",
			"__PIPELINE_USER_UID":      "fakeUser",
			"__PIPELINE_REQUESTER_UID": "fakeRequester",
		}

		e := &execution{
			ComponentExecution: base.ComponentExecution{Component: component, SystemVariables: sysVar, Setup: nil, Task: taskQuery},
		}

		e.execute = e.query

		input := QueryInput{
			Namespace: "fakeNs",
			CatalogID: "fakeID",
			Question:  "fakeQuestion",
			TopK:      1,
		}

		inputStruct, _ := base.ConvertToStructpb(input)

		clientMock := mock.NewArtifactPublicServiceClientMock(mc)

		clientMock.QuestionAnsweringMock.
			Expect(minimock.AnyContext, &artifactPB.QuestionAnsweringRequest{
				NamespaceId: "fakeNs",
				CatalogId:   "fakeID",
				Question:    "fakeQuestion",
				TopK:        1,
			}).
			Times(1).
			Return(&artifactPB.QuestionAnsweringResponse{
				Answer: "fakeAnswer",
				SimilarChunks: []*artifactPB.SimilarityChunk{
					{
						ChunkUid:        "fakeChunkID",
						SimilarityScore: float32(1),
						TextContent:     "fakeContent",
						SourceFile:      "fakeFile",
					},
				},
			}, nil)

		e.client = clientMock
		e.connection = fakeConnection{}

		output, err := e.execute(inputStruct)

		c.Assert(err, quicktest.IsNil)

		var outputStruct QueryOutput
		err = base.ConvertFromStructpb(output, &outputStruct)

		c.Assert(err, quicktest.IsNil)

		c.Assert(outputStruct.Answer, quicktest.Equals, "fakeAnswer")
		c.Assert(outputStruct.Chunks[0].ChunkUID, quicktest.Equals, "fakeChunkID")
		c.Assert(outputStruct.Chunks[0].SimilarityScore, quicktest.Equals, float32(1))
		c.Assert(outputStruct.Chunks[0].TextContent, quicktest.Equals, "fakeContent")
		c.Assert(outputStruct.Chunks[0].SourceFileName, quicktest.Equals, "fakeFile")

	})

}

func Test_matchFileStatus(t *testing.T) {
	c := quicktest.New(t)
	mc := minimock.NewController(t)

	testCases := []struct {
		name     string
		status   artifactPB.FileProcessStatus
		expected bool
	}{
		{
			name:     "process status completed",
			status:   artifactPB.FileProcessStatus_FILE_PROCESS_STATUS_COMPLETED,
			expected: true,
		},
		{
			name:     "process status failed",
			status:   artifactPB.FileProcessStatus_FILE_PROCESS_STATUS_FAILED,
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
				Namespace: "fakeNs",
				CatalogID: "fakeID",
				FileUID:   "fakeFileID",
			}

			inputStruct, _ := base.ConvertToStructpb(input)

			clientMock := mock.NewArtifactPublicServiceClientMock(mc)

			clientMock.ListCatalogFilesMock.
				Expect(minimock.AnyContext, &artifactPB.ListCatalogFilesRequest{
					NamespaceId: "fakeNs",
					CatalogId:   "fakeID",
					Filter: &artifactPB.ListCatalogFilesFilter{
						FileUids: []string{"fakeFileID"},
					},
				}).
				Times(1).
				Return(&artifactPB.ListCatalogFilesResponse{
					Files: []*artifactPB.File{
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
