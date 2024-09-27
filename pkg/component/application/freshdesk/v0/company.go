package freshdesk

import (
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

const (
	CompanyPath = "companies"
)

// API functions for Company

func (c *FreshdeskClient) GetCompany(companyID int64) (*TaskGetCompanyResponse, error) {
	resp := &TaskGetCompanyResponse{}

	httpReq := c.httpclient.R().SetResult(resp)
	if _, err := httpReq.Get(fmt.Sprintf("/%s/%d", CompanyPath, companyID)); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *FreshdeskClient) CreateCompany(req *TaskCreateCompanyReq) (*TaskCreateCompanyResponse, error) {
	resp := &TaskCreateCompanyResponse{}

	httpReq := c.httpclient.R().SetBody(req).SetResult(resp)
	if _, err := httpReq.Post("/" + CompanyPath); err != nil {
		return nil, err
	}
	return resp, nil

}

// Task 1: Get Company

type TaskGetCompanyInput struct {
	CompanyID int64 `json:"company-id"`
}

type TaskGetCompanyResponse struct {
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Note         string                 `json:"note"`
	Domains      []string               `json:"domains"`
	HealthScore  string                 `json:"health_score"`
	AccountTier  string                 `json:"account_tier"`
	RenewalDate  string                 `json:"renewal_date"`
	Industry     string                 `json:"industry"`
	CreatedAt    string                 `json:"created_at"`
	UpdatedAt    string                 `json:"updated_at"`
	CustomFields map[string]interface{} `json:"custom_fields"`
}

type TaskGetCompanyOutput struct {
	Name         string                 `json:"name"`
	Description  string                 `json:"description,omitempty"`
	Note         string                 `json:"note,omitempty"`
	Domains      []string               `json:"domains"`
	HealthScore  string                 `json:"health-score,omitempty"`
	AccountTier  string                 `json:"account-tier,omitempty"`
	RenewalDate  string                 `json:"renewal-date,omitempty"`
	Industry     string                 `json:"industry,omitempty"`
	CreatedAt    string                 `json:"created-at,omitempty"`
	UpdatedAt    string                 `json:"updated-at,omitempty"`
	CustomFields map[string]interface{} `json:"custom-fields,omitempty"`
}

func (e *execution) TaskGetCompany(in *structpb.Struct) (*structpb.Struct, error) {
	inputStruct := TaskGetCompanyInput{}
	err := base.ConvertFromStructpb(in, &inputStruct)

	if err != nil {
		return nil, fmt.Errorf("failed to convert input to struct: %v", err)
	}

	resp, err := e.client.GetCompany(inputStruct.CompanyID)

	if err != nil {
		return nil, err
	}

	outputStruct := TaskGetCompanyOutput{
		Name:        resp.Name,
		Description: resp.Description,
		Note:        resp.Note,
		Domains:     *checkForNilString(&resp.Domains),
		HealthScore: resp.HealthScore,
		AccountTier: resp.AccountTier,
		RenewalDate: convertTimestampResp(resp.RenewalDate),
		Industry:    resp.Industry,
		CreatedAt:   convertTimestampResp(resp.CreatedAt),
		UpdatedAt:   convertTimestampResp(resp.UpdatedAt),
	}

	if len(resp.CustomFields) > 0 {
		outputStruct.CustomFields = resp.CustomFields
	}

	output, err := base.ConvertToStructpb(outputStruct)

	if err != nil {
		return nil, fmt.Errorf("failed to convert output to struct: %v", err)
	}

	return output, nil
}

// Task 2: Create Company

type TaskCreateCompanyInput struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Note        string   `json:"note"`
	Domains     []string `json:"domains"`
	HealthScore string   `json:"health-score"`
	AccountTier string   `json:"account-tier"`
	RenewalDate string   `json:"renewal-date"`
	Industry    string   `json:"industry"`
}

type TaskCreateCompanyReq struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Note        string   `json:"note,omitempty"`
	Domains     []string `json:"domains,omitempty"`
	HealthScore string   `json:"health_score,omitempty"`
	AccountTier string   `json:"account_tier,omitempty"`
	RenewalDate string   `json:"renewal_date,omitempty"`
	Industry    string   `json:"industry,omitempty"`
}

type TaskCreateCompanyResponse struct {
	ID        int64  `json:"id"`
	CreatedAt string `json:"created_at"`
}

type TaskCreateCompanyOutput struct {
	ID        int64  `json:"company-id"`
	CreatedAt string `json:"created-at"`
}

func (e *execution) TaskCreateCompany(in *structpb.Struct) (*structpb.Struct, error) {
	inputStruct := TaskCreateCompanyInput{}
	err := base.ConvertFromStructpb(in, &inputStruct)

	if err != nil {
		return nil, fmt.Errorf("failed to convert input to struct: %v", err)
	}

	req := &TaskCreateCompanyReq{
		Name:        inputStruct.Name,
		Description: inputStruct.Description,
		Note:        inputStruct.Note,
		Domains:     inputStruct.Domains,
		HealthScore: inputStruct.HealthScore,
		AccountTier: inputStruct.AccountTier,
		RenewalDate: inputStruct.RenewalDate,
		Industry:    inputStruct.Industry,
	}

	resp, err := e.client.CreateCompany(req)

	if err != nil {
		return nil, err
	}

	outputStruct := TaskCreateCompanyOutput{
		ID:        resp.ID,
		CreatedAt: convertTimestampResp(resp.CreatedAt),
	}

	output, err := base.ConvertToStructpb(outputStruct)

	if err != nil {
		return nil, fmt.Errorf("failed to convert output to struct: %v", err)
	}

	return output, nil
}
