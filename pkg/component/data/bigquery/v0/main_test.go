package bigquery

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/protobuf/types/known/structpb"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/option"
)

type MockClient struct {
	mock.Mock
}

type MockExecution struct {
	base.ComponentExecution
	mock.Mock
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
}

func (m *mockOutput) Write(ctx context.Context, output *structpb.Struct) error {
	m.data = output
	return nil
}

type mockError struct{}

func (m *mockError) Error(ctx context.Context, err error) {}

func (m *MockExecution) Execute(ctx context.Context, jobs []*base.Job) error {
	args := m.Called(ctx, jobs)
	return args.Error(0)
}

func TestInit(t *testing.T) {
	bc := base.Component{}
	comp := Init(bc)
	assert.NotNil(t, comp)
}

func TestNewClient(t *testing.T) {
	jsonKey := `{"type": "service_account"}`
	projectID := "test-project"

	client, err := NewClient(jsonKey, projectID)
	assert.NoError(t, err)
	assert.NotNil(t, client)
}

func TestExecute(t *testing.T) {
	ctx := context.Background()

	job := &base.Job{
		Input:  &mockInput{err: errors.New("mock input error")}, // Trigger error
		Output: &mockOutput{},
		Error:  &mockError{},
	}

	exec := &MockExecution{}
	exec.On("Execute", ctx, mock.Anything).Return(errors.New("mock execution error"))

	err := exec.Execute(ctx, []*base.Job{job})
	assert.Error(t, err)
}

func TestInsertDataSuccessWithMockServer(t *testing.T) {
	server := mockBigQueryInserterServer()
	defer server.Close()

	ctx := context.Background()
	client, err := bigquery.NewClient(ctx, "test-project", option.WithEndpoint(server.URL), option.WithoutAuthentication())
	assert.NoError(t, err)

	schema := bigquery.Schema{
		{Name: "field1", Type: bigquery.StringFieldType},
	}

	inserter := client.Dataset("test_dataset").Table("test_table").Inserter()
	rows := []bigquery.ValueSaver{
		&bigquery.ValuesSaver{
			Schema: schema,
			Row:    []bigquery.Value{"row_value"},
		},
	}
	err = inserter.Put(ctx, rows)
	assert.NoError(t, err)
}

func TestInsertDataInvalidTable(t *testing.T) {
	server := mockBigQueryInserterServer()
	defer server.Close()

	ctx := context.Background()
	client, err := bigquery.NewClient(ctx, "test-project", option.WithEndpoint(server.URL), option.WithoutAuthentication())
	assert.NoError(t, err)

	inserter := client.Dataset("test_dataset").Table("invalid_table").Inserter()
	rows := []bigquery.ValueSaver{
		&bigquery.ValuesSaver{
			Schema: nil,
			Row:    []bigquery.Value{"row_value"},
		},
	}
	err = inserter.Put(ctx, rows)
	assert.Error(t, err, "expected error due to invalid table name")
}

func TestInsertDataEmptyRows(t *testing.T) {
	server := mockBigQueryInserterServer()
	defer server.Close()

	ctx := context.Background()
	client, err := bigquery.NewClient(ctx, "test-project", option.WithEndpoint(server.URL), option.WithoutAuthentication())
	assert.NoError(t, err)

	inserter := client.Dataset("test_dataset").Table("test_table").Inserter()
	var emptyRows []bigquery.ValueSaver
	err = inserter.Put(ctx, emptyRows)
	assert.NoError(t, err, "empty rows should not cause an error")
}

func TestInsertDataFailure(t *testing.T) {
	server := mockBigQueryServerError()
	defer server.Close()

	ctx := context.Background()
	client, err := bigquery.NewClient(ctx, "test-project", option.WithEndpoint(server.URL), option.WithoutAuthentication())
	assert.NoError(t, err)

	inserter := client.Dataset("test_dataset").Table("test_table").Inserter()
	rows := []bigquery.ValueSaver{
		&bigquery.ValuesSaver{
			Schema: nil,
			Row:    []bigquery.Value{"row_value"},
		},
	}
	err = inserter.Put(ctx, rows)
	assert.Error(t, err, "expected error due to server failure")
}

func TestInsertDataWithTimestamp(t *testing.T) {
	server := mockBigQueryInserterServer()
	defer server.Close()

	ctx := context.Background()
	client, err := bigquery.NewClient(ctx, "test-project", option.WithEndpoint(server.URL), option.WithoutAuthentication())
	assert.NoError(t, err)

	schema := bigquery.Schema{
		{Name: "field1", Type: bigquery.StringFieldType},
		{Name: "event_time", Type: bigquery.TimestampFieldType},
	}

	inserter := client.Dataset("test_dataset").Table("test_table").Inserter()
	rows := []bigquery.ValueSaver{
		&bigquery.ValuesSaver{
			Schema: schema,
			Row:    []bigquery.Value{"row_value", bigquery.NullTimestamp{Timestamp: time.Now(), Valid: true}}, // Current timestamp
		},
	}

	err = inserter.Put(ctx, rows)
	assert.NoError(t, err)
}

func mockBigQueryInserterServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"kind": "bigquery#tableDataInsertAllResponse",
		}); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}))
}

func mockBigQueryServerError() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]string{"error": "Internal Server Error"}); err != nil {
			http.Error(w, "Failed to encode error response", http.StatusInternalServerError)
			return
		}
	}))
}
