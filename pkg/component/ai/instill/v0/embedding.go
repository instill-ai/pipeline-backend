package instill

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/structpb"

	modelPB "github.com/instill-ai/protogen-go/model/model/v1alpha"
)

func (e *execution) executeEmbedding(grpcClient modelPB.ModelPublicServiceClient, nsID string, modelID string, version string, inputs []*structpb.Struct) ([]*structpb.Struct, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	ctx = metadata.NewOutgoingContext(ctx, getRequestMetadata(e.SystemVariables))

	res, err := grpcClient.TriggerNamespaceModel(ctx, &modelPB.TriggerNamespaceModelRequest{
		NamespaceId: nsID,
		ModelId:     modelID,
		Version:     version,
		TaskInputs:  inputs,
	})

	if err != nil || res == nil {
		return nil, fmt.Errorf("error triggering model: %v", err)
	}

	if len(res.TaskOutputs) > 0 {
		return res.TaskOutputs, nil
	}

	return nil, fmt.Errorf("no output from model")
}
