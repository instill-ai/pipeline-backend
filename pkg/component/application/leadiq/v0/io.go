package leadiq

import (
	"regexp"
)

// We need to expose those structs because we will need them to calculate
// the Instill Credit cost.

// FindProspectsInput is the input for the FindProspects task.
type FindProspectsInput struct {
	// CompanyName is the name of the company to find prospects for.
	CompanyName string `instill:"company-name"`
	// Limit is the maximum number of prospects to return.
	Limit int `instill:"limit"`
	// FilterBy is the filter to apply to the prospects.
	FilterBy FilterBy `instill:"filter-by"`
	// OrderBy is the order to sort the prospects by.
	OrderBy OrderBy `instill:"order-by"`
}

// FilterBy is the filter to apply to the prospects.
type FilterBy struct {
	// JobTitle is the job title to filter by.
	JobTitle string `instill:"job-title,omitempty"`
	// Seniority is the seniority to filter by.
	Seniority string `instill:"seniority,omitempty"`

	jobTitleRegex  *regexp.Regexp
	seniorityRegex *regexp.Regexp
}

// OrderBy is the order to sort the prospects by.
type OrderBy struct {
	// JobTitle is asc or desc to sort by job title.
	JobTitle string `instill:"job-title,omitempty"`
	// Seniority is asc or desc to sort by seniority.
	Seniority string `instill:"seniority,omitempty"`
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
	// CompanyDescription is the company description of the prospect.
	CompanyDescription string `instill:"company-description"`
}
