package hubspot

import (
	"fmt"
	"strconv"
	"strings"

	"google.golang.org/protobuf/types/known/structpb"

	hubspot "github.com/belong-inc/go-hubspot"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

// Get Company
type TaskGetCompanyInput struct {
	CompanyID string `json:"company-id"`
}

type TaskGetCompanyResp struct {
	OwnerID       string `json:"hubspot_owner_id,omitempty"`
	CompanyName   string `json:"name,omitempty"`
	CompanyDomain string `json:"domain,omitempty"`
	Description   string `json:"description,omitempty"`
	PhoneNumber   string `json:"phone,omitempty"`
	Industry      string `json:"industry,omitempty"`
	CompanyType   string `json:"type,omitempty"`
	City          string `json:"city,omitempty"`
	State         string `json:"state,omitempty"`
	Country       string `json:"country,omitempty"`
	PostalCode    string `json:"zip,omitempty"`
	TimeZone      string `json:"timezone,omitempty"`
	AnnualRevenue string `json:"annualrevenue,omitempty"`
	TotalRevenue  string `json:"totalrevenue,omitempty"`
	LinkedinPage  string `json:"linkedin_company_page,omitempty"`
}

type TaskGetCompanyOutput struct {
	OwnerID              string   `json:"owner-id,omitempty"`
	CompanyName          string   `json:"company-name,omitempty"`
	CompanyDomain        string   `json:"company-domain,omitempty"`
	Description          string   `json:"description,omitempty"`
	PhoneNumber          string   `json:"phone-number,omitempty"`
	Industry             string   `json:"industry,omitempty"`
	CompanyType          string   `json:"company-type,omitempty"`
	City                 string   `json:"city,omitempty"`
	State                string   `json:"state,omitempty"`
	Country              string   `json:"country,omitempty"`
	PostalCode           string   `json:"postal-code,omitempty"`
	TimeZone             string   `json:"time-zone,omitempty"`
	AnnualRevenue        float64  `json:"annual-revenue,omitempty"`
	TotalRevenue         float64  `json:"total-revenue,omitempty"`
	LinkedinPage         string   `json:"linkedin-page,omitempty"`
	AssociatedContactIDs []string `json:"associated-contact-ids"`
}

func (e *execution) GetCompany(input *structpb.Struct) (*structpb.Struct, error) {
	inputStruct := TaskGetCompanyInput{}

	err := base.ConvertFromStructpb(input, &inputStruct)

	if err != nil {
		return nil, fmt.Errorf("failed to convert input to struct: %v", err)
	}

	res, err := e.client.CRM.Company.Get(inputStruct.CompanyID, &TaskGetCompanyResp{}, &hubspot.RequestQueryOption{Associations: []string{"contacts"}})

	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return nil, fmt.Errorf("404: unable to read response from hubspot: no company was found")
		} else {
			return nil, err
		}
	}

	companyInfo := res.Properties.(*TaskGetCompanyResp)

	// get contacts associated with company

	var companyContactList []string
	if res.Associations != nil {
		// for company, it is possible to have duplicate contacts, so need to remove all the duplicates.

		hash := make(map[string]bool)

		companyContactAssociation := res.Associations.Contacts.Results

		for _, value := range companyContactAssociation {
			if _, ok := hash[value.ID]; !ok {
				hash[value.ID] = true
				companyContactList = append(companyContactList, value.ID)
			}
		}
	} else {
		companyContactList = []string{}
	}

	// convert to outputStruct

	var annualRevenue, totalRevenue float64

	if companyInfo.AnnualRevenue != "" {
		var err error
		annualRevenue, err = strconv.ParseFloat(companyInfo.AnnualRevenue, 64)

		if err != nil {
			return nil, err
		}
	}

	if companyInfo.TotalRevenue != "" {
		var err error
		totalRevenue, err = strconv.ParseFloat(companyInfo.TotalRevenue, 64)

		if err != nil {
			return nil, err
		}
	}

	outputStruct := TaskGetCompanyOutput{
		OwnerID:              companyInfo.OwnerID,
		CompanyName:          companyInfo.CompanyName,
		CompanyDomain:        companyInfo.CompanyDomain,
		Description:          companyInfo.Description,
		PhoneNumber:          companyInfo.PhoneNumber,
		Industry:             companyInfo.Industry,
		CompanyType:          companyInfo.CompanyType,
		City:                 companyInfo.City,
		State:                companyInfo.State,
		Country:              companyInfo.Country,
		PostalCode:           companyInfo.PostalCode,
		TimeZone:             companyInfo.TimeZone,
		AnnualRevenue:        annualRevenue,
		TotalRevenue:         totalRevenue,
		LinkedinPage:         companyInfo.LinkedinPage,
		AssociatedContactIDs: companyContactList,
	}

	output, err := base.ConvertToStructpb(outputStruct)

	if err != nil {
		return nil, fmt.Errorf("failed to convert output to struct: %v", err)
	}

	return output, nil
}

