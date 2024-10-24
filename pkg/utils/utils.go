package utils

import (
	"context"
	"strings"
	"time"

	"github.com/influxdata/influxdb-client-go/v2/api/write"
	"google.golang.org/protobuf/types/known/structpb"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"

	"github.com/instill-ai/pipeline-backend/pkg/constant"
	"github.com/instill-ai/pipeline-backend/pkg/resource"

	mgmtPB "github.com/instill-ai/protogen-go/core/mgmt/v1beta"
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

func IsAuditEvent(eventName string) bool {
	return strings.HasPrefix(eventName, CreateEvent) ||
		strings.HasPrefix(eventName, UpdateEvent) ||
		strings.HasPrefix(eventName, DeleteEvent) ||
		strings.HasPrefix(eventName, ActivateEvent) ||
		strings.HasPrefix(eventName, DeactivateEvent) ||
		strings.HasPrefix(eventName, ConnectEvent) ||
		strings.HasPrefix(eventName, DisconnectEvent) ||
		strings.HasPrefix(eventName, TriggerEvent) ||
		strings.HasPrefix(eventName, RenameEvent) ||
		strings.HasPrefix(eventName, ExecuteEvent)
}

func IsBillableEvent(eventName string) bool {
	return strings.HasPrefix(eventName, TriggerEvent)
}

type PipelineUsageMetricData struct {
	OwnerUID  string
	OwnerType mgmtPB.OwnerType

	// User represents the authenticated user. Only user authentication is
	// supported at the moment.
	UserUID  string
	UserType mgmtPB.OwnerType

	// Requester will differ from User impersonates another namespace when
	// triggering the pipeline. The only supported impersonation is from an
	// authenticated user to an organization they belong to.
	RequesterUID  string
	RequesterType mgmtPB.OwnerType

	TriggerMode         mgmtPB.Mode
	Status              mgmtPB.Status
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

type ConnectorUsageMetricData struct {
	OwnerUID               string
	OwnerType              mgmtPB.OwnerType
	UserUID                string
	UserType               mgmtPB.OwnerType
	Status                 mgmtPB.Status
	ConnectorID            string
	ConnectorUID           string
	ConnectorExecuteUID    string
	ConnectorDefinitionUID string
	ExecuteTime            string
	ComputeTimeDuration    float64
}

func NewConnectorDataPoint(data ConnectorUsageMetricData, pipelineMetadata *structpb.Value) *write.Point {
	pipelineOwnerUUID, _ := resource.GetRscPermalinkUID(pipelineMetadata.GetStructValue().GetFields()["owner"].GetStringValue())
	return influxdb2.NewPoint(
		"connector.execute",
		map[string]string{
			"status": data.Status.String(),
		},
		map[string]any{
			"pipeline_id":              pipelineMetadata.GetStructValue().GetFields()["id"].GetStringValue(),
			"pipeline_uid":             pipelineMetadata.GetStructValue().GetFields()["uid"].GetStringValue(),
			"pipeline_release_id":      pipelineMetadata.GetStructValue().GetFields()["release_id"].GetStringValue(),
			"pipeline_release_uid":     pipelineMetadata.GetStructValue().GetFields()["release_uid"].GetStringValue(),
			"pipeline_owner":           pipelineOwnerUUID,
			"pipeline_trigger_id":      pipelineMetadata.GetStructValue().GetFields()["trigger_id"].GetStringValue(),
			"connector_owner_uid":      data.OwnerUID,
			"connector_owner_type":     data.OwnerType,
			"connector_user_uid":       data.UserUID,
			"connector_user_type":      data.UserType,
			"connector_id":             data.ConnectorID,
			"connector_uid":            data.ConnectorUID,
			"connector_definition_uid": data.ConnectorDefinitionUID,
			"connector_execute_id":     data.ConnectorExecuteUID,
			"execute_time":             data.ExecuteTime,
			"compute_time_duration":    data.ComputeTimeDuration,
		},
		time.Now(),
	)
}

func GetRequesterUIDAndUserUID(ctx context.Context) (string, string) {
	requesterUID := resource.GetRequestSingleHeader(ctx, constant.HeaderRequesterUIDKey)
	userUID := resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey)
	if len(strings.TrimSpace(requesterUID)) == 0 {
		requesterUID = userUID
	}
	return requesterUID, userUID
}
