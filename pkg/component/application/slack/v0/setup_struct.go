package slack

type slackComponentSetup struct {
	BotToken  string  `instill:"bot-token"`
	UserToken *string `instill:"user-token"`
}
