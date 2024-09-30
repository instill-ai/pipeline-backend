package freshdesk

import (
	"fmt"
	"strings"

	"github.com/go-resty/resty/v2"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func (c *FreshdeskClient) GetAll(objectType string, pagination bool, paginationPath string) ([]TaskGetAllResponse, string, error) {

	resp := []TaskGetAllResponse{}

	httpReq := c.httpclient.R().SetResult(&resp)

	var rawResp *resty.Response
	var err error
	if !pagination {

		var path string
		if objectType != "Skills" {
			path = strings.ToLower(objectType)
		} else {
			path = "admin/skills"
		}

		rawResp, err = httpReq.Get(fmt.Sprintf("/%s", path))

	} else {
		rawResp, err = httpReq.Get(paginationPath)
	}

	if err != nil {
		return nil, "", err
	}

	// Will exist if there is a next page
	linkHeader := rawResp.Header().Get("Link")

	var nextPage string
	if linkHeader != "" {
		startIndex := strings.Index(linkHeader, "<")
		endIndex := strings.Index(linkHeader, ">")
		nextPage = linkHeader[startIndex+1 : endIndex]

		return resp, nextPage, nil
	}

	return resp, "", nil
}

// Task 1: Get All
type TaskGetAllInput struct {
	ObjectType string `json:"object-type"`
	Length     int    `json:"length"`
}

type TaskGetAllResponse struct {
	ID int64 `json:"id"`
}

type TaskGetAllOutput struct {
	IDs      []int64 `json:"ids"`
	IDLength int     `json:"id-length"`
}

func (e *execution) TaskGetAll(in *structpb.Struct) (*structpb.Struct, error) {
	inputStruct := TaskGetAllInput{}
	err := base.ConvertFromStructpb(in, &inputStruct)

	if err != nil {
		return nil, fmt.Errorf("failed to convert input to struct: %v", err)
	}

	if inputStruct.Length < 0 || inputStruct.Length > 500 {
		return nil, fmt.Errorf("please set the limit between 0 and 500")
	}

	resp, paginationPath, err := e.client.GetAll(inputStruct.ObjectType, false, "")

	if err != nil {
		return nil, err
	}

	counter := 0
	outputStruct := TaskGetAllOutput{}
	for _, value := range resp {
		outputStruct.IDs = append(outputStruct.IDs, value.ID)
		counter += 1
		if counter == inputStruct.Length {
			break
		}
	}

	if counter < inputStruct.Length {
		for paginationPath != "" && counter < inputStruct.Length {
			respPage, nextPage, err := e.client.GetAll(inputStruct.ObjectType, true, paginationPath)

			if err != nil {
				return nil, err
			}

			for _, value := range respPage {
				outputStruct.IDs = append(outputStruct.IDs, value.ID)
				counter += 1
				if counter == inputStruct.Length {
					break
				}
			}
			paginationPath = nextPage
		}
	}

	outputStruct.IDLength = len(outputStruct.IDs)

	output, err := base.ConvertToStructpb(outputStruct)

	if err != nil {
		return nil, fmt.Errorf("failed to convert output to struct: %v", err)
	}

	return output, nil
}
