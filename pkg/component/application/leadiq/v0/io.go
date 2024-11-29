package leadiq

import (
	"regexp"
)

// We need to expose those structs because we will need them to calculate
// the Instill Credit cost.

// FindProspectsInput is the input for the FindProspects task.
type FindProspectsInput struct {
	// Company is the company information to find prospects for.
	Company Company `instill:"company"`
	// Limit is the maximum number of prospects to return.
	Limit int `instill:"limit,default=10"`
	// FilterBy is the filter to apply to the prospects.
	FilterBy FilterBy `instill:"filter-by"`
}

// Company is the company information to find prospects for.
type Company struct {
	// Name is the name of the company.
	Names []string `instill:"names,omitempty"`
	// RevenueSize is the revenue size of the company.
	RevenueSize RevenueSize `instill:"revenue-size,omitempty"`
	// Countries is the list of countries the company operates in.
	Countries []string `instill:"countries,omitempty"`
	// States is the list of states the company operates in.
	States []string `instill:"states,omitempty"`
	// Cities is the list of cities the company operates in.
	Cities []string `instill:"cities,omitempty"`
	// Industries is the list of industries the company are.
	Industries []string `instill:"industries,omitempty"`
	// Descriptions is the list of descriptions of the company.
	Descriptions []string `instill:"descriptions,omitempty"`
	// Technologies is the list of technologies the company uses.
	Technologies []string `instill:"technologies,omitempty"`
}

// RevenueSize is the revenue size of the company.
type RevenueSize struct {
	// Min is the minimum revenue size.
	Min int `instill:"min"`
	// Max is the maximum revenue size.
	Max int `instill:"max"`
	// Description is the description of the revenue size.
	Description string `instill:"description,omitempty"`
}

// FilterBy is the filter to apply to the prospects.
type FilterBy struct {
	// JobTitle is the job title to filter by.
	JobTitle string `instill:"job-title,omitempty"`
	// Seniorities is the seniorities to filter by.
	Seniorities []string `instill:"seniorities,omitempty"`
	// Function is the function to filter by.
	Function string `instill:"function,omitempty"`

	jobTitleRegex *regexp.Regexp
	functionRegex *regexp.Regexp
}

// FindProspectsOutput is the output for the FindProspects task.
type FindProspectsOutput struct {
	// Prospects is the list of prospects found.
	Prospects []Prospect `instill:"prospects"`
}

// Prospect is a prospect found.
type Prospect struct {
	// Name is the name of the prospect.
	Name string `instill:"name"`
	// JobTitle is the job title of the prospect.
	JobTitle string `instill:"job-title"`
	// Seniority is the seniority of the prospect.
	Seniority string `instill:"seniority"`
	// Email is the email of the prospect.
	Email string `instill:"email"`
	// LinkedInURL is the LinkedIn URL of the prospect.
	LinkedInURL string `instill:"linkedin-url"`
	// CompanyName is the company name of the prospect.
	CompanyName string `instill:"company-name"`
	// CompanyDescription is the company description of the prospect.
	CompanyDescription string `instill:"company-description"`
	// CompanyIndustry is the company industry of the prospect.
	CompanyIndustry string `instill:"company-industry"`
	// CompanyAddress is the company address of the prospect.
	CompanyAddress string `instill:"company-address"`
	// CompanyTechnologies is the company technologies of the prospect.
	CompanyTechnologies []string `instill:"company-technologies"`
	// CompanyTechnologyCategories is the company technology categories of the prospect.
	CompanyTechnologyCategories []string `instill:"company-technology-categories"`
	// RevenueSize is the revenue size of the company of the prospect.
	RevenueSize RevenueSize `instill:"revenue-size"`

	function string
}
