package utils

import (
	"reflect"
	"time"

	"github.com/gofrs/uuid"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
	"google.golang.org/grpc/metadata"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"

	mgmtpb "github.com/instill-ai/protogen-go/mgmt/v1beta"
)

const (
	CreateEvent     string = "Create"
	UpdateEvent     string = "Update"
	DeleteEvent     string = "Delete"
	ActivateEvent   string = "Activate"
	DeactivateEvent string = "Deactivate"
	TriggerEvent    string = "Trigger"
	ConnectEvent    string = "Connect"
	DisconnectEvent string = "Disconnect"
	RenameEvent     string = "Rename"
	ExecuteEvent    string = "Execute"

	pipelineMeasurement = "pipeline.trigger.v1"
)

// Resource prefix constants for pipeline-backend AIP-compliant IDs
const (
	PrefixPipeline        = "pip"
	PrefixPipelineRelease = "rel"
	PrefixSecret          = "sec"
	PrefixConnection      = "con"
	PrefixTag             = "tag"
)

type PipelineUsageMetricData struct {
	OwnerUID  string
	OwnerType mgmtpb.OwnerType

	// User represents the authenticated user. Only user authentication is
	// supported at the moment.
	UserUID  string
	UserType mgmtpb.OwnerType

	// Requester will differ from User impersonates another namespace when
	// triggering the pipeline. The only supported impersonation is from an
	// authenticated user to an organization they belong to.
	RequesterUID  string
	RequesterType mgmtpb.OwnerType

	TriggerMode         mgmtpb.Mode
	Status              mgmtpb.Status
	PipelineID          string
	PipelineUID         string
	PipelineReleaseID   string
	PipelineReleaseUID  string
	PipelineTriggerUID  string
	TriggerTime         string
	ComputeTimeDuration float64
}

// NewPipelineDataPoint transforms the information of a pipeline trigger into
// an InfluxDB datapoint.
func NewPipelineDataPoint(data PipelineUsageMetricData) *write.Point {
	// The tags contain metadata, i.e. information we might filter or group by.
	tags := map[string]string{
		"status":         data.Status.String(),
		"trigger_mode":   data.TriggerMode.String(),
		"owner_uid":      data.OwnerUID,
		"owner_type":     data.OwnerType.String(),
		"user_uid":       data.UserUID,
		"user_type":      data.UserType.String(),
		"requester_uid":  data.RequesterUID,
		"requester_type": data.RequesterType.String(),
		"pipeline_id":    data.PipelineID,
		"pipeline_uid":   data.PipelineUID,
	}

	// Optional tags
	if data.PipelineReleaseID != "" {
		tags["pipeline_release_id"] = data.PipelineReleaseID
		tags["pipeline_release_uid"] = data.PipelineReleaseUID
	}

	fields := map[string]any{
		"pipeline_trigger_id":   data.PipelineTriggerUID,
		"trigger_time":          data.TriggerTime,
		"compute_time_duration": data.ComputeTimeDuration,
	}

	return influxdb2.NewPoint(pipelineMeasurement, tags, fields, time.Now())
}

// DeprecatedNewPipelineDatapoint transforms the information of a pipeline
// triger into an InfluxDB datapoint. This measurement is deprecated and will
// be retired with the new dashboard implementation.
func DeprecatedNewPipelineDatapoint(data PipelineUsageMetricData) *write.Point {
	return influxdb2.NewPoint(
		"pipeline.trigger",
		map[string]string{
			"status":       data.Status.String(),
			"trigger_mode": data.TriggerMode.String(),
		},
		map[string]any{
			"owner_uid":             data.OwnerUID,
			"owner_type":            data.OwnerType,
			"user_uid":              data.UserUID,
			"user_type":             data.UserType,
			"pipeline_id":           data.PipelineID,
			"pipeline_uid":          data.PipelineUID,
			"pipeline_release_id":   data.PipelineReleaseID,
			"pipeline_release_uid":  data.PipelineReleaseUID,
			"pipeline_trigger_id":   data.PipelineTriggerUID,
			"trigger_time":          data.TriggerTime,
			"compute_time_duration": data.ComputeTimeDuration,
		},
		time.Now(),
	)
}

// StructToMap converts a struct to a map with the given tag.
func StructToMap(s interface{}, tag string) map[string]interface{} {
	out := make(map[string]interface{})
	v := reflect.ValueOf(s)
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i).Interface()
		if jsonTag := field.Tag.Get(tag); jsonTag != "" {
			out[jsonTag] = value
		}
	}
	return out
}

// They are same logic in the some components like Instill Artifact, Instill Model.
// We can extract this logic to the shared package.
// But for now, we keep it here because we want to avoid that the components depend on pipeline shared package.
func GetRequestMetadata(vars map[string]any) metadata.MD {
	md := metadata.Pairs(
		"Authorization", getHeaderAuthorization(vars),
		"Instill-User-Uid", getInstillUserUID(vars),
		"Instill-Auth-Type", "user",
	)

	if requester := getInstillRequesterUID(vars); requester != "" {
		md.Set("Instill-Requester-Uid", requester)
	}
	return md
}

func getHeaderAuthorization(vars map[string]any) string {
	if v, ok := vars["__PIPELINE_HEADER_AUTHORIZATION"]; ok {
		return v.(string)
	}
	return ""
}
func getInstillUserUID(vars map[string]any) string {
	if v, ok := vars["__PIPELINE_USER_UID"]; ok {
		switch uid := v.(type) {
		case uuid.UUID:
			return uid.String()
		case string:
			return uid
		}
	}
	return ""
}

func getInstillRequesterUID(vars map[string]any) string {
	if v, ok := vars["__PIPELINE_REQUESTER_UID"]; ok {
		switch uid := v.(type) {
		case uuid.UUID:
			return uid.String()
		case string:
			return uid
		}
	}
	return ""
}
