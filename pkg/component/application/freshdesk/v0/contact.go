package freshdesk

import (
	"fmt"
	"strconv"
	"strings"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

// name, email, phone, mobile, description, job_title, tags, language, time_zone, company_id, unique_external_id, twitter_id, view_all_tickets, deletedc, other_companies, created_at, updated_at

const (
	ContactPath = "contacts"
)

// API functions for Contact

func (c *FreshdeskClient) GetContact(contactID int64) (*TaskGetContactResponse, error) {
	resp := &TaskGetContactResponse{}

	httpReq := c.httpclient.R().SetResult(resp)
	if _, err := httpReq.Get(fmt.Sprintf("/%s/%d", ContactPath, contactID)); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *FreshdeskClient) CreateContact(req *TaskCreateContactReq) (*TaskCreateContactResponse, error) {
	resp := &TaskCreateContactResponse{}

	httpReq := c.httpclient.R().SetBody(req).SetResult(resp)
	if _, err := httpReq.Post("/" + ContactPath); err != nil {
		return nil, err
	}
	return resp, nil

}

// Task 1: Get Contact

type TaskGetContactInput struct {
	ContactID int64 `json:"contact-id"`
}

type TaskGetContactResponse struct {
	Name              string                                   `json:"name"`
	Email             string                                   `json:"email"`
	Phone             string                                   `json:"phone"`
	Mobile            string                                   `json:"mobile"`
	Description       string                                   `json:"description"`
	Address           string                                   `json:"address"`
	JobTitle          string                                   `json:"job_title"`
	Tags              []string                                 `json:"tags"`
	Language          string                                   `json:"language"`
	TimeZone          string                                   `json:"time_zone"`
	CompanyID         int64                                    `json:"company_id"`
	UniqueExternalID  string                                   `json:"unique_external_id"`
	TwitterID         string                                   `json:"twitter_id"`
	ViewAllTickets    bool                                     `json:"view_all_tickets"`
	Deleted           bool                                     `json:"deleted"`
	Active            bool                                     `json:"active"`
	OtherEmails       []string                                 `json:"other_emails"`
	OtherCompanies    []taskGetContactResponseOtherCompany     `json:"other_companies"`
	OtherPhoneNumbers []taskGetContactResponseOtherPhoneNumber `json:"other_phone_numbers"`
	CreatedAt         string                                   `json:"created_at"`
	UpdatedAt         string                                   `json:"updated_at"`
	CustomFields      map[string]interface{}                   `json:"custom_fields"`
}

type taskGetContactResponseOtherCompany struct {
	CompanyID      int64 `json:"company_id"`
	ViewAllTickets bool  `json:"view_all_tickets"`
}

type taskGetContactResponseOtherPhoneNumber struct {
	PhoneNumber string `json:"value"`
}

type TaskGetContactOutput struct {
	Name              string                             `json:"name"`
	Email             string                             `json:"email,omitempty"`
	Phone             string                             `json:"phone,omitempty"`
	Mobile            string                             `json:"mobile,omitempty"`
	Description       string                             `json:"description,omitempty"`
	Address           string                             `json:"address,omitempty"`
	JobTitle          string                             `json:"job-title,omitempty"`
	Tags              []string                           `json:"tags"`
	Language          string                             `json:"language,omitempty"`
	TimeZone          string                             `json:"time-zone,omitempty"`
	CompanyID         int64                              `json:"company-id,omitempty"`
	UniqueExternalID  string                             `json:"unique-external-id,omitempty"`
	TwitterID         string                             `json:"twitter-id,omitempty"`
	ViewAllTickets    bool                               `json:"view-all-tickets"`
	Deleted           bool                               `json:"deleted"`
	Active            bool                               `json:"active"`
	OtherEmails       []string                           `json:"other-emails"`
	OtherCompaniesIDs []taskGetContactOutputOtherCompany `json:"other-companies-ids,omitempty"`
	OtherPhoneNumbers []string                           `json:"other-phone-numbers"`
	CreatedAt         string                             `json:"created-at"`
	UpdatedAt         string                             `json:"updated-at"`
	CustomFields      map[string]interface{}             `json:"custom-fields,omitempty"`
}

type taskGetContactOutputOtherCompany struct {
	CompanyID      int64 `json:"company-id"`
	ViewAllTickets bool  `json:"view-all-tickets"`
}

func (e *execution) TaskGetContact(in *structpb.Struct) (*structpb.Struct, error) {

	inputStruct := TaskGetContactInput{}
	err := base.ConvertFromStructpb(in, &inputStruct)

	if err != nil {
		return nil, fmt.Errorf("failed to convert input to struct: %v", err)
	}

	resp, err := e.client.GetContact(inputStruct.ContactID)

	if err != nil {
		return nil, err
	}

	outputStruct := TaskGetContactOutput{
		Name:             resp.Name,
		Email:            resp.Email,
		Phone:            resp.Phone,
		Mobile:           resp.Mobile,
		Description:      resp.Description,
		Address:          resp.Address,
		JobTitle:         resp.JobTitle,
		Tags:             resp.Tags,
		Language:         convertCodeToLanguage(resp.Language),
		TimeZone:         resp.TimeZone,
		CompanyID:        resp.CompanyID,
		UniqueExternalID: resp.UniqueExternalID,
		TwitterID:        resp.TwitterID,
		ViewAllTickets:   resp.ViewAllTickets,
		Deleted:          resp.Deleted,
		Active:           resp.Active,
		OtherEmails:      *checkForNilString(&resp.OtherEmails),
		CreatedAt:        convertTimestampResp(resp.CreatedAt),
		UpdatedAt:        convertTimestampResp(resp.UpdatedAt),
	}

	if len(resp.OtherCompanies) > 0 {
		outputStruct.OtherCompaniesIDs = make([]taskGetContactOutputOtherCompany, len(resp.OtherCompanies))
		for index, company := range resp.OtherCompanies {
			outputStruct.OtherCompaniesIDs[index].CompanyID = company.CompanyID
			outputStruct.OtherCompaniesIDs[index].ViewAllTickets = company.ViewAllTickets
		}
	}

	if len(resp.OtherPhoneNumbers) > 0 {
		outputStruct.OtherPhoneNumbers = make([]string, len(resp.OtherPhoneNumbers))
		for index, phone := range resp.OtherPhoneNumbers {
			outputStruct.OtherPhoneNumbers[index] = phone.PhoneNumber
		}
	} else {
		outputStruct.OtherPhoneNumbers = []string{}
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

// Task 2: Create Contact

type TaskCreateContactInput struct {
	Name              string   `json:"name"`
	Email             string   `json:"email"`
	Phone             string   `json:"phone"`
	Mobile            string   `json:"mobile"`
	Description       string   `json:"description"`
	Address           string   `json:"address"`
	JobTitle          string   `json:"job-title"`
	Tags              []string `json:"tags"`
	Language          string   `json:"language"`
	TimeZone          string   `json:"time-zone"`
	CompanyID         int64    `json:"company-id"`
	UniqueExternalID  string   `json:"unique-external-id"`
	TwitterID         string   `json:"twitter-id"`
	ViewAllTickets    bool     `json:"view-all-tickets"`
	OtherEmails       []string `json:"other-emails"`
	OtherCompanies    []string `json:"other-companies"`
	OtherPhoneNumbers []string `json:"other-phone-numbers"`
}

type TaskCreateContactReq struct {
	Name              string                                 `json:"name"`
	Email             string                                 `json:"email,omitempty"`
	Phone             string                                 `json:"phone,omitempty"`
	Mobile            string                                 `json:"mobile,omitempty"`
	Description       string                                 `json:"description,omitempty"`
	Address           string                                 `json:"address,omitempty"`
	JobTitle          string                                 `json:"job_title,omitempty"`
	Tags              []string                               `json:"tags,omitempty"`
	Language          string                                 `json:"language,omitempty"`
	TimeZone          string                                 `json:"time_zone,omitempty"`
	CompanyID         int64                                  `json:"company_id,omitempty"`
	UniqueExternalID  string                                 `json:"unique_external_id,omitempty"`
	TwitterID         string                                 `json:"twitter_id,omitempty"`
	ViewAllTickets    bool                                   `json:"view_all_tickets,omitempty"`
	OtherEmails       []string                               `json:"other_emails,omitempty"`
	OtherCompanies    []taskCreateContactReqOtherCompany     `json:"other_companies,omitempty"`
	OtherPhoneNumbers []taskCreateContactReqOtherPhoneNumber `json:"other_phone_numbers,omitempty"`
}

type taskCreateContactReqOtherCompany struct {
	CompanyID      int64 `json:"company_id"`
	ViewAllTickets bool  `json:"view_all_tickets"`
}

type taskCreateContactReqOtherPhoneNumber struct {
	PhoneNumber string `json:"value"`
}

type TaskCreateContactResponse struct {
	ID        int64  `json:"id"`
	CreatedAt string `json:"created_at"`
}

type TaskCreateContactOutput struct {
	ID        int64  `json:"contact-id"`
	CreatedAt string `json:"created-at"`
}

func (e *execution) TaskCreateContact(in *structpb.Struct) (*structpb.Struct, error) {
	inputStruct := TaskCreateContactInput{}
	err := base.ConvertFromStructpb(in, &inputStruct)

	if err != nil {
		return nil, fmt.Errorf("failed to convert input to struct: %v", err)
	}

	if inputStruct.Email == "" && inputStruct.Phone == "" && inputStruct.Mobile == "" {
		return nil, fmt.Errorf("please fill in at least one of the following fields: email, phone, mobile")
	}

	req := TaskCreateContactReq{
		Name:             inputStruct.Name,
		Email:            inputStruct.Email,
		Phone:            inputStruct.Phone,
		Mobile:           inputStruct.Mobile,
		Description:      inputStruct.Description,
		Address:          inputStruct.Address,
		JobTitle:         inputStruct.JobTitle,
		Tags:             inputStruct.Tags,
		Language:         convertLanguageToCode(inputStruct.Language),
		TimeZone:         inputStruct.TimeZone,
		CompanyID:        inputStruct.CompanyID,
		UniqueExternalID: inputStruct.UniqueExternalID,
		TwitterID:        inputStruct.TwitterID,
		ViewAllTickets:   inputStruct.ViewAllTickets,
		OtherEmails:      inputStruct.OtherEmails,
	}

	if len(inputStruct.OtherCompanies) > 0 {
		req.OtherCompanies = make([]taskCreateContactReqOtherCompany, len(inputStruct.OtherCompanies))
		for index, company := range inputStruct.OtherCompanies {
			// input format for other companies: company_id;view_all_tickets

			otherCompanyInfo := strings.Split(company, ";")
			if len(otherCompanyInfo) != 2 {
				return nil, fmt.Errorf("invalid format. The correct format is company_id;view_all_tickets(boolean \"true\"/\"false\"). Example: 123;true")
			}

			req.OtherCompanies[index].CompanyID, err = strconv.ParseInt(otherCompanyInfo[0], 10, 64)

			if err != nil {
				return nil, fmt.Errorf("error converting string to int64: %v", err)
			}

			if otherCompanyInfo[1] != "true" && otherCompanyInfo[1] != "false" {
				return nil, fmt.Errorf("invalid value for view_all_tickets. Please use either \"true\" or \"false\" Correct format: company_id;view_all_tickets(boolean \"true\"/\"false\"). Example: 123;true")
			}

			req.OtherCompanies[index].ViewAllTickets = otherCompanyInfo[1] == "true"

		}
	}

	if len(inputStruct.OtherPhoneNumbers) > 0 {
		req.OtherPhoneNumbers = make([]taskCreateContactReqOtherPhoneNumber, len(inputStruct.OtherPhoneNumbers))
		for index, phone := range inputStruct.OtherPhoneNumbers {
			req.OtherPhoneNumbers[index].PhoneNumber = phone
		}
	}

	resp, err := e.client.CreateContact(&req)

	if err != nil {
		return nil, err
	}

	outputStruct := TaskCreateContactOutput{
		ID:        resp.ID,
		CreatedAt: convertTimestampResp(resp.CreatedAt),
	}

	output, err := base.ConvertToStructpb(outputStruct)

	if err != nil {
		return nil, err
	}

	return output, nil
}
