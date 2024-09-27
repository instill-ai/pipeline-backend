package mockasana

type User struct {
	GID  string `json:"gid"`
	Name string `json:"name"`
}

type Like struct {
	LikeGID  string `json:"like-gid"`
	UserGID  string `json:"user-gid"`
	UserName string `json:"name"`
}

type RawLike struct {
	LikeGID string `json:"gid"`
	User    User   `json:"user"`
}

type SimpleProject struct {
	GID  string `json:"gid"`
	Name string `json:"name"`
}

type TaskParent struct {
	GID             string `json:"gid"`
	Name            string `json:"name"`
	ResourceSubtype string `json:"resource-subtype"`
	CreatedBy       User   `json:"created-by"`
}

type ScheduleDates struct {
	ShouldSkipWeekends bool   `json:"should_skip_weekends"`
	DueOn              string `json:"due_on,omitempty"`
	StartOn            string `json:"start_on,omitempty"`
}

var FakeUser = []User{
	{
		GID:  "123",
		Name: "Admin User",
	},
	{
		GID:  "456",
		Name: "Test User",
	},
}

var FakeLike = []RawLike{
	{
		LikeGID: "123",
		User:    FakeUser[0],
	},
}

type RawGoal struct {
	GID       string    `json:"gid"`
	Name      string    `json:"name"`
	Owner     User      `json:"owner"`
	Notes     string    `json:"notes"`
	HTMLNotes string    `json:"html_notes"`
	DueOn     string    `json:"due_on"`
	StartOn   string    `json:"start_on"`
	Liked     bool      `json:"liked"`
	Likes     []RawLike `json:"likes"`
}

var FakeGoal = []RawGoal{
	{
		GID:       "1234",
		Name:      "Test Goal",
		Owner:     FakeUser[0],
		Notes:     "Test Notes",
		HTMLNotes: "Test HTML Notes",
		DueOn:     "2021-01-01",
		StartOn:   "2021-01-01",
		Liked:     true,
		Likes:     FakeLike,
	},
	{
		GID:       "1234567890",
		Name:      "Test Goal (To be deleted)",
		Owner:     FakeUser[0],
		Notes:     "Test Notes",
		HTMLNotes: "Test HTML Notes",
		DueOn:     "2021-01-01",
		StartOn:   "2021-01-01",
		Liked:     true,
		Likes:     FakeLike,
	},
}

type RawProject struct {
	GID                 string            `json:"gid"`
	Name                string            `json:"name"`
	Owner               User              `json:"owner"`
	Notes               string            `json:"notes"`
	HTMLNotes           string            `json:"html_notes"`
	DueOn               string            `json:"due_on"`
	StartOn             string            `json:"start_on"`
	Completed           bool              `json:"completed"`
	Color               string            `json:"color"`
	PrivacySetting      string            `json:"privacy_setting"`
	Archived            bool              `json:"archived"`
	CompletedBy         User              `json:"completed_by"`
	CurrentStatus       map[string]string `json:"current_status"`
	CustomFields        map[string]string `json:"custom_fields"`
	CustomFieldSettings map[string]string `json:"custom_field_settings"`
	ModifiedAt          string            `json:"modified_at"`
}

var FakeProject = []RawProject{
	{
		GID:                 "1234",
		Name:                "Test Project",
		Owner:               FakeUser[0],
		Notes:               "Test Notes",
		HTMLNotes:           "Test HTML Notes",
		DueOn:               "2021-01-01",
		StartOn:             "2021-01-01",
		Completed:           true,
		Color:               "red",
		PrivacySetting:      "public_to_workspace",
		Archived:            true,
		CompletedBy:         FakeUser[0],
		CurrentStatus:       map[string]string{"status": "completed"},
		CustomFields:        map[string]string{"field": "value"},
		CustomFieldSettings: map[string]string{"field": "value"},
		ModifiedAt:          "2021-01-01",
	},
	{
		GID:                 "123",
		Name:                "Test Project",
		Owner:               FakeUser[0],
		Notes:               "Test Notes",
		HTMLNotes:           "Test HTML Notes",
		DueOn:               "2021-01-01",
		StartOn:             "2021-01-01",
		Completed:           true,
		Color:               "red",
		PrivacySetting:      "public_to_workspace",
		Archived:            true,
		CompletedBy:         FakeUser[0],
		CurrentStatus:       map[string]string{"status": "completed"},
		CustomFields:        map[string]string{"field": "value"},
		CustomFieldSettings: map[string]string{"field": "value"},
		ModifiedAt:          "2021-01-01",
	},
	{
		GID:                 "1234567890",
		Name:                "Test Project",
		Owner:               FakeUser[0],
		Notes:               "Test Notes",
		HTMLNotes:           "Test HTML Notes",
		DueOn:               "2021-01-01",
		StartOn:             "2021-01-01",
		Completed:           true,
		Color:               "red",
		PrivacySetting:      "public_to_workspace",
		Archived:            true,
		CompletedBy:         FakeUser[0],
		CurrentStatus:       map[string]string{"status": "completed"},
		CustomFields:        map[string]string{"field": "value"},
		CustomFieldSettings: map[string]string{"field": "value"},
		ModifiedAt:          "2021-01-01",
	},
}

