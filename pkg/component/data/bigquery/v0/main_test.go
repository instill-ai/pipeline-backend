package bigquery

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

type MockClient struct {
	mock.Mock
}

type MockExecution struct {
	execution
}

type mockInput struct {
	err error
}

func (m *mockInput) Read(ctx context.Context) (*structpb.Struct, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &structpb.Struct{Fields: map[string]*structpb.Value{"data": structpb.NewStringValue("mock data")}}, nil
}

type mockOutput struct {
	data *structpb.Struct
	err  error
}

func (m *mockOutput) Write(ctx context.Context, output *structpb.Struct) error {
	if m.err != nil {
		return m.err
	}
	m.data = output
	return nil
}

type mockError struct{}

func (m *mockError) Error(ctx context.Context, err error) {

}

func TestInit(t *testing.T) {
	bc := base.Component{}
	comp := Init(bc)
	assert.NotNil(t, comp)
}

func TestNewClient(t *testing.T) {

	jsonKey := `{"type": "service_account", "project_id": "test-project", "private_key": "-----BEGIN PRIVATE KEY-----\nYOUR_PRIVATE_KEY\n-----END PRIVATE KEY-----\n", "client_email": "test-email@project.iam.gserviceaccount.com"}`
	projectID := "test-project"

	client, err := NewClient(jsonKey, projectID)
	assert.NoError(t, err)
	assert.NotNil(t, client)
}

func TestExecute(t *testing.T) {
	ctx := context.Background()

	job := &base.Job{
		Input:  &mockInput{},
		Output: &mockOutput{},
		Error:  &mockError{},
	}

	exec := &MockExecution{}
	err := exec.Execute(ctx, []*base.Job{job})
	assert.NoError(t, err)

	job.Input = &mockInput{err: errors.New("input error")}
	err = exec.Execute(ctx, []*base.Job{job})
	assert.Error(t, err)

	err = exec.Execute(ctx, nil)
	assert.Error(t, err, "Expected error for nil jobs")

	job.Output = &mockOutput{err: errors.New("output error")}
	err = exec.Execute(ctx, []*base.Job{job})
	assert.Error(t, err, "Expected error for output write failure")

}

func TestGetDefinition(t *testing.T) {

	compConfig := &base.ComponentConfig{
		Setup: map[string]interface{}{
			"json-key":   `{"type": "service_account", "project_id": "test-project", "private_key": "-----BEGIN PRIVATE KEY-----\nYOUR_PRIVATE_KEY\n-----END PRIVATE KEY-----\n", "client_email": "test-email@project.iam.gserviceaccount.com"}`,
			"project-id": "test-project",
			"dataset-id": "test_dataset",
			"table-name": "test_table",
		},
	}

	comp := Init(base.Component{})

	def, err := comp.GetDefinition(nil, compConfig)
	assert.NoError(t, err)
	assert.NotNil(t, def)

	invalidConfig := &base.ComponentConfig{
		Setup: map[string]interface{}{
			"json-key":   "invalid-json",
			"project-id": "test-project",
		},
	}
	def, err = comp.GetDefinition(nil, invalidConfig)
	assert.Error(t, err, "Expected error for invalid config")
	assert.Nil(t, def, "Expected nil definition for invalid config")
}
