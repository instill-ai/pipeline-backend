package hubspot

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	hubspot "github.com/belong-inc/go-hubspot"
	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

// mockClient is in contact_test.go

// Mock Deal struct and its functions
type MockDeal struct{}

func (s *MockDeal) Get(dealID string, deal interface{}, option *hubspot.RequestQueryOption) (*hubspot.ResponseResource, error) {

	var fakeDeal TaskGetDealResp
	if dealID == "20620806729" {

		fakeDeal = TaskGetDealResp{
			DealName:   "Fake deal",
			Pipeline:   "default",
			DealStage:  "qualifiedtobuy",
			CreateDate: hubspot.NewTime(time.Date(2024, 7, 9, 0, 0, 0, 0, time.UTC)),
		}
	}

	ret := &hubspot.ResponseResource{
		Properties: &fakeDeal,
	}

	return ret, nil
}

func (s *MockDeal) Create(deal interface{}) (*hubspot.ResponseResource, error) {
	arbitraryDealID := "12345678900"

	fakeDealInfo := deal.(*TaskCreateDealReq)

	fakeDealInfo.DealID = arbitraryDealID

	ret := &hubspot.ResponseResource{
		Properties: fakeDealInfo,
	}

	return ret, nil
}

func (s *MockDeal) Update(dealID string, deal interface{}) (*hubspot.ResponseResource, error) {
	return nil, nil
}
func (s *MockDeal) AssociateAnotherObj(dealID string, conf *hubspot.AssociationConfig) (*hubspot.ResponseResource, error) {
	return nil, nil
}

func TestComponent_ExecuteGetDealTask(t *testing.T) {
	c := qt.New(t)
	ctx := context.Background()
	bc := base.Component{Logger: zap.NewNop()}
	component := Init(bc)

	tc := struct {
		name     string
		input    string
		wantResp TaskGetDealOutput
	}{
		name:  "ok - get deal",
		input: "20620806729",
		wantResp: TaskGetDealOutput{
			DealName:             "Fake deal",
			Pipeline:             "default",
			DealStage:            "qualifiedtobuy",
			CreateDate:           "2024-07-09 00:00:00 +0000 UTC",
			AssociatedContactIDs: []string{},
		},
	}

	c.Run(tc.name, func(c *qt.C) {
		setup, err := structpb.NewStruct(map[string]any{
			"token": bearerToken,
		})
		c.Assert(err, qt.IsNil)

		e := &execution{
			ComponentExecution: base.ComponentExecution{Component: component, SystemVariables: nil, Setup: setup, Task: taskGetDeal},
			client:             createMockClient(),
		}
		e.execute = e.GetDeal

		pbInput, err := structpb.NewStruct(map[string]any{
			"deal-id": tc.input,
		})

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

func TestComponent_ExecuteCreateDealTask(t *testing.T) {
	c := qt.New(t)
	ctx := context.Background()
	bc := base.Component{Logger: zap.NewNop()}
	component := Init(bc)

	tc := struct {
		name      string
		inputDeal TaskCreateDealInput
		wantResp  string
	}{
		name: "ok - create deal",
		inputDeal: TaskCreateDealInput{
			DealName:  "Test Creating Deal",
			Pipeline:  "default",
			DealStage: "contractsent",
		},
		wantResp: "12345678900",
	}

	c.Run(tc.name, func(c *qt.C) {
		setup, err := structpb.NewStruct(map[string]any{
			"token": bearerToken,
		})
		c.Assert(err, qt.IsNil)

		e := &execution{
			ComponentExecution: base.ComponentExecution{Component: component, SystemVariables: nil, Setup: setup, Task: taskCreateDeal},
			client:             createMockClient(),
		}
		e.execute = e.CreateDeal

		pbInput, err := base.ConvertToStructpb(tc.inputDeal)

		c.Assert(err, qt.IsNil)

		ir, ow, eh, job := base.GenerateMockJob(c)
		ir.ReadMock.Return(pbInput, nil)
		ow.WriteMock.Optional().Set(func(ctx context.Context, output *structpb.Struct) (err error) {
			resString := output.Fields["deal-id"].GetStringValue()

			c.Check(resString, qt.Equals, tc.wantResp)
			return nil
		})
		eh.ErrorMock.Optional()
		err = e.Execute(ctx, []*base.Job{job})
		c.Assert(err, qt.IsNil)

	})
}
