package leadiq

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/machinebox/graphql"
	"go.uber.org/zap"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/x/errmsg"
)

// 1. Fetch FlatAdvancedSearch from LeadIQ
// 2. Filter prospects based on FilterBy.Seniority and FilterBy.JobTitle
// 3. Fetch email from searchPeople if workEmail is hidden
// 4. Filter prospects based on FilterBy.Function
func (e *execution) executeFindProspects(ctx context.Context, job *base.Job) error {
	logger := e.GetLogger()
	client := newClient(e.GetSetup(), logger)
	var inputStruct FindProspectsInput

	if err := job.Input.ReadData(ctx, &inputStruct); err != nil {
		err = fmt.Errorf("reading input data: %w", err)
		job.Error.Error(ctx, err)
		return err
	}

	flatAdvancedSearchIn := buildFlatAdvancedSearchInput(inputStruct)
	flatAdvancedSearchResp := flatAdvancedSearchResp{}
	if err := e.sendRequest(ctx, client, logger, flatAdvancedSearchQuery, flatAdvancedSearchIn, &flatAdvancedSearchResp); err != nil {
		msg := fmt.Sprintf("LeadIQ API result error: %v", err)
		err = errmsg.AddMessage(fmt.Errorf("sending flat advanced search request to LeadIQ with error"), msg)
		job.Error.Error(ctx, err)
		return err
	}

	filterBy := inputStruct.FilterBy
	if err := filterBy.compile(); err != nil {
		err = fmt.Errorf("compiling filter by: %w", err)
		job.Error.Error(ctx, err)
		return err
	}

	prospects := make([]Prospect, 0, len(flatAdvancedSearchResp.FlatAdvancedSearch.People))

	for _, person := range flatAdvancedSearchResp.FlatAdvancedSearch.People {
		if !filterBy.beforeAPICallingMatch(person) {
			continue
		}
		// It is possible that the work email is hidden, so we need to find the first non-hidden email
		// When workEmail is hidden, we should call searchPeople to get the email.
		var workEmail string
		for _, email := range person.WorkEmails {
			workEmail = email
			if email != "hidden" {
				break
			}
		}

		prospect := Prospect{
			Name:                        person.Name,
			JobTitle:                    person.Title,
			Seniority:                   person.Seniority,
			Email:                       workEmail,
			LinkedInURL:                 person.LinkedInURL,
			CompanyName:                 person.Company.Name,
			CompanyDescription:          person.Company.Description,
			CompanyIndustry:             person.Company.Industry,
			CompanyAddress:              fmt.Sprintf("%s, %s, %s", person.Company.City, person.Company.State, person.Company.Country),
			CompanyTechnologies:         person.Company.CompanyTechnologies,
			CompanyTechnologyCategories: person.Company.CompanyTechnologyCategories,
			RevenueSize: RevenueSize{
				Min:         person.Company.RevenueRange.Start,
				Max:         person.Company.RevenueRange.End,
				Description: person.Company.RevenueRange.Description,
			},
		}
		prospects = append(prospects, prospect)
	}

	for i := range prospects {
		searchPeopleIn := buildSearchPeopleInput(prospects[i])
		searchPeopleResp := searchPeopleResp{}
		if err := e.sendRequest(ctx, client, logger, searchPeopleQuery, searchPeopleIn, &searchPeopleResp); err != nil {
			msg := fmt.Sprintf("LeadIQ API result error: %v", err)
			err = errmsg.AddMessage(fmt.Errorf("sending search people request to LeadIQ with error"), msg)
			job.Error.Error(ctx, err)
			return err
		}

		if !foundEmail(searchPeopleResp) {
			prospects[i].Email = ""
		} else {
			prospects[i].Email = searchPeopleResp.SearchPeople.Results[0].CurrentPositions[0].Emails[0].Value
		}

		// Function is enum in LeadIQ. If they don't have the function, we set it to "Other", which is
		// one of the Enum value.
		if !foundFunction(searchPeopleResp) {
			prospects[i].function = "Other"
		} else {
			prospects[i].function = searchPeopleResp.SearchPeople.Results[0].CurrentPositions[0].Function
		}

	}

	filteredProspects := make([]Prospect, 0, len(prospects))
	for _, prospect := range prospects {
		if filterBy.afterAPICallingMatch(prospect) {
			filteredProspects = append(filteredProspects, prospect)
		}
	}

	output := FindProspectsOutput{
		Prospects: filteredProspects,
	}

	if err := job.Output.WriteData(ctx, &output); err != nil {
		err = fmt.Errorf("writing output data: %w", err)
		job.Error.Error(ctx, err)
		return err
	}

	return nil
}

