package instillmodel

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"

	modelPB "github.com/instill-ai/protogen-go/model/model/v1alpha"
)

func (e *execution) trigger(input *structpb.Struct, job *base.Job, ctx context.Context) (*structpb.Struct, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	ctx = metadata.NewOutgoingContext(ctx, getRequestMetadata(e.SystemVariables))

	triggerInfo, err := getTriggerInfo(input)

	if err != nil {
		return nil, fmt.Errorf("getting trigger info: %w", err)
	}

	grpcClient := e.client

	res, err := grpcClient.TriggerNamespaceModel(ctx, &modelPB.TriggerNamespaceModelRequest{
		NamespaceId: triggerInfo.nsID,
		ModelId:     triggerInfo.modelID,
		Version:     triggerInfo.version,
		TaskInputs:  []*structpb.Struct{input},
	})

	if err != nil {
		return nil, fmt.Errorf("triggering model: %v", err)
	}

	if res == nil || len(res.TaskOutputs) == 0 {
		return nil, fmt.Errorf("triggering model: get empty task outputs")
	}

	return res.TaskOutputs[0], nil
}
