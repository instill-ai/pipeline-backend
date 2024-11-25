package leadiq

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"time"

	"github.com/machinebox/graphql"
	"go.uber.org/zap"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/x/errmsg"
)

// 1. Fetch FlatAdvancedSearch from LeadIQ
// 2. Filter prospects based on FilterBy
// 3. Order prospects based on OrderBy
// 4. Fetch email from searchPeople if workEmail is hidden
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

	prospects := make([]Prospect, 0, len(flatAdvancedSearchResp.FlatAdvancedSearch.People))

	filterBy := inputStruct.FilterBy
	if err := filterBy.compile(); err != nil {
		err = fmt.Errorf("compiling filter by: %w", err)
		job.Error.Error(ctx, err)
		return err
	}

	for _, person := range flatAdvancedSearchResp.FlatAdvancedSearch.People {
		if !filterBy.match(person) {
			continue
		}
		// It is possible that the work email is hidden, so we need to find the first non-hidden email
		// When workEmail is hidden, we should call searchPeople to get the email.
		// When there is no workEmail, we don't need to call searchPeople.
		var workEmail string
		for _, email := range person.WorkEmails {
			workEmail = email
			if email != "hidden" {
				break
			}
		}

		prospect := Prospect{
			Name:               person.Name,
			JobTitle:           person.Title,
			Seniority:          person.Seniority,
			Email:              workEmail,
			LinkedInURL:        person.LinkedInURL,
			CompanyDescription: person.Company.Description,
		}
		prospects = append(prospects, prospect)
	}

	orderBy := inputStruct.OrderBy
	sort.Slice(prospects, func(i, j int) bool {
		// Primary sort: JobTitle
		if orderBy.JobTitle == "asc" && prospects[i].JobTitle != prospects[j].JobTitle {
			return prospects[i].JobTitle < prospects[j].JobTitle
		}
		if orderBy.JobTitle == "desc" && prospects[i].JobTitle != prospects[j].JobTitle {
			return prospects[i].JobTitle > prospects[j].JobTitle
		}

		// Secondary sort: Seniority
		if orderBy.Seniority == "asc" {
			return prospects[i].Seniority < prospects[j].Seniority
		}
		if orderBy.Seniority == "desc" {
			return prospects[i].Seniority > prospects[j].Seniority
		}
		return false
	})

	for i := range prospects {
		prospectName := prospects[i].Name
		searchPeopleIn := buildSearchPeopleInput(inputStruct, prospectName)
		searchPeopleResp := searchPeopleResp{}
		if err := e.sendRequest(ctx, client, logger, searchPeopleQuery, searchPeopleIn, &searchPeopleResp); err != nil {
			msg := fmt.Sprintf("LeadIQ API result error: %v", err)
			err = errmsg.AddMessage(fmt.Errorf("sending search people request to LeadIQ with error"), msg)
			job.Error.Error(ctx, err)
			return err
		}

		if !foundEmail(searchPeopleResp) {
			continue
		}

		prospects[i].Email = searchPeopleResp.SearchPeople.Results[0].CurrentPositions[0].Emails[0].Value
	}

	output := FindProspectsOutput{
		Prospects: prospects,
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

	// Set API Key. graphql.NewRequest does not support setting headers.
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

			// If the error is too many requests, we use backoff retry strategy.
			if statusCode != "429" {
				logger.Error("Failed to send request to LeadIQ with status code",
					zap.String("status code", statusCode),
					zap.Error(err),
				)
				err = fmt.Errorf("sending request to LeadIQ: %w, status code with %s", err, statusCode)
				return err
			}

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
	return map[string]interface{}{
		"companyFilter": map[string]interface{}{
			"names": []string{in.CompanyName},
		},
		"limit": in.Limit,
	}
}

func buildSearchPeopleInput(in FindProspectsInput, prospectName string) map[string]interface{} {
	return map[string]interface{}{
		"fullName": prospectName,
		"company": map[string]string{
			"name": in.CompanyName,
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
	if f.Seniority != "" {
		f.seniorityRegex, err = regexp.Compile(f.Seniority)
		if err != nil {
			return fmt.Errorf("compiling seniority filter: %w", err)
		}
	}
	return nil
}

func (f *FilterBy) match(person person) bool {
	if f.jobTitleRegex != nil && !f.jobTitleRegex.MatchString(person.Title) {
		return false
	}
	if f.seniorityRegex != nil && !f.seniorityRegex.MatchString(person.Seniority) {
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
	Emails []email `json:"emails"`
}

type email struct {
	Status string `json:"status"`
	Type   string `json:"type"`
	Value  string `json:"value"`
}
