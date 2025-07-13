package jira

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-resty/resty/v2"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"

	errorsx "github.com/instill-ai/x/errors"
)

func (c *client) listIssues(ctx context.Context, job *base.Job) error {

	var (
		input listIssuesInput
		jql   string
	)

	if err := job.Input.ReadData(ctx, &input); err != nil {
		return fmt.Errorf("reading input data: %w", err)
	}

	boards, err := listBoards(c, &listBoardsInput{Name: input.BoardName})
	if err != nil {
		return fmt.Errorf("listing boards: %w", err)
	}

	if len(boards.Boards) == 0 {
		return errorsx.AddMessage(
			fmt.Errorf("board not found"),
			fmt.Sprintf("board with name %s not found", input.BoardName),
		)
	} else if len(boards.Boards) > 1 {
		return errorsx.AddMessage(
			fmt.Errorf("multiple boards found"),
			fmt.Sprintf("multiple boards are found with the partial name \"%s\". Please provide a more specific name", input.BoardName),
		)
	}

	board := boards.Boards[0]

	boardDetails, err := getBoard(c, board.ID)
	if err != nil {
		return fmt.Errorf("getting board details: %w", err)
	}
	projectKey := boardDetails.Location.ProjectKey
	if projectKey == "" {
		projectKey = strings.Split(board.Name, "-")[0]
	}
	apiEndpoint := fmt.Sprintf("rest/agile/1.0/board/%d", board.ID)
	switch input.RangeData.Range {
	case "All":
		// https://developer.atlassian.com/cloud/jira/software/rest/api-group-board/#api-rest-agile-1-0-board-boardid-issue-get
		apiEndpoint = apiEndpoint + "/issue"
	case "Epics only":
		// https://developer.atlassian.com/cloud/jira/software/rest/api-group-board/#api-rest-agile-1-0-board-boardid-epic-get
		apiEndpoint = apiEndpoint + "/epic"
	case "Issues of an epic":
		// API not working: https://developer.atlassian.com/cloud/jira/software/rest/api-group-board/#api-rest-agile-1-0-board-boardid-epic-epicid-issue-get
		// use JQL instead
		jql = fmt.Sprintf("project=\"%s\" AND parent=\"%s\"", projectKey, input.RangeData.EpicKey)
	case "Issues of a sprint":
		// API not working: https://developer.atlassian.com/cloud/jira/software/rest/api-group-board/#api-rest-agile-1-0-board-boardid-sprint-sprintid-issue-get
		// use JQL instead
		jql = fmt.Sprintf("project=\"%s\" AND sprint=\"%s\"", projectKey, input.RangeData.SprintName)
	case "In backlog only":
		// https://developer.atlassian.com/cloud/jira/software/rest/api-group-board/#api-rest-agile-1-0-board-boardid-backlog-get
		apiEndpoint = apiEndpoint + "/backlog"
	case "Issues without epic assigned":
		// https://developer.atlassian.com/cloud/jira/software/rest/api-group-board/#api-rest-agile-1-0-board-boardid-epic-none-issue-get
		apiEndpoint = apiEndpoint + "/epic/none/issue"
	case "Standard Issues":
		// https://support.atlassian.com/jira-cloud-administration/docs/what-are-issue-types/
		jql = fmt.Sprintf("project=\"%s\" AND issuetype not in (Epic, subtask)", projectKey)
	case "JQL query":
		jql = input.RangeData.JQL
	default:
		return errorsx.AddMessage(
			fmt.Errorf("invalid range"),
			fmt.Sprintf("%s is an invalid range", input.RangeData.Range),
		)
	}

	var resp *resty.Response
	if jql != "" {
		resp, err = nextGenIssuesSearch(c.Client, nextGenSearchReq{
			JQL:        jql,
			MaxResults: input.MaxResults,
			StartAt:    input.StartAt,
		},
		)
	} else {
		req := c.R().SetResult(&listIssuesResp{})
		err = addQueryOptions(req, map[string]interface{}{
			"maxResults": input.MaxResults,
			"startAt":    input.StartAt,
		})
		if err != nil {
			return fmt.Errorf("adding query options: %w", err)
		}
		resp, err = req.Get(apiEndpoint)
	}

	if err != nil {
		return fmt.Errorf("getting issues: %w", err)
	}

	issues, ok := resp.Result().(*listIssuesResp)
	if !ok {
		return errorsx.AddMessage(
			fmt.Errorf("failed to convert response to `List Issue` Output"),
			fmt.Sprintf("failed to convert %v to `List Issue` Output", resp.Result()),
		)
	}

	if issues.Issues == nil {
		issues.Issues = []Issue{}
	}

	output := listIssuesOutput{
		Issues:     issues.Issues,
		StartAt:    issues.StartAt,
		MaxResults: issues.MaxResults,
		Total:      issues.Total,
	}
	for idx, issue := range output.Issues {
		output.Issues[idx] = *extractIssue(&issue)
		if input.RangeData.Range == "Epics only" {
			output.Issues[idx].IssueType = "Epic"
		}
	}
	return job.Output.WriteData(ctx, output)
}
