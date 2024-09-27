package hubspot

import (
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"

	hubspot "github.com/belong-inc/go-hubspot"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

// Get All is a custom feature
// Will implement it following go-hubspot sdk format

// API function for Get All

type GetAllService interface {
	Get(objectType string, param string) (*TaskGetAllResp, error)
}

type GetAllServiceOp struct {
	client *hubspot.Client
}

func (s *GetAllServiceOp) Get(objectType string, param string) (*TaskGetAllResp, error) {
	resource := &TaskGetAllResp{}

	var relativePath string
	switch objectType {
	case "Companies", "Deals", "Contacts", "Tickets":
		relativePath = "/crm/v3/objects/" + objectType
	case "Threads":
		relativePath = "/conversations/v3/conversations/threads"
	case "Owners":
		relativePath = "/crm/v3/owners"
	}

	relativePath += param

	if err := s.client.Get(relativePath, resource, nil); err != nil {
		return nil, err
	}
	return resource, nil
}

// Get All

type TaskGetAllInput struct {
	ObjectType string `json:"object-type"`
}

type TaskGetAllResp struct {
	Results []taskGetAllRespResult `json:"results"`
	Paging  *taskGetAllRespPaging  `json:"paging,omitempty"`
}

type taskGetAllRespResult struct {
	ID string `json:"id"`
}

type taskGetAllRespPaging struct {
	Next struct {
		Link  string `json:"link"`
		After string `json:"after"`
	} `json:"next"`
}

type TaskGetAllOutput struct {
	ObjectIDs       []string `json:"object-ids"`
	ObjectIDsLength int      `json:"object-ids-length"`
}

func (e *execution) GetAll(input *structpb.Struct) (*structpb.Struct, error) {
	inputStruct := TaskGetAllInput{}

	err := base.ConvertFromStructpb(input, &inputStruct)

	if err != nil {
		return nil, fmt.Errorf("failed to convert input to struct: %v", err)
	}

	outputStruct := TaskGetAllOutput{}

	var param string
	// need to use for loop because each API call can only retrieve 10 objects. So need to do multiple API calls to get all objects if it is more than 10.
	for {
		res, err := e.client.GetAll.Get(inputStruct.ObjectType, param)

		if err != nil {
			return nil, err
		}

		for _, result := range res.Results {
			outputStruct.ObjectIDs = append(outputStruct.ObjectIDs, result.ID)
		}

		if res.Paging == nil || len(res.Results) == 0 {
			if len(outputStruct.ObjectIDs) == 0 {
				outputStruct.ObjectIDs = []string{}
			}
			break
		}

		param = fmt.Sprintf("?after=%s", res.Paging.Next.After)
	}

	outputStruct.ObjectIDsLength = len(outputStruct.ObjectIDs)

	output, err := base.ConvertToStructpb(outputStruct)
	if err != nil {
		return nil, fmt.Errorf("failed to convert output to struct: %v", err)
	}

	return output, nil
}
