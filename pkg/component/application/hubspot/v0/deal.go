package hubspot

import (
	"fmt"
	"strconv"
	"strings"

	"google.golang.org/protobuf/types/known/structpb"

	hubspot "github.com/belong-inc/go-hubspot"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

// Get Deal

type TaskGetDealInput struct {
	DealID string `json:"deal-id"`
}

type TaskGetDealResp struct {
	OwnerID    string          `json:"hubspot_owner_id,omitempty"`
	DealName   string          `json:"dealname"`
	Pipeline   string          `json:"pipeline"`
	DealStage  string          `json:"dealstage"`
	Amount     string          `json:"amount,omitempty"`
	DealType   string          `json:"dealtype,omitempty"`
	CloseDate  *hubspot.HsTime `json:"closedate,omitempty"`
	CreateDate *hubspot.HsTime `json:"createdate"`
}

type TaskGetDealOutput struct {
	OwnerID              string   `json:"owner-id,omitempty"`
	DealName             string   `json:"deal-name"`
	Pipeline             string   `json:"pipeline"`
	DealStage            string   `json:"deal-stage"`
	Amount               float64  `json:"amount,omitempty"`
	DealType             string   `json:"deal-type,omitempty"`
	CreateDate           string   `json:"create-date"`
	CloseDate            string   `json:"close-date,omitempty"`
	AssociatedContactIDs []string `json:"associated-contact-ids"`
}

func (e *execution) GetDeal(input *structpb.Struct) (*structpb.Struct, error) {
	inputStruct := TaskGetDealInput{}

	err := base.ConvertFromStructpb(input, &inputStruct)

	if err != nil {
		return nil, fmt.Errorf("failed to convert input to struct: %v", err)
	}

	// get deal information

	res, err := e.client.CRM.Deal.Get(inputStruct.DealID, &TaskGetDealResp{}, &hubspot.RequestQueryOption{Associations: []string{"contacts"}})

	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return nil, fmt.Errorf("404: unable to read response from hubspot: no deal was found")
		} else {
			return nil, err
		}
	}

	dealInfo := res.Properties.(*TaskGetDealResp)

	// get contacts associated with deal

	var dealContactList []string
	if res.Associations != nil {
		dealContactAssociation := res.Associations.Contacts.Results
		dealContactList = make([]string, len(dealContactAssociation))
		for index, value := range dealContactAssociation {
			dealContactList[index] = value.ID
		}
	} else {
		dealContactList = []string{}
	}

	// convert to outputStruct

	var amount float64

	if dealInfo.Amount != "" {
		var err error
		amount, err = strconv.ParseFloat(dealInfo.Amount, 64)

		if err != nil {
			return nil, err
		}
	}

	var closeDate string

	if dealInfo.CloseDate == nil {
		closeDate = ""
	} else {
		closeDate = dealInfo.CloseDate.String()
	}

	outputStruct := TaskGetDealOutput{
		OwnerID:              dealInfo.OwnerID,
		DealName:             dealInfo.DealName,
		Pipeline:             dealInfo.Pipeline,
		DealStage:            dealInfo.DealStage,
		Amount:               amount,
		DealType:             dealInfo.DealType,
		CreateDate:           dealInfo.CreateDate.String(),
		CloseDate:            closeDate,
		AssociatedContactIDs: dealContactList,
	}

	output, err := base.ConvertToStructpb(outputStruct)
	if err != nil {
		return nil, fmt.Errorf("failed to convert output to struct: %v", err)
	}

	return output, nil
}

// Create Deal

type TaskCreateDealInput struct {
	OwnerID                   string   `json:"owner-id"`
	DealName                  string   `json:"deal-name"`
	Pipeline                  string   `json:"pipeline"`
	DealStage                 string   `json:"deal-stage"`
	Amount                    float64  `json:"amount"`
	DealType                  string   `json:"deal-type"`
	CloseDate                 string   `json:"close-date"`
	CreateContactsAssociation []string `json:"create-contacts-association"`
}

type TaskCreateDealReq struct {
	OwnerID   string `json:"hubspot_owner_id,omitempty"`
	DealName  string `json:"dealname"`
	Pipeline  string `json:"pipeline"`
	DealStage string `json:"dealstage"`
	Amount    string `json:"amount,omitempty"`
	DealType  string `json:"dealtype,omitempty"`
	CloseDate string `json:"closedate,omitempty"`
	DealID    string `json:"hs_object_id"`
}

type TaskCreateDealOutput struct {
	DealID string `json:"deal-id"`
}