// Create Company
type TaskCreateCompanyInput struct {
	OwnerID                   string   `json:"owner-id"`
	CompanyName               string   `json:"company-name"`
	CompanyDomain             string   `json:"company-domain"`
	Description               string   `json:"description"`
	PhoneNumber               string   `json:"phone-number"`
	Industry                  string   `json:"industry"`
	CompanyType               string   `json:"company-type"`
	City                      string   `json:"city"`
	State                     string   `json:"state"`
	Country                   string   `json:"country"`
	PostalCode                string   `json:"postal-code"`
	TimeZone                  string   `json:"time-zone"`
	AnnualRevenue             float64  `json:"annual-revenue"`
	LinkedinPage              string   `json:"linkedin-page"`
	CreateContactsAssociation []string `json:"create-contacts-association"`
}

type TaskCreateCompanyReq struct {
	OwnerID       string `json:"hubspot_owner_id,omitempty"`
	CompanyName   string `json:"name,omitempty"`
	CompanyDomain string `json:"domain,omitempty"`
	Description   string `json:"description,omitempty"`
	PhoneNumber   string `json:"phone,omitempty"`
	Industry      string `json:"industry,omitempty"`
	CompanyType   string `json:"type,omitempty"`
	City          string `json:"city,omitempty"`
	State         string `json:"state,omitempty"`
	Country       string `json:"country,omitempty"`
	PostalCode    string `json:"zip,omitempty"`
	TimeZone      string `json:"timezone,omitempty"`
	AnnualRevenue string `json:"annualrevenue,omitempty"`
	LinkedinPage  string `json:"linkedin_company_page,omitempty"`
	CompanyID     string `json:"hs_object_id"`
}

type TaskCreateCompanyOutput struct {
	CompanyID string `json:"company-id"`
}

func (e *execution) CreateCompany(input *structpb.Struct) (*structpb.Struct, error) {

	inputStruct := TaskCreateCompanyInput{}
	err := base.ConvertFromStructpb(input, &inputStruct)

	if err != nil {
		return nil, fmt.Errorf("failed to convert input to struct: %v", err)
	}

	var annualRevenue string
	if inputStruct.AnnualRevenue != 0 {
		annualRevenue = strconv.FormatFloat(inputStruct.AnnualRevenue, 'f', -1, 64)
	}

	req := TaskCreateCompanyReq{
		OwnerID:       inputStruct.OwnerID,
		CompanyName:   inputStruct.CompanyName,
		CompanyDomain: inputStruct.CompanyDomain,
		Description:   inputStruct.Description,
		PhoneNumber:   inputStruct.PhoneNumber,
		Industry:      inputStruct.Industry,
		CompanyType:   inputStruct.CompanyType,
		City:          inputStruct.City,
		State:         inputStruct.State,
		Country:       inputStruct.Country,
		PostalCode:    inputStruct.PostalCode,
		TimeZone:      inputStruct.TimeZone,
		AnnualRevenue: annualRevenue,
		LinkedinPage:  inputStruct.LinkedinPage,
	}

	res, err := e.client.CRM.Company.Create(&req)

	if err != nil {
		return nil, err
	}

	// get company ID
	companyID := res.Properties.(*TaskCreateCompanyReq).CompanyID

	outputStruct := TaskCreateCompanyOutput{CompanyID: companyID}

	output, err := base.ConvertToStructpb(outputStruct)

	if err != nil {
		return nil, fmt.Errorf("failed to convert output to struct: %v", err)
	}

	// This section is for creating associations (company -> object)
	if len(inputStruct.CreateContactsAssociation) != 0 {
		err := CreateAssociation(&outputStruct.CompanyID, &inputStruct.CreateContactsAssociation, "company", "contact", e)

		if err != nil {
			return nil, err
		}
	}

	return output, nil
}
