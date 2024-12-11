package jira

import (
	"fmt"
	"reflect"

	_ "embed"

	"github.com/go-resty/resty/v2"

	jsoniter "github.com/json-iterator/go"

	"github.com/instill-ai/x/errmsg"
)

// Issue is the Jira issue object.
type Issue struct {
	ID          string                 `json:"id"`
	Key         string                 `json:"key"`
	Description string                 `json:"description"`
	Summary     string                 `json:"summary"`
	Fields      map[string]interface{} `json:"fields"`
	Self        string                 `json:"self"`
	IssueType   string                 `json:"issue-type"`
	Status      string                 `json:"status"`
}

func extractIssue(issue *Issue) *Issue {
	if issue.Description == "" && issue.Fields["description"] != nil {
		description, ok := issue.Fields["description"].(string)
		if ok {
			issue.Description = description
		}
	}
	if issue.Summary == "" && issue.Fields["summary"] != nil {
		summary, ok := issue.Fields["summary"].(string)
		if ok {
			issue.Summary = summary
		}
	}
	if issue.IssueType == "" && issue.Fields["issuetype"] != nil {
		if issueType, ok := issue.Fields["issuetype"]; ok {
			if issue.IssueType, ok = issueType.(map[string]interface{})["name"].(string); !ok {
				issue.IssueType = ""
			}
		}
	}
	if issue.Status == "" && issue.Fields["status"] != nil {
		if status, ok := issue.Fields["status"]; ok {
			if issue.Status, ok = status.(map[string]interface{})["name"].(string); !ok {
				issue.Status = ""
			}
		}
	}
	return issue
}

func getIssue(client *resty.Client, issueKey string, updateHistory bool) (*Issue, error) {
	apiEndpoint := fmt.Sprintf("rest/agile/1.0/issue/%s", issueKey)
	req := client.R().SetResult(&Issue{})

	err := addQueryOptions(req, map[string]interface{}{
		"updateHistory": updateHistory,
	})
	if err != nil {
		return nil, fmt.Errorf("adding query options: %w", err)
	}

	resp, err := req.Get(apiEndpoint)
	if resp != nil && resp.StatusCode() == 404 {
		return nil, fmt.Errorf(
			err.Error(),
			errmsg.Message(err)+"Please check you have the correct permissions to access this resource.",
		)
	}
	if err != nil {
		return nil, fmt.Errorf("getting issue: %w", err)
	}

	issue, ok := resp.Result().(*Issue)
	if !ok {
		return nil, errmsg.AddMessage(
			fmt.Errorf("failed to convert response to `Get Issue` Output"),
			fmt.Sprintf("failed to convert %v to `Get Issue` Output", resp.Result()),
		)
	}

	return extractIssue(issue), nil
}

type issueRange struct {
	Range      string `json:"range,omitempty"`
	EpicKey    string `json:"epic-key,omitempty"`
	SprintName string `json:"sprint-name,omitempty"`
	JQL        string `json:"jql,omitempty"`
}

type listIssuesResp struct {
	Issues     []Issue `json:"issues"`
	StartAt    int     `json:"startAt"`
	MaxResults int     `json:"maxResults"`
	Total      int     `json:"total"`
}

// https://support.atlassian.com/jira-software-cloud/docs/jql-fields/
type nextGenSearchReq struct {
	JQL        string `json:"jql,omitempty" api:"jql"`
	MaxResults int    `json:"maxResults,omitempty" api:"maxResults"`
	StartAt    int    `json:"startAt,omitempty" api:"startAt"`
}

// https://developer.atlassian.com/cloud/jira/platform/rest/v2/api-group-issue-search/#api-rest-api-2-search-get
// https://developer.atlassian.com/cloud/jira/platform/rest/v2/api-group-issue-search/#api-rest-api-2-search-post
func nextGenIssuesSearch(client *resty.Client, opt nextGenSearchReq) (*resty.Response, error) {

	var err error
	apiEndpoint := "/rest/api/2/search"

	req := client.R().SetResult(&listIssuesResp{})
	var resp *resty.Response
	if len(opt.JQL) < 50 {
		// 50 is an arbitrary number to determine if the JQL is too long to be a query param
		if err := addQueryOptions(req, opt); err != nil {
			return nil, err
		}
		resp, err = req.Get(apiEndpoint)
	} else {
		req.SetBody(opt)
		resp, err = req.Post(apiEndpoint)
	}

	if err != nil {
		return nil, err
	}
	return resp, nil
}

type additionalFields struct {
	Add    any `json:"add,omitempty"`
	Copy   any `json:"copy,omitempty"`
	Set    any `json:"set,omitempty"`
	Edit   any `json:"edit,omitempty"`
	Remove any `json:"remove,omitempty"`
}
type issueType struct {
	IssueType       string `json:"issue-type"`
	ParentKey       string `json:"parent-key"`
	CustomIssueType string `json:"custom-issue-type"`
}

type createIssueReq struct {
	Fields map[string]interface{}        `json:"fields"`
	Update map[string][]additionalFields `json:"update"`
}
type createIssueResp struct {
	ID         string `json:"id"`
	Key        string `json:"key"`
	Self       string `json:"self"`
	Transition struct {
		Status          string `json:"status"`
		ErrorCollection struct {
			ErrorMessages []string               `json:"errorMessages"`
			Errors        map[string]interface{} `json:"errors"`
		} `json:"errorCollection"`
	} `json:"transition"`
}

