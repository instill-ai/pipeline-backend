package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"

	"github.com/instill-ai/pipeline-backend/internal/resource"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"

	mgmtPB "github.com/instill-ai/protogen-go/core/mgmt/v1alpha"
	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1alpha"
)

func GenOwnerPermalink(owner *mgmtPB.User) string {
	return "users/" + owner.GetUid()
}

func InjectOwnerToContext(ctx context.Context, owner *mgmtPB.User) context.Context {
	ctx = metadata.AppendToOutgoingContext(ctx, "Jwt-Sub", owner.GetUid())
	return ctx
}
func InjectOwnerToContextWithOwnerPermalink(ctx context.Context, permalink string) context.Context {
	uid, _ := resource.GetRscPermalinkUID(permalink)
	ctx = metadata.AppendToOutgoingContext(ctx, "Jwt-Sub", uid.String())
	return ctx
}
func InjectOwnerToContextWithUserUid(ctx context.Context, userUid uuid.UUID) context.Context {
	ctx = metadata.AppendToOutgoingContext(ctx, "Jwt-Sub", userUid.String())
	return ctx
}

const (
	CreateEvent     string = "Create"
	UpdateEvent     string = "Update"
	DeleteEvent     string = "Delete"
	ActivateEvent   string = "Activate"
	DeactivateEvent string = "Deactivate"
	TriggerEvent    string = "Trigger"
)

func IsAuditEvent(eventName string) bool {
	return strings.HasPrefix(eventName, CreateEvent) ||
		strings.HasPrefix(eventName, UpdateEvent) ||
		strings.HasPrefix(eventName, DeleteEvent) ||
		strings.HasPrefix(eventName, ActivateEvent) ||
		strings.HasPrefix(eventName, DeactivateEvent) ||
		strings.HasPrefix(eventName, TriggerEvent)
}

func IsBillableEvent(eventName string) bool {
	return strings.HasPrefix(eventName, TriggerEvent)
}

type UsageMetricData struct {
	OwnerUID            string
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

func NewDataPoint(data UsageMetricData) *write.Point {
	return influxdb2.NewPoint(
		"pipeline.trigger",
		map[string]string{
			"status":       data.Status.String(),
			"trigger_mode": data.TriggerMode.String(),
		},
		map[string]interface{}{
			"owner_uid":             data.OwnerUID,
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

// we only support the simple case for now
//
//	"dependencies": {
//		"texts": "[*c1.texts, *c2.texts]",
//		"images": "[*c1.images, *c2.images]",
//		"structured_data": "{**c1.structured_data, **c2.structured_data}",
//		"metadata": "{**c1.metadata, **c2.metadata}"
//	}
//
//	"dependencies": {
//		"texts": "[*c2.texts]",
//		"images": "[*c1.images]",
//		"structured_data": "{**c1.structured_data}",
//		"metadata": "{**c1.metadata, **c2.metadata}"
//	}
func ParseDependency(dep map[string]string) ([]string, map[string][]string, error) {
	parentMap := map[string]bool{}
	depMap := map[string][]string{}
	for _, key := range []string{"images", "audios", "texts", "structured_data", "metadata"} {
		depMap[key] = []string{}

		if str, ok := dep[key]; ok {
			str = strings.ReplaceAll(str, " ", "")
			str = str[1 : len(str)-1]
			if len(str) > 0 {
				items := strings.Split(str, ",")
				for idx := range items {

					name := strings.Split(items[idx], ".")[0]
					depKey := strings.Split(items[idx], ".")[1]
					name = strings.ReplaceAll(name, "*", "")
					parentMap[name] = true
					depMap[key] = append(depMap[key], fmt.Sprintf("%s.%s", name, depKey))
				}
			}

		}
	}
	parent := []string{}
	for k := range parentMap {
		parent = append(parent, k)
	}
	return parent, depMap, nil
}

func GenerateTraces(comps []*datamodel.Component, memory []map[string]interface{}, computeTime map[string]float32, batchSize int) (map[string]*pipelinePB.Trace, error) {
	trace := map[string]*pipelinePB.Trace{}
	for compIdx := range comps {
		inputs := []*structpb.Struct{}
		outputs := []*structpb.Struct{}

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

		trace[comps[compIdx].Id] = &pipelinePB.Trace{
			Success:              true,
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
	return strings.HasPrefix(resourceName, "connector-resources/")
}
func IsConnectorWithNamespace(resourceName string) bool {
	return len(strings.Split(resourceName, "/")) > 3 && strings.Split(resourceName, "/")[2] == "connector-resources"
}

func IsConnectorDefinition(resourceName string) bool {
	return strings.HasPrefix(resourceName, "connector-definitions/")
}

func IsOperatorDefinition(resourceName string) bool {
	return strings.HasPrefix(resourceName, "operator-definitions/")
}
