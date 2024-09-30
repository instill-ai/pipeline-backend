package hubspot

import (
	"fmt"
	"strings"

	"google.golang.org/protobuf/types/known/structpb"

	hubspot "github.com/belong-inc/go-hubspot"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

// following go-hubspot sdk format

// API functions for Ticket

type TicketService interface {
	Get(ticketID string) (*hubspot.ResponseResource, error)
	Create(ticket *TaskCreateTicketReq) (*hubspot.ResponseResource, error)
	Update(ticketID string, ticket *TaskUpdateTicketReq) (*hubspot.ResponseResource, error)
}

type TicketServiceOp struct {
	client     *hubspot.Client
	ticketPath string
}

var ticketProperties = []string{
	"hubspot_owner_id",
	"subject",
	"hs_pipeline_stage",
	"hs_pipeline",
	"hs_ticket_category",
	"hs_ticket_priority",
	"source_type",
	"hs_object_source_label",
	"createdate",
	"hs_lastmodifieddate",
}

func (s *TicketServiceOp) Get(ticketID string) (*hubspot.ResponseResource, error) {
	resource := &hubspot.ResponseResource{Properties: &TaskGetTicketResp{}}
	option := &hubspot.RequestQueryOption{Properties: ticketProperties, Associations: []string{"contacts"}}
	if err := s.client.Get(s.ticketPath+"/"+ticketID, resource, option); err != nil {
		return nil, err
	}

	return resource, nil
}

func (s *TicketServiceOp) Create(ticket *TaskCreateTicketReq) (*hubspot.ResponseResource, error) {
	req := &hubspot.RequestPayload{Properties: ticket}
	resource := &hubspot.ResponseResource{Properties: ticket}
	if err := s.client.Post(s.ticketPath, req, resource); err != nil {
		return nil, err
	}
	return resource, nil
}

func (s *TicketServiceOp) Update(ticketID string, ticket *TaskUpdateTicketReq) (*hubspot.ResponseResource, error) {
	req := &hubspot.RequestPayload{Properties: ticket}
	resource := &hubspot.ResponseResource{} //leave the Properties blank because we don't use any values from the properties.
	if err := s.client.Patch(s.ticketPath+"/"+ticketID, req, resource); err != nil {
		return nil, err
	}
	return resource, nil
}

// Get Ticket

type TaskGetTicketInput struct {
	TicketID string `json:"ticket-id"`
}

type TaskGetTicketResp struct {
	OwnerID          string          `json:"hubspot_owner_id,omitempty"`
	TicketName       string          `json:"subject"`
	TicketStatus     string          `json:"hs_pipeline_stage"`
	Pipeline         string          `json:"hs_pipeline"`
	Category         string          `json:"hs_ticket_category,omitempty"`
	Priority         string          `json:"hs_ticket_priority,omitempty"`
	Source           string          `json:"source_type,omitempty"`
	RecordSource     string          `json:"hs_object_source_label,omitempty"`
	CreateDate       *hubspot.HsTime `json:"createdate"`
	LastModifiedDate *hubspot.HsTime `json:"hs_lastmodifieddate"`
	TicketID         string          `json:"hs_object_id"`
}

type TaskGetTicketOutput struct {
	OwnerID              string   `json:"owner-id,omitempty"`
	TicketName           string   `json:"ticket-name"`
	TicketStatus         string   `json:"ticket-status"`
	Pipeline             string   `json:"pipeline"`
	Category             []string `json:"categories"`
	Priority             string   `json:"priority,omitempty"`
	Source               string   `json:"source,omitempty"`
	RecordSource         string   `json:"record-source,omitempty"`
	CreateDate           string   `json:"create-date"`
	LastModifiedDate     string   `json:"last-modified-date"`
	AssociatedContactIDs []string `json:"associated-contact-ids"`
}

func (e *execution) GetTicket(input *structpb.Struct) (*structpb.Struct, error) {

	inputStruct := TaskGetTicketInput{}

	err := base.ConvertFromStructpb(input, &inputStruct)

	if err != nil {
		return nil, fmt.Errorf("failed to convert input to struct: %v", err)
	}

	res, err := e.client.Ticket.Get(inputStruct.TicketID)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return nil, fmt.Errorf("404: unable to read response from hubspot: no ticket was found")
		} else {
			return nil, err
		}
	}

	ticketInfo := res.Properties.(*TaskGetTicketResp)

	// get contacts associated with ticket

	var ticketContactList []string
	if res.Associations != nil {
		ticketContactAssociation := res.Associations.Contacts.Results
		ticketContactList = make([]string, len(ticketContactAssociation))
		for index, value := range ticketContactAssociation {
			ticketContactList[index] = value.ID
		}
	} else {
		ticketContactList = []string{}
	}

	var categoryValues []string
	if ticketInfo.Category != "" {
		categoryValues = strings.Split(ticketInfo.Category, ";")
	} else {
		categoryValues = []string{}
	}

	outputStruct := TaskGetTicketOutput{
		OwnerID:              ticketInfo.OwnerID,
		TicketName:           ticketInfo.TicketName,
		TicketStatus:         ticketInfo.TicketStatus,
		Pipeline:             ticketInfo.Pipeline,
		Category:             categoryValues,
		Priority:             ticketInfo.Priority,
		Source:               ticketInfo.Source,
		RecordSource:         ticketInfo.RecordSource,
		CreateDate:           ticketInfo.CreateDate.String(),
		LastModifiedDate:     ticketInfo.LastModifiedDate.String(),
		AssociatedContactIDs: ticketContactList,
	}

	output, err := base.ConvertToStructpb(outputStruct)
	if err != nil {
		return nil, fmt.Errorf("failed to convert output to struct: %v", err)
	}

	return output, nil
}

