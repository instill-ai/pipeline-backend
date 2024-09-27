package jira

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func router(middlewares ...func(http.Handler) http.Handler) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	for _, m := range middlewares {
		r.Use(m)
	}
	r.Get("/_edge/tenant_info", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"cloudId":"12345678-1234-1234-1234-123456789012"}`))
	})

	r.Post("/rest/agile/1.0/epic/{epic-key}/issue", mockMoveIssueToEpic)
	r.Get("/rest/agile/1.0/issue/{issueIdOrKey:[a-zA-z0-9-]+}", mockGetIssue)
	r.Get("/rest/agile/1.0/sprint/{sprintId}", mockGetSprint)
	r.Put("/rest/agile/1.0/sprint/{sprintId}", mockUpdateSprint)
	r.Post("/rest/agile/1.0/sprint", mockCreateSprint)

	r.Get("/rest/agile/1.0/board/{boardId}/issue", mockListIssues)           // list all issues
	r.Get("/rest/agile/1.0/board/{boardId}/epic", mockListIssues)            // list all epic
	r.Get("/rest/agile/1.0/board/{boardId}/sprint", mockListSprints)         // list all sprint
	r.Get("/rest/agile/1.0/board/{boardId}/backlog", mockListIssues)         // list all issues in backlog
	r.Get("/rest/agile/1.0/board/{boardId}/epic/none/issue", mockListIssues) // list all issues without epic assigned
	r.Get("/rest/agile/1.0/board/{boardId}", mockGetBoard)
	r.Get("/rest/agile/1.0/board", mockListBoards)

	r.Get("/rest/api/2/search", mockIssuesSearch)
	r.Post("/rest/api/2/search", mockIssuesSearch)

	r.Put("/rest/api/2/issue/{issue-key}", mockUpdateIssue)
	r.Post("/rest/api/2/issue", mockCreateIssue)
	return r
}

func mockListBoards(res http.ResponseWriter, req *http.Request) {
	var err error
	opt := req.URL.Query()
	boardType := opt.Get("type")
	startAt := opt.Get("startAt")
	maxResults := opt.Get("maxResults")
	name := opt.Get("name")
	projectKeyOrID := opt.Get("projectKeyOrId")
	// filter boards
	var boards []FakeBoard
	pjNotFound := projectKeyOrID != ""
	for _, board := range fakeBoards {
		if boardType != "" && board.BoardType != boardType {
			continue
		}
		if name != "" && !strings.Contains(board.Name, name) {
			continue
		}
		if projectKeyOrID != "" {
			if !strings.EqualFold(board.Name, projectKeyOrID) {
				continue
			}
			pjNotFound = false
		}
		boards = append(boards, board)
	}
	if pjNotFound {
		res.WriteHeader(http.StatusBadRequest)
		_, _ = res.Write([]byte(`{"errorMessages":["No project could be found with key or id"]}`))
		return
	}
	// pagination
	start, end := 0, len(boards)
	if startAt != "" {
		start, err = strconv.Atoi(startAt)
		if err != nil {
			return
		}
	}
	maxResultsNum := len(boards)
	if maxResults != "" {
		maxResultsNum, err = strconv.Atoi(maxResults)
		if err != nil {
			return
		}
		end = start + maxResultsNum
		if end > len(boards) {
			end = len(boards)
		}
	}
	// response
	res.WriteHeader(http.StatusOK)
	respText := `{"values":[`
	if len(boards) != 0 {
		for i, board := range boards[start:end] {
			if i > 0 {
				respText += ","
			}
			respText += fmt.Sprintf(`{"id":%d,"name":"%s","type":"%s","self":"%s"}`, board.ID, board.Name, board.BoardType, board.getSelf())
		}
	}
	respText += `],`
	respText += `"total":` + strconv.Itoa(len(boards)) + `,"startAt":` + strconv.Itoa(start) + `,"maxResults":` + strconv.Itoa(maxResultsNum) + `,"isLast":` + strconv.FormatBool(end == len(boards)) + `}`
	_, _ = res.Write([]byte(respText))
}

func mockGetBoard(res http.ResponseWriter, req *http.Request) {
	var err error
	boardID := chi.URLParam(req, "boardId")
	// filter boards
	var board *FakeBoard
	for _, b := range fakeBoards {
		if boardID != "" && strconv.Itoa(b.ID) != boardID {
			continue
		}
		board = &b
	}
	if board == nil {
		res.WriteHeader(http.StatusNotFound)
		_, _ = res.Write([]byte(`{"errorMessages":["Board does not exist or you do not have permission to see it"]}`))
		return
	}
	// response
	res.WriteHeader(http.StatusOK)
	respText, err := json.Marshal(board)
	if err != nil {
		return
	}
	_, _ = res.Write([]byte(respText))
}

func mockGetIssue(res http.ResponseWriter, req *http.Request) {
	var err error

	issueID := chi.URLParam(req, "issueIdOrKey")
	if issueID == "" {
		res.WriteHeader(http.StatusBadRequest)
		_, _ = res.Write([]byte(`{"errorMessages":["Issue id or key is required"]}`))
		return
	}
	// find issue
	var issue *FakeIssue
	for _, i := range fakeIssues {
		if i.ID == issueID || i.Key == issueID {
			issue = &i
			issue.getSelf()
			break
		}
	}
	if issue == nil {
		res.WriteHeader(http.StatusNotFound)
		_, _ = res.Write([]byte(`{"errorMessages":["Issue does not exist or you do not have permission to see it"]}`))
		return
	}
	fmt.Println(issue)
	// response
	res.WriteHeader(http.StatusOK)
	respText, err := json.Marshal(issue)
	if err != nil {
		return
	}
	_, _ = res.Write(respText)
}

func mockGetSprint(res http.ResponseWriter, req *http.Request) {
	var err error
	sprintID := chi.URLParam(req, "sprintId")
	if sprintID == "" {
		res.WriteHeader(http.StatusBadRequest)
		_, _ = res.Write([]byte(`{"errorMessages":["Sprint id is required"]}`))
		return
	}
	// find sprint
	var sprint *FakeSprint
	for _, s := range fakeSprints {
		if strconv.Itoa(s.ID) == sprintID {
			sprint = &s
			sprint.getSelf()
			break
		}
	}
	if sprint == nil {
		res.WriteHeader(http.StatusNotFound)
		_, _ = res.Write([]byte(`{"errorMessages":["Sprint does not exist or you do not have permission to see it"]}`))
		return
	}
	// response
	res.WriteHeader(http.StatusOK)
	respText, err := json.Marshal(sprint)
	if err != nil {
		return
	}
	_, _ = res.Write(respText)
}

type MockListIssuesResponse struct {
	Issues     []FakeIssue `json:"issues"`
	Total      int         `json:"total"`
	StartAt    int         `json:"startAt"`
	MaxResults int         `json:"maxResults"`
}

func mockListIssues(res http.ResponseWriter, req *http.Request) {
	var err error
	opt := req.URL.Query()
	boardID := chi.URLParam(req, "boardId")
	jql := opt.Get("jql")
	startAt := opt.Get("startAt")
	maxResults := opt.Get("maxResults")
	// find board
	var board *FakeBoard
	for _, b := range fakeBoards {
		if strconv.Itoa(b.ID) == boardID {
			board = &b
			break
		}
	}
	if board == nil {
		res.WriteHeader(http.StatusNotFound)
		_, _ = res.Write([]byte(`{"errorMessages":["Board does not exist or you do not have permission to see it"]}`))
		return
	}
	// filter issues
	var issues []FakeIssue
	for _, issue := range fakeIssues {
		prefix := strings.Split(issue.Key, "-")[0]
		if board.Name != "" && prefix != board.Name {
			continue
		}
		if jql != "" {
			// Skip JQL filter as there is no need to implement it
			continue
		}
		issue.getSelf()
		issues = append(issues, issue)
	}
	// response
	res.WriteHeader(http.StatusOK)
	startAtNum := 0
	if startAt != "" {
		startAtNum, err = strconv.Atoi(startAt)
		if err != nil {
			return
		}
	}
	maxResultsNum, err := strconv.Atoi(maxResults)
	if err != nil {
		return
	}
	resp := MockListIssuesResponse{
		Issues:     issues,
		Total:      len(issues),
		StartAt:    startAtNum,
		MaxResults: maxResultsNum,
	}
	respText, err := json.Marshal(resp)
	if err != nil {
		return
	}
	_, _ = res.Write([]byte(respText))
}

type MockListSprintsResponse struct {
	Values     []FakeSprint `json:"values"`
	StartAt    int          `json:"startAt"`
	MaxResults int          `json:"maxResults"`
	Total      int          `json:"total"`
}

func mockListSprints(res http.ResponseWriter, req *http.Request) {
	var err error
	opt := req.URL.Query()
	boardID := chi.URLParam(req, "boardId")
	state := opt.Get("state")
	startAt := opt.Get("startAt")
	maxResults := opt.Get("maxResults")
	// find board
	var board *FakeBoard
	for _, b := range fakeBoards {
		if strconv.Itoa(b.ID) == boardID {
			board = &b
			break
		}
	}
	if board == nil {
		res.WriteHeader(http.StatusNotFound)
		_, _ = res.Write([]byte(`{"errorMessages":["Board does not exist or you do not have permission to see it"]}`))
		return
	}
	// filter sprints
	var sprints []FakeSprint
	for _, sprint := range fakeSprints {
		if sprint.ID != board.ID {
			continue
		}
		if state != "" && sprint.State != state {
			continue
		}
		sprints = append(sprints, sprint)
	}
	// pagination
	start, end := 0, len(sprints)
	if startAt != "" {
		start, err = strconv.Atoi(startAt)
		if err != nil {
			return
		}
	}
	maxResultsNum := len(sprints)
	if maxResults != "" {
		maxResultsNum, err = strconv.Atoi(maxResults)
		if err != nil {
			return
		}
		end = start + maxResultsNum
		if end > len(sprints) {
			end = len(sprints)
		}
	}
	// response
	res.WriteHeader(http.StatusOK)

	resp := MockListSprintsResponse{
		Values:     sprints[start:end],
		StartAt:    start,
		MaxResults: maxResultsNum,
		Total:      len(sprints),
	}
	for i := range resp.Values {
		resp.Values[i].getSelf()

	}
	respText, err := json.Marshal(resp)
	if err != nil {
		return
	}
	_, _ = res.Write([]byte(respText))
}

type MockIssuesSearchRequest struct {
	JQL        string `json:"jql"`
	StartAt    int    `json:"startAt"`
	MaxResults int    `json:"maxResults"`
}

func mockIssuesSearch(res http.ResponseWriter, req *http.Request) {
	var err error
	var (
		opt        url.Values
		jql        string
		startAt    string
		maxResults string
	)
	if req.Method == http.MethodGet {
		opt = req.URL.Query()
		jql = opt.Get("jql")
		startAt = opt.Get("startAt")
		maxResults = opt.Get("maxResults")
	} else if req.Method == http.MethodPost {
		body := MockIssuesSearchRequest{}
		err = json.NewDecoder(req.Body).Decode(&body)
		if err != nil {
			fmt.Println(err)
			return
		}
		jql = body.JQL
		startAt = strconv.Itoa(body.StartAt)
		maxResults = strconv.Itoa(body.MaxResults)
	} else {
		res.WriteHeader(http.StatusMethodNotAllowed)
		_, _ = res.Write([]byte(`{"errorMessages":["Method not allowed"]}`))
		return
	}
	// filter issues
	var issues []FakeIssue
	for _, issue := range fakeIssues {
		if jql != "" {
			// Skip JQL filter as there is no need to implement it
			continue
		}
		issue.getSelf()
		issues = append(issues, issue)
	}
	// response
	res.WriteHeader(http.StatusOK)
	startAtNum := 0
	if startAt != "" {
		startAtNum, err = strconv.Atoi(startAt)
		if err != nil {
			return
		}
	}
	maxResultsNum, err := strconv.Atoi(maxResults)
	if err != nil {
		return
	}
	resp := MockListIssuesResponse{
		Issues:     issues,
		Total:      len(issues),
		StartAt:    startAtNum,
		MaxResults: maxResultsNum,
	}
	respText, err := json.Marshal(resp)
	if err != nil {
		return
	}
	_, _ = res.Write([]byte(respText))
}
