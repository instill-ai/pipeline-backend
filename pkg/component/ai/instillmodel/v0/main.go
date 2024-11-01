//go:generate compogen readme ./config ./README.mdx
package instillmodel

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	_ "embed"

	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util"

	modelPB "github.com/instill-ai/protogen-go/model/model/v1alpha"
	pb "github.com/instill-ai/protogen-go/vdp/pipeline/v1beta"
)

const (
	taskEmbedding            string = "TASK_EMBEDDING"
	taskChat                 string = "TASK_CHAT"
	taskCompletion           string = "TASK_COMPLETION"
	taskTextToImage          string = "TASK_TEXT_TO_IMAGE"
	taskClassification       string = "TASK_CLASSIFICATION"
	taskDetection            string = "TASK_DETECTION"
	taskKeyPoint             string = "TASK_KEYPOINT"
	taskOCR                  string = "TASK_OCR"
	taskSemanticSegmentation string = "TASK_SEMANTIC_SEGMENTATION"
	taskInstanceSegmentation string = "TASK_INSTANCE_SEGMENTATION"
)

var (
	//go:embed config/definition.json
	definitionJSON []byte
	//go:embed config/tasks.json
	tasksJSON []byte
	once      sync.Once
	comp      *component
)

type component struct {
	base.Component
}

type execution struct {
	base.ComponentExecution

	execute    func(context.Context, *base.Job) error
	client     modelPB.ModelPublicServiceClient
	connection Connection
}

// Connection is the interface for the gRPC connection.
type Connection interface {
	Close() error
}

// Init initializes the component with the definition and tasks.
func Init(bc base.Component) *component {
	once.Do(func() {
		comp = &component{Component: bc}
		err := comp.LoadDefinition(definitionJSON, nil, tasksJSON, nil)
		if err != nil {
			panic(err)
		}
	})
	return comp
}

// CreateExecution initializes a component executor that can be used in a
// pipeline trigger.
func (c *component) CreateExecution(x base.ComponentExecution) (base.IExecution, error) {
	e := &execution{ComponentExecution: x}

	client, connection := initModelPublicServiceClient(getModelServerURL(e.SystemVariables))

	e.client = client
	e.connection = connection

	switch x.Task {
	case taskEmbedding, taskChat, taskCompletion, taskTextToImage, taskClassification, taskDetection, taskKeyPoint, taskOCR, taskSemanticSegmentation, taskInstanceSegmentation:
		e.execute = e.trigger
	default:
		return nil, fmt.Errorf("task %s not supported", x.Task)
	}

	return e, nil
}

// Execute runs the component with the given jobs.
func (e *execution) Execute(ctx context.Context, jobs []*base.Job) error {
	defer e.connection.Close()
	return base.ConcurrentExecutor(ctx, jobs, e.execute)
}

func (c *component) Test(sysVars map[string]any, setup *structpb.Struct) error {
	gRPCCLient, gRPCCLientConn := initModelPublicServiceClient(getModelServerURL(sysVars))
	if gRPCCLientConn != nil {
		defer gRPCCLientConn.Close()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ctx = metadata.NewOutgoingContext(ctx, getRequestMetadata(sysVars))
	_, err := gRPCCLient.ListModels(ctx, &modelPB.ListModelsRequest{})
	if err != nil {
		return err
	}

	return nil
}

// Generate the `model_name` enum based on the task.
func (c *component) GetDefinition(sysVars map[string]any, compConfig *base.ComponentConfig) (*pb.ComponentDefinition, error) {

	oriDef, err := c.Component.GetDefinition(nil, nil)
	if err != nil {
		return nil, err
	}
	if sysVars == nil && compConfig == nil {
		return oriDef, nil
	}
	def := proto.Clone(oriDef).(*pb.ComponentDefinition)

	if getModelServerURL(sysVars) == "" {
		return def, nil
	}

	gRPCCLient, gRPCCLientConn := initModelPublicServiceClient(getModelServerURL(sysVars))
	if gRPCCLientConn != nil {
		defer gRPCCLientConn.Close()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ctx = metadata.NewOutgoingContext(ctx, getRequestMetadata(sysVars))

	pageToken := ""
	pageSize := int32(100)
	modelNameMap := map[string]*structpb.ListValue{}
	for {
		resp, err := gRPCCLient.ListModels(ctx, &modelPB.ListModelsRequest{PageToken: &pageToken, PageSize: &pageSize, View: modelPB.View_VIEW_BASIC.Enum()})
		if err != nil {
			return def, nil
		}
		for _, m := range resp.Models {
			if _, ok := modelNameMap[m.Task.String()]; !ok {
				modelNameMap[m.Task.String()] = &structpb.ListValue{}
			}
			namePaths := strings.Split(m.Name, "/")
			for _, v := range m.Versions {
				modelNameMap[m.Task.String()].Values = append(modelNameMap[m.Task.String()].Values, structpb.NewStringValue(fmt.Sprintf("%s/%s/%s", namePaths[1], namePaths[3], v)))
			}
		}

		pageToken = resp.NextPageToken
		if pageToken == "" {
			break
		}
	}

	for _, sch := range def.Spec.ComponentSpecification.Fields["oneOf"].GetListValue().Values {
		task := sch.GetStructValue().Fields["properties"].GetStructValue().Fields["task"].GetStructValue().Fields["const"].GetStringValue()
		if _, ok := modelNameMap[task]; ok {
			addModelEnum(sch.GetStructValue().Fields, modelNameMap[task])
		}

	}

	return def, nil
}

func getModelServerURL(vars map[string]any) string {
	if v, ok := vars["__MODEL_BACKEND"]; ok {
		return v.(string)
	}
	return ""
}

func getRequestMetadata(vars map[string]any) metadata.MD {
	md := metadata.Pairs(
		"Authorization", util.GetHeaderAuthorization(vars),
		"Instill-User-Uid", util.GetInstillUserUID(vars),
		"Instill-Auth-Type", "user",
	)

	if requester := util.GetInstillRequesterUID(vars); requester != "" {
		md.Set("Instill-Requester-Uid", requester)
	}

	return md
}

func addModelEnum(compSpec map[string]*structpb.Value, modelName *structpb.ListValue) {
	if compSpec == nil {
		return
	}
	for key, sch := range compSpec {
		if key == "model-name" {
			sch.GetStructValue().Fields["enum"] = structpb.NewListValue(modelName)
		}

		if sch.GetStructValue() != nil {
			addModelEnum(sch.GetStructValue().Fields, modelName)
		}
		if sch.GetListValue() != nil {
			for _, v := range sch.GetListValue().Values {
				if v.GetStructValue() != nil {
					addModelEnum(v.GetStructValue().Fields, modelName)
				}
			}
		}
	}
}

type triggerInfo struct {
	nsID    string
	modelID string
	version string
}

func getTriggerInfo(input *structpb.Struct) (*triggerInfo, error) {
	if input == nil {
		return nil, fmt.Errorf("input is nil")
	}
	data, ok := input.Fields["data"]
	if !ok {
		return nil, fmt.Errorf("data field not found")
	}
	model, ok := data.GetStructValue().Fields["model"]

	if !ok {
		return nil, fmt.Errorf("model field not found")
	}
	modelNameSplits := strings.Split(model.GetStringValue(), "/")
	if len(modelNameSplits) != 3 {
		return nil, fmt.Errorf("model name should be in the format of <namespace>/<model>/<version>")
	}
	return &triggerInfo{
		nsID:    modelNameSplits[0],
		modelID: modelNameSplits[1],
		version: modelNameSplits[2],
	}, nil
}
