package otel

type SystemLogMessage struct {
	TraceId      string
	SpanId       string
	Owner        string
	Resource     string
	ResourceUUID string
}

type ErrorLogMessage struct {
	ServiceName string
	TraceInfo   struct {
		TraceId string
		SpanId  string
	}
	StatusCode   int
	ErrorMessage string
}

type AuditLogMessage struct {
	ServiceName string
	TraceInfo   struct {
		TraceId string
		SpanId  string
	}
	UserInfo struct {
		UserID   string
		UserUUID string
		Token    string
	}
	EventInfo struct {
		Name string
	}
	ResourceInfo struct {
		ResourceName  string
		ResourceUUID  string
		ResourceState string
		Billable      bool
	}
	Result   string
	Metadata string
}
