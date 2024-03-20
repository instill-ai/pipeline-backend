package utils

import (
	"fmt"
	"strings"
	"time"

	"github.com/influxdata/influxdb-client-go/v2/api/write"
	"google.golang.org/protobuf/types/known/structpb"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"

	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/internal/resource"

	componentBase "github.com/instill-ai/component/pkg/base"
	connector "github.com/instill-ai/component/pkg/connector"
	connectorAirbyte "github.com/instill-ai/component/pkg/connector/airbyte/v0"
	mgmtPB "github.com/instill-ai/protogen-go/core/mgmt/v1beta"
)

const (
	CreateEvent          string = "Create"
	UpdateEvent          string = "Update"
	DeleteEvent          string = "Delete"
	ActivateEvent        string = "Activate"
	DeactivateEvent      string = "Deactivate"
	TriggerEvent         string = "Trigger"
	ConnectEvent         string = "Connect"
	DisconnectEvent      string = "Disconnect"
	RenameEvent          string = "Rename"
	ExecuteEvent         string = "Execute"
	credentialMaskString string = "*****MASK*****"
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
	OwnerUID            string
	OwnerType           mgmtPB.OwnerType
	UserUID             string
	UserType            mgmtPB.OwnerType
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

func NewPipelineDataPoint(data PipelineUsageMetricData) *write.Point {
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

func IsConnector(name string) bool {
	return strings.HasPrefix(name, "connectors/")
}
func IsConnectorWithNamespace(name string) bool {
	return len(strings.Split(name, "/")) > 3 && strings.Split(name, "/")[2] == "connectors"
}

func IsConnectorDefinition(name string) bool {
	return strings.HasPrefix(name, "connector-definitions/")
}

func IsOperatorDefinition(name string) bool {
	return strings.HasPrefix(name, "operator-definitions/")
}

func MaskCredentialFields(connector componentBase.IConnector, defID string, config *structpb.Struct) {
	maskCredentialFields(connector, defID, config, "")
}

func maskCredentialFields(connector componentBase.IConnector, defID string, config *structpb.Struct, prefix string) {

	for k, v := range config.GetFields() {
		key := prefix + k
		if connector.IsCredentialField(defID, key) {
			config.GetFields()[k] = structpb.NewStringValue(credentialMaskString)
		}
		if v.GetStructValue() != nil {
			maskCredentialFields(connector, defID, v.GetStructValue(), fmt.Sprintf("%s.", key))
		}

	}
}

func RemoveCredentialFieldsWithMaskString(connector componentBase.IConnector, defID string, config *structpb.Struct) {
	removeCredentialFieldsWithMaskString(connector, defID, config, "")
}

func KeepCredentialFieldsWithMaskString(connector componentBase.IConnector, defID string, config *structpb.Struct) {
	keepCredentialFieldsWithMaskString(connector, defID, config, "")
}

func removeCredentialFieldsWithMaskString(connector componentBase.IConnector, defID string, config *structpb.Struct, prefix string) {

	for k, v := range config.GetFields() {
		key := prefix + k
		if connector.IsCredentialField(defID, key) {
			if v.GetStringValue() == credentialMaskString {
				delete(config.GetFields(), k)
			}
		}
		if v.GetStructValue() != nil {
			removeCredentialFieldsWithMaskString(connector, defID, v.GetStructValue(), fmt.Sprintf("%s.", key))
		}

	}
}
func keepCredentialFieldsWithMaskString(connector componentBase.IConnector, defID string, config *structpb.Struct, prefix string) {

	for k, v := range config.GetFields() {
		key := prefix + k
		if !connector.IsCredentialField(defID, key) {
			delete(config.GetFields(), k)
		}
		if v.GetStructValue() != nil {
			keepCredentialFieldsWithMaskString(connector, defID, v.GetStructValue(), fmt.Sprintf("%s.", key))
		}

	}
}

func GetConnectorOptions() connector.ConnectorOptions {
	return connector.ConnectorOptions{
		Airbyte: connectorAirbyte.ConnectorOptions{
			MountSourceVDP:        config.Config.Connector.Airbyte.MountSource.VDP,
			MountTargetVDP:        config.Config.Connector.Airbyte.MountTarget.VDP,
			MountSourceAirbyte:    config.Config.Connector.Airbyte.MountSource.Airbyte,
			MountTargetAirbyte:    config.Config.Connector.Airbyte.MountTarget.Airbyte,
			ExcludeLocalConnector: config.Config.Connector.Airbyte.ExcludeLocalConnector,
		},
	}

}
