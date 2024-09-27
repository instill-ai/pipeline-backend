package asana

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

type Job struct {
	GID        string  `json:"gid" api:"gid"`
	NewTask    Task    `json:"task" api:"new_task.name"`
	NewProject Project `json:"project" api:"new_project.name"`
}

type SimpleProject struct {
	GID  string `json:"gid"`
	Name string `json:"name"`
}

type Goal struct {
	GID       string `json:"gid" api:"gid"`
	Name      string `json:"name" api:"name"`
	Owner     User   `json:"owner" api:"owner.name"`
	Notes     string `json:"notes" api:"notes"`
	HTMLNotes string `json:"html-notes" api:"html_notes"`
	DueOn     string `json:"due-on" api:"due_on"`
	StartOn   string `json:"start-on" api:"start_on"`
	Liked     bool   `json:"liked" api:"liked"`
	Likes     []Like `json:"likes" api:"likes.user.name"`
}

type TaskParent struct {
	GID             string `json:"gid"`
	Name            string `json:"name"`
	ResourceSubtype string `json:"resource-subtype"`
	CreatedBy       User   `json:"created-by"`
}
type Task struct {
	GID             string          `json:"gid" api:"gid"`
	Name            string          `json:"name" api:"name"`
	Notes           string          `json:"notes" api:"notes"`
	HTMLNotes       string          `json:"html-notes" api:"html_notes"`
	Projects        []SimpleProject `json:"projects" api:"projects.name"`
	DueOn           string          `json:"due-on" api:"due_on"`
	StartOn         string          `json:"start-on" api:"start_on"`
	Liked           bool            `json:"liked" api:"liked"`
	Likes           []Like          `json:"likes" api:"likes.user.name"`
	ApprovalStatus  string          `json:"approval-status" api:"approval_status"`
	ResourceSubtype string          `json:"resource-subtype" api:"resource_subtype"`
	Completed       bool            `json:"completed" api:"completed"`
	Assignee        string          `json:"assignee" api:"assignee.name"`
	Parent          string          `json:"parent" api:"parent.name"`
}

type Project struct {
	GID                 string                   `json:"gid" api:"gid"`
	Name                string                   `json:"name" api:"name"`
	Owner               User                     `json:"owner" api:"owner.name"`
	Notes               string                   `json:"notes" api:"notes"`
	HTMLNotes           string                   `json:"html-notes" api:"html_notes"`
	DueOn               string                   `json:"due-on" api:"due_on"`
	StartOn             string                   `json:"start-on" api:"start_on"`
	Completed           bool                     `json:"completed" api:"completed"`
	Color               string                   `json:"color" api:"color"`
	PrivacySetting      string                   `json:"privacy-setting" api:"privacy_setting"`
	Archived            bool                     `json:"archived" api:"archived"`
	CompletedBy         User                     `json:"completed-by" api:"completed_by.name"`
	CurrentStatus       []map[string]interface{} `json:"current-status" api:"current_status.created_by.name,current_status.author.name,current_status.color,current_status.html_text,current_status.modified_at,current_status.text,current_status.title"`
	CustomFields        []map[string]interface{} `json:"custom-fields" api:"custom_fields.type,custom_fields.text_value,custom_fields.resource_subtype,custom_fields.representation_type,custom_fields.date_value.date_time,custom_fields.enabled,custom_fields.enum_options.name,custom_fields.enum_value.name,custom_fields.id_prefix,custom_fields.is_formula_field,custom_fields.multi_enum_values.name,custom_fields.name,custom_fields.number_value"`
	CustomFieldSettings []map[string]interface{} `json:"custom-field-settings" api:"custom_field_settings.custom_field.created_by.name,custom_field_settings.custom_field.currency_code,custom_field_settings.custom_field.custom_label,custom_field_settings.custom_field.date_value.date_time,custom_field_settings.custom_field.description,custom_field_settings.custom_field.enum_value.name,custom_field_settings.custom_field.enum_options.name,custom_field_settings.custom_field.name,custom_field_settings.custom_field.people_value.name,custom_field_settings.custom_field.type,custom_field_settings.custom_field.text_value,custom_field_settings.custom_field.resource_subtype"`
	ModifiedAt          string                   `json:"modified-at" api:"modified_at"`
}

type Portfolio struct {
	GID                 string                   `json:"gid" api:"gid"`
	Name                string                   `json:"name" api:"name"`
	Owner               User                     `json:"owner" api:"owner.name"`
	DueOn               string                   `json:"due-on" api:"due_on"`
	StartOn             string                   `json:"start-on" api:"start_on"`
	Color               string                   `json:"color" api:"color"`
	Public              bool                     `json:"public" api:"public"`
	CreatedBy           User                     `json:"created-by" api:"created_by.name"`
	CurrentStatus       []map[string]interface{} `json:"current-status" api:"current_status.created_by.name,current_status.author.name,current_status.color,current_status.html_text,current_status.modified_at,current_status.text,current_status.title"`
	CustomFields        []map[string]interface{} `json:"custom-fields" api:"custom_fields.type,custom_fields.text_value,custom_fields.resource_subtype,custom_fields.representation_type,custom_fields.date_value.date_time,custom_fields.enabled,custom_fields.enum_options.name,custom_fields.enum_value.name,custom_fields.id_prefix,custom_fields.is_formula_field,custom_fields.multi_enum_values.name,custom_fields.name,custom_fields.number_value"`
	CustomFieldSettings []map[string]interface{} `json:"custom-field-settings" api:"custom_field_settings.custom_field.created_by.name,custom_field_settings.custom_field.currency_code,custom_field_settings.custom_field.custom_label,custom_field_settings.custom_field.date_value.date_time,custom_field_settings.custom_field.description,custom_field_settings.custom_field.enum_value.name,custom_field_settings.custom_field.enum_options.name,custom_field_settings.custom_field.name,custom_field_settings.custom_field.people_value.name,custom_field_settings.custom_field.type,custom_field_settings.custom_field.text_value,custom_field_settings.custom_field.resource_subtype"`
}
