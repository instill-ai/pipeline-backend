package jira

import (
	"fmt"

	_ "embed"

	"github.com/instill-ai/x/errmsg"
)

// Board is the Jira board object.
type Board struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Self      string `json:"self"`
	BoardType string `json:"type"`
}

type listBoardsResp struct {
	Boards     []Board `json:"boards"`
	StartAt    int     `json:"startAt"`
	MaxResults int     `json:"maxResults"`
	Total      int     `json:"total"`
	IsLast     bool    `json:"isLast"`
}

func listBoards(c *client, opt *listBoardsInput) (*listBoardsResp, error) {
	apiEndpoint := "rest/agile/1.0/board"

	req := c.R().SetResult(&listBoardsResp{})
	err := addQueryOptions(req, *opt)
	if err != nil {
		return nil, err
	}
	resp, err := req.Get(apiEndpoint)
	if err != nil {
		return nil, err
	}

	boards := resp.Result().(*listBoardsResp)

	return boards, err
}

type getBoardResp struct {
	Location struct {
		DisplayName string `json:"displayName"`
		Name        string `json:"name"`

		ProjectKey     string `json:"projectKey"`
		ProjectID      int    `json:"projectId"`
		ProjectName    string `json:"projectName"`
		ProjectTypeKey string `json:"projectTypeKey"`
		UserAccountID  string `json:"userAccountId"`
		UserID         string `json:"userId"`
	} `json:"location"`
	Board
}

func getBoard(c *client, boardID int) (*getBoardResp, error) {
	apiEndpoint := fmt.Sprintf("rest/agile/1.0/board/%v", boardID)

	req := c.R().SetResult(&getBoardResp{})
	resp, err := req.Get(apiEndpoint)
	if err != nil {
		return nil, fmt.Errorf(
			err.Error(), errmsg.Message(err),
		)
	}
	result := resp.Result().(*getBoardResp)

	return result, err
}