func convertCreateIssueReq(issue *createIssueInput) *createIssueReq {
	newReq := &createIssueReq{
		Fields: map[string]interface{}{
			"project": map[string]interface{}{
				"key": issue.ProjectKey,
			},
			"issuetype": map[string]interface{}{
				"name": issue.IssueType.IssueType,
			},
			"summary":     issue.Summary,
			"description": issue.Description,
		},
	}
	if issue.IssueType.ParentKey != "" {
		newReq.Fields["parent"] = map[string]interface{}{
			"key": issue.IssueType.ParentKey,
		}
	}
	if issue.IssueType.CustomIssueType != "" {
		newReq.Fields["issuetype"] = map[string]interface{}{
			"name": issue.IssueType.CustomIssueType,
		}
	}
	return newReq
}

type updateField struct {
	Action    string `json:"action"`
	FieldName string `json:"field-name"`
	Value     any    `json:"value"`
}

type update struct {
	UpdateType   string        `json:"update"`
	UpdateFields []updateField `json:"update-fields"`
	EpicKey      string        `json:"epic-key"`
}

type updateIssueReq struct {
	Body struct {
		Update map[string][]additionalFields `json:"update,omitempty"`
		Fields map[string]interface{}        `json:"fields,omitempty"`
	}
	Query struct {
		NotifyUsers bool `json:"notify-users" api:"notifyUsers"`
		ReturnIssue bool `json:"return-issue" api:"returnIssue"`
	}
}
type updateIssueResp struct {
	Issue
}

func moveIssueToEpic(client *resty.Client, issueKey, epicKey string) error {
	apiEndpoint := fmt.Sprintf("/rest/agile/1.0/epic/%s/issue", epicKey)
	req := client.R().SetBody(fmt.Sprintf(`{"issues":["%s"]}`, issueKey))
	resp, err := req.Post(apiEndpoint)
	if err != nil {
		return err
	}
	if resp.StatusCode() != 204 {
		return errmsg.AddMessage(
			fmt.Errorf("failed to move issue to epic"),
			fmt.Sprintf(`failed to move issue "%s" to epic "%s"`, issueKey, epicKey),
		)
	}
	return nil
}

func updateIssue(client *resty.Client, input *updateIssueInput) (*updateIssueResp, error) {
	if input.Update.UpdateType != "Custom Update" {
		return nil, errmsg.AddMessage(
			fmt.Errorf("invalid update type"),
			fmt.Sprintf("%s is an invalid update type", input.Update.UpdateType),
		)
	}
	updateInfo := make(map[string][]additionalFields)
	fieldsInfo := make(map[string]interface{})
	for _, field := range input.Update.UpdateFields {
		if field.FieldName == "" {
			return nil, errmsg.AddMessage(
				fmt.Errorf("field name is required"),
				"field name is required",
			)
		}
		if updateInfo[field.FieldName] == nil {
			updateInfo[field.FieldName] = []additionalFields{}
		}
		switch field.Action {
		case "set":
			if v := reflect.ValueOf(field.Value); v.Kind() != reflect.Slice || v.Len() <= 1 {
				fieldsInfo[field.FieldName] = field.Value
				delete(updateInfo, field.FieldName)
			} else {
				updateInfo[field.FieldName] = append(updateInfo[field.FieldName], additionalFields{Set: field.Value})
			}
		case "add":
			updateInfo[field.FieldName] = append(updateInfo[field.FieldName], additionalFields{Add: field.Value})
		case "remove":
			updateInfo[field.FieldName] = append(updateInfo[field.FieldName], additionalFields{Remove: field.Value})
		case "edit":
			updateInfo[field.FieldName] = append(updateInfo[field.FieldName], additionalFields{Edit: field.Value})
		case "copy":
			updateInfo[field.FieldName] = append(updateInfo[field.FieldName], additionalFields{Copy: field.Value})
		default:
			return nil, errmsg.AddMessage(
				fmt.Errorf("invalid action"),
				fmt.Sprintf("%s is an invalid action", field.Action),
			)
		}
	}
	apiEndpoint := "rest/api/2/issue/" + input.IssueKey
	updateIssueReq := updateIssueReq{
		Body: struct {
			Update map[string][]additionalFields `json:"update,omitempty"`
			Fields map[string]interface{}        `json:"fields,omitempty"`
		}{
			Update: updateInfo,
			Fields: fieldsInfo,
		},
		Query: struct {
			NotifyUsers bool `json:"notify-users" api:"notifyUsers"`
			ReturnIssue bool `json:"return-issue" api:"returnIssue"`
		}{
			NotifyUsers: input.NotifyUsers,
			ReturnIssue: true,
		},
	}

	body, err := jsoniter.Marshal(updateIssueReq.Body)
	if err != nil {
		return nil, err
	}
	req := client.R().SetResult(&updateIssueResp{}).SetBody(string(body))
	err = addQueryOptions(req, updateIssueReq.Query)
	if err != nil {
		return nil, err
	}
	resp, err := req.Put(apiEndpoint)
	if err != nil {
		return nil, err
	}

	updatedIssue, ok := resp.Result().(*updateIssueResp)

	if !ok {
		return nil, errmsg.AddMessage(
			fmt.Errorf("failed to convert response to `Update Issue` Output"),
			fmt.Sprintf("failed to convert %v to `Update Issue` Output", resp.Result()),
		)
	}
	return updatedIssue, nil
}