// Create Ticket
type TaskCreateTicketInput struct {
	OwnerID                   string   `json:"owner-id"`
	TicketName                string   `json:"ticket-name"`
	TicketStatus              string   `json:"ticket-status"`
	Pipeline                  string   `json:"pipeline"`
	Category                  []string `json:"categories"`
	Priority                  string   `json:"priority"`
	Source                    string   `json:"source"`
	CreateContactsAssociation []string `json:"create-contacts-association"`
}

type TaskCreateTicketReq struct {
	OwnerID      string `json:"hubspot_owner_id,omitempty"`
	TicketName   string `json:"subject"`
	TicketStatus string `json:"hs_pipeline_stage"`
	Pipeline     string `json:"hs_pipeline"`
	Category     string `json:"hs_ticket_category,omitempty"`
	Priority     string `json:"hs_ticket_priority,omitempty"`
	Source       string `json:"source_type,omitempty"`
	TicketID     string `json:"hs_object_id"`
}

type TaskCreateTicketOutput struct {
	TicketID string `json:"ticket-id"`
}

func (e *execution) CreateTicket(input *structpb.Struct) (*structpb.Struct, error) {

	inputStruct := TaskCreateTicketInput{}
	err := base.ConvertFromStructpb(input, &inputStruct)

	if err != nil {
		return nil, fmt.Errorf("failed to convert input to struct: %v", err)
	}

	req := TaskCreateTicketReq{
		OwnerID:      inputStruct.OwnerID,
		TicketName:   inputStruct.TicketName,
		TicketStatus: inputStruct.TicketStatus,
		Pipeline:     inputStruct.Pipeline,
		Category:     strings.Join(inputStruct.Category, ";"),
		Priority:     inputStruct.Priority,
		Source:       inputStruct.Source,
	}

	res, err := e.client.Ticket.Create(&req)

	if err != nil {
		return nil, err
	}

	// get ticket ID
	ticketID := res.Properties.(*TaskCreateTicketReq).TicketID

	outputStruct := TaskCreateTicketOutput{TicketID: ticketID}

	output, err := base.ConvertToStructpb(outputStruct)

	if err != nil {
		return nil, fmt.Errorf("failed to convert output to struct: %v", err)
	}

	// This section is for creating associations (ticket -> object)
	if len(inputStruct.CreateContactsAssociation) != 0 {
		err := CreateAssociation(&outputStruct.TicketID, &inputStruct.CreateContactsAssociation, "ticket", "contact", e)

		if err != nil {
			return nil, err
		}
	}

	return output, nil
}

// Update Ticket
type TaskUpdateTicketInput struct {
	TicketID                  string   `json:"ticket-id"`
	OwnerID                   string   `json:"owner-id"`
	TicketName                string   `json:"ticket-name"`
	TicketStatus              string   `json:"ticket-status"`
	Pipeline                  string   `json:"pipeline"`
	Category                  []string `json:"categories"`
	Priority                  string   `json:"priority"`
	Source                    string   `json:"source"`
	CreateContactsAssociation []string `json:"create-contacts-association"`
}

type TaskUpdateTicketReq struct {
	OwnerID      string `json:"hubspot_owner_id,omitempty"`
	TicketName   string `json:"subject"`
	TicketStatus string `json:"hs_pipeline_stage"`
	Pipeline     string `json:"hs_pipeline"`
	Category     string `json:"hs_ticket_category,omitempty"`
	Priority     string `json:"hs_ticket_priority,omitempty"`
	Source       string `json:"source_type,omitempty"`
}

type TaskUpdateTicketOutput struct {
	UpdatedAt string `json:"updated-at"` //mostly just used to signal that it is updated successfully.
} // unlike UpdateDeal, UpdateTicket doesn't have UpdatedByUserID because the API response doesn't return that value for some reason.

func (e *execution) UpdateTicket(input *structpb.Struct) (*structpb.Struct, error) {

	inputStruct := TaskUpdateTicketInput{}
	err := base.ConvertFromStructpb(input, &inputStruct)

	if err != nil {
		return nil, fmt.Errorf("failed to convert input to struct: %v", err)
	}

	req := TaskUpdateTicketReq{
		OwnerID:      inputStruct.OwnerID,
		TicketName:   inputStruct.TicketName,
		TicketStatus: inputStruct.TicketStatus,
		Pipeline:     inputStruct.Pipeline,
		Category:     strings.Join(inputStruct.Category, ";"),
		Priority:     inputStruct.Priority,
		Source:       inputStruct.Source,
	}

	res, err := e.client.Ticket.Update(inputStruct.TicketID, &req)

	if err != nil {
		return nil, err
	}

	outputStruct := TaskUpdateTicketOutput{UpdatedAt: res.UpdatedAt.String()}

	output, err := base.ConvertToStructpb(outputStruct)

	if err != nil {
		return nil, fmt.Errorf("failed to convert output to struct: %v", err)
	}

	// This section is for creating associations (ticket -> object)
	if len(inputStruct.CreateContactsAssociation) != 0 {
		err := CreateAssociation(&inputStruct.TicketID, &inputStruct.CreateContactsAssociation, "ticket", "contact", e)

		if err != nil {
			return nil, err
		}
	}

	return output, nil
}
