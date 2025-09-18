package instillmodel

import (
	"context"
	"testing"

	"google.golang.org/protobuf/types/known/structpb"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func TestComponent_GetDefinition(t *testing.T) {
	c := qt.New(t)

	// Create component
	comp := Init(base.Component{})

	// Test case 1: Get definition without system variables
	t.Run("get definition without system variables", func(t *testing.T) {
		def, err := comp.GetDefinition(nil, nil)

		c.Assert(err, qt.IsNil)
		c.Assert(def, qt.Not(qt.IsNil))
		c.Assert(def.Id, qt.Equals, "instill-model")
		c.Assert(def.Spec, qt.Not(qt.IsNil))
	})

	// Test case 2: Get definition with empty system variables
	t.Run("get definition with empty system variables", func(t *testing.T) {
		sysVars := map[string]any{}

		def, err := comp.GetDefinition(sysVars, nil)

		c.Assert(err, qt.IsNil)
		c.Assert(def, qt.Not(qt.IsNil))
	})

	// Test case 3: Get definition with system variables but no model backend
	t.Run("get definition with system variables but no model backend", func(t *testing.T) {
		sysVars := map[string]any{
			"__PIPELINE_USER_UID": "test-user",
		}

		def, err := comp.GetDefinition(sysVars, nil)

		c.Assert(err, qt.IsNil)
		c.Assert(def, qt.Not(qt.IsNil))
	})
}

func TestComponent_Test(t *testing.T) {
	c := qt.New(t)

	// Create component
	comp := Init(base.Component{})

	// Test case 1: Test with empty system variables
	t.Run("test with empty system variables", func(t *testing.T) {
		sysVars := map[string]any{}

		err := comp.Test(sysVars, nil)

		// Should fail because no model backend URL is provided
		c.Assert(err, qt.Not(qt.IsNil))
	})

	// Test case 2: Test with invalid model backend URL
	t.Run("test with invalid model backend URL", func(t *testing.T) {
		sysVars := map[string]any{
			"__MODEL_BACKEND": "invalid-url",
		}

		err := comp.Test(sysVars, nil)

		// Should fail because the URL is invalid
		c.Assert(err, qt.Not(qt.IsNil))
	})
}

func TestComponent_CreateExecution(t *testing.T) {
	c := qt.New(t)

	// Create component
	comp := Init(base.Component{})

	// Test case 1: Create execution successfully
	t.Run("create execution successfully", func(t *testing.T) {
		baseExec := base.ComponentExecution{
			Task: "TASK_TEXT_GENERATION",
			SystemVariables: map[string]any{
				"__PIPELINE_USER_UID": "test-user",
			},
		}

		exec, err := comp.CreateExecution(baseExec)

		c.Assert(err, qt.IsNil)
		c.Assert(exec, qt.Not(qt.IsNil))

		// Verify it's the correct type
		instillExec, ok := exec.(*execution)
		c.Assert(ok, qt.IsTrue)
		c.Assert(instillExec.Task, qt.Equals, "TASK_TEXT_GENERATION")
	})
}

func TestExecution_Execute(t *testing.T) {
	c := qt.New(t)

	// Create component and execution
	comp := Init(base.Component{})
	baseExec := base.ComponentExecution{
		Task: "TASK_TEXT_GENERATION",
		SystemVariables: map[string]any{
			"__PIPELINE_USER_UID": "test-user",
		},
	}

	exec, err := comp.CreateExecution(baseExec)
	c.Assert(err, qt.IsNil)

	// Test case 1: Execute with empty jobs
	t.Run("execute with empty jobs", func(t *testing.T) {
		jobs := []*base.Job{}

		err := exec.Execute(context.Background(), jobs)

		c.Assert(err, qt.ErrorMatches, "invalid input")
	})

	// Test case 2: Execute with invalid task
	t.Run("execute with invalid task", func(t *testing.T) {
		// Create execution with invalid task
		invalidBaseExec := base.ComponentExecution{
			Task: "INVALID_TASK",
			SystemVariables: map[string]any{
				"__PIPELINE_USER_UID": "test-user",
			},
		}

		invalidExec, err := comp.CreateExecution(invalidBaseExec)
		c.Assert(err, qt.IsNil)

		// Create mock job
		input, _ := structpb.NewStruct(map[string]any{
			"model-name": "test-namespace/test-model/v1.0",
			"prompt":     "test prompt",
		})

		mockJob := &base.Job{
			Input:  &mockInputReader{data: input},
			Output: &mockOutputWriter{},
			Error:  &mockErrorHandler{},
		}

		jobs := []*base.Job{mockJob}

		err = invalidExec.Execute(context.Background(), jobs)

		c.Assert(err, qt.ErrorMatches, "unsupported task.*")
	})
}

// Mock implementations for testing
type mockInputReader struct {
	data *structpb.Struct
	err  error
}

func (m *mockInputReader) ReadData(ctx context.Context, input any) error {
	return m.err
}

func (m *mockInputReader) Read(ctx context.Context) (*structpb.Struct, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.data, nil
}

type mockOutputWriter struct {
	data *structpb.Struct
	err  error
}

func (m *mockOutputWriter) WriteData(ctx context.Context, output any) error {
	return m.err
}

func (m *mockOutputWriter) Write(ctx context.Context, data *structpb.Struct) error {
	if m.err != nil {
		return m.err
	}
	m.data = data
	return nil
}

type mockErrorHandler struct {
	err error
}

func (m *mockErrorHandler) Error(ctx context.Context, err error) {
	m.err = err
}

// Helper functions for testing
func TestGetModelServerURL(t *testing.T) {
	c := qt.New(t)

	// Test case 1: With model backend variable
	t.Run("with model backend variable", func(t *testing.T) {
		vars := map[string]any{
			"__MODEL_BACKEND": "https://model-backend.example.com",
		}

		url := getModelServerURL(vars)
		c.Assert(url, qt.Equals, "https://model-backend.example.com")
	})

	// Test case 2: Without model backend variable
	t.Run("without model backend variable", func(t *testing.T) {
		vars := map[string]any{
			"other_var": "value",
		}

		url := getModelServerURL(vars)
		c.Assert(url, qt.Equals, "")
	})

	// Test case 3: Empty variables
	t.Run("empty variables", func(t *testing.T) {
		vars := map[string]any{}

		url := getModelServerURL(vars)
		c.Assert(url, qt.Equals, "")
	})
}

func TestGetRequestMetadata(t *testing.T) {
	c := qt.New(t)

	// Test case 1: With all required variables
	t.Run("with all required variables", func(t *testing.T) {
		vars := map[string]any{
			"__PIPELINE_USER_UID":      "test-user",
			"__PIPELINE_REQUESTER_UID": "test-requester",
		}

		md := getRequestMetadata(vars)

		c.Assert(md, qt.Not(qt.IsNil))
		c.Assert(len(md.Get("Instill-User-Uid")), qt.Equals, 1)
		c.Assert(md.Get("Instill-User-Uid")[0], qt.Equals, "test-user")
		c.Assert(len(md.Get("Instill-Requester-Uid")), qt.Equals, 1)
		c.Assert(md.Get("Instill-Requester-Uid")[0], qt.Equals, "test-requester")
		c.Assert(len(md.Get("Instill-Auth-Type")), qt.Equals, 1)
		c.Assert(md.Get("Instill-Auth-Type")[0], qt.Equals, "user")
	})

	// Test case 2: Without requester UID
	t.Run("without requester uid", func(t *testing.T) {
		vars := map[string]any{
			"__PIPELINE_USER_UID": "test-user",
		}

		md := getRequestMetadata(vars)

		c.Assert(md, qt.Not(qt.IsNil))
		c.Assert(len(md.Get("Instill-User-Uid")), qt.Equals, 1)
		c.Assert(md.Get("Instill-User-Uid")[0], qt.Equals, "test-user")
		c.Assert(len(md.Get("Instill-Requester-Uid")), qt.Equals, 0)
	})
}
