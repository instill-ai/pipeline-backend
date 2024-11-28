package smartlead

type createCampaignInput struct {
	Name string `instill:"name"`
}

type createCampaignOutput struct {
	ID        string `instill:"id"`
	CreatedAt string `instill:"created-at"`
}

type setupCampaignInput struct {
	CampaignName  string   `instill:"campaign-name"`
	Timezone      string   `instill:"timezone"`
	DaysOfTheWeek []string `instill:"days-of-the-week"`
	// With (HH:MM) format
	StartHour                   string   `instill:"start-hour"`
	EndHour                     string   `instill:"end-hour"`
	MinTimeBtwEmails            int      `instill:"min-time-btw-emails"`
	MaxNewLeadsPerDay           int      `instill:"max-new-leads-per-day"`
	ScheduleStartTime           string   `instill:"schedule-start-time"`
	TrackSettings               []string `instill:"track-settings"`
	StopLeadSettings            string   `instill:"stop-lead-settings"`
	SendAsPlainText             bool     `instill:"send-as-plain-text"`
	FollowUpPercentage          int      `instill:"follow-up-percentage"`
	AddUnsubscribeTag           bool     `instill:"add-unsubscribe-tag"`
	IgnoreSsMailboxSendingLimit bool     `instill:"ignore-ss-mailbox-sending-limit"`
}

type setupCampaignOutput struct {
	ScheduleSettingResult string `instill:"schedule-setting-result"`
	GeneralSettingResult  string `instill:"general-setting-result"`
}

type saveSequencesInput struct {
	CampaignName string     `instill:"campaign-name"`
	Sequences    []sequence `instill:"sequences"`
}

type sequence struct {
	SeqID             string `instill:"seq-id"`
	SeqNumber         int    `instill:"seq-number"`
	SequenceDelayDays int    `instill:"sequence-delay-days"`
	Subject           string `instill:"subject"`
	EmailBody         string `instill:"email-body"`
}

type saveSequencesOutput struct {
	Result string `instill:"result"`
}

type getSequencesInput struct {
	CampaignName string `instill:"campaign-name"`
}

type getSequencesOutput struct {
	Sequences []sequence `instill:"sequences"`
}

type addLeadsInput struct {
	CampaignName string   `instill:"campaign-name"`
	Leads        []lead   `instill:"leads"`
	Settings     settings `instill:"settings"`
}

type lead struct {
	Email        string        `instill:"email"`
	FirstName    string        `instill:"first-name"`
	LastName     string        `instill:"last-name"`
	Company      string        `instill:"company"`
	Location     string        `instill:"location"`
	CustomFields []customField `instill:"custom-fields"`
}

type customField struct {
	Key   string `instill:"key"`
	Value string `instill:"value"`
}

type settings struct {
	IgnoreGlobalBlockList               bool `instill:"ignore-global-block-list"`
	IgnoreUnsubscribeList               bool `instill:"ignore-unsubscribe-list"`
	IgnoreCommunityBounceList           bool `instill:"ignore-community-bounce-list"`
	IgnoreDuplicateLeadsInOtherCampaign bool `instill:"ignore-duplicate-leads-in-other-campaign"`
}

type addLeadsOutput struct {
	UploadCount            int    `instill:"upload-count"`
	TotalLeads             int    `instill:"total-leads"`
	AlreadyAddedToCampaign int    `instill:"already-added-to-campaign"`
	InvalidEmailCount      int    `instill:"invalid-email-count"`
	Error                  string `instill:"error"`
}

type addSenderEmailInput struct {
	CampaignName string `instill:"campaign-name"`
	SenderEmail  string `instill:"sender-email"`
}

type addSenderEmailOutput struct {
	Result string `instill:"result"`
}

type updateCampaignStatusInput struct {
	CampaignName string `instill:"campaign-name"`
	Status       string `instill:"status"`
}

type updateCampaignStatusOutput struct {
	Result string `instill:"result"`
}

type getCampaignMetricInput struct {
	CampaignName string `instill:"campaign-name"`
	StartDate    string `instill:"start-date"`
	EndDate      string `instill:"end-date"`
}

type getCampaignMetricOutput struct {
	SentCount        int `instill:"sent-count"`
	UniqueSentCount  int `instill:"unique-sent-count"`
	OpenCount        int `instill:"open-count"`
	UniqueOpenCount  int `instill:"unique-open-count"`
	ClickCount       int `instill:"click-count"`
	UniqueClickCount int `instill:"unique-click-count"`
	ReplyCount       int `instill:"reply-count"`
	TotalCount       int `instill:"total-count"`
	BounceCount      int `instill:"bounce-count"`
}

type listLeadsStatusInput struct {
	CampaignName string `instill:"campaign-name"`
	Limit        int    `instill:"limit"`
}

type listLeadsStatusOutput struct {
	Leads []leadStatus `instill:"leads"`
}

type leadStatus struct {
	Email      string `instill:"email"`
	Status     string `instill:"status"`
	LastSeqNum int    `instill:"last-seq-num"`
}
