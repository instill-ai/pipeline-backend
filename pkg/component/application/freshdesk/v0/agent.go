package freshdesk

import (
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

// This file is used to handle "Get Agent", "Get Role", "Get Skill"  and "Get Group" tasks.
// Role, Skill and Group are common to all agents, so they are handled in the same file.

const (
	AgentPath = "agents"
	RolePath  = "roles"
	GroupPath = "groups"
	SkillPath = "admin/skills"
)

// API function for Agent

func (c *FreshdeskClient) GetAgent(agentID int64) (*TaskGetAgentResponse, error) {
	resp := &TaskGetAgentResponse{}

	httpReq := c.httpclient.R().SetResult(resp)
	if _, err := httpReq.Get(fmt.Sprintf("/%s/%d", AgentPath, agentID)); err != nil {
		return nil, err
	}
	return resp, nil
}

// API function for Role

func (c *FreshdeskClient) GetRole(roleID int64) (*TaskGetRoleResponse, error) {
	resp := &TaskGetRoleResponse{}

	httpReq := c.httpclient.R().SetResult(resp)
	if _, err := httpReq.Get(fmt.Sprintf("/%s/%d", RolePath, roleID)); err != nil {
		return nil, err
	}
	return resp, nil
}

// API function for Group

func (c *FreshdeskClient) GetGroup(groupID int64) (*TaskGetGroupResponse, error) {
	resp := &TaskGetGroupResponse{}

	httpReq := c.httpclient.R().SetResult(resp)
	if _, err := httpReq.Get(fmt.Sprintf("/%s/%d", GroupPath, groupID)); err != nil {
		return nil, err
	}
	return resp, nil
}

// API function for Skill

func (c *FreshdeskClient) GetSkill(skillID int64) (*TaskGetSkillResponse, error) {
	resp := &TaskGetSkillResponse{}

	httpReq := c.httpclient.R().SetResult(resp)
	if _, err := httpReq.Get(fmt.Sprintf("/%s/%d", SkillPath, skillID)); err != nil {
		return nil, err
	}
	return resp, nil
}

// Task 1: Get Agent

type TaskGetAgentInput struct {
	AgentID int64 `json:"agent-id"`
}

type TaskGetAgentResponse struct {
	Contact     taskGetAgentResponseContact `json:"contact"`
	Type        string                      `json:"type"`
	TicketScope int                         `json:"ticket_scope"`
	Available   bool                        `json:"available"`
	GroupIDs    []int64                     `json:"group_ids"`
	RoleIDs     []int64                     `json:"role_ids"`
	SkillIDs    []int64                     `json:"skill_ids"`
	Occasional  bool                        `json:"occasional"`
	Signature   string                      `json:"signature"`
	FocusMode   bool                        `json:"focus_mode"`
	Deactivated bool                        `json:"deactivated"`
	CreatedAt   string                      `json:"created_at"`
	UpdatedAt   string                      `json:"updated_at"`
}

type taskGetAgentResponseContact struct {
	Name        string `json:"name"`
	Active      bool   `json:"active"`
	Email       string `json:"email"`
	JobTitle    string `json:"job_title"`
	Language    string `json:"language"`
	LastLoginAt string `json:"last_login_at"`
	Mobile      string `json:"mobile"`
	Phone       string `json:"phone"`
	TimeZone    string `json:"time_zone"`
}

type TaskGetAgentOutput struct {
	Name        string  `json:"name,omitempty"`
	Active      bool    `json:"active"`
	Email       string  `json:"email"`
	JobTitle    string  `json:"job-title,omitempty"`
	Language    string  `json:"language,omitempty"`
	LastLoginAt string  `json:"last-login-at"`
	Mobile      string  `json:"mobile,omitempty"`
	Phone       string  `json:"phone,omitempty"`
	TimeZone    string  `json:"time-zone,omitempty"`
	Type        string  `json:"type"`
	TicketScope string  `json:"ticket-scope"`
	Available   bool    `json:"available"`
	GroupIDs    []int64 `json:"group-ids"`
	RoleIDs     []int64 `json:"role-ids"`
	SkillIDs    []int64 `json:"skill-ids"`
	Occasional  bool    `json:"occasional"`
	Signature   string  `json:"signature,omitempty"`
	FocusMode   bool    `json:"focus-mode"`
	Deactivated bool    `json:"deactivated"`
	CreatedAt   string  `json:"created-at"`
	UpdatedAt   string  `json:"updated-at"`
}

func (e *execution) TaskGetAgent(in *structpb.Struct) (*structpb.Struct, error) {
	inputStruct := TaskGetAgentInput{}
	err := base.ConvertFromStructpb(in, &inputStruct)

	if err != nil {
		return nil, fmt.Errorf("failed to convert input to struct: %v", err)
	}

	resp, err := e.client.GetAgent(inputStruct.AgentID)

	if err != nil {
		return nil, err
	}

	outputStruct := TaskGetAgentOutput{
		Name:        resp.Contact.Name,
		Active:      resp.Contact.Active,
		Email:       resp.Contact.Email,
		JobTitle:    resp.Contact.JobTitle,
		Language:    convertCodeToLanguage(resp.Contact.Language),
		LastLoginAt: convertTimestampResp(resp.Contact.LastLoginAt),
		Mobile:      resp.Contact.Mobile,
		Phone:       resp.Contact.Phone,
		TimeZone:    resp.Contact.TimeZone,
		Type:        resp.Type,
		TicketScope: convertTicketScopeResponse(resp.TicketScope),
		Available:   resp.Available,
		GroupIDs:    resp.GroupIDs,
		RoleIDs:     resp.RoleIDs,
		SkillIDs:    resp.SkillIDs,
		Occasional:  resp.Occasional,
		Signature:   resp.Signature,
		FocusMode:   resp.FocusMode,
		Deactivated: resp.Deactivated,
		CreatedAt:   convertTimestampResp(resp.CreatedAt),
		UpdatedAt:   convertTimestampResp(resp.UpdatedAt),
	}

	output, err := base.ConvertToStructpb(outputStruct)

	if err != nil {
		return nil, fmt.Errorf("failed to convert output to struct: %v", err)
	}

	return output, nil
}

// Task 2: Get Role
type TaskGetRoleInput struct {
	RoleID int64 `json:"role-id"`
}

type TaskGetRoleResponse struct {
	Description string `json:"description"`
	Name        string `json:"name"`
	Default     bool   `json:"default"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
	AgentType   int    `json:"agent_type"`
}

type TaskGetRoleOutput struct {
	Description string `json:"description"`
	Name        string `json:"name"`
	Default     bool   `json:"default"`
	CreatedAt   string `json:"created-at"`
	UpdatedAt   string `json:"updated-at"`
	AgentType   string `json:"agent-type"`
}

func (e *execution) TaskGetRole(in *structpb.Struct) (*structpb.Struct, error) {

	inputStruct := TaskGetRoleInput{}
	err := base.ConvertFromStructpb(in, &inputStruct)

	if err != nil {
		return nil, fmt.Errorf("failed to convert input to struct: %v", err)
	}

	resp, err := e.client.GetRole(inputStruct.RoleID)

	if err != nil {
		return nil, err
	}

	outputStruct := TaskGetRoleOutput{
		Description: resp.Description,
		Name:        resp.Name,
		Default:     resp.Default,
		CreatedAt:   convertTimestampResp(resp.CreatedAt),
		UpdatedAt:   convertTimestampResp(resp.UpdatedAt),
		AgentType:   convertAgentTypeResponse(resp.AgentType),
	}

	output, err := base.ConvertToStructpb(outputStruct)

	if err != nil {
		return nil, fmt.Errorf("failed to convert output to struct: %v", err)
	}

	return output, nil

}

// Task 3: Get Group
type TaskGetGroupInput struct {
	GroupID int64 `json:"group-id"`
}

type TaskGetGroupResponse struct {
	Name                    string  `json:"name"`
	Description             string  `json:"description"`
	AgentIDs                []int64 `json:"agent_ids"`
	AutoTicketAssign        int     `json:"auto_ticket_assign"`
	EscalateTo              int64   `json:"escalate_to"`
	UnassignedDuration      string  `json:"unassigned_for"`
	GroupType               string  `json:"group_type"`
	AgentAvailabilityStatus bool    `json:"agent_availability_status"`
	CreatedAt               string  `json:"created_at"`
	UpdatedAt               string  `json:"updated_at"`
}

type TaskGetGroupOutput struct {
	Name                    string  `json:"name"`
	Description             string  `json:"description"`
	AgentIDs                []int64 `json:"agent-ids"`
	AutoTicketAssign        string  `json:"auto-ticket-assign"`
	EscalateTo              int64   `json:"escalate-to,omitempty"`
	UnassignedDuration      string  `json:"unassigned-duration,omitempty"`
	GroupType               string  `json:"group-type,omitempty"`
	AgentAvailabilityStatus bool    `json:"agent-availability-status"`
	CreatedAt               string  `json:"created-at"`
	UpdatedAt               string  `json:"updated-at"`
}

func (e *execution) TaskGetGroup(in *structpb.Struct) (*structpb.Struct, error) {
	inputStruct := TaskGetGroupInput{}
	err := base.ConvertFromStructpb(in, &inputStruct)

	if err != nil {
		return nil, fmt.Errorf("failed to convert input to struct: %v", err)
	}

	resp, err := e.client.GetGroup(inputStruct.GroupID)

	if err != nil {
		return nil, err
	}

	outputStruct := TaskGetGroupOutput{
		Name:                    resp.Name,
		AgentIDs:                resp.AgentIDs,
		AutoTicketAssign:        convertAutoTicketAssignResponse(resp.AutoTicketAssign),
		Description:             resp.Description,
		EscalateTo:              resp.EscalateTo,
		UnassignedDuration:      resp.UnassignedDuration,
		GroupType:               resp.GroupType,
		AgentAvailabilityStatus: resp.AgentAvailabilityStatus,
		CreatedAt:               convertTimestampResp(resp.CreatedAt),
		UpdatedAt:               convertTimestampResp(resp.UpdatedAt),
	}

	output, err := base.ConvertToStructpb(outputStruct)

	if err != nil {
		return nil, fmt.Errorf("failed to convert output to struct: %v", err)
	}

	return output, nil

}

// Task 4: Get Skill

type TaskGetSkillInput struct {
	SkillID int64 `json:"skill-id"`
}

type TaskGetSkillResponse struct {
	Name       string                          `json:"name"`
	Rank       int                             `json:"rank"`
	Conditions []taskGetSkillResponseCondition `json:"conditions"`
	CreatedAt  string                          `json:"created_at"`
	UpdatedAt  string                          `json:"updated_at"`
}

type taskGetSkillResponseCondition struct {
	ChannelConditions []taskGetSkillResponseChannelCondition `json:"channel_conditions"`
}

type taskGetSkillResponseChannelCondition struct {
	MatchType  string                   `json:"match_type"`
	Properties []map[string]interface{} `json:"properties"`
}

type TaskGetSkillOutput struct {
	Name               string                   `json:"name"`
	Rank               int                      `json:"rank"`
	ConditionMatchType string                   `json:"condition-match-type"`
	Conditions         []map[string]interface{} `json:"conditions"`
	CreatedAt          string                   `json:"created-at"`
	UpdatedAt          string                   `json:"updated-at"`
}

func (e *execution) TaskGetSkill(in *structpb.Struct) (*structpb.Struct, error) {
	inputStruct := TaskGetSkillInput{}
	err := base.ConvertFromStructpb(in, &inputStruct)

	if err != nil {
		return nil, fmt.Errorf("failed to convert input to struct: %v", err)
	}

	resp, err := e.client.GetSkill(inputStruct.SkillID)

	if err != nil {
		return nil, err
	}

	outputStruct := TaskGetSkillOutput{
		Name:               resp.Name,
		Rank:               resp.Rank,
		ConditionMatchType: resp.Conditions[0].ChannelConditions[0].MatchType, //only Condition[0] and ChannelCondition[0] are taken because no matter what kind of skill I make, there is only one of each.
		Conditions:         resp.Conditions[0].ChannelConditions[0].Properties,
		CreatedAt:          convertTimestampResp(resp.CreatedAt),
		UpdatedAt:          convertTimestampResp(resp.UpdatedAt),
	}

	output, err := base.ConvertToStructpb(outputStruct)

	if err != nil {
		return nil, fmt.Errorf("failed to convert output to struct: %v", err)
	}

	return output, nil
}

func convertTicketScopeResponse(in int) string { //used in Get Agent task
	switch in {
	case 1:
		return "Global Access"
	case 2:
		return "Group Access"
	case 3:
		return "Restricted Access"
	default:
		return "Unknown"
	}
}

func convertAgentTypeResponse(in int) string { //used in Get Role task
	switch in {
	case 1:
		return "Support Agent"
	case 2:
		return "Field Agent"
	case 3:
		return "Collaborator"
	default:
		return "Unknown agent type"
	}
}

func convertAutoTicketAssignResponse(in int) string {
	switch in {
	case 0:
		return "Disabled"
	case 1:
		return "Round Robin"
	case 2:
		return "Skill Based Round Robin"
	case 3:
		return "Load Based Round Robin"
	case 12:
		return "Omniroute"
	default:
		return "Unknown automatic ticket assign type"
	}
}
