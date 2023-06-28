package utils

import (
	"context"
	"strings"
	"time"

	"google.golang.org/grpc/metadata"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
	"github.com/instill-ai/pipeline-backend/internal/resource"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"

	mgmtPB "github.com/instill-ai/protogen-go/base/mgmt/v1alpha"
	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
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
	uid, _ := resource.GetPermalinkUID(permalink)
	ctx = metadata.AppendToOutgoingContext(ctx, "Jwt-Sub", uid)
	return ctx
}

func GetResourceFromRecipe(recipe *datamodel.Recipe, t connectorPB.ConnectorType) []string {
	resources := []string{}
	for _, component := range recipe.Components {
		if component.Type == t.String() {
			resources = append(resources, component.ResourceName)
		}
	}
	return resources
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

func NewDataPoint(
	ownerUUID string,
	pipelineRunID string,
	pipeline *datamodel.Pipeline,
	startTime time.Time,
) *write.Point {
	return influxdb2.NewPoint(
		"pipeline.trigger",
		map[string]string{
			"pipeline_mode": pipelinePB.Pipeline_Mode(pipeline.Mode).String(),
		},
		map[string]interface{}{
			"owner_uid":          ownerUUID,
			"pipeline_name":        pipeline.ID,
			"pipeline_permalink":       pipeline.UID.String(),
			"pipeline_trigger_id": pipelineRunID,
			"trigger_time":       startTime.Format(time.RFC3339Nano),
		},
		time.Now(),
	)
}
