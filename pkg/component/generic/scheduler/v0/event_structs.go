package scheduler

type EventCronJobTriggered struct {
	Cron string `instill:"cron"`
}

type EventCronJobTriggeredMessage struct {
	UID     string `instill:"uid"`
	EventID string `instill:"eventID"`
}
