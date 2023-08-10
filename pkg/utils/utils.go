package utils

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/influxdata/influxdb-client-go/v2/api/write"
	"google.golang.org/grpc/metadata"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"

	"github.com/instill-ai/pipeline-backend/internal/resource"

	mgmtPB "github.com/instill-ai/protogen-go/base/mgmt/v1alpha"
)

func GenOwnerPermalink(owner *mgmtPB.User) string {
	return "users/" + owner.GetUid()
}

func InjectOwnerToContext(ctx context.Context, owner *mgmtPB.User) context.Context {
	ctx = metadata.AppendToOutgoingContext(ctx, "Jwt-Sub", owner.GetUid())
	return ctx
}
func InjectOwnerToContextWithOwnerPermalink(ctx context.Context, permalink string) context.Context {
	uid, _ := resource.GetPermalinkUID(permalink)
	ctx = metadata.AppendToOutgoingContext(ctx, "Jwt-Sub", uid)
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