type RawTask struct {
	GID             string          `json:"gid"`
	Name            string          `json:"name"`
	Notes           string          `json:"notes"`
	HTMLNotes       string          `json:"html_notes"`
	Projects        []SimpleProject `json:"projects"`
	DueOn           string          `json:"due_on"`
	StartOn         string          `json:"start_on"`
	DueAt           string          `json:"due_at,omitempty"`
	StartAt         string          `json:"start_at,omitempty"`
	Liked           bool            `json:"liked"`
	Likes           []RawLike       `json:"likes"`
	ApprovalStatus  string          `json:"approval_status"`
	ResourceSubtype string          `json:"resource_subtype"`
	Completed       bool            `json:"completed"`
	Assignee        User            `json:"assignee"`
	Parent          TaskParent      `json:"parent"`
}

var FakeTask = []RawTask{
	{
		GID:       "1234",
		Name:      "Test Task",
		Notes:     "Test Notes",
		HTMLNotes: "Test HTML Notes",
		Projects: []SimpleProject{
			{
				GID:  "1234",
				Name: "Test Project",
			},
		},
		DueOn:           "2021-01-01",
		StartOn:         "2021-01-01",
		Liked:           true,
		Likes:           FakeLike,
		ApprovalStatus:  "approved",
		ResourceSubtype: "default_task",
		Completed:       true,
		Assignee:        FakeUser[0],
		Parent: TaskParent{
			GID:             "1234",
			Name:            "Test Task",
			ResourceSubtype: "default_task",
			CreatedBy:       FakeUser[0],
		},
	},
	{
		GID:       "43210",
		Name:      "Test Task",
		Notes:     "Test Notes",
		HTMLNotes: "Test HTML Notes",
		Projects: []SimpleProject{
			{
				GID:  "1234",
				Name: "Test Project",
			},
		},
		DueOn:           "2021-01-01",
		StartOn:         "2021-01-01",
		Liked:           true,
		Likes:           FakeLike,
		ApprovalStatus:  "approved",
		ResourceSubtype: "default_task",
		Completed:       true,
		Assignee:        FakeUser[0],
		Parent: TaskParent{
			GID:             "1234",
			Name:            "Test Task",
			ResourceSubtype: "default_task",
			CreatedBy:       FakeUser[0],
		},
	},
	{
		GID:       "1234567890",
		Name:      "Test Task",
		Notes:     "Test Notes",
		HTMLNotes: "Test HTML Notes",
		Projects: []SimpleProject{
			{
				GID:  "1234",
				Name: "Test Project",
			},
		},
		DueOn:           "2021-01-01",
		StartOn:         "2021-01-01",
		Liked:           true,
		Likes:           FakeLike,
		ApprovalStatus:  "approved",
		ResourceSubtype: "default_task",
		Completed:       true,
		Assignee:        FakeUser[0],
		Parent: TaskParent{
			GID:             "1234",
			Name:            "Test Task",
			ResourceSubtype: "default_task",
			CreatedBy:       FakeUser[0],
		},
	},
}

type RawPortfolio struct {
	GID                 string            `json:"gid"`
	Name                string            `json:"name"`
	Owner               User              `json:"owner"`
	Notes               string            `json:"notes"`
	HTMLNotes           string            `json:"html_notes"`
	DueOn               string            `json:"due_on"`
	StartOn             string            `json:"start_on"`
	Color               string            `json:"color"`
	Public              bool              `json:"public"`
	CreatedBy           User              `json:"created_by"`
	CurrentStatus       map[string]string `json:"current_status"`
	CustomFields        map[string]string `json:"custom_fields"`
	CustomFieldSettings map[string]string `json:"custom_field_settings"`
}

var FakePortfolio = []RawPortfolio{
	{
		GID:                 "1234",
		Name:                "Test Portfolio",
		Owner:               FakeUser[0],
		DueOn:               "2021-01-01",
		StartOn:             "2021-01-01",
		Color:               "red",
		Public:              true,
		CreatedBy:           FakeUser[0],
		CurrentStatus:       map[string]string{"title": "On track"},
		CustomFields:        map[string]string{"field": "value"},
		CustomFieldSettings: map[string]string{"field": "value"},
	},
	{
		GID:                 "4321",
		Name:                "Test Portfolio",
		Owner:               FakeUser[0],
		DueOn:               "2021-01-01",
		StartOn:             "2021-01-01",
		Color:               "red",
		Public:              true,
		CreatedBy:           FakeUser[0],
		CurrentStatus:       map[string]string{"title": "On track"},
		CustomFields:        map[string]string{"field": "value"},
		CustomFieldSettings: map[string]string{"field": "value"},
	},
	{
		GID:                 "1234567890",
		Name:                "Test Portfolio",
		Owner:               FakeUser[0],
		DueOn:               "2021-01-01",
		StartOn:             "2021-01-01",
		Color:               "red",
		Public:              true,
		CreatedBy:           FakeUser[0],
		CurrentStatus:       map[string]string{"title": "On track"},
		CustomFields:        map[string]string{"field": "value"},
		CustomFieldSettings: map[string]string{"field": "value"},
	},
}
