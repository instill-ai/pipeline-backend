package slack

import (
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
)

type rawSlackMessage struct {
	Type           string       `instill:"type"`
	Token          string       `instill:"token"`
	TeamID         string       `instill:"team_id"`
	APIAppID       string       `instill:"api_app_id"`
	Event          format.Value `instill:"event"`
	EventContext   string       `instill:"event_context"`
	EventID        string       `instill:"event_id"`
	EventTime      int64        `instill:"event_time"`
	Authorizations []struct {
		EnterpriseID        *string `instill:"enterprise_id"`
		TeamID              string  `instill:"team_id"`
		UserID              string  `instill:"user_id"`
		IsBot               bool    `instill:"is_bot"`
		IsEnterpriseInstall bool    `instill:"is_enterprise_install"`
	} `instill:"authorizations"`
	IsExtSharedChannel  bool    `instill:"is_ext_shared_channel"`
	ContextTeamID       string  `instill:"context_team_id"`
	ContextEnterpriseID *string `instill:"context_enterprise_id"`
}

type rawSlackEventType struct {
	Type string `instill:"type"`
}

type rawSlackMessageEvent struct {
	Type    string `instill:"type"`
	Channel string `instill:"channel"`
	User    string `instill:"user"`
	Text    string `instill:"text"`
	TS      string `instill:"ts"`
}

type channel struct {
	ID   string `instill:"id"`
	Name string `instill:"name"`
}

type user struct {
	ID       string      `instill:"id"`
	Name     string      `instill:"name"`
	RealName string      `instill:"real-name"`
	Profile  userProfile `instill:"profile"`
}

type userProfile struct {
	DisplayName string `instill:"display-name"`
}

type slackEventNewMessage struct {
	Timestamp string  `instill:"timestamp"`
	Channel   channel `instill:"channel"`
	User      user    `instill:"user"`
	Text      string  `instill:"text"`
}

type slackEventNewMessageConfig struct {
	ChannelNames []string `instill:"channel-names"`
}
