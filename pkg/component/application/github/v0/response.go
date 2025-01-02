package github

import "github.com/google/go-github/v62/github"

// Response represents pagination information from GitHub API responses
type Response struct {
	NextPage      int    `instill:"next-page"`
	PrevPage      int    `instill:"prev-page"`
	FirstPage     int    `instill:"first-page"`
	LastPage      int    `instill:"last-page"`
	NextPageToken string `instill:"next-page-token"`
	Cursor        string `instill:"cursor"`
	Before        string `instill:"before"`
	After         string `instill:"after"`
}

func (client *Client) extractResponse(oriResponse *github.Response) *Response {
	if oriResponse == nil {
		return nil
	}

	return &Response{
		NextPage:      oriResponse.NextPage,
		PrevPage:      oriResponse.PrevPage,
		FirstPage:     oriResponse.FirstPage,
		LastPage:      oriResponse.LastPage,
		NextPageToken: oriResponse.NextPageToken,
		Cursor:        oriResponse.Cursor,
		Before:        oriResponse.Before,
		After:         oriResponse.After,
	}
}
