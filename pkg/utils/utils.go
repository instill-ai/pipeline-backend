package utils

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"

	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/internal/resource"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"

	componentBase "github.com/instill-ai/component/pkg/base"
	connector "github.com/instill-ai/connector/pkg"
	connectorAirbyte "github.com/instill-ai/connector/pkg/airbyte"
	mgmtPB "github.com/instill-ai/protogen-go/core/mgmt/v1beta"
	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1beta"
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
		map[string]interface{}{
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
	ConnectorDefinitionUid string
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
		map[string]interface{}{
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
			"connector_definition_uid": data.ConnectorDefinitionUid,
			"connector_execute_id":     data.ConnectorExecuteUID,
			"execute_time":             data.ExecuteTime,
			"compute_time_duration":    data.ComputeTimeDuration,
		},
		time.Now(),
	)
}

func GenerateTraces(comps []*datamodel.Component, memory []map[string]interface{}, status []map[string]*ComponentStatus, computeTime map[string]float32, batchSize int) (map[string]*pipelinePB.Trace, error) {
	trace := map[string]*pipelinePB.Trace{}
	for compIdx := range comps {
		inputs := []*structpb.Struct{}
		outputs := []*structpb.Struct{}
		var traceStatuses []pipelinePB.Trace_Status
		for dataIdx := 0; dataIdx < batchSize; dataIdx++ {
			if status[dataIdx][comps[compIdx].Id].Completed {
				traceStatuses = append(traceStatuses, pipelinePB.Trace_STATUS_COMPLETED)
			} else if status[dataIdx][comps[compIdx].Id].Skipped {
				traceStatuses = append(traceStatuses, pipelinePB.Trace_STATUS_SKIPPED)
			} else if status[dataIdx][comps[compIdx].Id].Error {
				traceStatuses = append(traceStatuses, pipelinePB.Trace_STATUS_ERROR)
			} else {
				traceStatuses = append(traceStatuses, pipelinePB.Trace_STATUS_UNSPECIFIED)
			}

		}

		if comps[compIdx].DefinitionName != "operator-definitions/2ac8be70-0f7a-4b61-a33d-098b8acfa6f3" &&
			comps[compIdx].DefinitionName != "operator-definitions/4f39c8bc-8617-495d-80de-80d0f5397516" {
			for dataIdx := 0; dataIdx < batchSize; dataIdx++ {
				if _, ok := memory[dataIdx][comps[compIdx].Id].(map[string]interface{})["input"]; ok {
					data, err := json.Marshal(memory[dataIdx][comps[compIdx].Id].(map[string]interface{})["input"])
					if err != nil {
						return nil, err
					}
					inputStruct := &structpb.Struct{}
					err = protojson.Unmarshal(data, inputStruct)
					if err != nil {
						return nil, err
					}
					inputs = append(inputs, inputStruct)
				}

			}
			for dataIdx := 0; dataIdx < batchSize; dataIdx++ {
				if _, ok := memory[dataIdx][comps[compIdx].Id].(map[string]interface{})["output"]; ok {
					data, err := json.Marshal(memory[dataIdx][comps[compIdx].Id].(map[string]interface{})["output"])
					if err != nil {
						return nil, err
					}
					outputStruct := &structpb.Struct{}
					err = protojson.Unmarshal(data, outputStruct)
					if err != nil {
						return nil, err
					}
					outputs = append(outputs, outputStruct)
				}

			}
		}

		trace[comps[compIdx].Id] = &pipelinePB.Trace{
			Statuses:             traceStatuses,
			Inputs:               inputs,
			Outputs:              outputs,
			ComputeTimeInSeconds: computeTime[comps[compIdx].Id],
		}
	}
	return trace, nil
}

func GenerateGlobalValue(pipelineUid uuid.UUID, recipe *datamodel.Recipe, ownerPermalink string) (map[string]interface{}, error) {
	global := map[string]interface{}{}

	global["pipeline"] = map[string]interface{}{
		"uid":    pipelineUid.String(),
		"recipe": recipe,
	}
	global["owner"] = map[string]interface{}{
		"uid": uuid.FromStringOrNil(strings.Split(ownerPermalink, "/")[1]),
	}

	return global, nil
}

func IsConnector(resourceName string) bool {
	return strings.HasPrefix(resourceName, "connectors/")
}
func IsConnectorWithNamespace(resourceName string) bool {
	return len(strings.Split(resourceName, "/")) > 3 && strings.Split(resourceName, "/")[2] == "connectors"
}

func IsConnectorDefinition(resourceName string) bool {
	return strings.HasPrefix(resourceName, "connector-definitions/")
}

func IsOperatorDefinition(resourceName string) bool {
	return strings.HasPrefix(resourceName, "operator-definitions/")
}

func MaskCredentialFields(connector componentBase.IConnector, defId string, config *structpb.Struct) {
	maskCredentialFields(connector, defId, config, "")
}

func maskCredentialFields(connector componentBase.IConnector, defId string, config *structpb.Struct, prefix string) {

	for k, v := range config.GetFields() {
		key := prefix + k
		if connector.IsCredentialField(defId, key) {
			config.GetFields()[k] = structpb.NewStringValue(credentialMaskString)
		}
		if v.GetStructValue() != nil {
			maskCredentialFields(connector, defId, v.GetStructValue(), fmt.Sprintf("%s.", key))
		}

	}
}

func RemoveCredentialFieldsWithMaskString(connector componentBase.IConnector, defId string, config *structpb.Struct) {
	removeCredentialFieldsWithMaskString(connector, defId, config, "")
}

func KeepCredentialFieldsWithMaskString(connector componentBase.IConnector, defId string, config *structpb.Struct) {
	keepCredentialFieldsWithMaskString(connector, defId, config, "")
}

func removeCredentialFieldsWithMaskString(connector componentBase.IConnector, defId string, config *structpb.Struct, prefix string) {

	for k, v := range config.GetFields() {
		key := prefix + k
		if connector.IsCredentialField(defId, key) {
			if v.GetStringValue() == credentialMaskString {
				delete(config.GetFields(), k)
			}
		}
		if v.GetStructValue() != nil {
			removeCredentialFieldsWithMaskString(connector, defId, v.GetStructValue(), fmt.Sprintf("%s.", key))
		}

	}
}
func keepCredentialFieldsWithMaskString(connector componentBase.IConnector, defId string, config *structpb.Struct, prefix string) {

	for k, v := range config.GetFields() {
		key := prefix + k
		if !connector.IsCredentialField(defId, key) {
			delete(config.GetFields(), k)
		}
		if v.GetStructValue() != nil {
			keepCredentialFieldsWithMaskString(connector, defId, v.GetStructValue(), fmt.Sprintf("%s.", key))
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
