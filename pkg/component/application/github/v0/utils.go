package github

import (
	"fmt"
	"time"

	"github.com/instill-ai/x/errmsg"
)

type PageOptions struct {
	Page    int `json:"page"`
	PerPage int `json:"per-page"`
}

func parseTime(since string) (*time.Time, error) {
	sinceTime, err := time.Parse(time.RFC3339, since)
	if err != nil {
		return nil, errmsg.AddMessage(
			fmt.Errorf("invalid time format"),
			fmt.Sprintf("Cannot parse time: \"%s\". Please provide RFC3339 format(like %s)", since, time.RFC3339),
		)
	}
	return &sinceTime, nil
}
