package jira

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type mockCreateIssueReq struct {
	Fields map[string]interface{}        `json:"fields"`
	Update map[string][]additionalFields `json:"update"`
}

type mockCreateIssueResp struct {
	ID         string `json:"id"`
	Key        string `json:"key"`
	Self       string `json:"self"`
	Transition struct {
		Status          string `json:"status"`
		ErrorCollection struct {
			ErrorMessages []string               `json:"errorMessages"`
			Errors        map[string]interface{} `json:"errors"`
		} `json:"errorCollection"`
	}
}

func mockCreateIssue(res http.ResponseWriter, req *http.Request) {
	var err error
	if req.Method != http.MethodPost {
		http.Error(res, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	body := mockCreateIssueReq{}
	err = json.NewDecoder(req.Body).Decode(&body)
	if err != nil {
		fmt.Println(err)
		return
	}
	fields := body.Fields
	// update := body.Update
	project, ok := fields["project"].(map[string]interface{})["key"].(string)
	if !ok {
		http.Error(res, "Invalid project", http.StatusBadRequest)
		return
	}
	badResp := mockCreateIssueResp{
		ID:   "",
		Key:  "",
		Self: "",
		Transition: struct {
			Status          string `json:"status"`
			ErrorCollection struct {
				ErrorMessages []string               `json:"errorMessages"`
				Errors        map[string]interface{} `json:"errors"`
			} `json:"errorCollection"`
		}{
			Status: "Failed",
			ErrorCollection: struct {
				ErrorMessages []string               `json:"errorMessages"`
				Errors        map[string]interface{} `json:"errors"`
			}{
				ErrorMessages: []string{"Invalid project"},
				Errors:        map[string]interface{}{},
			},
		},
	}
	if project == "INVALID" {
		res.WriteHeader(http.StatusBadRequest)
		err = json.NewEncoder(res).Encode(badResp)
		if err != nil {
			fmt.Println(err)
			return
		}
		return
	}
	key := project + "-1"
	ID := "30000"
	successResp := mockCreateIssueResp{
		ID:   ID,
		Key:  key,
		Self: "http://localhost:8080/rest/api/2/issue/10000",
		Transition: struct {
			Status          string `json:"status"`
			ErrorCollection struct {
				ErrorMessages []string               `json:"errorMessages"`
				Errors        map[string]interface{} `json:"errors"`
			} `json:"errorCollection"`
		}{
			Status: "Success",
			ErrorCollection: struct {
				ErrorMessages []string               `json:"errorMessages"`
				Errors        map[string]interface{} `json:"errors"`
			}{
				ErrorMessages: []string{},
				Errors:        map[string]interface{}{},
			},
		},
	}
	res.WriteHeader(http.StatusOK)
	err = json.NewEncoder(res).Encode(successResp)
	if err != nil {
		fmt.Println(err)
		return
	}

	fakeIssues = append(fakeIssues, fakeIssue{
		ID:     ID,
		Key:    key,
		Fields: fields,
	})
}
