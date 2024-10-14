package slack

import (
	"github.com/slack-go/slack"
)

func newClient(token string) *slack.Client {
	return slack.New(token)
}