func (e *execution) sendRequest(ctx context.Context, client *graphql.Client, logger *zap.Logger, query string, input map[string]interface{}, output any) error {
	req := graphql.NewRequest(query)

	// graphql.NewRequest does not support setting headers.
	// So, we have to set the headers here.
	req.Header.Set("Authorization", "Basic "+getAPIKey(e.GetSetup()))

	req.Var("input", input)
	sleepTime := 5 * time.Second
	retries := 0
	maxRetries := 5
	var err error

	// Backoff retry logic
	for {
		err = client.Run(ctx, req, &output)
		if err != nil {
			match := e.httpStatusCodeCompiler.FindStringSubmatch(err.Error())

			if len(match) == 0 {
				logger.Error("Failed to send request to LeadIQ with unknown status code",
					zap.Error(err),
				)
				err = fmt.Errorf("sending request to LeadIQ: %w", err)
				return err
			}

			statusCode := match[1]

			if statusCode != "429" {
				logger.Error("Failed to send request to LeadIQ with status code",
					zap.String("status code", statusCode),
					zap.Error(err),
				)
				err = fmt.Errorf("sending request to LeadIQ: %w, status code with %s", err, statusCode)
				return err
			}

			// If the error is too many requests, we use backoff retry strategy.
			logger.Error("Rate limited by LeadIQ",
				zap.Error(err),
				zap.Int("retries", retries),
			)
			retries++
			time.Sleep(sleepTime)
			sleepTime *= 2
			if retries >= maxRetries {
				err = fmt.Errorf("max retries reached, error: %v", err)
				return err
			}
			continue
		}
		return nil
	}
}

func buildFlatAdvancedSearchInput(in FindProspectsInput) map[string]interface{} {
	companyFilter := make(map[string]interface{})

	// Now, we don't support optional fields in the Console/API input.
	// So, we set the default value with the " " string.
	// If it is the default value, it means the users don't provide any information.
	// So, we won't add the field to the companyFilter.
	filterNonBlank := func(slice []string) []string {
		result := []string{}
		for _, s := range slice {
			trimmed := strings.TrimSpace(s)
			if trimmed != "" {
				result = append(result, trimmed)
			}
		}
		return result
	}

	names := filterNonBlank(in.Company.Names)
	if len(names) > 0 {
		companyFilter["names"] = names
	}

	// If there are names and revenue size at the same time, LeadIQ won't return any result. So, we will ignore the revenue size when the users provide the names.
	if len(names) == 0 && (in.Company.RevenueSize.Min != 0 || in.Company.RevenueSize.Max != 0) {
		revenueRanges := map[string]interface{}{}
		if in.Company.RevenueSize.Min != 0 {
			revenueRanges["start"] = in.Company.RevenueSize.Min
		}
		if in.Company.RevenueSize.Max != 0 {
			revenueRanges["end"] = in.Company.RevenueSize.Max
		}
		companyFilter["revenueRanges"] = revenueRanges
	}

	if countries := filterNonBlank(in.Company.Countries); len(countries) > 0 {
		companyFilter["countries"] = countries
	}

	if states := filterNonBlank(in.Company.States); len(states) > 0 {
		companyFilter["states"] = states
	}

	if cities := filterNonBlank(in.Company.Cities); len(cities) > 0 {
		companyFilter["cities"] = cities
	}

	if industries := filterNonBlank(in.Company.Industries); len(industries) > 0 {
		companyFilter["industries"] = industries
	}

	if descriptions := filterNonBlank(in.Company.Descriptions); len(descriptions) > 0 {
		companyFilter["descriptions"] = descriptions
	}

	if technologies := filterNonBlank(in.Company.Technologies); len(technologies) > 0 {
		companyFilter["technologies"] = technologies
	}

	builtInput := map[string]interface{}{
		"companyFilter": companyFilter,
		"limit":         in.Limit,
	}

	return builtInput
}

