package instill

import (
	"context"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/gojuno/minimock/v3"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"

	mock "github.com/instill-ai/pipeline-backend/pkg/component/internal/mock"
	modelPB "github.com/instill-ai/protogen-go/model/model/v1alpha"
)

var input = `{
	"data": {
		"model": "admin/dummy-model/latest"
	}
}`

var output = `{
	"task_outputs": [
		{
			"data": {
				"output": "output"
			}
		}
	]
}`

// Test_Execute tests the Execute function with the model server returning a valid response.
func Test_Execute(t *testing.T) {
	c := qt.New(t)
	ctx := context.Background()
	mc := minimock.NewController(c)

	grpcServer := grpc.NewServer()
	testServer := mock.NewModelPublicServiceServerMock(mc)

	testServer.TriggerNamespaceModelMock.Set(func(ctx context.Context, in *modelPB.TriggerNamespaceModelRequest) (*modelPB.TriggerNamespaceModelResponse, error) {
		c.Check(in.NamespaceId, qt.Equals, "admin")
		c.Check(in.ModelId, qt.Equals, "dummy-model")
		c.Check(in.Version, qt.Equals, "latest")
		c.Check(in.TaskInputs, qt.HasLen, 1)

		outputStruct := new(structpb.Struct)

		err := protojson.Unmarshal([]byte(output), outputStruct)

		c.Assert(err, qt.IsNil)

		return &modelPB.TriggerNamespaceModelResponse{
			TaskOutputs: []*structpb.Struct{outputStruct},
		}, nil
	})

	modelPB.RegisterModelPublicServiceServer(grpcServer, testServer)
	lis, err := net.Listen("tcp", ":0")
	c.Assert(err, qt.IsNil)

	go func() {
		err := grpcServer.Serve(lis)
		c.Check(err, qt.IsNil)
	}()
	defer grpcServer.Stop()

	mockAddress := lis.Addr().String()

	bc := base.Component{}

	comp := Init(bc)

	tasks := []string{
		taskEmbedding,
		taskChat,
		taskCompletion,
		taskTextToImage,
		taskClassification,
		taskDetection,
		taskKeyPoint,
		taskOCR,
		taskSemanticSegmentation,
		taskInstanceSegmentation,
	}

	for _, task := range tasks {

		x, err := comp.CreateExecution(base.ComponentExecution{
			Task: task,
			SystemVariables: map[string]any{
				"__MODEL_BACKEND":                 mockAddress,
				"__PIPELINE_HEADER_AUTHORIZATION": "Bearer inst token",
				"__PIPELINE_USER_UID":             "user1",
				"__PIPELINE_REQUESTER_UID":        "requester1",
			},
		})

		c.Assert(err, qt.IsNil)

		pbIn := new(structpb.Struct)
		err = protojson.Unmarshal([]byte(input), pbIn)

		c.Assert(err, qt.IsNil)

		ir, ow, eh, job := mock.GenerateMockJob(c)

		ir.ReadMock.Return(pbIn, nil)
		ow.WriteMock.Optional().Return(nil)

		eh.ErrorMock.Optional().Set(func(ctx context.Context, err error) {
			c.Check(err, qt.IsNil)
		})

		err = x.Execute(ctx, []*base.Job{job})
		c.Check(err, qt.IsNil)
	}
}

// Test_ExecuteServerError tests the case where the model server returns nothing.
func Test_ExecuteServerError(t *testing.T) {
	c := qt.New(t)
	ctx := context.Background()
	mc := minimock.NewController(c)

	grpcServer := grpc.NewServer()
	testServer := mock.NewModelPublicServiceServerMock(mc)

	testServer.TriggerNamespaceModelMock.Set(func(ctx context.Context, in *modelPB.TriggerNamespaceModelRequest) (*modelPB.TriggerNamespaceModelResponse, error) {
		return nil, nil
	})

	modelPB.RegisterModelPublicServiceServer(grpcServer, testServer)

	lis, err := net.Listen("tcp", ":0")

	c.Assert(err, qt.IsNil)

	go func() {
		err := grpcServer.Serve(lis)
		c.Check(err, qt.IsNil)
	}()
	defer grpcServer.Stop()

	mockAddress := lis.Addr().String()

	bc := base.Component{}

	comp := Init(bc)

	x, err := comp.CreateExecution(base.ComponentExecution{
		// All tasks are same behavior. We can just test one.
		Task: taskEmbedding,
		SystemVariables: map[string]any{
			"__MODEL_BACKEND":                 mockAddress,
			"__PIPELINE_HEADER_AUTHORIZATION": "Bearer inst token",
			"__PIPELINE_USER_UID":             "user1",
			"__PIPELINE_REQUESTER_UID":        "requester1",
		},
	})

	c.Assert(err, qt.IsNil)

	pbIn := new(structpb.Struct)
	err = protojson.Unmarshal([]byte(input), pbIn)

	c.Assert(err, qt.IsNil)

	ir, ow, eh, job := mock.GenerateMockJob(c)

	ir.ReadMock.Return(pbIn, nil)
	ow.WriteMock.Optional().Return(nil)

	eh.ErrorMock.Optional().Set(func(ctx context.Context, err error) {
		c.Check(err.Error(), qt.Contains, "get empty task outputs")
	})
	err = x.Execute(ctx, []*base.Job{job})
	c.Check(err, qt.IsNil)
}

var invalidInput = `{
	"data": {
		"model": "admin/dummy-model"
		}
}`

// Test_ExecuteClientError tests the case where the input is invalid.
func Test_ExecuteClientError(t *testing.T) {
	c := qt.New(t)
	ctx := context.Background()
	mc := minimock.NewController(c)

	grpcServer := grpc.NewServer()
	testServer := mock.NewModelPublicServiceServerMock(mc)

	modelPB.RegisterModelPublicServiceServer(grpcServer, testServer)

	lis, err := net.Listen("tcp", ":0")

	c.Assert(err, qt.IsNil)

	go func() {
		err := grpcServer.Serve(lis)
		c.Check(err, qt.IsNil)
	}()
	defer grpcServer.Stop()

	mockAddress := lis.Addr().String()

	bc := base.Component{}

	comp := Init(bc)

	x, err := comp.CreateExecution(base.ComponentExecution{
		// All tasks are same behavior. We can just test one.
		Task: taskEmbedding,
		SystemVariables: map[string]any{
			"__MODEL_BACKEND":                 mockAddress,
			"__PIPELINE_HEADER_AUTHORIZATION": "Bearer inst token",
			"__PIPELINE_USER_UID":             "user1",
			"__PIPELINE_REQUESTER_UID":        "requester1",
		},
	})

	c.Assert(err, qt.IsNil)

	pbIn := new(structpb.Struct)
	err = protojson.Unmarshal([]byte(invalidInput), pbIn)

	c.Assert(err, qt.IsNil)

	ir, ow, eh, job := mock.GenerateMockJob(c)

	ir.ReadMock.Return(pbIn, nil)
	ow.WriteMock.Optional().Return(nil)

	eh.ErrorMock.Optional().Set(func(ctx context.Context, err error) {
		c.Check(err.Error(), qt.Contains, "model name should be in the format of <namespace>/<model>/<version>")
	})
	err = x.Execute(ctx, []*base.Job{job})
	c.Check(err, qt.IsNil)
}