func (e *execution) CreateDeal(input *structpb.Struct) (*structpb.Struct, error) {

	inputStruct := TaskCreateDealInput{}
	err := base.ConvertFromStructpb(input, &inputStruct)

	if err != nil {
		return nil, fmt.Errorf("failed to convert input to struct: %v", err)
	}

	var amount string
	if inputStruct.Amount != 0 {
		amount = strconv.FormatFloat(inputStruct.Amount, 'f', -1, 64)
	}

	req := TaskCreateDealReq{
		OwnerID:   inputStruct.OwnerID,
		DealName:  inputStruct.DealName,
		Pipeline:  inputStruct.Pipeline,
		DealStage: inputStruct.DealStage,
		Amount:    amount,
		DealType:  inputStruct.DealType,
		CloseDate: inputStruct.CloseDate,
	}

	res, err := e.client.CRM.Deal.Create(&req)

	if err != nil {
		return nil, err
	}

	// get deal ID
	dealID := res.Properties.(*TaskCreateDealReq).DealID

	outputStruct := TaskCreateDealOutput{DealID: dealID}

	output, err := base.ConvertToStructpb(outputStruct)

	if err != nil {
		return nil, fmt.Errorf("failed to convert output to struct: %v", err)
	}

	// This section is for creating associations (deal -> object)
	if len(inputStruct.CreateContactsAssociation) != 0 {
		err := CreateAssociation(&outputStruct.DealID, &inputStruct.CreateContactsAssociation, "deal", "contact", e)

		if err != nil {
			return nil, err
		}
	}

	return output, nil
}

// Update Deal

type TaskUpdateDealInput struct {
	DealID                    string   `json:"deal-id"`
	OwnerID                   string   `json:"owner-id,omitempty"`
	DealName                  string   `json:"deal-name"`
	Pipeline                  string   `json:"pipeline"`
	DealStage                 string   `json:"deal-stage"`
	Amount                    float64  `json:"amount"`
	DealType                  string   `json:"deal-type"`
	CloseDate                 string   `json:"close-date"`
	CreateContactsAssociation []string `json:"create-contacts-association"`
}

type TaskUpdateDealReq struct {
	OwnerID         string `json:"hubspot_owner_id,omitempty"`
	DealName        string `json:"dealname"`
	Pipeline        string `json:"pipeline"`
	DealStage       string `json:"dealstage"`
	Amount          string `json:"amount,omitempty"`
	DealType        string `json:"dealtype,omitempty"`
	CloseDate       string `json:"closedate,omitempty"`
	UpdatedByUserID string `json:"hs_updated_by_user_id,omitempty"`
}

// no response struct because it uses the req struct as well to store the response.

type TaskUpdateDealOutput struct {
	UpdatedByUserID string `json:"updated-by-user-id"`
	UpdatedAt       string `json:"updated-at"` //mostly just used to signal that it is updated successfully.
}

func (e *execution) UpdateDeal(input *structpb.Struct) (*structpb.Struct, error) {
	inputStruct := TaskUpdateDealInput{}
	err := base.ConvertFromStructpb(input, &inputStruct)

	if err != nil {
		return nil, fmt.Errorf("failed to convert input to struct: %v", err)
	}

	var amount string
	if inputStruct.Amount != 0 {
		amount = strconv.FormatFloat(inputStruct.Amount, 'f', -1, 64)
	}

	req := TaskUpdateDealReq{
		OwnerID:   inputStruct.OwnerID,
		DealName:  inputStruct.DealName,
		Pipeline:  inputStruct.Pipeline,
		DealStage: inputStruct.DealStage,
		Amount:    amount,
		DealType:  inputStruct.DealType,
		CloseDate: inputStruct.CloseDate,
	}

	res, err := e.client.CRM.Deal.Update(inputStruct.DealID, &req)

	if err != nil {
		return nil, err
	}

	// get the user ID which updated the deal
	userID := res.Properties.(*TaskUpdateDealReq).UpdatedByUserID

	outputStruct := TaskUpdateDealOutput{
		UpdatedByUserID: userID,
		UpdatedAt:       res.UpdatedAt.String(),
	}

	output, err := base.ConvertToStructpb(outputStruct)

	if err != nil {
		return nil, fmt.Errorf("failed to convert output to struct: %v", err)
	}

	// This section is for creating associations (deal -> object)
	if len(inputStruct.CreateContactsAssociation) != 0 {
		err := CreateAssociation(&inputStruct.DealID, &inputStruct.CreateContactsAssociation, "deal", "contact", e)

		if err != nil {
			return nil, err
		}
	}

	return output, nil
}
