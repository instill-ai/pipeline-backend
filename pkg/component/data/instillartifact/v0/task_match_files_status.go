package instillartifact

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"

	artifactPB "github.com/instill-ai/protogen-go/artifact/artifact/v1alpha"
)

func (e *execution) matchFileStatus(input *structpb.Struct) (*structpb.Struct, error) {

	inputStruct := MatchFileStatusInput{}
	err := base.ConvertFromStructpb(input, &inputStruct)
	if err != nil {
		return nil, fmt.Errorf("failed to convert input to struct: %w", err)
	}

	artifactClient, connection := e.client, e.connection

	defer connection.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	ctx = metadata.NewOutgoingContext(ctx, getRequestMetadata(e.SystemVariables))

	for {
		matchRes, err := artifactClient.ListCatalogFiles(ctx, &artifactPB.ListCatalogFilesRequest{
			NamespaceId: inputStruct.Namespace,
			CatalogId:   inputStruct.CatalogID,
			Filter: &artifactPB.ListCatalogFilesFilter{
				FileUids: []string{inputStruct.FileUID},
			},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to match file status: %w", err)
		}

		if matchRes.Files[0].ProcessStatus == artifactPB.FileProcessStatus_FILE_PROCESS_STATUS_COMPLETED {
			return base.ConvertToStructpb(MatchFileStatusOutput{
				Succeeded: true,
			})
		}

		if matchRes.Files[0].ProcessStatus == artifactPB.FileProcessStatus_FILE_PROCESS_STATUS_FAILED {
			return base.ConvertToStructpb(MatchFileStatusOutput{
				Succeeded: false,
			})
		}
	}
}