func buildSearchPeopleInput(prospect Prospect) map[string]interface{} {
	return map[string]interface{}{
		"fullName": prospect.Name,
		"company": map[string]string{
			"name": prospect.CompanyName,
		},
	}
}

func (f *FilterBy) compile() error {
	var err error
	if f.JobTitle != "" {
		f.jobTitleRegex, err = regexp.Compile(f.JobTitle)
		if err != nil {
			return fmt.Errorf("compiling job title filter: %w", err)
		}
	}
	if f.Function != "" {
		f.functionRegex, err = regexp.Compile(f.Function)
		if err != nil {
			return fmt.Errorf("compiling seniority filter: %w", err)
		}
	}
	return nil
}

func (f *FilterBy) beforeAPICallingMatch(person person) bool {
	if f.jobTitleRegex != nil && !f.jobTitleRegex.MatchString(person.Title) {
		return false
	}
	if len(f.Seniorities) > 0 {
		found := false
		for _, seniority := range f.Seniorities {
			if seniority == person.Seniority {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func (f *FilterBy) afterAPICallingMatch(prospect Prospect) bool {
	if f.functionRegex != nil && !f.functionRegex.MatchString(prospect.function) {
		return false
	}
	return true
}

func foundEmail(searchPeopleResp searchPeopleResp) bool {
	if len(searchPeopleResp.SearchPeople.Results) == 0 {
		return false
	}

	result := searchPeopleResp.SearchPeople.Results[0]
	if len(result.CurrentPositions) == 0 {
		return false
	}

	currentPosition := result.CurrentPositions[0]
	if len(currentPosition.Emails) == 0 {
		return false
	}

	email := currentPosition.Emails[0]
	return email.Status == "Verified"
}

func foundFunction(searchPeopleResp searchPeopleResp) bool {
	if len(searchPeopleResp.SearchPeople.Results) == 0 {
		return false
	}

	result := searchPeopleResp.SearchPeople.Results[0]
	if len(result.CurrentPositions) == 0 {
		return false
	}

	function := result.CurrentPositions[0].Function

	return function != ""
}

// flatAdvancedSearchResp is the response from the flatAdvancedSearch query.
// We only extract the fields we need.
type flatAdvancedSearchResp struct {
	FlatAdvancedSearch flatAdvancedSearch `json:"flatAdvancedSearch"`
}

type flatAdvancedSearch struct {
	People []person `json:"people"`
}

type person struct {
	Company     company  `json:"company"`
	Name        string   `json:"name"`
	LinkedInURL string   `json:"linkedinUrl"`
	Seniority   string   `json:"seniority"`
	Title       string   `json:"title"`
	WorkEmails  []string `json:"workEmails"`
}

type company struct {
	Name                        string       `json:"name"`
	Description                 string       `json:"description"`
	Industry                    string       `json:"industry"`
	Country                     string       `json:"country"`
	State                       string       `json:"state"`
	City                        string       `json:"city"`
	CompanyTechnologies         []string     `json:"companyTechnologies"`
	CompanyTechnologyCategories []string     `json:"companyTechnologyCategories"`
	RevenueRange                revenueRange `json:"revenueRange"`
}

type revenueRange struct {
	Start       int    `json:"start"`
	End         int    `json:"end"`
	Description string `json:"description"`
}

// searchPeopleResp is the response from the searchPeople query.
// We only extract the fields we need.
type searchPeopleResp struct {
	SearchPeople searchPeople `json:"searchPeople"`
}

type searchPeople struct {
	Results []result `json:"results"`
}

type result struct {
	CurrentPositions []currentPosition `json:"currentPositions"`
	LinkedIn         struct {
		LinkedInURL string `json:"linkedinUrl"`
	} `json:"linkedin"`
}

type currentPosition struct {
	CompanyInfo struct {
		Name string `json:"name"`
	} `json:"companyInfo"`
	Emails   []email `json:"emails"`
	Function string  `json:"function"`
}

type email struct {
	Status string `json:"status"`
	Type   string `json:"type"`
	Value  string `json:"value"`
}
