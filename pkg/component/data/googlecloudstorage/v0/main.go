//go:generate compogen readme ./config ./README.mdx
package googlecloudstorage

import (
	"context"
	"fmt"
	"sync"

	_ "embed"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

const (
	taskUpload       = "TASK_UPLOAD"
	taskReadObjects  = "TASK_READ_OBJECTS"
	taskCreateBucket = "TASK_CREATE_BUCKET"
)

//go:embed config/definition.yaml
var definitionYAML []byte

//go:embed config/setup.yaml
var setupYAML []byte

//go:embed config/tasks.yaml
var tasksYAML []byte

var once sync.Once
var comp *component

type component struct {
	base.Component
}

type execution struct {
	base.ComponentExecution
}

func Init(bc base.Component) *component {
	once.Do(func() {
		comp = &component{Component: bc}
		err := comp.LoadDefinition(definitionYAML, setupYAML, tasksYAML, nil, nil)
		if err != nil {
			panic(err)
		}
	})
	return comp
}

func (c *component) CreateExecution(x base.ComponentExecution) (base.IExecution, error) {
	return &execution{
		ComponentExecution: x,
	}, nil
}

func NewClient(jsonKey string) (*storage.Client, error) {
	return storage.NewClient(context.Background(), option.WithCredentialsJSON([]byte(jsonKey)))
}

func getJSONKey(setup *structpb.Struct) string {
	return setup.GetFields()["json-key"].GetStringValue()
}

func (e *execution) Execute(ctx context.Context, jobs []*base.Job) error {

	client, err := NewClient(getJSONKey(e.Setup))
	if err != nil || client == nil {
		return fmt.Errorf("error creating GCS client: %v", err)
	}
	defer client.Close()
	for _, job := range jobs {
		input, err := job.Input.Read(ctx)
		if err != nil {
			job.Error.Error(ctx, err)
			continue
		}

		var output *structpb.Struct
		bucketName := input.GetFields()["bucket-name"].GetStringValue()
		switch e.Task {
		case taskUpload, "":
			objectName := input.GetFields()["object-name"].GetStringValue()
			data := input.GetFields()["data"].GetStringValue()
			err = uploadToGCS(client, bucketName, objectName, data)
			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}

			gsutilURI := fmt.Sprintf("gs://%s/%s", bucketName, objectName)
			authenticatedURL := fmt.Sprintf("https://storage.cloud.google.com/%s/%s?authuser=1", bucketName, objectName)
			publicURL := ""

			// Check whether the object is public or not
			publicAccess, err := isObjectPublic(client, bucketName, objectName)
			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}
			if publicAccess {
				publicURL = fmt.Sprintf("https://storage.googleapis.com/%s/%s", bucketName, objectName)
			}

			output = &structpb.Struct{Fields: map[string]*structpb.Value{
				"authenticated-url": {Kind: &structpb.Value_StringValue{StringValue: authenticatedURL}},
				"gsutil-uri":        {Kind: &structpb.Value_StringValue{StringValue: gsutilURI}},
				"public-url":        {Kind: &structpb.Value_StringValue{StringValue: publicURL}},
				"public-access":     {Kind: &structpb.Value_BoolValue{BoolValue: publicAccess}},
				"status":            {Kind: &structpb.Value_StringValue{StringValue: "success"}}}}

		case taskReadObjects:
			inputStruct := ReadInput{
				BucketName: bucketName,
			}

			err := base.ConvertFromStructpb(input, &inputStruct)
			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}
			outputStruct, err := readObjects(inputStruct, client, ctx)
			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}
			output, err = base.ConvertToStructpb(outputStruct)

			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}

		case taskCreateBucket:
			inputStruct := CreateBucketInput{}
			err := base.ConvertFromStructpb(input, &inputStruct)
			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}

			outputStruct, err := createBucket(inputStruct, client, ctx)
			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}

			output, err = base.ConvertToStructpb(outputStruct)
			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}
		}
		err = job.Output.Write(ctx, output)
		if err != nil {
			job.Error.Error(ctx, err)
			continue
		}
	}
	return nil
}

func (c *component) Test(sysVars map[string]any, setup *structpb.Struct) error {

	client, err := NewClient(getJSONKey(setup))
	if err != nil {
		return fmt.Errorf("error creating GCS client: %v", err)
	}
	if client == nil {
		return fmt.Errorf("GCS client is nil")
	}
	defer client.Close()
	return nil
}
