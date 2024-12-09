package jira

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type mockUpdateIssueReq struct {
	IssueKey    string                        `json:"issue-key"`
	Update      map[string][]additionalFields `json:"update"`
	Fields      map[string]interface{}        `json:"fields"`
	NotifyUsers bool                          `json:"notify-users" api:"notifyUsers"`
	ReturnIssue bool                          `json:"return-issue" api:"returnIssue"`
}
type mockUpdateIssueResp struct {
	Issue
	NotifyUsers bool `json:"notify-users" api:"notifyUsers"`
	ReturnIssue bool `json:"return-issue" api:"returnIssue"`
}

// UpdateIssue updates an issue in Jira.
func mockUpdateIssue(res http.ResponseWriter, req *http.Request) {
	var mocUpdateIssueReq mockUpdateIssueReq
	err := json.NewDecoder(req.Body).Decode(&mocUpdateIssueReq)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}
	issueKey := chi.URLParam(req, "issue-key")
	if issueKey == "" {
		http.Error(res, "issue key is required", http.StatusBadRequest)
		return
	}
	var fi *fakeIssue
	for _, i := range fakeIssues {
		if i.ID == issueKey || i.Key == issueKey {
			fi = &i
			fi.getSelf()
			break
		}
	}
	if fi == nil {
		http.Error(res, "issue not found", http.StatusNotFound)
		return
	}
	opt := req.URL.Query()
	notifyUsers := opt.Get("notifyUsers")
	returnIssue := opt.Get("returnIssue")
	for key, fields := range mocUpdateIssueReq.Update {
		for _, field := range fields {
			if field.Set != "" {
				fi.Fields[key] = field.Set
			}
		}
	}
	for key, field := range mocUpdateIssueReq.Fields {
		if field != "" {
			fi.Fields[key] = field
		}
	}
	newIssue := Issue{
		ID:          fi.ID,
		Key:         fi.Key,
		Self:        fi.Self,
		Fields:      fi.Fields,
		Description: fi.Fields["description"].(string),
		IssueType:   fi.Fields["issuetype"].(map[string]interface{})["name"].(string),
		Summary:     fi.Fields["summary"].(string),
		Status:      fi.Fields["status"].(map[string]interface{})["name"].(string),
	}
	for issue := range fakeIssues {
		if fakeIssues[issue].ID == newIssue.ID {
			fakeIssues[issue] = fakeIssue{
				ID:     newIssue.ID,
				Key:    newIssue.Key,
				Self:   newIssue.Self,
				Fields: newIssue.Fields,
			}
			break
		}
	}
	resp := mockUpdateIssueResp{
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
