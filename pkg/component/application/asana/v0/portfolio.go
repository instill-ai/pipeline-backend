package asana

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

type PortfolioTaskOutput struct {
	Portfolio
}

type PortfolioTaskResp struct {
	Data struct {
		GID                 string                   `json:"gid"`
		Name                string                   `json:"name"`
		Owner               User                     `json:"owner"`
		DueOn               string                   `json:"due_on"`
		StartOn             string                   `json:"start_on"`
		Color               string                   `json:"color"`
		Public              bool                     `json:"public"`
		CreatedBy           User                     `json:"created_by"`
		CurrentStatus       []map[string]interface{} `json:"current_status"`
		CustomFields        []map[string]interface{} `json:"custom_fields"`
		CustomFieldSettings []map[string]interface{} `json:"custom_field_settings"`
	} `json:"data"`
}

func portfolioResp2Output(resp *PortfolioTaskResp) PortfolioTaskOutput {
	out := PortfolioTaskOutput{
		Portfolio: Portfolio{
			GID:                 resp.Data.GID,
			Name:                resp.Data.Name,
			Owner:               resp.Data.Owner,
			DueOn:               resp.Data.DueOn,
			StartOn:             resp.Data.StartOn,
			Color:               resp.Data.Color,
			Public:              resp.Data.Public,
			CreatedBy:           resp.Data.CreatedBy,
			CurrentStatus:       resp.Data.CurrentStatus,
			CustomFields:        resp.Data.CustomFields,
			CustomFieldSettings: resp.Data.CustomFieldSettings,
		},
	}
	if out.CurrentStatus == nil {
		out.CurrentStatus = []map[string]interface{}{}
	}
	if out.CustomFields == nil {
		out.CustomFields = []map[string]interface{}{}
	}
	if out.CustomFieldSettings == nil {
		out.CustomFieldSettings = []map[string]interface{}{}
	}
	return out
}

type GetPortfolioInput struct {
	Action string `json:"action"`
	ID     string `json:"portfolio-gid"`
}

func (c *Client) GetPortfolio(ctx context.Context, props *structpb.Struct) (*structpb.Struct, error) {
	var input GetPortfolioInput
	if err := base.ConvertFromStructpb(props, &input); err != nil {
		return nil, err
	}

	apiEndpoint := fmt.Sprintf("/portfolios/%s", input.ID)
	req := c.Client.R().SetResult(&PortfolioTaskResp{})

	wantOptFields := parseWantOptionFields(Portfolio{})
	if err := addQueryOptions(req, map[string]interface{}{"opt_fields": wantOptFields}); err != nil {
		return nil, err
	}
	resp, err := req.Get(apiEndpoint)
	if err != nil {
		return nil, err
	}

	portfolio := resp.Result().(*PortfolioTaskResp)
	out := portfolioResp2Output(portfolio)
	return base.ConvertToStructpb(out)
}

type UpdatePortfolioInput struct {
	Action    string `json:"action"`
	ID        string `json:"portfolio-gid"`
	Name      string `json:"name"`
	Color     string `json:"color"`
	Public    bool   `json:"public"`
	Workspace string `json:"workspace"`
}

type UpdatePortfolioReq struct {
	Name      string `json:"name,omitempty"`
	Color     string `json:"color,omitempty"`
	Public    bool   `json:"public"`
	Workspace string `json:"workspace,omitempty"`
}

func (c *Client) UpdatePortfolio(ctx context.Context, props *structpb.Struct) (*structpb.Struct, error) {
	var input UpdatePortfolioInput
	if err := base.ConvertFromStructpb(props, &input); err != nil {
		return nil, err
	}

	apiEndpoint := fmt.Sprintf("/portfolios/%s", input.ID)
	req := c.Client.R().SetResult(&PortfolioTaskResp{}).SetBody(
		map[string]interface{}{
			"data": &UpdatePortfolioReq{
				Name:      input.Name,
				Color:     input.Color,
				Public:    input.Public,
				Workspace: input.Workspace,
			},
		})

	wantOptFields := parseWantOptionFields(Portfolio{})
	if err := addQueryOptions(req, map[string]interface{}{"opt_fields": wantOptFields}); err != nil {
		return nil, err
	}

	resp, err := req.Put(apiEndpoint)

	if err != nil {
		return nil, err
	}
	portfolio := resp.Result().(*PortfolioTaskResp)
	out := portfolioResp2Output(portfolio)
	return base.ConvertToStructpb(out)
}

type CreatePortfolioInput struct {
	Action    string `json:"action"`
	Name      string `json:"name"`
	Color     string `json:"color"`
	Public    bool   `json:"public"`
	Workspace string `json:"workspace"`
}

type CreatePortfolioReq struct {
	Name      string `json:"name"`
	Color     string `json:"color,omitempty"`
	Public    bool   `json:"public"`
	Workspace string `json:"workspace"`
}

func (c *Client) CreatePortfolio(ctx context.Context, props *structpb.Struct) (*structpb.Struct, error) {
	var input CreatePortfolioInput
	if err := base.ConvertFromStructpb(props, &input); err != nil {
		return nil, err
	}

	apiEndpoint := "/portfolios"
	req := c.Client.R().SetResult(&PortfolioTaskResp{}).SetBody(
		map[string]interface{}{
			"data": &CreatePortfolioReq{
				Name:      input.Name,
				Color:     input.Color,
				Public:    input.Public,
				Workspace: input.Workspace,
			},
		})
	wantOptFields := parseWantOptionFields(Portfolio{})
	if err := addQueryOptions(req, map[string]interface{}{"opt_fields": wantOptFields}); err != nil {
		return nil, err
	}

	resp, err := req.Post(apiEndpoint)
	if err != nil {
		return nil, err
	}
	portfolio := resp.Result().(*PortfolioTaskResp)
	out := portfolioResp2Output(portfolio)
	return base.ConvertToStructpb(out)
}

type DeletePortfolioInput struct {
	Action string `json:"action"`
	ID     string `json:"portfolio-gid"`
}

func (c *Client) DeletePortfolio(ctx context.Context, props *structpb.Struct) (*structpb.Struct, error) {
	var input DeletePortfolioInput
	if err := base.ConvertFromStructpb(props, &input); err != nil {
		return nil, err
	}

	apiEndpoint := fmt.Sprintf("/portfolios/%s", input.ID)
	req := c.R()

	_, err := req.Delete(apiEndpoint)
	if err != nil {
		return nil, err
	}
	out := portfolioResp2Output(&PortfolioTaskResp{})
	return base.ConvertToStructpb(out)
}
