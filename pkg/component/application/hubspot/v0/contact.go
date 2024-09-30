package hubspot

import (
	"fmt"
	"strings"

	"google.golang.org/protobuf/types/known/structpb"

	hubspot "github.com/belong-inc/go-hubspot"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

// Get Contact

type TaskGetContactInput struct {
	ContactIDOrEmail string `json:"contact-id-or-email"`
}

type TaskGetContactResp struct {
	OwnerID        string `json:"hubspot_owner_id,omitempty"`
	Email          string `json:"email,omitempty"`
	FirstName      string `json:"firstname,omitempty"`
	LastName       string `json:"lastname,omitempty"`
	PhoneNumber    string `json:"phone,omitempty"`
	Company        string `json:"company,omitempty"`
	JobTitle       string `json:"jobtitle,omitempty"`
	LifecycleStage string `json:"lifecyclestage,omitempty"`
	LeadStatus     string `json:"hs_lead_status,omitempty"`
	ContactID      string `json:"hs_object_id"`
}

type TaskGetContactOutput struct {
	OwnerID        string `json:"owner-id,omitempty"`
	Email          string `json:"email,omitempty"`
	FirstName      string `json:"first-name,omitempty"`
	LastName       string `json:"last-name,omitempty"`
	PhoneNumber    string `json:"phone-number,omitempty"`
	Company        string `json:"company,omitempty"`
	JobTitle       string `json:"job-title,omitempty"`
	LifecycleStage string `json:"lifecycle-stage,omitempty"`
	LeadStatus     string `json:"lead-status,omitempty"`
	ContactID      string `json:"contact-id"`
}

func (e *execution) GetContact(input *structpb.Struct) (*structpb.Struct, error) {
	inputStruct := TaskGetContactInput{}

	err := base.ConvertFromStructpb(input, &inputStruct)

	if err != nil {
		return nil, fmt.Errorf("failed to convert input to struct: %v", err)
	}

	uniqueKey := inputStruct.ContactIDOrEmail

	// If user enter email instead of contact ID
	if strings.Contains(uniqueKey, "@") {
		uniqueKey += "?idProperty=email"
	}

	res, err := e.client.CRM.Contact.Get(uniqueKey, &TaskGetContactResp{}, &hubspot.RequestQueryOption{CustomProperties: []string{"phone"}})

	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return nil, fmt.Errorf("404: unable to read response from hubspot: no contact was found")
		} else {
			return nil, err
		}
	}

	contactInfo := res.Properties.(*TaskGetContactResp)

	outputStruct := TaskGetContactOutput(*contactInfo)

	output, err := base.ConvertToStructpb(outputStruct)

	if err != nil {
		return nil, fmt.Errorf("failed to convert output to struct: %v", err)
	}

	return output, nil
}

// Create Contact

type TaskCreateContactInput struct {
	OwnerID                    string   `json:"owner-id"`
	Email                      string   `json:"email"`
	FirstName                  string   `json:"first-name"`
	LastName                   string   `json:"last-name"`
	PhoneNumber                string   `json:"phone-number"`
	Company                    string   `json:"company"`
	JobTitle                   string   `json:"job-title"`
	LifecycleStage             string   `json:"lifecycle-stage"`
	LeadStatus                 string   `json:"lead-status"`
	CreateDealsAssociation     []string `json:"create-deals-association"`
	CreateCompaniesAssociation []string `json:"create-companies-association"`
	CreateTicketsAssociation   []string `json:"create-tickets-association"`
}

type TaskCreateContactReq struct {
	OwnerID        string `json:"hubspot_owner_id,omitempty"`
	Email          string `json:"email,omitempty"`
	FirstName      string `json:"firstname,omitempty"`
	LastName       string `json:"lastname,omitempty"`
	PhoneNumber    string `json:"phone,omitempty"`
	Company        string `json:"company,omitempty"`
	JobTitle       string `json:"jobtitle,omitempty"`
	LifecycleStage string `json:"lifecyclestage,omitempty"`
	LeadStatus     string `json:"hs_lead_status,omitempty"`
	ContactID      string `json:"hs_object_id"`
}

type TaskCreateContactOutput struct {
	ContactID string `json:"contact-id"`
}

func (e *execution) CreateContact(input *structpb.Struct) (*structpb.Struct, error) {
	inputStruct := TaskCreateContactInput{}

	err := base.ConvertFromStructpb(input, &inputStruct)

	if err != nil {
		return nil, fmt.Errorf("failed to convert input to struct: %v", err)
	}

	req := TaskCreateContactReq{
		OwnerID:        inputStruct.OwnerID,
		Email:          inputStruct.Email,
		FirstName:      inputStruct.FirstName,
		LastName:       inputStruct.LastName,
		PhoneNumber:    inputStruct.PhoneNumber,
		Company:        inputStruct.Company,
		JobTitle:       inputStruct.JobTitle,
		LifecycleStage: inputStruct.LifecycleStage,
		LeadStatus:     inputStruct.LeadStatus,
	}

	res, err := e.client.CRM.Contact.Create(&req)

	if err != nil {
		return nil, err
	}

	contactID := res.Properties.(*TaskCreateContactReq).ContactID

	outputStruct := TaskCreateContactOutput{ContactID: contactID}

	output, err := base.ConvertToStructpb(outputStruct)

	if err != nil {
		return nil, fmt.Errorf("failed to convert output to struct: %v", err)
	}

	// This section is for creating associations (contact -> object)

	if len(inputStruct.CreateDealsAssociation) != 0 {
		err := CreateAssociation(&outputStruct.ContactID, &inputStruct.CreateDealsAssociation, "contact", "deal", e)

		if err != nil {
			return nil, err
		}
	}
	if len(inputStruct.CreateCompaniesAssociation) != 0 {
		err := CreateAssociation(&outputStruct.ContactID, &inputStruct.CreateCompaniesAssociation, "contact", "company", e)

		if err != nil {
			return nil, err
		}
	}
	if len(inputStruct.CreateTicketsAssociation) != 0 {
		err := CreateAssociation(&outputStruct.ContactID, &inputStruct.CreateTicketsAssociation, "contact", "ticket", e)

		if err != nil {
			return nil, err
		}
	}
	return output, nil
}
