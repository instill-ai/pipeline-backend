package hubspot

import (
	"context"
	"testing"

	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

// mockClient is in contact_test.go

// Mock Retrieve Association struct and its functions

type MockRetrieveAssociation struct{}

func (s *MockRetrieveAssociation) GetThreadID(contactID string, paging bool, pagingPath string) (*TaskRetrieveAssociationThreadResp, error) {

	var fakeThreadID TaskRetrieveAssociationThreadResp
	if contactID == "32027696539" {
		fakeThreadID = TaskRetrieveAssociationThreadResp{
			Results: []struct {
				ID string `json:"id"`
			}{
				{ID: "7509711154"},
			},
		}
	}
	return &fakeThreadID, nil
}

func (s *MockRetrieveAssociation) GetCrmID(contactID string, objectType string, paging bool, pagingPath string) (interface{}, error) {

	var fakeCrmID TaskRetrieveAssociationCrmResp
	if contactID == "32027696539" {
		fakeCrmID = TaskRetrieveAssociationCrmResp{
			Results: []taskRetrieveAssociationCrmRespResult{
				{
					IDArray: []struct {
						ID string `json:"id"`
					}{
						{ID: "12345678900"},
					},
				},
			},
		}
	}
	return &fakeCrmID, nil

}

func TestComponent_ExecuteRetrieveAssociationTask(t *testing.T) {
	c := qt.New(t)
	ctx := context.Background()
	bc := base.Component{Logger: zap.NewNop()}
	connector := Init(bc)

	testcases := []struct {
		name     string
		input    TaskRetrieveAssociationInput
		wantResp interface{}
	}{
		{
			name: "ok - retrieve association: thread ID",
			input: TaskRetrieveAssociationInput{
				ContactID:  "32027696539",
				ObjectType: "Threads",
			},
			wantResp: TaskRetrieveAssociationOutput{
				ObjectIDs: []string{
					"7509711154",
				},
				ObjectIDsLength: 1,
			},
		},
		{
			name: "ok - retrieve association: deal ID",
			input: TaskRetrieveAssociationInput{
				ContactID:  "32027696539",
				ObjectType: "Deals",
			},
			wantResp: TaskRetrieveAssociationOutput{
				ObjectIDs: []string{
					"12345678900",
				},
				ObjectIDsLength: 1,
			},
		},
	}

	for _, tc := range testcases {
		c.Run(tc.name, func(c *qt.C) {
			setup, err := structpb.NewStruct(map[string]any{
				"token": bearerToken,
			})
			c.Assert(err, qt.IsNil)

			e := &execution{
				ComponentExecution: base.ComponentExecution{Component: connector, SystemVariables: nil, Setup: setup, Task: taskRetrieveAssociation},
				client:             createMockClient(),
			}
			e.execute = e.RetrieveAssociation

			pbInput, err := base.ConvertToStructpb(tc.input)

			c.Assert(err, qt.IsNil)

			ir, ow, eh, job := base.GenerateMockJob(c)
			ir.ReadMock.Return(pbInput, nil)
			ow.WriteMock.Optional().Set(func(ctx context.Context, output *structpb.Struct) (err error) {
				resJSON, err := protojson.Marshal(output)
				c.Assert(err, qt.IsNil)

				c.Check(resJSON, qt.JSONEquals, tc.wantResp)
				return nil
			})
			eh.ErrorMock.Optional()

			err = e.Execute(ctx, []*base.Job{job})
			c.Assert(err, qt.IsNil)

		})
	}

}
