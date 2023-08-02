package utils

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/influxdata/influxdb-client-go/v2/api/write"
	"google.golang.org/grpc/metadata"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"

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

func LoadPipelineUnstructuredData(unstructuredData []*pipelinePB.PipelineDataPayload_UnstructuredData) ([][]byte, error) {
	var data [][]byte
	for idx := range unstructuredData {
		switch unstructuredData[idx].UnstructuredData.(type) {
		case *pipelinePB.PipelineDataPayload_UnstructuredData_Blob:
			data = append(data, unstructuredData[idx].GetBlob())
		case *pipelinePB.PipelineDataPayload_UnstructuredData_Url:
			url := unstructuredData[idx].GetUrl()
			response, err := http.Get(url)
			if err != nil {

				return nil, fmt.Errorf("unable to download url at %v", url)
			}
			defer response.Body.Close()

			buff := new(bytes.Buffer) // pointer
			_, err = buff.ReadFrom(response.Body)
			if err != nil {
				return nil, fmt.Errorf("unable to read content body from data at %v", url)
			}
			data = append(data, buff.Bytes())
		}
	}
	return data, nil
}

func DumpPipelineUnstructuredData(data [][]byte) []*pipelinePB.PipelineDataPayload_UnstructuredData {
	unstructuredData := []*pipelinePB.PipelineDataPayload_UnstructuredData{}
	for idx := range data {
		unstructuredData = append(unstructuredData, &pipelinePB.PipelineDataPayload_UnstructuredData{
			UnstructuredData: &pipelinePB.PipelineDataPayload_UnstructuredData_Blob{
				Blob: data[idx],
			},
		})
	}
	return unstructuredData
}
