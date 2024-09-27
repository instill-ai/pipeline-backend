package freshdesk

import (
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

const (
	ProductPath = "products"
)

// API function for Product

func (c *FreshdeskClient) GetProduct(productID int64) (*TaskGetProductResponse, error) {
	resp := &TaskGetProductResponse{}

	httpReq := c.httpclient.R().SetResult(resp)
	if _, err := httpReq.Get(fmt.Sprintf("/%s/%d", ProductPath, productID)); err != nil {
		return nil, err
	}
	return resp, nil
}

// Task 1: Get Product

type TaskGetProductInput struct {
	ProductID int64 `json:"product-id"`
}

type TaskGetProductResponse struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	PrimaryEmail string `json:"primary_email"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
	Default      bool   `json:"default"`
}

type TaskGetProductOutput struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	PrimaryEmail string `json:"primary-email"`
	CreatedAt    string `json:"created-at"`
	UpdatedAt    string `json:"updated-at"`
	Default      bool   `json:"default"`
}

func (e *execution) TaskGetProduct(in *structpb.Struct) (*structpb.Struct, error) {
	inputStruct := TaskGetProductInput{}
	err := base.ConvertFromStructpb(in, &inputStruct)

	if err != nil {
		return nil, fmt.Errorf("failed to convert input to struct: %v", err)
	}

	resp, err := e.client.GetProduct(inputStruct.ProductID)

	if err != nil {
		return nil, err
	}

	outputStruct := TaskGetProductOutput{
		Name:         resp.Name,
		Description:  resp.Description,
		PrimaryEmail: resp.PrimaryEmail,
		CreatedAt:    convertTimestampResp(resp.CreatedAt),
		UpdatedAt:    convertTimestampResp(resp.UpdatedAt),
		Default:      resp.Default,
	}

	output, err := base.ConvertToStructpb(outputStruct)

	if err != nil {
		return nil, fmt.Errorf("failed to convert output to struct: %v", err)
	}

	return output, nil

}
