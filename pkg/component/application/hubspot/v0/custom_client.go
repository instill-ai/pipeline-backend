package hubspot

import (
	hubspot "github.com/belong-inc/go-hubspot"
)

// need to create CustomClient because the go-hubspot sdk we are using does not support threads (conversation inbox)
// future functionalities that go-huspot sdk doesn't support will go here or need to be modified will go here.
type CustomClient struct {
	*hubspot.Client
	Thread              ThreadService
	RetrieveAssociation RetrieveAssociationService
	Ticket              TicketService
	Owner               OwnerService
	GetAll              GetAllService
}

func NewCustomClient(setAuthMethod hubspot.AuthMethod, opts ...hubspot.Option) (*CustomClient, error) {

	// call default NewClient
	c, err := hubspot.NewClient(setAuthMethod, opts...)

	if err != nil {
		return nil, err
	}

	customC := &CustomClient{
		Client: c,
		Thread: &ThreadServiceOp{
			threadPath: "conversations/v3/conversations/threads",
			client:     c,
		},
		RetrieveAssociation: &RetrieveAssociationServiceOp{
			retrieveCrmIDPath:    "crm/v3/associations/Contacts",
			retrieveThreadIDPath: "conversations/v3/conversations/threads?associatedContactId=",
			client:               c,
		},
		Ticket: &TicketServiceOp{
			ticketPath: "crm/v3/objects/tickets",
			client:     c,
		},
		Owner: &OwnerServiceOp{
			ownerPath: "crm/v3/owners",
			client:    c,
		},
		GetAll: &GetAllServiceOp{
			client: c,
		},
	}

	return customC, nil
}
