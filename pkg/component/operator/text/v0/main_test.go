package text

import (
	"context"
	"testing"

	"github.com/frankban/quicktest"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/mock"
)

func TestOperator(t *testing.T) {
	c := quicktest.New(t)

	testcases := []struct {
		name  string
		task  string
		input structpb.Struct
	}{
		{
			name: "chunk texts",
			task: "TASK_CHUNK_TEXT",
			input: structpb.Struct{
				Fields: map[string]*structpb.Value{
					"text": {Kind: &structpb.Value_StringValue{StringValue: "Hello world. This is a test."}},
					"strategy": {Kind: &structpb.Value_StructValue{StructValue: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							"setting": {Kind: &structpb.Value_StructValue{StructValue: &structpb.Struct{
								Fields: map[string]*structpb.Value{
									"chunk-method": {Kind: &structpb.Value_StringValue{StringValue: "Token"}},
								},
							}}}},
						},
					}}}},
				},
			},
		},
		{
			name:  "error case",
			task:  "FAKE_TASK",
			input: structpb.Struct{},
		},
		{
			name: "data cleansing",
			task: "TASK_CLEAN_DATA",
			input: structpb.Struct{
				Fields: map[string]*structpb.Value{
					"texts": {Kind: &structpb.Value_ListValue{ListValue: &structpb.ListValue{
						Values: []*structpb.Value{
							{Kind: &structpb.Value_StringValue{StringValue: "Sample text 1."}},
							{Kind: &structpb.Value_StringValue{StringValue: "Sample text 2."}},
						},
					}}},
					"setting": {Kind: &structpb.Value_StructValue{StructValue: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							"clean-method": {Kind: &structpb.Value_StringValue{StringValue: "Regex"}},
							"exclude-patterns": {Kind: &structpb.Value_ListValue{ListValue: &structpb.ListValue{
								Values: []*structpb.Value{
									{Kind: &structpb.Value_StringValue{StringValue: "exclude this"}},
								},
							}}},
						},
					}}},
				},
			},
		},
	}
	bc := base.Component{}
	ctx := context.Background()
	for i := range testcases {
		tc := &testcases[i]
		c.Run(tc.name, func(c *quicktest.C) {
			component := Init(bc)
			c.Assert(component, quicktest.IsNotNil)

			execution, err := component.CreateExecution(base.ComponentExecution{
				Component: component,
				Task:      tc.task,
			})
			c.Assert(err, quicktest.IsNil)
			c.Assert(execution, quicktest.IsNotNil)

			ir, ow, eh, job := mock.GenerateMockJob(c)
			ir.ReadMock.Return(&tc.input, nil)
			ow.WriteMock.Optional().Set(func(ctx context.Context, output *structpb.Struct) (err error) {
				if tc.name == "error case" {
					c.Assert(output, quicktest.IsNil)
					return
				}
				return nil
			})
			eh.ErrorMock.Optional().Set(func(ctx context.Context, err error) {
				if tc.name == "error case" {
					c.Assert(err, quicktest.ErrorMatches, "not supported task: FAKE_TASK")
				}
			})
			err = execution.Execute(ctx, []*base.Job{job})
			c.Assert(err, quicktest.IsNil)
		})
	}
}
