package hubspot

import (
	"fmt"
	"strings"

	"google.golang.org/protobuf/types/known/structpb"

	hubspot "github.com/belong-inc/go-hubspot"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

// Retrieve Association is a custom feature
// Will implement it following go-hubspot sdk format

// API functions for Retrieve Association

type RetrieveAssociationService interface {
	GetThreadID(contactID string, paging bool, pagingPath string) (*TaskRetrieveAssociationThreadResp, error)
	GetCrmID(contactID string, objectType string, paging bool, pagingPath string) (interface{}, error)
}

type RetrieveAssociationServiceOp struct {
	retrieveCrmIDPath    string
	retrieveThreadIDPath string
	client               *hubspot.Client
}

func (s *RetrieveAssociationServiceOp) GetThreadID(contactID string, paging bool, pagingPath string) (*TaskRetrieveAssociationThreadResp, error) {
	resource := &TaskRetrieveAssociationThreadResp{}

	var path string
	if !paging {
		path = s.retrieveThreadIDPath + contactID
	} else {
		path = pagingPath
	}

	if err := s.client.Get(path, resource, nil); err != nil {
		return nil, err
	}
	return resource, nil
}

func (s *RetrieveAssociationServiceOp) GetCrmID(contactID string, objectType string, paging bool, pagingPath string) (interface{}, error) {

	if !paging {
		resource := &TaskRetrieveAssociationCrmResp{}

		contactIDInput := TaskRetrieveAssociationCrmReqID{ContactID: contactID}

		req := &TaskRetrieveAssociationCrmReq{}
		req.Input = append(req.Input, contactIDInput)

		path := s.retrieveCrmIDPath + "/" + objectType + "/batch/read"

		if err := s.client.Post(path, req, resource); err != nil {
			return nil, err
		}

		return resource, nil
	} else {

		resource := &TaskRetrieveAssociationCrmPagingResp{}

		if err := s.client.Get(pagingPath, resource, nil); err != nil {
			return nil, err
		}

		return resource, nil

	}

}

// Retrieve Association: use contact id to get the object ID associated with it

type TaskRetrieveAssociationInput struct {
	ContactID  string `json:"contact-id"`
	ObjectType string `json:"object-type"`
}

// This struct is used for both CRM and Threads
type taskRetrieveAssociationRespPaging struct {
	Next struct {
		Link  string `json:"link"`
		After string `json:"after"`
	} `json:"next"`
}

// Retrieve Association Task is mainly divided into two:
// 1. GetThreadID
// 2. GetCrmID
// Basically, these two will have seperate structs for handling request/response

// For GetThreadID

type TaskRetrieveAssociationThreadResp struct {
	Results []struct {
		ID string `json:"id"`
	} `json:"results"`
	Paging *taskRetrieveAssociationRespPaging `json:"paging,omitempty"`
}

// For GetCrmID

type TaskRetrieveAssociationCrmReq struct {
	Input []TaskRetrieveAssociationCrmReqID `json:"inputs"`
}

type TaskRetrieveAssociationCrmReqID struct {
	ContactID string `json:"id"`
}

// RetrieveAssociation CRM can have 2 responses
// if it is more than 100, it will have  a paging link, which user can do another API call to it.
// it has a different response format

type TaskRetrieveAssociationCrmResp struct {
	Results []taskRetrieveAssociationCrmRespResult `json:"results"`
}

type taskRetrieveAssociationCrmRespResult struct {
	IDArray []struct {
		ID string `json:"id"`
	} `json:"to"`
	Paging *taskRetrieveAssociationRespPaging `json:"paging,omitempty"`
}

type TaskRetrieveAssociationCrmPagingResp struct {
	Results []struct {
		ID string `json:"id"`
	} `json:"results"`
	Paging *taskRetrieveAssociationRespPaging `json:"paging,omitempty"`
}

// Retrieve Association Output

type TaskRetrieveAssociationOutput struct {
	ObjectIDs       []string `json:"object-ids"`
	ObjectIDsLength int      `json:"object-ids-length"`
}

func (e *execution) RetrieveAssociation(input *structpb.Struct) (*structpb.Struct, error) {
	inputStruct := TaskRetrieveAssociationInput{}
	err := base.ConvertFromStructpb(input, &inputStruct)

	if err != nil {
		return nil, fmt.Errorf("failed to convert input to struct: %v", err)
	}

	// API calls to retrieve association for Threads and CRM objects are different

	var objectIDs []string

	switch inputStruct.ObjectType {
	case "Contacts", "Companies", "Deals", "Tickets":
		// To handle CRM objects
		res, err := e.client.RetrieveAssociation.GetCrmID(inputStruct.ContactID, inputStruct.ObjectType, false, "")

		if err != nil {
			return nil, err
		}

		crmRes := res.(*TaskRetrieveAssociationCrmResp)

		// no object ID associated with the contact ID, just break
		if len(crmRes.Results) == 0 {
			break
		}

		// only take the first Result, because the input is only one contact id
		objectIDs = make([]string, len(crmRes.Results[0].IDArray))
		for index, value := range crmRes.Results[0].IDArray {
			objectIDs[index] = value.ID
		}

		// if there is a paging link, do another API call to get the rest of the object IDs
		if crmRes.Results[0].Paging != nil {
			for {
				// need to trim because go-hubspot sdk get function will add the base URL
				pagingRelativePath := strings.TrimPrefix(crmRes.Results[0].Paging.Next.Link, "https://api.hubapi.com/")

				res, err := e.client.RetrieveAssociation.GetCrmID(inputStruct.ContactID, inputStruct.ObjectType, true, pagingRelativePath)

				if err != nil {
					return nil, err
				}

				crmPagingRes := res.(*TaskRetrieveAssociationCrmPagingResp)

				for _, value := range crmPagingRes.Results {
					objectIDs = append(objectIDs, value.ID)
				}

				if crmPagingRes.Paging == nil || len(crmPagingRes.Results) == 0 {
					break
				}
			}
		}

	case "Threads":
		res, err := e.client.RetrieveAssociation.GetThreadID(inputStruct.ContactID, false, "")

		if err != nil {
			return nil, err
		}

		if len(res.Results) == 0 {
			break
		}

		objectIDs = make([]string, len(res.Results))
		for index, value := range res.Results {
			objectIDs[index] = value.ID
		}

		if res.Paging != nil {
			for {
				// need to trim because go-hubspot sdk get function will add the base URL
				pagingRelativePath := strings.TrimPrefix(res.Paging.Next.Link, "https://api.hubapi.com/")

				res, err := e.client.RetrieveAssociation.GetThreadID(inputStruct.ContactID, true, pagingRelativePath)

				if err != nil {
					return nil, err
				}

				for _, value := range res.Results {
					objectIDs = append(objectIDs, value.ID)
				}

				if res.Paging == nil || len(res.Results) == 0 {
					break
				}
			}
		}
	}

	if len(objectIDs) == 0 {
		objectIDs = []string{}
	}

	outputStruct := TaskRetrieveAssociationOutput{
		ObjectIDs:       objectIDs,
		ObjectIDsLength: len(objectIDs),
	}

	output, err := base.ConvertToStructpb(outputStruct)

	if err != nil {
		return nil, fmt.Errorf("failed to convert output to struct: %v", err)
	}

	return output, nil
}

// Create Association (not a task)
// This section (create association) is used in:
// create contact task to create contact -> objects (company, ticket, deal) association
// create company task to create company -> contact association
// create deal task to create deal -> contact association
// create ticket task to create ticket -> contact association

type CreateAssociationReq struct {
	Associations []association `json:"inputs"`
}

type association struct {
	From struct {
		ID string `json:"id"`
	} `json:"from"`
	To struct {
		ID string `json:"id"`
	} `json:"to"`
	Type string `json:"type"`
}

type CreateAssociationResponse struct {
	Status string `json:"status"`
}

// CreateAssociation is used to create batch associations between objects

func CreateAssociation(fromID *string, toIDs *[]string, fromObjectType string, toObjectType string, e *execution) error {
	req := &CreateAssociationReq{
		Associations: make([]association, len(*toIDs)),
	}

	//for any association created related to company, it will use non-primary label.
	//for more info: https://developers.hubspot.com/beta-docs/guides/api/crm/associations#association-type-id-values

	var associationType string
	if toObjectType == "company" {
		switch fromObjectType { //use switch here in case other association of object -> company want to be created in the future
		case "contact":
			associationType = "279"
		}

	} else if fromObjectType == "company" {
		switch toObjectType {
		case "contact":
			associationType = "280"
		}
	} else {
		associationType = fmt.Sprintf("%s_to_%s", fromObjectType, toObjectType)
	}

	for index, toID := range *toIDs {

		req.Associations[index] = association{
			From: struct {
				ID string `json:"id"`
			}{
				ID: *fromID,
			},
			To: struct {
				ID string `json:"id"`
			}{
				ID: toID,
			},
			Type: associationType,
		}
	}

	createAssociationPath := fmt.Sprintf("crm/v3/associations/%s/%s/batch/create", fromObjectType, toObjectType)

	resp := &CreateAssociationResponse{}

	if err := e.client.Post(createAssociationPath, req, resp); err != nil {
		return err
	}

	if resp.Status != "COMPLETE" {
		return fmt.Errorf("failed to create association")
	}

	return nil
}
