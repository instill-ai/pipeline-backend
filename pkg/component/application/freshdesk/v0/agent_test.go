package freshdesk

import (
	"context"
	"testing"

	"github.com/gojuno/minimock/v3"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func TestComponent_ExecuteGetAgentTask(t *testing.T) {
	mc := minimock.NewController(t)
	c := qt.New(t)
	bc := base.Component{Logger: zap.NewNop()}
	connector := Init(bc)
	ctx := context.Background()

	FreshdeskClientMock := NewFreshdeskInterfaceMock(mc)

	FreshdeskClientMock.GetAgentMock.
		When(154023630520).
		Then(
			&TaskGetAgentResponse{
				Contact: taskGetAgentResponseContact{
					Name:     "Fake Agent",
					Email:    "randomemail@gmail.com",
					JobTitle: "Random Job Title",
					Language: "ja-JP",
					Phone:    "1234567890",
					TimeZone: "Santiago",
				},
				Type:        "collaborator",
				TicketScope: 2,
				Available:   false,
				GroupIDs:    []int64{154000444629},
				RoleIDs:     []int64{154001049982, 154001049983},
				SkillIDs:    []int64{1},
				Occasional:  false,
				Signature:   "Random Signature",
				FocusMode:   true,
				Deactivated: false,
				CreatedAt:   "2024-08-29T10:03:14Z",
				UpdatedAt:   "2024-08-29T10:03:14Z",
			}, nil)

	tc := struct {
		name       string
		input      TaskGetAgentInput
		wantOutput TaskGetAgentOutput
	}{
		name: "ok - task get agent",
		input: TaskGetAgentInput{
			AgentID: 154023630520,
		},
		wantOutput: TaskGetAgentOutput{
			Name:        "Fake Agent",
			Email:       "randomemail@gmail.com",
			JobTitle:    "Random Job Title",
			Language:    "Japanese",
			Phone:       "1234567890",
			TimeZone:    "Santiago",
			Type:        "collaborator",
			TicketScope: "Group Access",
			Available:   false,
			GroupIDs:    []int64{154000444629},
			RoleIDs:     []int64{154001049982, 154001049983},
			SkillIDs:    []int64{1},
			Occasional:  false,
			Signature:   "Random Signature",
			FocusMode:   true,
			Deactivated: false,
			CreatedAt:   "2024-08-29 10:03:14 UTC",
			UpdatedAt:   "2024-08-29 10:03:14 UTC",
		},
	}

	c.Run(tc.name, func(c *qt.C) {
		setup, err := structpb.NewStruct(map[string]any{
			"api-key": apiKey,
			"domain":  domain,
		})
		c.Assert(err, qt.IsNil)

		e := &execution{
			ComponentExecution: base.ComponentExecution{Component: connector, SystemVariables: nil, Setup: setup, Task: taskGetAgent},
			client:             FreshdeskClientMock,
		}
		e.execute = e.TaskGetAgent

		pbIn, err := base.ConvertToStructpb(tc.input)
		c.Assert(err, qt.IsNil)

		ir, ow, eh, job := base.GenerateMockJob(c)
		ir.ReadMock.Return(pbIn, nil)
		ow.WriteMock.Optional().Set(func(ctx context.Context, output *structpb.Struct) (err error) {

			outJSON, err := protojson.Marshal(output)
			c.Assert(err, qt.IsNil)

			c.Check(outJSON, qt.JSONEquals, tc.wantOutput)
			return nil
		})
		eh.ErrorMock.Optional()

		err = e.Execute(ctx, []*base.Job{job})

		c.Assert(err, qt.IsNil)

	})
}

func TestComponent_ExecuteGetRoleTask(t *testing.T) {
	mc := minimock.NewController(t)
	c := qt.New(t)
	bc := base.Component{Logger: zap.NewNop()}
	connector := Init(bc)
	ctx := context.Background()

	FreshdeskClientMock := NewFreshdeskInterfaceMock(mc)

	FreshdeskClientMock.GetRoleMock.
		When(154001049978).
		Then(
			&TaskGetRoleResponse{
				Description: "Has complete control over the help desk and the organisation including access to Account or Billing related information, and receives Invoices.",
				Name:        "Fake Role",
				Default:     true,
				CreatedAt:   "2024-08-18T03:47:58Z",
				UpdatedAt:   "2024-08-18T03:47:58Z",
				AgentType:   1,
			}, nil)

	tc := struct {
		name       string
		input      TaskGetRoleInput
		wantOutput TaskGetRoleOutput
	}{
		name: "ok - task get role",
		input: TaskGetRoleInput{
			RoleID: 154001049978,
		},
		wantOutput: TaskGetRoleOutput{
			Description: "Has complete control over the help desk and the organisation including access to Account or Billing related information, and receives Invoices.",
			Name:        "Fake Role",
			Default:     true,
			CreatedAt:   "2024-08-18 03:47:58 UTC",
			UpdatedAt:   "2024-08-18 03:47:58 UTC",
			AgentType:   "Support Agent",
		},
	}

	c.Run(tc.name, func(c *qt.C) {
		setup, err := structpb.NewStruct(map[string]any{
			"api-key": apiKey,
			"domain":  domain,
		})
		c.Assert(err, qt.IsNil)

		e := &execution{
			ComponentExecution: base.ComponentExecution{Component: connector, SystemVariables: nil, Setup: setup, Task: taskGetRole},
			client:             FreshdeskClientMock,
		}
		e.execute = e.TaskGetRole

		pbIn, err := base.ConvertToStructpb(tc.input)
		c.Assert(err, qt.IsNil)

		ir, ow, eh, job := base.GenerateMockJob(c)
		ir.ReadMock.Return(pbIn, nil)
		ow.WriteMock.Optional().Set(func(ctx context.Context, output *structpb.Struct) (err error) {

			outJSON, err := protojson.Marshal(output)
			c.Assert(err, qt.IsNil)

			c.Check(outJSON, qt.JSONEquals, tc.wantOutput)
			return nil
		})
		eh.ErrorMock.Optional()

		err = e.Execute(ctx, []*base.Job{job})

		c.Assert(err, qt.IsNil)

	})
}

func TestComponent_ExecuteGetGroupTask(t *testing.T) {
	mc := minimock.NewController(t)
	c := qt.New(t)
	bc := base.Component{Logger: zap.NewNop()}
	connector := Init(bc)
	ctx := context.Background()

	FreshdeskClientMock := NewFreshdeskInterfaceMock(mc)

	FreshdeskClientMock.GetGroupMock.
		When(154000458525).
		Then(
			&TaskGetGroupResponse{
				Name:                    "Random Group",
				Description:             "Just a random group",
				AgentIDs:                []int64{154023114553},
				AutoTicketAssign:        12,
				EscalateTo:              154023114553,
				UnassignedDuration:      "8h",
				GroupType:               "support_agent_group",
				AgentAvailabilityStatus: true,
				CreatedAt:               "2024-08-29T10:31:44Z",
				UpdatedAt:               "2024-08-29T10:31:44Z",
			}, nil)

	tc := struct {
		name       string
		input      TaskGetGroupInput
		wantOutput TaskGetGroupOutput
	}{
		name: "ok - task get group",
		input: TaskGetGroupInput{
			GroupID: 154000458525,
		},
		wantOutput: TaskGetGroupOutput{
			Name:                    "Random Group",
			Description:             "Just a random group",
			AgentIDs:                []int64{154023114553},
			AutoTicketAssign:        "Omniroute",
			EscalateTo:              154023114553,
			UnassignedDuration:      "8h",
			GroupType:               "support_agent_group",
			AgentAvailabilityStatus: true,
			CreatedAt:               "2024-08-29 10:31:44 UTC",
			UpdatedAt:               "2024-08-29 10:31:44 UTC",
		},
	}

	c.Run(tc.name, func(c *qt.C) {
		setup, err := structpb.NewStruct(map[string]any{
			"api-key": apiKey,
			"domain":  domain,
		})
		c.Assert(err, qt.IsNil)

		e := &execution{
			ComponentExecution: base.ComponentExecution{Component: connector, SystemVariables: nil, Setup: setup, Task: taskGetGroup},
			client:             FreshdeskClientMock,
		}
		e.execute = e.TaskGetGroup

		pbIn, err := base.ConvertToStructpb(tc.input)
		c.Assert(err, qt.IsNil)

		ir, ow, eh, job := base.GenerateMockJob(c)
		ir.ReadMock.Return(pbIn, nil)
		ow.WriteMock.Optional().Set(func(ctx context.Context, output *structpb.Struct) (err error) {

			outJSON, err := protojson.Marshal(output)
			c.Assert(err, qt.IsNil)

			c.Check(outJSON, qt.JSONEquals, tc.wantOutput)
			return nil
		})
		eh.ErrorMock.Optional()

		err = e.Execute(ctx, []*base.Job{job})

		c.Assert(err, qt.IsNil)

	})
}

func TestComponent_ExecuteGetSkillTask(t *testing.T) {
	mc := minimock.NewController(t)
	c := qt.New(t)
	bc := base.Component{Logger: zap.NewNop()}
	connector := Init(bc)
	ctx := context.Background()

	FreshdeskClientMock := NewFreshdeskInterfaceMock(mc)

	FreshdeskClientMock.GetSkillMock.
		When(1).
		Then(
			&TaskGetSkillResponse{
				Name:      "Random Skill",
				Rank:      2,
				CreatedAt: "2024-08-22T09:14:32Z",
				UpdatedAt: "2024-08-22T12:56:57Z",
				Conditions: []taskGetSkillResponseCondition{
					{
						ChannelConditions: []taskGetSkillResponseChannelCondition{
							{
								MatchType: "any",
								Properties: []map[string]interface{}{
									{
										"value":         []interface{}{3},
										"operator":      "in",
										"field_name":    "priority",
										"resource_type": "ticket",
									},
									{
										"value":         []interface{}{"zh-CN"},
										"operator":      "in",
										"field_name":    "language",
										"resource_type": "contact",
									},
								},
							},
						},
					},
				},
			}, nil)

	tc := struct {
		name       string
		input      TaskGetSkillInput
		wantOutput TaskGetSkillOutput
	}{
		name: "ok - task get skill",
		input: TaskGetSkillInput{
			SkillID: 1,
		},
		wantOutput: TaskGetSkillOutput{
			Name:               "Random Skill",
			Rank:               2,
			CreatedAt:          "2024-08-22 09:14:32 UTC",
			UpdatedAt:          "2024-08-22 12:56:57 UTC",
			ConditionMatchType: "any",
			Conditions: []map[string]interface{}{
				{
					"value":         []interface{}{3},
					"operator":      "in",
					"field_name":    "priority",
					"resource_type": "ticket",
				},
				{
					"value":         []interface{}{"zh-CN"},
					"operator":      "in",
					"field_name":    "language",
					"resource_type": "contact"},
			},
		},
	}

	c.Run(tc.name, func(c *qt.C) {
		setup, err := structpb.NewStruct(map[string]any{
			"api-key": apiKey,
			"domain":  domain,
		})
		c.Assert(err, qt.IsNil)

		e := &execution{
			ComponentExecution: base.ComponentExecution{Component: connector, SystemVariables: nil, Setup: setup, Task: taskGetSkill},
			client:             FreshdeskClientMock,
		}
		e.execute = e.TaskGetSkill

		pbIn, err := base.ConvertToStructpb(tc.input)
		c.Assert(err, qt.IsNil)

		ir, ow, eh, job := base.GenerateMockJob(c)
		ir.ReadMock.Return(pbIn, nil)
		ow.WriteMock.Optional().Set(func(ctx context.Context, output *structpb.Struct) (err error) {

			outJSON, err := protojson.Marshal(output)
			c.Assert(err, qt.IsNil)

			c.Check(outJSON, qt.JSONEquals, tc.wantOutput)
			return nil
		})
		eh.ErrorMock.Optional()

		err = e.Execute(ctx, []*base.Job{job})

		c.Assert(err, qt.IsNil)

	})
}
