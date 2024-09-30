package freshdesk

import (
	"fmt"
	"strings"

	"github.com/go-resty/resty/v2"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

const (
	TicketPath = "tickets"
)

// API functions for Ticket

func (c *FreshdeskClient) GetTicket(ticketID int64) (*TaskGetTicketResponse, error) {

	resp := &TaskGetTicketResponse{}

	httpReq := c.httpclient.R().SetResult(resp)
	if _, err := httpReq.Get(fmt.Sprintf("/%s/%d", TicketPath, ticketID)); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *FreshdeskClient) CreateTicket(req *TaskCreateTicketReq) (*TaskCreateTicketResponse, error) {
	resp := &TaskCreateTicketResponse{}
	httpReq := c.httpclient.R().SetBody(req).SetResult(resp)
	if _, err := httpReq.Post("/" + TicketPath); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *FreshdeskClient) ReplyToTicket(ticketID int64, req *TaskReplyToTicketReq) (*TaskReplyToTicketResponse, error) {
	resp := &TaskReplyToTicketResponse{}
	httpReq := c.httpclient.R().SetBody(req).SetResult(resp)
	if _, err := httpReq.Post(fmt.Sprintf("/%s/%d/reply", TicketPath, ticketID)); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *FreshdeskClient) CreateTicketNote(ticketID int64, req *TaskCreateTicketNoteReq) (*TaskCreateTicketNoteResponse, error) {
	resp := &TaskCreateTicketNoteResponse{}
	httpReq := c.httpclient.R().SetBody(req).SetResult(resp)
	if _, err := httpReq.Post(fmt.Sprintf("/%s/%d/notes", TicketPath, ticketID)); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *FreshdeskClient) GetAllConversations(ticketID int64, pagination bool, paginationPath string) ([]TaskGetAllConversationsResponse, string, error) {
	resp := []TaskGetAllConversationsResponse{}

	httpReq := c.httpclient.R().SetResult(&resp)

	var rawResp *resty.Response
	var err error
	if !pagination {
		rawResp, err = httpReq.Get(fmt.Sprintf("/%s/%d/conversations", TicketPath, ticketID))

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

//Task 1: Get Ticket

type TaskGetTicketInput struct {
	TicketID int64 `json:"ticket-id"`
}

type TaskGetTicketResponse struct {
	Subject                string                          `json:"subject"`
	DescriptionText        string                          `json:"description_text"`
	Source                 int                             `json:"source"`
	Status                 int                             `json:"status"`
	Priority               int                             `json:"priority"`
	TicketType             string                          `json:"type"`
	AssociationType        int                             `json:"association_type"`
	AssociatedTicketList   []int                           `json:"associated_tickets_list"`
	Tags                   []string                        `json:"tags"`
	CCEmails               []string                        `json:"cc_emails"`
	ForwardEmails          []string                        `json:"fwd_emails"`
	ReplyCCEmails          []string                        `json:"reply_cc_emails"`
	RequesterID            int64                           `json:"requester_id"`
	ResponderID            int64                           `json:"responder_id"`
	CompanyID              int64                           `json:"company_id"`
	GroupID                int64                           `json:"group_id"`
	ProductID              int64                           `json:"product_id"`
	SupportEmail           string                          `json:"support_email"`
	ToEmails               []string                        `json:"to_emails"`
	Spam                   bool                            `json:"spam"`
	IsEscalated            bool                            `json:"is_escalated"`
	DueBy                  string                          `json:"due_by"`
	FirstResponseDueBy     string                          `json:"fr_due_by"`
	FirstResponseEscalated bool                            `json:"fr_escalated"`
	NextResponseDueBy      string                          `json:"nr_due_by"`
	NextResponseEscalated  bool                            `json:"nr_escalated"`
	CreatedAt              string                          `json:"created_at"`
	UpdatedAt              string                          `json:"updated_at"`
	Attachments            []taskGetTicketOutputAttachment `json:"attachments"`
	SentimentScore         int                             `json:"sentiment_score"`
	InitialSentimentScore  int                             `json:"initial_sentiment_score"`
	CustomFields           map[string]interface{}          `json:"custom_fields"`
}

type TaskGetTicketOutput struct {
	Subject                string                          `json:"subject"`
	DescriptionText        string                          `json:"description-text"`
	Source                 string                          `json:"source"`
	Status                 string                          `json:"status"`
	Priority               string                          `json:"priority"`
	TicketType             string                          `json:"ticket-type,omitempty"`
	AssociationType        string                          `json:"association-type"`
	AssociatedTicketList   []int                           `json:"associated-ticket-list,omitempty"`
	Tags                   []string                        `json:"tags"`
	CCEmails               []string                        `json:"cc-emails"`
	ForwardEmails          []string                        `json:"forward-emails"`
	ReplyCCEmails          []string                        `json:"reply-cc-emails"`
	RequesterID            int64                           `json:"requester-id"`
	ResponderID            int64                           `json:"responder-id,omitempty"`
	CompanyID              int64                           `json:"company-id,omitempty"`
	GroupID                int64                           `json:"group-id,omitempty"`
	ProductID              int64                           `json:"product-id,omitempty"`
	SupportEmail           string                          `json:"support-email,omitempty"`
	ToEmails               []string                        `json:"to-emails"`
	Spam                   bool                            `json:"spam"`
	DueBy                  string                          `json:"due-by,omitempty"`
	IsEscalated            bool                            `json:"is-escalated"`
	FirstResponseDueBy     string                          `json:"first-response-due-by,omitempty"`
	FirstResponseEscalated bool                            `json:"first-response-escalated"`
	NextResponseDueBy      string                          `json:"next-response-due-by,omitempty"`
	NextResponseEscalated  bool                            `json:"next-response-escalated"`
	CreatedAt              string                          `json:"created-at"`
	UpdatedAt              string                          `json:"updated-at"`
	Attachments            []taskGetTicketOutputAttachment `json:"attachments,omitempty"`
	SentimentScore         int                             `json:"sentiment-score"`
	InitialSentimentScore  int                             `json:"initial-sentiment-score"`
	CustomFields           map[string]interface{}          `json:"custom-fields,omitempty"`
}

type taskGetTicketOutputAttachment struct {
	Name        string `json:"name"`
	ContentType string `json:"content-type"`
	URL         string `json:"url"`
}

func (e *execution) TaskGetTicket(in *structpb.Struct) (*structpb.Struct, error) {
	inputStruct := TaskGetTicketInput{}
	err := base.ConvertFromStructpb(in, &inputStruct)

	if err != nil {
		return nil, fmt.Errorf("failed to convert input to struct: %v", err)
	}

	resp, err := e.client.GetTicket(inputStruct.TicketID)
	if err != nil {
		return nil, err
	}

	outputStruct := TaskGetTicketOutput{
		Subject:                resp.Subject,
		DescriptionText:        resp.DescriptionText,
		Source:                 convertSourceToString(resp.Source),
		Status:                 convertStatusToString(resp.Status),
		Priority:               convertPriorityToString(resp.Priority),
		TicketType:             resp.TicketType,
		AssociationType:        convertAssociationType(resp.AssociationType),
		AssociatedTicketList:   resp.AssociatedTicketList,
		Tags:                   *checkForNilString(&resp.Tags),
		CCEmails:               *checkForNilString(&resp.CCEmails),
		ForwardEmails:          *checkForNilString(&resp.ForwardEmails),
		ReplyCCEmails:          *checkForNilString(&resp.ReplyCCEmails),
		RequesterID:            resp.RequesterID,
		ResponderID:            resp.ResponderID,
		CompanyID:              resp.CompanyID,
		GroupID:                resp.GroupID,
		ProductID:              resp.ProductID,
		SupportEmail:           resp.SupportEmail,
		ToEmails:               *checkForNilString(&resp.ToEmails),
		Spam:                   resp.Spam,
		DueBy:                  convertTimestampResp(resp.DueBy),
		IsEscalated:            resp.IsEscalated,
		FirstResponseDueBy:     convertTimestampResp(resp.FirstResponseDueBy),
		FirstResponseEscalated: resp.FirstResponseEscalated,
		NextResponseDueBy:      convertTimestampResp(resp.NextResponseDueBy),
		NextResponseEscalated:  resp.NextResponseEscalated,
		CreatedAt:              convertTimestampResp(resp.CreatedAt),
		UpdatedAt:              convertTimestampResp(resp.UpdatedAt),
		Attachments:            resp.Attachments,
		SentimentScore:         resp.SentimentScore,
		InitialSentimentScore:  resp.InitialSentimentScore,
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

// Create Ticket
type TaskCreateTicketInput struct {
	// Only one is needed: requester-id or email
	RequesterID      int64    `json:"requester-id"`
	Email            string   `json:"email"`
	Subject          string   `json:"subject"`
	Description      string   `json:"description"`
	Source           string   `json:"source"`
	Status           string   `json:"status"`
	Priority         string   `json:"priority"`
	Type             string   `json:"ticket-type"`
	CompanyID        int64    `json:"company-id"`
	ProductID        int64    `json:"product-id"`
	GroupID          int64    `json:"group-id"`
	ResponderID      int64    `json:"responder-id"`
	Tags             []string `json:"tags"`
	CCEmails         []string `json:"cc-emails"`
	ParentID         int64    `json:"parent-id"`
	RelatedTicketIDs []int64  `json:"related-ticket-ids"`
}

type TaskCreateTicketReq struct {
	RequesterID      int64    `json:"requester_id,omitempty"`
	Email            string   `json:"email,omitempty"`
	Subject          string   `json:"subject"`
	Description      string   `json:"description"`
	Source           int      `json:"source"`
	Status           int      `json:"status"`
	Priority         int      `json:"priority"`
	Type             string   `json:"type,omitempty"`
	CompanyID        int64    `json:"company_id,omitempty"`
	ProductID        int64    `json:"product_id,omitempty"`
	GroupID          int64    `json:"group_id,omitempty"`
	ResponderID      int64    `json:"responder_id,omitempty"`
	Tags             []string `json:"tags,omitempty"`
	CCEmails         []string `json:"cc_emails,omitempty"`
	ParentID         int64    `json:"parent_id,omitempty"`
	RelatedTicketIDs []int64  `json:"related_ticket_ids,omitempty"`
}

type TaskCreateTicketResponse struct {
	ID        int64  `json:"id"`
	CreatedAt string `json:"created_at"`
}

type TaskCreateTicketOutput struct {
	ID        int64  `json:"ticket-id"`
	CreatedAt string `json:"created-at"`
}

func (e *execution) TaskCreateTicket(in *structpb.Struct) (*structpb.Struct, error) {
	inputStruct := TaskCreateTicketInput{}
	err := base.ConvertFromStructpb(in, &inputStruct)

	if err != nil {
		return nil, fmt.Errorf("failed to convert input to struct: %v", err)
	}

	if inputStruct.ParentID != 0 && len(inputStruct.RelatedTicketIDs) > 0 {
		return nil, fmt.Errorf("only one of parent-id or related-ticket-ids can be provided")
	}

	req := TaskCreateTicketReq{
		Subject:     inputStruct.Subject,
		Description: inputStruct.Description,
		Source:      convertSourceToInt(inputStruct.Source),
		Status:      convertStatusToInt(inputStruct.Status),
		Priority:    convertPriorityToInt(inputStruct.Priority),
		Type:        inputStruct.Type,
		CompanyID:   inputStruct.CompanyID,
		ProductID:   inputStruct.ProductID,
		GroupID:     inputStruct.GroupID,
		ResponderID: inputStruct.ResponderID,
		Tags:        inputStruct.Tags,
		CCEmails:    inputStruct.CCEmails,
	}

	if inputStruct.RequesterID != 0 {
		req.RequesterID = inputStruct.RequesterID
	} else if inputStruct.Email != "" {
		req.Email = inputStruct.Email
	} else {
		return nil, fmt.Errorf("either Requester ID or email is required")
	}

	if inputStruct.ParentID != 0 {
		req.ParentID = inputStruct.ParentID
	}

	if len(inputStruct.RelatedTicketIDs) > 0 {
		req.RelatedTicketIDs = inputStruct.RelatedTicketIDs
	}

	resp, err := e.client.CreateTicket(&req)
	if err != nil {
		return nil, err
	}

	outputStruct := TaskCreateTicketOutput{
		ID:        resp.ID,
		CreatedAt: convertTimestampResp(resp.CreatedAt),
	}

	output, err := base.ConvertToStructpb(outputStruct)
	if err != nil {
		return nil, fmt.Errorf("failed to convert output to struct: %v", err)
	}

	return output, nil
}

// Task 3: Reply To A Ticket

type TaskReplyToTicketInput struct {
	TicketID  int64    `json:"ticket-id"`
	Body      string   `json:"body"`
	FromEmail string   `json:"from-email"`
	UserID    int64    `json:"user-id"` //user ID can either be the requester or the agent
	CCEmails  []string `json:"cc-emails"`
	BCCEmails []string `json:"bcc-emails"`
}

type TaskReplyToTicketReq struct {
	Body      string   `json:"body"`
	FromEmail string   `json:"from_email,omitempty"`
	UserID    int64    `json:"user_id,omitempty"`
	CCEmails  []string `json:"cc_emails,omitempty"`
	BCCEmails []string `json:"bcc_emails,omitempty"`
}

type TaskReplyToTicketResponse struct {
	ConversationID int64  `json:"id"`
	CreatedAt      string `json:"created_at"`
}

type TaskReplyToTicketOutput struct {
	ConversationID int64  `json:"conversation-id"`
	CreatedAt      string `json:"created-at"`
}

func (e *execution) TaskReplyToTicket(in *structpb.Struct) (*structpb.Struct, error) {
	inputStruct := TaskReplyToTicketInput{}
	err := base.ConvertFromStructpb(in, &inputStruct)

	if err != nil {
		return nil, fmt.Errorf("failed to convert input to struct: %v", err)
	}

	req := TaskReplyToTicketReq{
		Body:      inputStruct.Body,
		FromEmail: inputStruct.FromEmail,
		UserID:    inputStruct.UserID,
		CCEmails:  inputStruct.CCEmails,
		BCCEmails: inputStruct.BCCEmails,
	}

	resp, err := e.client.ReplyToTicket(inputStruct.TicketID, &req)

	if err != nil {
		return nil, err
	}

	outputStruct := TaskReplyToTicketOutput{
		ConversationID: resp.ConversationID,
		CreatedAt:      convertTimestampResp(resp.CreatedAt),
	}

	output, err := base.ConvertToStructpb(outputStruct)

	if err != nil {
		return nil, fmt.Errorf("failed to convert output to struct: %v", err)
	}

	return output, nil
}

// Task 4: Create Ticket Note

type TaskCreateTicketNoteInput struct {
	TicketID     int64    `json:"ticket-id"`
	Body         string   `json:"body"`
	NotifyEmails []string `json:"notify-emails"`
	UserID       int64    `json:"user-id"`
	Private      bool     `json:"private"`
	Incoming     bool     `json:"incoming"`
}

type TaskCreateTicketNoteReq struct {
	Body         string   `json:"body"`
	NotifyEmails []string `json:"notify_emails,omitempty"`
	UserID       int64    `json:"user_id,omitempty"`
	Private      bool     `json:"private"`
	Incoming     bool     `json:"incoming"`
}

type TaskCreateTicketNoteResponse struct {
	ConversationID int64  `json:"id"`
	CreatedAt      string `json:"created_at"`
}

type TaskCreateTicketNoteOutput struct {
	ConversationID int64  `json:"conversation-id"`
	CreatedAt      string `json:"created-at"`
}

func (e *execution) TaskCreateTicketNote(in *structpb.Struct) (*structpb.Struct, error) {
	inputStruct := TaskCreateTicketNoteInput{}
	err := base.ConvertFromStructpb(in, &inputStruct)

	if err != nil {
		return nil, fmt.Errorf("failed to convert input to struct: %v", err)
	}

	req := TaskCreateTicketNoteReq{
		Body:         inputStruct.Body,
		NotifyEmails: inputStruct.NotifyEmails,
		UserID:       inputStruct.UserID,
		Private:      inputStruct.Private,
		Incoming:     inputStruct.Incoming,
	}

	resp, err := e.client.CreateTicketNote(inputStruct.TicketID, &req)

	if err != nil {
		return nil, err
	}

	outputStruct := TaskCreateTicketNoteOutput{
		ConversationID: resp.ConversationID,
		CreatedAt:      convertTimestampResp(resp.CreatedAt),
	}

	output, err := base.ConvertToStructpb(outputStruct)

	if err != nil {
		return nil, fmt.Errorf("failed to convert output to struct: %v", err)
	}

	return output, nil
}

// Task 5: Get Conversations

type TaskGetAllConversationsInput struct {
	TicketID int64 `json:"ticket-id"`
}

type TaskGetAllConversationsResponse struct {
	BodyText       string   `json:"body_text"`
	ConversationID int64    `json:"id"`
	SupportEmail   string   `json:"support_email"`
	ToEmails       []string `json:"to_emails"`
	FromEmail      string   `json:"from_email"`
	CCEmails       []string `json:"cc_emails"`
	BCCEmails      []string `json:"bcc_emails"`
	Incoming       bool     `json:"incoming"`
	Private        bool     `json:"private"`
	UserID         int64    `json:"user_id"`
	CreatedAt      string   `json:"created_at"`
}

type TaskGetAllConversationsOutput struct {
	Conversations       []taskGetAllConversationsOutputConversation `json:"conversations"`
	ConversationsLength int                                         `json:"conversations-length"`
}

type taskGetAllConversationsOutputConversation struct {
	BodyText       string   `json:"body-text"`
	ConversationID int64    `json:"conversation-id"`
	SupportEmail   string   `json:"support-email,omitempty"`
	ToEmails       []string `json:"to-emails"`
	FromEmail      string   `json:"from-email,omitempty"`
	CCEmails       []string `json:"cc-emails"`
	BCCEmails      []string `json:"bcc-emails"`
	Incoming       bool     `json:"incoming"`
	Private        bool     `json:"private"`
	UserID         int64    `json:"user-id,omitempty"`
	CreatedAt      string   `json:"created-at"`
}

func (e *execution) TaskGetAllConversations(in *structpb.Struct) (*structpb.Struct, error) {
	inputStruct := TaskGetAllConversationsInput{}
	err := base.ConvertFromStructpb(in, &inputStruct)

	if err != nil {
		return nil, fmt.Errorf("failed to convert input to struct: %v", err)
	}

	outputStruct := TaskGetAllConversationsOutput{}

	var resp []TaskGetAllConversationsResponse
	var paginationPath string

	for {

		if paginationPath == "" {
			resp, paginationPath, err = e.client.GetAllConversations(inputStruct.TicketID, false, "")
		} else {
			resp, paginationPath, err = e.client.GetAllConversations(inputStruct.TicketID, true, paginationPath)
		}

		if err != nil {
			return nil, err
		}

		for _, conversation := range resp {
			outputConvo := taskGetAllConversationsOutputConversation{
				BodyText:       conversation.BodyText,
				ConversationID: conversation.ConversationID,
				SupportEmail:   conversation.SupportEmail,
				ToEmails:       *checkForNilString(&conversation.ToEmails),
				FromEmail:      conversation.FromEmail,
				CCEmails:       *checkForNilString(&conversation.CCEmails),
				BCCEmails:      *checkForNilString(&conversation.BCCEmails),
				Incoming:       conversation.Incoming,
				Private:        conversation.Private,
				UserID:         conversation.UserID,
				CreatedAt:      convertTimestampResp(conversation.CreatedAt),
			}

			outputStruct.Conversations = append(outputStruct.Conversations, outputConvo)
		}

		if paginationPath == "" {
			break
		}
	}

	outputStruct.ConversationsLength = len(outputStruct.Conversations)
	if outputStruct.ConversationsLength == 0 {
		outputStruct.Conversations = []taskGetAllConversationsOutputConversation{}
	}

	output, err := base.ConvertToStructpb(outputStruct)

	if err != nil {
		return nil, err
	}

	return output, nil
}

func convertSourceToString(source int) string {
	switch source {
	case 1:
		return "Email"
	case 2:
		return "Portal"
	case 3:
		return "Phone"
	case 4:
		return "Forum"
	case 5:
		return "Twitter"
	case 6:
		return "Facebook"
	case 7:
		return "Chat"
	case 8:
		return "MobiHelp"
	case 9:
		return "Feedback Widget"
	case 10:
		return "Outbound Email"
	case 11:
		return "Ecommerce"
	case 12:
		return "Bot"
	case 13:
		return "Whatsapp"
	default:
		return fmt.Sprintf("Unknown source, received: %d", source)
	}
}

func convertSourceToInt(source string) int {
	// For creating ticket, the only source that can be used is 1,2,3,5,6,7,9,11,10
	switch source {
	case "Email":
		return 1
	case "Portal":
		return 2
	case "Phone":
		return 3
	case "Twitter":
		return 5
	case "Facebook":
		return 6
	case "Chat":
		return 7
	case "Feedback Widget":
		return 9
	case "Outbound Email":
		return 10
	case "Ecommerce":
		return 11
	}
	return 0
}

func convertStatusToString(status int) string {
	switch status {
	case 2:
		return "Open"
	case 3:
		return "Pending"
	case 4:
		return "Resolved"
	case 5:
		return "Closed"
	case 6:
		return "Waiting on Customer"
	case 7:
		return "Waiting on Third Party"
	default:
		return fmt.Sprintf("Unknown status, received: %d", status)
	}
}

func convertStatusToInt(status string) int {
	switch status {
	case "Open":
		return 2
	case "Pending":
		return 3
	case "Resolved":
		return 4
	case "Closed":
		return 5
	case "Waiting on Customer":
		return 6
	case "Waiting on Third Party":
		return 7
	}
	return 0
}

func convertPriorityToString(priority int) string {
	switch priority {
	case 1:
		return "Low"
	case 2:
		return "Medium"
	case 3:
		return "High"
	case 4:
		return "Urgent"
	default:
		return fmt.Sprintf("Unknown priority, received: %d", priority)
	}
}

func convertPriorityToInt(priority string) int {
	switch priority {
	case "Low":
		return 1
	case "Medium":
		return 2
	case "High":
		return 3
	case "Urgent":
		return 4
	}
	return 0
}

func convertAssociationType(associationType int) string {
	switch associationType {
	case 1:
		return "Parent"
	case 2:
		return "Child"
	case 3:
		return "Tracker"
	case 4:
		return "Related"
	default:
		return "No association"
	}
}
