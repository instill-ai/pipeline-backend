package instill

import (
	"google.golang.org/protobuf/types/known/structpb"

	modelPB "github.com/instill-ai/protogen-go/model/model/v1alpha"
)

func (e *execution) executeEmbedding(grpcClient modelPB.ModelPublicServiceClient, nsID string, modelID string, version string, inputs []*structpb.Struct) ([]*structpb.Struct, error) {
	return nil, nil
}
