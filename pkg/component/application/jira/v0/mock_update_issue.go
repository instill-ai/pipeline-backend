package jira

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type MockUpdateIssueRequset struct {
	IssueKey    string                        `json:"issue-key"`
	Update      map[string][]AdditionalFields `json:"update"`
	Fields      map[string]interface{}        `json:"fields"`
	NotifyUsers bool                          `json:"notify-users" api:"notifyUsers"`
	ReturnIssue bool                          `json:"return-issue" api:"returnIssue"`
}
type MockUpdateIssueResp struct {
	Issue
	NotifyUsers bool `json:"notify-users" api:"notifyUsers"`
	ReturnIssue bool `json:"return-issue" api:"returnIssue"`
}

// UpdateIssue updates an issue in Jira.
func mockUpdateIssue(res http.ResponseWriter, req *http.Request) {
	var request MockUpdateIssueRequset
	err := json.NewDecoder(req.Body).Decode(&request)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}
	issueKey := chi.URLParam(req, "issue-key")
	if issueKey == "" {
		http.Error(res, "issue key is required", http.StatusBadRequest)
		return
	}
	var issue *FakeIssue
	for _, i := range fakeIssues {
		if i.ID == issueKey || i.Key == issueKey {
			issue = &i
			issue.getSelf()
			break
		}
	}
	if issue == nil {
		http.Error(res, "issue not found", http.StatusNotFound)
		return
	}
	opt := req.URL.Query()
	notifyUsers := opt.Get("notifyUsers")
	returnIssue := opt.Get("returnIssue")
	for key, fields := range request.Update {
		for _, field := range fields {
			if field.Set != "" {
				issue.Fields[key] = field.Set
			}
		}
	}
	for key, field := range request.Fields {
		if field != "" {
			issue.Fields[key] = field
		}
	}
	newIssue := Issue{
		ID:          issue.ID,
		Key:         issue.Key,
		Self:        issue.Self,
		Fields:      issue.Fields,
		Description: issue.Fields["description"].(string),
		IssueType:   issue.Fields["issuetype"].(map[string]interface{})["name"].(string),
		Summary:     issue.Fields["summary"].(string),
		Status:      issue.Fields["status"].(map[string]interface{})["name"].(string),
	}
	for issue := range fakeIssues {
		if fakeIssues[issue].ID == newIssue.ID {
			fakeIssues[issue] = FakeIssue{
				ID:     newIssue.ID,
				Key:    newIssue.Key,
				Self:   newIssue.Self,
				Fields: newIssue.Fields,
			}
			break
		}
	}
	resp := MockUpdateIssueResp{
		Issue:       newIssue,
		NotifyUsers: notifyUsers == "true",
		ReturnIssue: returnIssue == "true",
	}
	err = json.NewEncoder(res).Encode(resp)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
}

func mockMoveIssueToEpic(res http.ResponseWriter, _ *http.Request) {
	http.Error(res, "The request contains a next-gen issue", http.StatusBadRequest)
}
